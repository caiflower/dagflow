## Approach

### Current State

```
FlowConverter creates:
  1. Regular subtask for branch node (branchPassthroughProvider)
  2. Dedicated branch subtask via AddBranch (nil provider, Condition closure)

  predecessor → [branch: passthrough] → [branch_branch_0: condition closure] → successors
                     ↑ useless                ↑ inline non-persistable logic
```

### Target State

```
FlowConverter:
  - Does NOT create a subtask for branch nodes
  - Uses the branch node's own protocol provider as ConditionProvider
  - Validates that provider.Execute returns string (ID or taskName)
  - Calls task.AddBranch(predecessor, branch) with that provider

  predecessor → [branch_branch_0: user's protocol as ConditionProvider] → successor_A
                                                                        → successor_B
```

Key design decisions:

1. **Remove `branchPassthroughProvider` entirely** — the engine's `executeBranchCondition` handles execution on any worker
2. **Remove `makeBranchCondition` closure** — branch routing is determined by the user's chosen protocol (e.g., HTTP, gRPC, local)
3. **Branch node's protocol provider → ConditionProvider** — `providerFactory(protocol, config)` creates the provider; it's passed directly to `NewBranch(provider, endNodes)`
4. **Validate return type** — after provider creation, wrap it to assert `Execute` returns `string` (ID or taskName). If not, fail fast during conversion
5. **No breaking change to Flow model** — `FlowNode.Type = "branch"` and `FlowEdge.Expr` remain; edges still carry optional `Expr` for the provider to consume via Config

## Data Flow

```
Flow JSON
  │
  ▼
FlowConverter.FlowToTask()
  │
  ├─ For task nodes: create SubTask with provider
  ├─ For branch nodes: skip subtask creation, collect as pending branches
  │     provider = providerFactory(node.Protocol, node.Config)
  │     validate provider returns string (type assertion or wrapper)
  ├─ For edges: skip edges where source or target is a branch node
  │
  ├─ For each branch node:
  │     find predecessors and successors from edges
  │     for each predecessor:
  │       task.AddBranch(predecessor, NewBranch(
  │         provider,           // user's protocol as ConditionProvider
  │         successorNames      // resolved end nodes
  │       ))
  │
  └─ task.Compile()
```

## Validation

The converter wraps the provider to assert the return type:

```go
// Type-check: provider.Execute must return string (ID or taskName)
testResult, err := provider.Execute(ctx, &executor.TaskData{})
if err != nil {
    return nil, fmt.Errorf("branch provider for %s failed: %w", node.Name, err)
}
if _, ok := testResult.(string); !ok {
    return nil, fmt.Errorf("branch provider for %s must return string (ID or taskName), got %T", node.Name, testResult)
}
```

## Risks

- Some protocols (e.g., remote HTTP) can't be validated at conversion time without a live endpoint — skip validation for non-local providers, defer to runtime
- Edge case: branch node with 0 predecessors → skip
- Edge case: branch node with < 2 successors → skip
- `FlowEdge.Expr` is preserved in node config for the provider to use
