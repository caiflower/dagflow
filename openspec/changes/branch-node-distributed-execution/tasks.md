## 1. Branch Subtask Data Model and Serialization

- [ ] 1.1 Extend `dagGraph.AddBranch` to create a dedicated branch subtask node (generate ID like `branch_<parentID>_<idx>`) with auto-registered `branchExecutor` provider and control edges to all end nodes
- [ ] 1.2 Update `Task.toBeans()` to serialize branch subtasks as regular `model.Subtask` rows with `Settings.BranchConfig` containing end node names and condition provider identifier
- [ ] 1.3 Update `Task.initByBean()` to detect branch subtasks via `Settings.BranchConfig != nil` and restore their `Branch` info from the global `_branchRegistry` and `_branchConditionProviderRegistry`
- [ ] 1.4 Add backward compatibility: if a subtask has `BranchConfig` in its `Settings` but no dedicated branch subtask row, preserve the old `processBranches()` path

## 2. Branch Executor Implementation

- [ ] 2.1 Implement `branchExecutor` as a built-in `executor.ExecutorProvider` that reads `BranchConfig` from the subtask's `Settings`, resolves the condition provider from the global registry, executes it, and returns the selected key as output
- [ ] 2.2 Register `branchExecutor` globally under a well-known protocol name (e.g., `"builtin:branch"`) so it can be resolved on any worker node
- [ ] 2.3 Handle error cases: missing condition provider on worker, condition returning key not in end nodes, condition returning non-string result

## 3. DAG Compilation with Branch Subtask Nodes

- [ ] 3.1 Update `validateBranches` in `compile.go` to recognize branch subtask nodes and validate their end node references
- [ ] 3.2 Ensure `compiledDAG.GetExecutableNodes()` includes branch subtasks when their parent completes successfully
- [ ] 3.3 Update edge construction so branch subtask completion triggers successor evaluation (control edges from branch subtask to each end node)

## 4. Distributed Branch Execution in Dispatch

- [ ] 4.1 Remove or deprecate `processBranches()` call from `analysisTask()` for tasks using the new branch subtask model
- [ ] 4.2 Implement post-execution branch result handling: after branch subtask succeeds, read its output (selected key), skip unselected end nodes, and activate the selected end node
- [ ] 4.3 Ensure branch subtask allocation follows the same `allocateWorker` → `deliverToCluster` flow as regular subtasks
- [ ] 4.4 Handle branch subtask retry: on failure, retry per configured count/interval using the existing retry infrastructure

## 5. Receiver Integration

- [ ] 5.1 Update receiver to detect branch subtasks (check `Settings.BranchConfig`) and route them to the `branchExecutor` instead of the regular provider lookup
- [ ] 5.2 Ensure branch subtask completion triggers the same `notifyLeaderHandleTaskImmediately` flow so the leader processes branch results

## 6. Backward Compatibility and Testing

- [ ] 6.1 Ensure existing tasks with `BranchConfig` on parent subtask (no dedicated branch subtask row) continue to work via the old `processBranches()` path
- [ ] 6.2 Update existing tests in `backend/taskx/dag_test.go`, `backend/taskx/dispatch_test.go`, `backend/taskx/integration_test.go` for the new branch subtask model
- [ ] 6.3 Add new tests: branch subtask creation, serialization/deserialization round-trip, branch executor error handling, distributed branch execution with mock cluster
