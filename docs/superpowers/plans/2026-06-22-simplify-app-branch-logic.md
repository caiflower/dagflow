---
change: simplify-app-branch-logic
design-doc: docs/superpowers/specs/2026-06-22-simplify-app-branch-logic-design.md
base-ref: 2a07473747b0b25c210519e949d09e38074c062c
archived-with: 2026-06-22-simplify-app-branch-logic
---

# Simplify App-Layer Branch Logic — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove redundant `branchPassthroughProvider` and `makeBranchCondition` from `FlowConverter`; use branch node's own protocol provider directly as `ConditionProvider`.

**Architecture:** Branch nodes in Flow skip regular subtask creation. Instead, `FlowToTask` collects branch nodes, creates their protocol providers, validates return type, then wires them via `task.AddBranch(predecessor, NewBranch(provider, endNodes))`. Edges through branch nodes are skipped — `AddBranch` handles internal wiring.

**Tech Stack:** Go 1.24, `github.com/caiflower/dagflow/taskx`, `github.com/stretchr/testify/assert`

## Global Constraints

- Go 1.24
- Use `github.com/caiflower/common-tools/pkg/json` for JSON (not `encoding/json`)
- Follow existing code patterns in `internal/converter/`
- Tests use `github.com/stretchr/testify/assert`

archived-with: 2026-06-22-simplify-app-branch-logic
---

### Task 1: Remove dead code from flow_converter.go

**Files:**
- Modify: `backend/internal/converter/flow_converter.go`

**Interfaces:**
- Consumes: none (pure removal)
- Produces: none (removes exports)

- [ ] **Step 1: Delete `branchPassthroughProvider` struct and methods**

Remove lines containing:
```go
// branchPassthroughProvider 分支节点执行器
// 透传输入数据，分支路由决策由 taskx AddBranch 的 Condition 处理
type branchPassthroughProvider struct {
	node  FlowNode
	edges []FlowEdge
}

func (p *branchPassthroughProvider) Execute(ctx context.Context, data *executor.TaskData) (any, error) {
	// ...
}

func (p *branchPassthroughProvider) Protocol() executor.Protocol {
	return "branch"
}
```

- [ ] **Step 2: Delete `makeBranchCondition` function**

Remove:
```go
func makeBranchCondition(outEdges []FlowEdge) func(ctx interface{}, input any) (string, error) {
	// ...
}
```

- [ ] **Step 3: Delete `matchExpr` function**

Remove:
```go
func matchExpr(expr string, input any) bool {
	// ...
}
```

- [ ] **Step 4: Verify compilation**

Run: `cd backend && GOTOOLCHAIN=local GOWORK=off go build github.com/caiflower/dagflow/internal/converter`
Expected: FAIL (FlowToTask still references deleted code — fixed in Task 2)

- [ ] **Step 5: Commit**

```bash
git add backend/internal/converter/flow_converter.go
git commit -m "refactor: remove branchPassthroughProvider, makeBranchCondition, matchExpr"
```

archived-with: 2026-06-22-simplify-app-branch-logic
---

### Task 2: Refactor FlowToTask branch node handling

**Files:**
- Modify: `backend/internal/converter/flow_converter.go`

**Interfaces:**
- Consumes: `taskx.NewBranch(provider, endNodes)`, `task.AddBranch(subtask, branch)`
- Produces: corrected `FlowToTask` that skips branch subtask creation, uses protocol provider as ConditionProvider

- [ ] **Step 1: Add helper types and functions**

After imports, before `FlowToTask`, add:

```go
// branchInfo holds pending branch node data for wiring after all subtasks are created
type branchInfo struct {
	node     FlowNode
	provider executor.ExecutorProvider
}

// validateBranchProvider checks that a branch provider returns string (ID or taskName)
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

// isLocalProvider returns true for providers that can be validated at conversion time
func isLocalProvider(p executor.ExecutorProvider) bool {
	return p.Protocol() == "local" || p.Protocol() == "branch" || p.Protocol() == ""
}
```

- [ ] **Step 2: Modify subtask creation loop**

Replace the branch node handling in the loop (where `n.Type == "branch"` is checked):

