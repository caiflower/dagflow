---
comet_change: branch-node-distributed-execution
role: technical-design
canonical_spec: openspec
---

# Branch Node Distributed Execution — Technical Design

## Problem Summary

1. Branch nodes are not first-class DB entities — they exist only as JSON metadata (`BranchConfig`) in the parent subtask's `Settings` field. No state tracking, no retry, no timeout.
2. Branch condition evaluation (`processBranches()`) runs exclusively on the leader node, creating a single-node hotspot under high task throughput.

## Solution

Make branch nodes full DB subtask rows with a built-in `branchExecutor` that runs on any cluster worker, fully leveraging the existing subtask infrastructure.

## Architecture

```
Before (current):                    After (proposed):

  Parent Subtask                       Parent Subtask
       │                                     │
       │ (has BranchConfig                   │ (control edge)
       │  in Settings field)                 ▼
       │                              Branch Subtask [NEW]
       ├──────────┬──────────┐         │ Settings.BranchConfig
       ▼          ▼          ▼         ├──────────┬──────────┐
    Target A  Target B  Target C       ▼          ▼          ▼
                                   Target A  Target B  Target C

  processBranches() runs             branchExecutor runs
  on leader ONLY                     on ANY worker node
```

## Key Design Decisions

### Decision 1: Branch as a Subtask Type

Branch nodes are stored as regular `subtask` rows with `Settings.BranchConfig`. The receiver detects them by checking `Settings.BranchConfig != nil` and routes to the built-in `branchExecutor`.

**Rationale:** Reuses the entire subtask lifecycle (allocation, state machine, retry, timeout, delivery) without any schema changes.

### Decision 2: Built-in branchExecutor

A globally registered `branchExecutor` (protocol `"builtin:branch"`) handles condition evaluation on the assigned worker. It resolves the condition provider from `_branchConditionProviderRegistry`, executes it, and writes the selected key to output.

**Rationale:** Branch execution becomes just another subtask — fully distributable.

### Decision 3: Conditional Edge Activation via Output

After the branch subtask succeeds, the leader reads its output (selected key), skips unselected end nodes via `SkipSubtask()`, and the selected target proceeds naturally through the DAG.

**Rationale:** Fits the existing DAG execution model where subtask completion triggers successor evaluation.

### Decision 4: Backward Compatibility

`initByBean()` checks: if a subtask row has `BranchConfig` in its `Settings`, use the new path. If a parent subtask has legacy `BranchConfig` attached (no dedicated branch subtask row), preserve the old `processBranches()` behavior.

## Implementation Plan by File

### `dag.go`
- `AddBranch()`: create `dagNode` for branch subtask with generated ID `branch_<parentID>_<idx>`, store `BranchConfig` in `Settings`
- Add control edges: `parent → branch` and `branch → each endNode`

### `task.go`
- `toBeans()`: serialize branch subtasks as regular `model.Subtask` rows
- `initByBean()`: detect `Settings.BranchConfig`, restore from global registries
- Backward compat flag for legacy `BranchConfig` on parent subtask

### `compile.go`
- `validateBranches()`: validate branch subtask nodes
- `GetExecutableNodes()`: branch subtask becomes executable when parent completes

### `dispatch.go`
- Deprecate `processBranches()` for new tasks
- `analysisTask()`: branch subtasks flow through normal `runningSubtasks` path
- Post-execution: read branch output, skip unselected end nodes

### `executor.go`
- Register `branchExecutor` under `"builtin:branch"`
- Resolve condition provider, execute, handle errors

### `receiver_internal.go`
- Route subtasks with `Settings.BranchConfig` to `branchExecutor`

## Data Flow (New Task)

```
1. SubmitTask() → toBeans() creates branch subtask rows
2. Leader analysisTask() → branch subtask as pending
3. allocateWorker() → assign to worker W
4. deliverToCluster() → gRPC to worker W
5. Worker receiver → detects BranchConfig → branchExecutor
6. branchExecutor → resolve provider → execute → write output
7. Worker notifies leader → HandleTaskImmediately
8. Leader reads output (selected key) → SkipSubtask(unselected)
9. Leader dispatches selected target
```

## Error Handling

| Condition | Behavior |
|-----------|----------|
| Condition provider not registered on worker | Subtask fails with error |
| Selected key not in end nodes | Subtask fails with error |
| Condition returns non-string | Subtask fails with error |
| Branch subtask execution fails | Retry per configured count/interval |

## Testing Strategy

- **Unit**: branchExecutor error cases, serialization round-trip, DAG node creation
- **Integration**: end-to-end branch task with mock cluster
- **Backward compat**: legacy `BranchConfig` tasks still execute correctly
