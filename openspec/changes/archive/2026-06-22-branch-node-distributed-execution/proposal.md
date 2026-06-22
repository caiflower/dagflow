## Why

Branch nodes currently exist only as metadata attached to parent subtasks (serialized in the `Settings` JSON field), which prevents them from being independently tracked, monitored, or retried. Additionally, branch condition evaluation and unselected-node skipping run exclusively on the leader node via `handleTaskImmediately()`, creating a single-node hotspot under high task throughput. This change makes branch nodes first-class cluster citizens by persisting them as subtasks in the DB and enabling distributed condition execution across worker nodes.

## What Changes

- **New**: Branch nodes become full DB subtask rows with state tracking, retry, timeout, and worker assignment — just like regular subtasks
- **New**: Branch condition evaluation moves from the leader's `processBranches()` to per-subtask execution on the assigned worker node
- **New**: Branch node output stores the selected branch key, enabling downstream nodes to be conditionally executed
- **Modified**: `processBranches()` is replaced by a branch executor that runs as a standard subtask executor on any cluster node
- **Modified**: DAG compilation treats branch subtasks as real nodes in the graph, with control edges to all branch target nodes
- **Modified**: The `Settings.BranchConfig` field on parent subtasks is deprecated; branch info lives in its own subtask row
- **Compatibility**: Existing `BranchConfig` in `Settings` is still read for backward compatibility during a migration window, but new tasks use the dedicated subtask approach

## Capabilities

### New Capabilities
- `branch-subtask-persistence`: Branch nodes are stored as full subtask rows in the DB with their own ID, state, retry, timeout, and worker assignment. Branch configuration (end nodes, condition provider) is stored in the subtask's `Settings` field.
- `branch-distributed-execution`: Branch condition evaluation executes on whichever cluster node is assigned the branch subtask, eliminating the leader hotspot.

### Modified Capabilities
<!-- No existing capabilities are modified at the spec level -->

## Impact

- **Affected code**: `backend/taskx/task.go` (branch serialization/deserialization), `backend/taskx/dag.go` (Branch/BranchConfig/SubtaskSettings structs), `backend/taskx/compile.go` (DAG compilation with branch nodes), `backend/taskx/dispatch.go` (`processBranches`, `analysisTask`), `backend/taskx/executor.go` (branch executor registration), `backend/taskx/receiver_internal.go` (branch subtask execution), `backend/taskx/dao/model/subtask.go` (Settings field usage)
- **APIs**: No external API changes. Internal gRPC proto unchanged — branch subtasks use existing `DeliverSubtask` delivery.
- **Dependencies**: No new dependencies.
- **Database**: The `subtask` table already has a `settings` JSON column — no schema migration needed for the new branch subtask rows. The `branch_config` field within settings gains new semantics.