```go
	for _, n := range nodes {
		if n.Type == "start" || n.Type == "end" {
			continue
		}

		if n.Type == "branch" {
			provider, err := providerFactory(n.Protocol, n.Config)
			if err != nil {
				return nil, fmt.Errorf("branch node %s: %w", n.Name, err)
			}
			// Validate return type for local providers
			if isLocalProvider(provider) {
				if err := validateBranchProvider(provider); err != nil {
					return nil, fmt.Errorf("branch node %s: %w", n.Name, err)
				}
			}
			pendingBranches = append(pendingBranches, branchInfo{node: n, provider: provider})
			continue
		}

		provider, err := providerFactory(n.Protocol, n.Config)
		if err != nil {
			return nil, fmt.Errorf("node %s: %w", n.Name, err)
		}

		st := taskx.NewSubtask(n.Name, provider)
		if input, ok := nodeInputs[n.Name]; ok && input != "" {
			st.SetInput(input)
		}
		subtaskMap[n.ID] = st
		if err := task.AddSubtask(st); err != nil {
			return nil, fmt.Errorf("add subtask %s: %w", n.Name, err)
		}
	}
```

Declare `pendingBranches` before the loop:
```go
var pendingBranches []branchInfo
```

- [ ] **Step 3: Modify edge creation loop**

Skip edges involving branch nodes:

```go
	for _, e := range edges {
		// Skip edges involving branch nodes — AddBranch handles wiring internally
		if isBranchNode(nodes, e.Source) || isBranchNode(nodes, e.Target) {
			continue
		}
		src, ok1 := subtaskMap[e.Source]
		dst, ok2 := subtaskMap[e.Target]
		if !ok1 || !ok2 {
			continue
		}
		switch e.Type {
		case "data":
			if err := task.AddDataEdge(src, dst); err != nil {
				return nil, fmt.Errorf("add data edge %s->%s: %w", e.Source, e.Target, err)
			}
		case "control+data":
			if err := task.AddEdge(src, dst); err != nil {
				return nil, fmt.Errorf("add edge %s->%s: %w", e.Source, e.Target, err)
			}
		default:
			if err := task.AddControlEdge(src, dst); err != nil {
				return nil, fmt.Errorf("add control edge %s->%s: %w", e.Source, e.Target, err)
			}
		}
	}
```

Add helper:
```go
func isBranchNode(nodes []FlowNode, id string) bool {
	for _, n := range nodes {
		if n.ID == id && n.Type == "branch" {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Replace branch wiring section**

Replace the entire `for _, n := range nodes { if n.Type != "branch" ... }` block with:

```go
	// Wire branch nodes: create branch subtasks via AddBranch with protocol provider
	for _, bi := range pendingBranches {
		// Find predecessors (nodes with edges to this branch node)
		var predIDs []string
		for _, e := range edges {
			if e.Target == bi.node.ID {
				predIDs = append(predIDs, e.Source)
			}
		}
		if len(predIDs) == 0 {
			continue
		}

		// Find successor names
		succNames := make(map[string]bool)
		for _, e := range edges {
			if e.Source == bi.node.ID {
				succNames[e.Target] = true
			}
		}
		if len(succNames) < 2 {
			continue
		}

		// Resolve successor names from target node IDs
		resolvedEndNodes := make(map[string]bool, len(succNames))
		for targetID := range succNames {
			// Find the target subtask by looking through nodes for this ID
			for _, n := range nodes {
				if n.ID == targetID && n.Type == "task" {
					resolvedEndNodes[n.Name] = true
					break
				}
			}
		}

		// Create branch for each predecessor
		for _, predID := range predIDs {
			predSt, ok := subtaskMap[predID]
			if !ok {
				continue
			}
			if err := task.AddBranch(predSt, taskx.NewBranch(bi.provider, resolvedEndNodes)); err != nil {
				return nil, fmt.Errorf("add branch %s: %w", bi.node.Name, err)
			}
		}
	}
```

- [ ] **Step 5: Remove unused imports**

Remove `"encoding/json"` from imports (was used by `branchPassthroughProvider.Execute`). Verify other removed imports.

- [ ] **Step 6: Verify compilation**

Run: `cd backend && GOTOOLCHAIN=local GOWORK=off go build github.com/caiflower/dagflow/internal/converter`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add backend/internal/converter/flow_converter.go
git commit -m "refactor: use protocol provider as branch ConditionProvider in FlowConverter"
```

