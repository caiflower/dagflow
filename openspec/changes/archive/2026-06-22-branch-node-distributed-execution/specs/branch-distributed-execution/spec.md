## ADDED Requirements

### Requirement: Branch condition evaluation on assigned worker
The system SHALL execute branch condition evaluation on the cluster node assigned to the branch subtask, not on the leader node.

#### Scenario: Branch subtask dispatched to worker
- **WHEN** the leader allocates a pending branch subtask to a worker node
- **THEN** the worker node receives the branch subtask via `DeliverSubtask` and executes the condition provider locally

#### Scenario: Condition provider not found on worker
- **WHEN** a worker node receives a branch subtask but the condition provider is not registered in its global registry
- **THEN** the branch subtask fails with an error indicating the missing provider

### Requirement: Unselected branch targets skipped after branch completion
After a branch subtask succeeds, the system SHALL skip all end nodes that were NOT selected by the branch condition, setting their state to `skipped`.

#### Scenario: Branch selects target A, skips B and C
- **WHEN** a branch subtask with end nodes [A, B, C] succeeds with output selecting "A"
- **THEN** subtasks B and C are set to `skipped` state, and subtask A proceeds to execution

#### Scenario: Branch selects target with no matching end node
- **WHEN** a branch subtask's output selects a key not in its end nodes
- **THEN** the branch subtask fails with an error

### Requirement: Leader delegates branch execution to workers
The leader SHALL NOT execute `processBranches()` directly. Instead, branch subtasks follow the standard subtask allocation and delivery flow via `allocateWorker` and `deliverToCluster`.

#### Scenario: Leader processes task with branch subtasks
- **WHEN** the leader's `analysisTask` encounters a task with pending branch subtasks
- **THEN** the branch subtasks are returned as `runningSubtasks` and allocated to workers via `allocateWorker`

#### Scenario: No leader hotspot from branch evaluation
- **WHEN** a task has 100 concurrent branch subtasks
- **THEN** the branch condition evaluations are distributed across all available cluster workers, not concentrated on the leader
