## Context

Currently, branch nodes in the taskx DAG framework are not first-class entities. They exist as metadata on a parent subtask:

1. **Branch configuration** is stored in the parent subtask's `Settings` JSON field (`BranchConfig` with `EndNodes` and `ConditionProvider` name).
2. **Branch condition execution** happens in `processBranches()` which runs inside `analysisTask()` on the **leader node only** (gated by `t.Cluster.IsLeader()` in `handleTaskImmediately`).
3. **Branch selection** (skipping unselected targets) also happens on the leader.

This means:
- Branch nodes have no DB row, no state, no retry capability, no timeout.
- Under high task volume, the leader becomes a hotspot for branch condition evaluation.

## Goals / Non-Goals

**Goals:**
- Store branch nodes as full subtask rows in DB with their own ID, state, retry, timeout, worker assignment.
- Move branch condition evaluation from the leader to the assigned worker node, enabling distributed execution.
- Store the selected branch key in the branch subtask's output, so downstream nodes can be conditionally triggered.
- Maintain backward compatibility: existing tasks with `BranchConfig` in `Settings` continue to work.

**Non-Goals:**
- Changing the branch API surface (`AddBranch`, `NewBranch`, `NewBranchFunc`).
- Modifying the gRPC proto or cluster delivery protocol.
- Adding new database tables or columns.
- Dynamic branch targets (end nodes remain static, defined at DAG construction time).

## Decisions

### Decision 1: Branch as a Subtask Type

Branch nodes are stored as regular subtask rows with a `Settings` field containing `BranchConfig`. The receiver detects branch subtasks by checking if `Settings.BranchConfig != nil` and invokes a special branch executor instead of the regular `ExecutorProvider`.

**Rationale:** Reuses the existing subtask infrastructure (worker allocation, state machine, retry, timeout, delivery) without schema changes. The `Settings` JSON column already supports `branch_config`.

**Alternative considered:** A new `branch` DB table. Rejected because it duplicates the subtask table's structure and would require new DAO, new allocation, new delivery paths.

### Decision 2: Branch Executor as a Built-in ExecutorProvider

A built-in `branchExecutor` (registered globally like other providers) handles branch condition evaluation. It:
1. Looks up the `BranchConfig` from the subtask's `Settings`.
2. Resolves the `ConditionProvider` from the global `_branchConditionProviderRegistry` using the stored name.
3. Calls `ConditionProvider.Execute()` to get the selected key.
4. Writes the selected key to the subtask's output.
5. Returns success.

The leader no longer calls `processBranches()` — instead, the DAG compilation adds control edges from the branch subtask to each end node, and the branch subtask's output determines which control edge is "activated."

**Rationale:** Treats branch evaluation as just another subtask execution, fully distributable across the cluster.

**Alternative considered:** Keep `processBranches()` on the leader but have it RPC to workers for condition evaluation. Rejected because it adds complexity without fully eliminating the leader dependency.

### Decision 3: Conditional Edge Activation via Output

After the branch subtask succeeds, the receiver (or post-execution hook) reads the branch output (selected key) and:
- Marks the selected end node's control dependency as satisfied.
- Skips unselected end nodes (sets state to `Skipped`).

This replaces the current `processBranches()` logic entirely. The `SkipSubtask` method and `analysisTask` flow remain largely unchanged — the branch subtask simply produces output that guides conditional skipping.

**Rationale:** Fits naturally into the existing DAG execution model where subtask completion triggers successor evaluation. The only addition is conditional skipping based on branch output.

**Alternative considered:** Dynamic DAG rewiring at runtime. Rejected because it complicates the DAG compilation and caching model.

### Decision 4: Backward Compatibility

During `initByBean()`, if a subtask has `BranchConfig` in its `Settings` but no dedicated branch subtask row, the old behavior is preserved: the branch config on the parent subtask is read and `processBranches()` handles it on the leader.

New tasks use the `AddBranch` API which now creates a dedicated branch subtask row instead of attaching config to the parent.

**Migration path:** No automatic migration. New tasks get the new behavior. Existing running tasks continue with the old behavior. The `BranchConfig` on the parent subtask is deprecated but not removed.

### Decision 5: DAG Compilation Changes

When `AddBranch` is called, the `dagGraph` now:
1. Creates a branch subtask node with a generated ID (e.g., `branch_<parentID>_<index>`).
2. Adds a control edge from the parent node to the branch subtask.
3. Adds control edges from the branch subtask to each end node.
4. Stores the `BranchConfig` in the branch subtask's `Settings`.

The `compiledDAG.GetBranchesMap()` continues to work for backward compat, but `GetExecutableNodes()` now includes branch subtasks like any other node.

## Risks / Trade-offs

- **[Risk] Increased DB rows**: Each branch adds 1 subtask row. Mitigation: Branches are relatively few per task; overhead is negligible.
- **[Risk] Race between branch completion and successor dispatch**: If the leader dispatches a successor before the branch sub task marks unselected ones as skipped. Mitigation: The branch subtask's completion is processed atomically — skipping happens before successor dispatch via the same `analysisTask` flow.
- **[Risk] Cluster node without registered branch condition**: A worker node may not have the global branch registry populated. Mitigation: `registerBranch`/`registerBranchConditionProvider` are called at init time on all nodes; the receiver falls back to error if the provider is not found.
- **[Trade-off] Old branch code coexists with new**: Two code paths exist during the transition. Mitigation: Clear deprecation comments; old path can be removed in a follow-up once all tasks migrate.