archived-with: 2026-06-22-simplify-app-branch-logic
---

### Task 3: Add flow_converter_test.go

**Files:**
- Create: `backend/internal/converter/flow_converter_test.go`

**Interfaces:**
- Consumes: `FlowToTask`, `FlowNode`, `FlowEdge`, `taskx.NewTask`, `taskx.NewSubtask`, `executor.NewLocalExecutor`
- Produces: test coverage for branch conversion

- [ ] **Step 1: Create test file with package and imports**

```go
package converter

import (
	"context"
	"testing"

	"github.com/caiflower/dagflow/taskx"
	"github.com/caiflower/dagflow/taskx/executor"
	"github.com/stretchr/testify/assert"
)
```

- [ ] **Step 2: Write helper — echo provider factory**

```go
func echoProviderFactory(protocol string, config map[string]any) (executor.ExecutorProvider, error) {
	return executor.NewLocalExecutor(func(ctx context.Context, input map[string]any) (string, error) {
		return "", nil
	}), nil
}
```

- [ ] **Step 3: Write helper — branch provider factory (returns target name)**

```go
func branchProviderFactory(targetName string) func(string, map[string]any) (executor.ExecutorProvider, error) {
	return func(protocol string, config map[string]any) (executor.ExecutorProvider, error) {
		return executor.NewLocalExecutor(func(ctx context.Context, input map[string]any) (string, error) {
			return targetName, nil
		}), nil
	}
}
```

- [ ] **Step 4: Test — single branch routing**

```go
func TestFlowToTask_SingleBranch(t *testing.T) {
	nodes := []FlowNode{
		{ID: "n1", Name: "start", Type: "task", Protocol: "local"},
		{ID: "n2", Name: "branch1", Type: "branch", Protocol: "local"},
		{ID: "n3", Name: "pathA", Type: "task", Protocol: "local"},
		{ID: "n4", Name: "pathB", Type: "task", Protocol: "local"},
		{ID: "n5", Name: "end", Type: "task", Protocol: "local"},
	}
	edges := []FlowEdge{
		{ID: "e1", Source: "n1", Target: "n2"},
		{ID: "e2", Source: "n2", Target: "n3"},
		{ID: "e3", Source: "n2", Target: "n4"},
		{ID: "e4", Source: "n3", Target: "n5"},
		{ID: "e5", Source: "n4", Target: "n5"},
	}

	providerFactory := branchProviderFactory("pathA")
	task, err := FlowToTaskWithNodes("test-branch", nodes, edges, providerFactory, nil)
	assert.NoError(t, err)

	_, err = task.Compile()
	assert.NoError(t, err)
	assert.NotNil(t, task)

	// Verify branch subtask exists in dag branches
	compiled := task.GetCompiled()
	assert.NotNil(t, compiled)
	branches := compiled.GetBranchesMap()
	assert.NotEmpty(t, branches, "should have branch definitions")
}
```

- [ ] **Step 5: Test — nested branch**

```go
func TestFlowToTask_NestedBranch(t *testing.T) {
	nodes := []FlowNode{
		{ID: "n1", Name: "start", Type: "task", Protocol: "local"},
		{ID: "n2", Name: "outer", Type: "branch", Protocol: "local"},
		{ID: "n3", Name: "inner", Type: "branch", Protocol: "local"},
		{ID: "n4", Name: "pathA", Type: "task", Protocol: "local"},
		{ID: "n5", Name: "pathB", Type: "task", Protocol: "local"},
		{ID: "n6", Name: "end", Type: "task", Protocol: "local"},
	}
	edges := []FlowEdge{
		{ID: "e1", Source: "n1", Target: "n2"},
		{ID: "e2", Source: "n2", Target: "n3"},
		{ID: "e3", Source: "n3", Target: "n4"},
		{ID: "e4", Source: "n3", Target: "n5"},
		{ID: "e5", Source: "n2", Target: "n6"}, // fallback path
		{ID: "e6", Source: "n4", Target: "n6"},
		{ID: "e7", Source: "n5", Target: "n6"},
	}

	// outer selects "inner", inner selects "pathA"
	providerFactory := func(protocol string, config map[string]any) (executor.ExecutorProvider, error) {
		// Determine target based on node context via config
		return executor.NewLocalExecutor(func(ctx context.Context, input map[string]any) (string, error) {
			return "pathA", nil
		}), nil
	}

	task, err := FlowToTaskWithNodes("test-nested-branch", nodes, edges, providerFactory, nil)
	assert.NoError(t, err)

	_, err = task.Compile()
	assert.NoError(t, err)
}
```

