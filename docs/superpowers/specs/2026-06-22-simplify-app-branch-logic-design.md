---
comet_change: simplify-app-branch-logic
role: technical-design
canonical_spec: openspec
archived-with: 2026-06-22-simplify-app-branch-logic
status: final
---

# Application-Layer Branch Logic Simplification

## Overview

Align `FlowConverter` with taskx engine's first-class branch subtask model. Remove the redundant `branchPassthroughProvider` and `makeBranchCondition` closure; instead use the branch node's own protocol provider directly as `ConditionProvider`.

## Current vs Target

```
BEFORE:
  predecessor → [branch: passthrough] → [branch_x_0: closure] → successor_A
                                                                → successor_B
                   ↑ dummy executor              ↑ non-persistable closure

AFTER:
  predecessor → [branch_x_0: user's protocol] → successor_A
                                                → successor_B
                   ↑ persistable ConditionProvider
```

## Implementation

### 1. Remove dead code
- Delete `branchPassthroughProvider` struct and its `Execute`/`Protocol` methods
- Delete `makeBranchCondition` function
- Delete `matchExpr` function (or keep if used elsewhere — verify)

### 2. Refactor `FlowToTask` branch node handling

```go
// For branch nodes: skip subtask creation, collect as pending
pendingBranches := []branchInfo{}
for _, n := range nodes {
    if n.Type == "branch" {
        provider, err := providerFactory(n.Protocol, n.Config)
        if err != nil {
            return nil, fmt.Errorf("branch node %s: %w", n.Name, err)
        }
        // Validate return type (local providers only)
        if isLocalProvider(provider) {
            if err := validateBranchProvider(provider); err != nil {
                return nil, fmt.Errorf("branch node %s: %w", n.Name, err)
            }
        }
        pendingBranches = append(pendingBranches, branchInfo{
            node:     n,
            provider: provider,
        })
        continue
    }
    // ... normal subtask creation ...
    subtaskMap[n.ID] = st
}

// Wire branch nodes
for _, bi := range pendingBranches {
    predIDs := findPredecessors(bi.node.ID, edges)
    succNames := findSuccessorNames(bi.node.ID, edges, subtaskMap)
    if len(predIDs) == 0 || len(succNames) < 2 {
        continue
    }
    for _, predID := range predIDs {
        predSt := subtaskMap[predID]
        endNodes := make(map[string]bool, len(succNames))
        for _, name := range succNames {
            endNodes[name] = true
        }
        if err := task.AddBranch(predSt, taskx.NewBranch(bi.provider, endNodes)); err != nil {
            return nil, fmt.Errorf("add branch %s: %w", bi.node.Name, err)
        }
    }
}
```

### 3. Edge filtering

Skip edges where source or target is a branch node — those connections are handled by `AddBranch` internally (DataEdge from predecessor, ControlEdge to successors).

### 4. Type validation

```go
func validateBranchProvider(p executor.ExecutorProvider) error {
    result, err := p.Execute(context.Background(), &executor.TaskData{SubTaskId: "validate"})
    if err != nil {
        return fmt.Errorf("branch provider execution failed: %w", err)
    }
    if _, ok := result.(string); !ok {
        return fmt.Errorf("branch provider must return string (ID or taskName), got %T", result)
    }
    return nil
}
```

Remote providers (HTTP, gRPC) skip compile-time validation — their type is checked at runtime by `executeBranchCondition` which already asserts `result.(string)`.

## Edge Cases

| Case | Behavior |
|------|----------|
| Branch node with 0 predecessors | Skip (no routing context) |
| Branch node with < 2 successors | Skip (not a real branch) |
| Multiple predecessors for one branch | Create branch subtask for each predecessor |
| Remote protocol (HTTP/gRPC) | Skip compile-time validation |
| `FlowEdge.Expr` | Preserved in node Config for provider use |

## Testing Strategy

New `flow_converter_test.go`:

1. **Single branch**: start → branch(pathA) → pathA/pathB → end. Verify pathA=Succeeded, pathB=Skipped, end=Succeeded
2. **Nested branch**: start → outer → innerA/innerB → end. Verify correct skip cascade
3. **Type validation**: non-string provider → error at conversion
4. **Skip non-branch**: 0 successors → branch node skipped gracefully
5. **Roundtrip**: compile → serialize task → restore from DB → re-compile → verify branch routing survives

## Files Changed

| File | Change |
|------|--------|
| `internal/converter/flow_converter.go` | Remove `branchPassthroughProvider`, `makeBranchCondition`, `matchExpr`; refactor `FlowToTask` branch handling |
| `internal/converter/flow_converter_test.go` | New: 5 test cases |
| `internal/service/execution_service.go` | Minor: remove unused `NodeType` field if safe |
