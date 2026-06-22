## ADDED Requirements

### Requirement: Branch node stored as subtask
The system SHALL persist each branch node as a dedicated row in the `subtask` table with its own ID, task ID, state, retry count, timeout, worker assignment, and a `Settings` JSON field containing `BranchConfig`.

#### Scenario: Branch subtask created via AddBranch
- **WHEN** a task calls `AddBranch(parentNode, branch)` on a DAG
- **THEN** a new subtask row is created in the `subtask` table with `settings` containing the serialized `BranchConfig` (end node names and condition provider identifier)

#### Scenario: Branch subtask restored from DB
- **WHEN** the dispatcher initializes a task from the database
- **THEN** branch subtasks are rebuilt with their `BranchConfig` restored from the `Settings` column, and the `ConditionProvider` is resolved from the global `_branchConditionProviderRegistry`

### Requirement: Branch subtask state lifecycle
The branch subtask SHALL follow the standard subtask state machine: `pending` → `running` → `succeeded` / `failed`. On failure, it SHALL be retried per the configured retry count and interval.

#### Scenario: Branch condition evaluation succeeds
- **WHEN** the branch executor returns a valid selected key
- **THEN** the branch subtask transitions to `succeeded` and stores the selected key in its output

#### Scenario: Branch condition evaluation fails
- **WHEN** the branch executor returns an error
- **THEN** the branch subtask transitions to `failed` and may be retried if retry count > 0

### Requirement: Branch configuration stored in subtask Settings
The branch subtask's `Settings` field SHALL contain a `BranchConfig` object with `end_nodes` (list of target node names) and `condition_provider` (string identifier for the global registry lookup).

#### Scenario: BranchConfig serialized on task submission
- **WHEN** a task is submitted via `SubmitTask`
- **THEN** the branch subtask's `Settings` JSON includes `branch_config` with the end node names and condition provider identifier