- [ ] **Step 6: Test — skip branch with < 2 successors**

```go
func TestFlowToTask_SkipNonBranch(t *testing.T) {
	nodes := []FlowNode{
		{ID: "n1", Name: "start", Type: "task", Protocol: "local"},
		{ID: "n2", Name: "single", Type: "branch", Protocol: "local"},
		{ID: "n3", Name: "end", Type: "task", Protocol: "local"},
	}
	edges := []FlowEdge{
		{ID: "e1", Source: "n1", Target: "n2"},
		{ID: "e2", Source: "n2", Target: "n3"}, // only 1 successor
	}

	providerFactory := echoProviderFactory
	task, err := FlowToTaskWithNodes("test-skip", nodes, edges, providerFactory, nil)
	assert.NoError(t, err)
	// Should compile without error, branch node is skipped
	_, err = task.Compile()
	assert.NoError(t, err)
}
```

- [ ] **Step 7: Test — type validation rejects non-string provider**

```go
func TestFlowToTask_BranchProviderMustReturnString(t *testing.T) {
	nodes := []FlowNode{
		{ID: "n1", Name: "start", Type: "task", Protocol: "local"},
		{ID: "n2", Name: "badBranch", Type: "branch", Protocol: "local"},
		{ID: "n3", Name: "pathA", Type: "task", Protocol: "local"},
		{ID: "n4", Name: "pathB", Type: "task", Protocol: "local"},
	}
	edges := []FlowEdge{
		{ID: "e1", Source: "n1", Target: "n2"},
		{ID: "e2", Source: "n2", Target: "n3"},
		{ID: "e3", Source: "n2", Target: "n4"},
	}

	// Provider returns int instead of string
	badFactory := func(protocol string, config map[string]any) (executor.ExecutorProvider, error) {
		return executor.NewLocalExecutor(func(ctx context.Context, input map[string]any) (int, error) {
			return 42, nil
		}), nil
	}

	_, err := FlowToTaskWithNodes("test-bad", nodes, edges, badFactory, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must return string")
}
```

- [ ] **Step 8: Add `FlowToTaskWithNodes` helper to flow_converter.go**

Expose a test-friendly variant that skips DB model wrapping:

