## Why

The taskx engine now supports branch nodes as first-class DB subtasks with distributed execution and persistable `ConditionProvider`. However, the application-layer `FlowConverter` still uses the old pattern: creating a dummy `branchPassthroughProvider` subtask plus a closure-based `Condition`, resulting in redundant subtask creation and non-persistable branch conditions that cannot survive DB restore. Simplifying the app layer to align with the engine's current capabilities reduces complexity and improves reliability.

## What Changes

- Remove `branchPassthroughProvider` — branch nodes no longer need a dummy executor; branch routing is handled entirely by the engine's `AddBranch` + `executeBranchCondition`
- Stop creating regular subtasks for branch nodes in `FlowConverter`; instead use `AddBranch` to create dedicated branch subtasks that sit between predecessors and successors
- Convert `makeBranchCondition` closure into a persistable `ConditionProvider` using `executor.NewLocalExecutor`
- Add comprehensive unit tests for `FlowConverter` covering branch scenarios (single branch, nested branch, expr-based routing, roundtrip persistence)
- **BREAKING**: Branch nodes no longer appear as regular subtasks in DB — only the dedicated branch subtask (keyed `branch_<parent>_<index>`) exists

## Capabilities

### New Capabilities
- `app-branch-converter`: Simplified application-layer branch conversion that leverages taskx engine's `AddBranch` with `ConditionProvider` for persistable, distributed branch execution

### Modified Capabilities
<!-- None — this is a pure simplification, no spec-level requirement changes -->

## Impact

- `internal/converter/flow_converter.go` — major simplification: remove `branchPassthroughProvider`, remove branch subtask creation, use `ConditionProvider`
- `internal/converter/flow_converter_test.go` — new file with branch-specific test cases
- `internal/service/execution_service.go` — minor: remove `NodeType` unused fields after simplification
