## Tasks

- [x] Remove `branchPassthroughProvider` and `makeBranchCondition` from `flow_converter.go`
- [x] Refactor `FlowToTask` to skip subtask creation for branch nodes; use branch node's protocol provider as `ConditionProvider`
- [x] Add return-type validation: branch provider's `Execute` must return `string` (ID or taskName)
- [x] Wire branch node edges: predecessors → branch subtask → successors via `AddBranch`
- [x] Handle edge cases: 0 predecessors, < 2 successors
- [x] Add `flow_converter_test.go` with test cases:
  - [x] Single branch: start → branch → pathA/pathB → end
  - [x] Nested branch: start → outer → innerA/innerB → end
  - [x] Expr-based routing via protocol provider
  - [x] Skip non-branch (0 successors)
  - [x] Type validation: non-string return → error
- [x] Run full test suite (`go test ./internal/... ./backend/...`)