```go
// FlowToTaskWithNodes creates a taskx.Task directly from nodes and edges (for testing)
func FlowToTaskWithNodes(name string, nodes []FlowNode, edges []FlowEdge, providerFactory func(string, map[string]any) (executor.ExecutorProvider, error), nodeInputs map[string]string) (*taskx.Task, error) {
	task := taskx.NewTask(name)

	var pendingBranches []branchInfo
	subtaskMap := make(map[string]*taskx.Subtask)

	for _, n := range nodes {
		if n.Type == "start" || n.Type == "end" {
			continue
		}

		if n.Type == "branch" {
			provider, err := providerFactory(n.Protocol, n.Config)
			if err != nil {
				return nil, fmt.Errorf("branch node %s: %w", n.Name, err)
			}
			if isLocalProvider(provider) {
				if err := validateBranchProvider(provider); err != nil {
					return nil, fmt.Errorf("branch node %s: %w", n.Name, err)
				}
			}
			pendingBranches = append(pendingBranches, branchInfo{node: n, provider: provider})
			continue
		}

		provider, err := providerFactory(n.Protocol, n.Config)
		if err != nil {
			return nil, fmt.Errorf("node %s: %w", n.Name, err)
		}

		st := taskx.NewSubtask(n.Name, provider)
		if input, ok := nodeInputs[n.Name]; ok && input != "" {
			st.SetInput(input)
		}
		subtaskMap[n.ID] = st
		if err := task.AddSubtask(st); err != nil {
			return nil, fmt.Errorf("add subtask %s: %w", n.Name, err)
		}
	}

	// Build edges (skip branch nodes)
	for _, e := range edges {
		if isBranchNode(nodes, e.Source) || isBranchNode(nodes, e.Target) {
			continue
		}
		src, ok1 := subtaskMap[e.Source]
		dst, ok2 := subtaskMap[e.Target]
		if !ok1 || !ok2 {
			continue
		}
		switch e.Type {
		case "data":
			if err := task.AddDataEdge(src, dst); err != nil {
				return nil, fmt.Errorf("add data edge %s->%s: %w", e.Source, e.Target, err)
			}
		case "control+data":
			if err := task.AddEdge(src, dst); err != nil {
				return nil, fmt.Errorf("add edge %s->%s: %w", e.Source, e.Target, err)
			}
		default:
			if err := task.AddControlEdge(src, dst); err != nil {
				return nil, fmt.Errorf("add control edge %s->%s: %w", e.Source, e.Target, err)
			}
		}
	}

	// Wire branches
	for _, bi := range pendingBranches {
		var predIDs []string
		for _, e := range edges {
			if e.Target == bi.node.ID {
				predIDs = append(predIDs, e.Source)
			}
		}
		if len(predIDs) == 0 {
			continue
		}

		succNames := make(map[string]bool)
		for _, e := range edges {
			if e.Source == bi.node.ID {
				succNames[e.Target] = true
			}
		}
		if len(succNames) < 2 {
			continue
		}

		resolvedEndNodes := make(map[string]bool, len(succNames))
		for targetID := range succNames {
			for _, n := range nodes {
				if n.ID == targetID && n.Type == "task" {
					resolvedEndNodes[n.Name] = true
					break
				}
			}
		}

		for _, predID := range predIDs {
			predSt, ok := subtaskMap[predID]
			if !ok {
				continue
			}
			if err := task.AddBranch(predSt, taskx.NewBranch(bi.provider, resolvedEndNodes)); err != nil {
				return nil, fmt.Errorf("add branch %s: %w", bi.node.Name, err)
			}
		}
	}

	return task, nil
}
```

Refactor `FlowToTask` to call `FlowToTaskWithNodes` internally after parsing.

- [ ] **Step 9: Run tests**

Run: `cd backend && GOTOOLCHAIN=local GOWORK=off go test ./internal/converter/ -v -run "TestFlowToTask" -count=1`
Expected: 4 tests PASS

- [ ] **Step 10: Commit**

```bash
git add backend/internal/converter/flow_converter.go backend/internal/converter/flow_converter_test.go
git commit -m "test: add FlowConverter branch tests and FlowToTaskWithNodes helper"
```

archived-with: 2026-06-22-simplify-app-branch-logic
---

### Task 4: Run full test suite and cleanup

**Files:**
- Modify: `backend/internal/converter/flow_converter.go` (remove old `FlowToTask` duplication)
- Modify: `backend/internal/service/execution_service.go` (if needed)

- [ ] **Step 1: Refactor FlowToTask to call FlowToTaskWithNodes**

Replace `FlowToTask` body with delegation to `FlowToTaskWithNodes` after JSON parsing:

```go
func FlowToTask(flow *model.Flow, providerFactory func(protocol string, config map[string]any) (executor.ExecutorProvider, error), nodeInputs map[string]string) (*taskx.Task, error) {
	nodes, edges, err := ParseFlowJSON(flow)
	if err != nil {
		return nil, err
	}
	return FlowToTaskWithNodes(flow.Name, nodes, edges, providerFactory, nodeInputs)
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd backend && GOTOOLCHAIN=local GOWORK=off go build ./internal/...`
Expected: PASS

- [ ] **Step 3: Run full test suite**

Run: `cd backend && GOTOOLCHAIN=local GOWORK=off go test ./internal/... ./taskx/... -count=1 -timeout 180s`
Expected: all PASS

- [ ] **Step 4: Commit**

```bash
git add backend/internal/converter/flow_converter.go
git commit -m "refactor: delegate FlowToTask to FlowToTaskWithNodes"
```
