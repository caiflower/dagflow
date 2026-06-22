# Branch Node Distributed Execution Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make branch nodes first-class DB subtasks with distributed cluster execution, eliminating the leader hotspot.

**Architecture:** Branch nodes become regular `subtask` rows with `Settings.BranchConfig`. A built-in `branchExecutor` replaces the leader-only `processBranches()`. The DAG adds control edges from branch subtask to each end node, and the branch output determines which path activates.

**Tech Stack:** Go 1.24, bun ORM, common-tools cluster framework

## Global Constraints

- Backward compatibility: existing tasks with legacy `BranchConfig` on parent subtask must continue working
- No new DB schema or tables — reuse existing `subtask.settings` JSON column
- No gRPC proto changes — branch subtasks use existing `DeliverSubtask` delivery
- Use `github.com/caiflower/common-tools/pkg/tools` for JSON, not `encoding/json`
- Branch subtask ID format: `branch_<parentSubtaskID>_<index>`

---

### Task 1: Branch Subtask Data Model (dag.go + task.go)

**Files:**
- Modify: `backend/taskx/dag.go:180-220` (structs), `backend/taskx/dag.go:345-375` (AddBranch)
- Modify: `backend/taskx/task.go:448-490` (AddBranch), `backend/taskx/task.go:1040-1100` (toBeans), `backend/taskx/task.go:1175-1210` (initByBean)

**Interfaces:**
- Produces: Updated `dagGraph.AddBranch` that creates branch subtask nodes; updated `Task.AddBranch` that creates branch subtask DB rows; updated `initByBean` that restores branch subtasks from DB; updated `toBeans` that serializes branch subtasks

- [ ] **Step 1: Update dagGraph.AddBranch to create branch subtask nodes**

In `backend/taskx/dag.go`, modify `AddBranch`:

```go
// AddBranch adds a conditional branch and creates a dedicated branch subtask node.
func (g *dagGraph) AddBranch(nodeKey string, branch *Branch) error {
	if g.compiled {
		return ErrGraphCompiled
	}
	parentNode := g.nodes[nodeKey]
	if parentNode == nil {
		return errors.New("node not found: " + nodeKey)
	}

	// Generate branch subtask node key (ID will be set by Task.toBeans)
	branchKey := fmt.Sprintf("branch_%s_%d", nodeKey, len(g.branches[nodeKey]))

	// Create branch subtask node in DAG
	g.nodes[branchKey] = &dagNode{
		key:         branchKey,
		triggerMode: AllPredecessor,
		state:       NodePending,
		priority:    parentNode.priority,
		timeout:     30, // default 30s for branch condition evaluation
	}

	// Add control edge: parent → branch subtask
	if err := g.AddEdge(nodeKey, branchKey, ControlEdge); err != nil {
		return err
	}

	// Add control edges: branch subtask → each end node
	for endKey := range branch.EndNodes {
		if err := g.AddEdge(branchKey, endKey, ControlEdge); err != nil {
			return err
		}
	}

	// Store branch metadata keyed by branch subtask node
	g.branches[branchKey] = append(g.branches[branchKey], branch)
	return nil
}
```

- [ ] **Step 2: Update Task.AddBranch to register branch subtask**

In `backend/taskx/task.go`, modify `AddBranch` to create a SubTask for the branch:

```go
func (t *Task) AddBranch(node *Subtask, branch *Branch) error {
	// Resolve EndNodes: name -> ID
	resolvedEndNodes := make(map[string]bool, len(branch.EndNodes))
	for key := range branch.EndNodes {
		resolvedEndNodes[t.resolveSubtaskKey(key)] = true
	}
	branch.EndNodes = resolvedEndNodes

	// Wrap Condition/ConditionProvider for name resolution (same as before)
	if branch.Condition != nil {
		origCond := branch.Condition
		branch.Condition = func(ctx interface{}, input any) (string, error) {
			selected, err := origCond(ctx, input)
			if err != nil {
				return "", err
			}
			return t.resolveSubtaskKey(selected), nil
		}
	}
	if branch.ConditionProvider != nil {
		origProvider := branch.ConditionProvider
		branch.ConditionProvider = &nameResolvingProvider{
			inner: origProvider,
			resolve: func(nameOrID string) string {
				return t.resolveSubtaskKey(nameOrID)
			},
		}
	}

	// Create branch subtask (visible in DB)
	branchSubtask := NewSubtask(
		fmt.Sprintf("branch_%s", node.GetName()),
		"builtin:branch",
	)
	branchSubtask.subtask.ID = fmt.Sprintf("branch_%s_%d", node.GetID(), len(t.dag.branches[node.GetID()]))

	// Build BranchConfig
	endNodeNames := make([]string, 0, len(branch.EndNodes))
	for k := range branch.EndNodes {
		if s, exists := t.subtaskMap[k]; exists {
			endNodeNames = append(endNodeNames, s.GetName())
		}
	}
	providerName := ""
	if branch.ConditionProvider != nil {
		providerName = string(branch.ConditionProvider.Protocol())
	}
	branchSubtask.subtask.Settings = tools.ToJson(SubtaskSettings{
		BranchConfig: &BranchConfig{
			EndNodes:          endNodeNames,
			ConditionProvider: providerName,
		},
	})

	t.subtaskMap[branchSubtask.GetID()] = branchSubtask

	// Register branch to global registries
	registerBranch(t.task.TaskName, branchSubtask.GetID(), branch)
	if branch.ConditionProvider != nil {
		registerBranchConditionProvider(t.task.TaskName, branchSubtask.GetID(), branch.ConditionProvider)
	}

	// Add to DAG with new semantics (creates control edges)
	return t.dag.AddBranch(node.GetID(), branch)
}
```

- [ ] **Step 3: Update toBeans to skip duplicate serialization of branch subtasks**

In `backend/taskx/task.go`, modify `toBeans()` ~line 1046: Remove the `Settings.BranchConfig` serialization on the parent subtask (branch subtasks are now separate rows) and ensure branch subtasks in `subtaskMap` are serialized as regular rows:

```go
// In toBeans(), for each subtask in t.subtaskMap:
for _, subtask := range t.subtaskMap {
    bean := &model.Subtask{
        // ... existing fields ...
    }
    // Branch config is now stored on the branch subtask itself, not on the parent
    // The branch subtask already has its Settings populated in AddBranch
    // For backward compat: keep old behavior if no dedicated branch subtask exists
    if branches, ok := t.dag.branches[subtask.GetID()]; ok && len(branches) > 0 {
        // Only serialize BranchConfig on parent if this is a legacy task (no branch subtask rows)
        isLegacy := true
        for _, br := range branches {
            for endKey := range br.EndNodes {
                // Check if there's a branch subtask node for this branch
                branchKey := fmt.Sprintf("branch_%s_%d", subtask.GetID(), 0)
                if _, exists := t.subtaskMap[branchKey]; exists {
                    isLegacy = false
                    break
                }
            }
        }
        if isLegacy {
            // Legacy path: serialize to parent's Settings
            settings := SubtaskSettings{}
            for _, br := range branches {
                endNodeNames := make([]string, 0, len(br.EndNodes))
                for k := range br.EndNodes {
                    if s, exists := t.subtaskMap[k]; exists {
                        endNodeNames = append(endNodeNames, s.GetName())
                    }
                }
                providerName := ""
                if br.ConditionProvider != nil {
                    providerName = string(br.ConditionProvider.Protocol())
                }
                settings.BranchConfig = &BranchConfig{
                    EndNodes:          endNodeNames,
                    ConditionProvider: providerName,
                }
            }
            bean.Settings = tools.ToJson(settings)
        }
    }
    subtaskBeans = append(subtaskBeans, bean)
}
```

- [ ] **Step 4: Update initByBean to restore branch subtasks**

In `backend/taskx/task.go`, modify `initByBean()` ~line 1175:

```go
// After restoring edges and before Compile(), add branch restoration:

// Restore branch info from global registry AND from branch subtask rows
if registeredBranches := getRegisteredBranches(taskBean.TaskName); len(registeredBranches) > 0 {
    for nodeKey, branches := range registeredBranches {
        for i, br := range branches {
            if br.ConditionProvider == nil && br.Condition == nil {
                if p := getBranchConditionProvider(taskBean.TaskName, nodeKey, i); p != nil {
                    br.ConditionProvider = p
                }
            }
        }
        t.dag.branches[nodeKey] = branches
    }
}

// Also detect branch subtask rows (Settings.BranchConfig != nil)
for _, subtask := range subtaskBeans {
    if subtask.Settings == "" {
        continue
    }
    var settings SubtaskSettings
    if err := tools.Unmarshal([]byte(subtask.Settings), &settings); err != nil {
        continue
    }
    if settings.BranchConfig == nil {
        continue
    }
    // This is a branch subtask row — restore its Branch to the DAG
    // The branch subtask is already in subtaskMap/subtaskLookup
    // Control edges from parent→branch and branch→endNodes are restored from task_edge table
}
```

- [ ] **Step 5: Commit**

```bash
git add backend/taskx/dag.go backend/taskx/task.go
git commit -m "feat: branch subtask data model with DB persistence"
```

---

### Task 2: Branch Executor Implementation (executor.go)

**Files:**
- Modify: `backend/taskx/executor.go` (append new code)

**Interfaces:**
- Produces: `branchExecutor` implementing `executor.ExecutorProvider`, registered globally under protocol `"builtin:branch"`

- [ ] **Step 1: Write branchExecutor**

In `backend/taskx/executor.go`, append at end of file:

```go
// branchExecutor is the built-in executor for branch subtasks.
// It reads BranchConfig from the subtask's Settings, resolves the condition
// provider from the global registry, executes it, and returns the selected key.
type branchExecutor struct{}

func (e *branchExecutor) Protocol() executor.Protocol {
	return "builtin:branch"
}

func (e *branchExecutor) Execute(ctx context.Context, data *executor.TaskData) (any, error) {
	// data.Input contains the serialized TaskInput which has SubTaskId
	// We need to load the subtask from DB to get its Settings
	// However, the receiver already has the subtask — we need a different approach.
	// The branchExecutor receives TaskData.Input = the branch subtask's Settings JSON
	// and the parent node's output as the condition input.
	return nil, errors.New("branchExecutor: Execute should be called through the receiver's branch execution path")
}

// executeBranchCondition reads the branch subtask's Settings, resolves the
// condition provider, and executes it with the parent node's output as input.
func executeBranchCondition(taskName, nodeKey string, settingsJSON string, conditionInput any) (string, error) {
	var settings SubtaskSettings
	if err := tools.Unmarshal([]byte(settingsJSON), &settings); err != nil {
		return "", fmt.Errorf("branch: failed to parse settings: %w", err)
	}
	if settings.BranchConfig == nil {
		return "", errors.New("branch: no BranchConfig in settings")
	}

	// Resolve condition provider from global registry
	provider := getBranchConditionProvider(taskName, nodeKey, 0)
	if provider == nil {
		return "", fmt.Errorf("branch: condition provider not found for %s/%s", taskName, nodeKey)
	}

	// Prepare TaskData with the parent node's output as input
	taskData := &executor.TaskData{
		SubTaskId: nodeKey,
	}
	if conditionInput != nil {
		if s, ok := conditionInput.(string); ok {
			taskData.Input = s
		}
	}

	result, err := provider.Execute(context.Background(), taskData)
	if err != nil {
		return "", fmt.Errorf("branch: condition execution failed: %w", err)
	}

	selected, ok := result.(string)
	if !ok {
		return "", fmt.Errorf("branch: condition returned non-string result: %v", result)
	}

	// Validate selected key is in end nodes
	found := false
	for _, n := range settings.BranchConfig.EndNodes {
		if n == selected {
			found = true
			break
		}
	}
	if !found {
		return "", fmt.Errorf("branch: selected key %q not in end nodes %v", selected, settings.BranchConfig.EndNodes)
	}

	return selected, nil
}
```

- [ ] **Step 2: Register branchExecutor during init**

In `backend/taskx/executor.go`, add an `init()` or register in package-level var:

```go
func init() {
	// Register the built-in branch executor under a well-known protocol
	registerProvider("__builtin__", "branch", &branchExecutor{})
}
```

- [ ] **Step 3: Commit**

```bash
git add backend/taskx/executor.go
git commit -m "feat: built-in branchExecutor for distributed branch condition evaluation"
```

---

### Task 3: DAG Compilation with Branch Subtask Nodes (compile.go)

**Files:**
- Modify: `backend/taskx/compile.go:30-100` (dagGraph fields, compilation), `backend/taskx/compile.go:163-175` (validateBranches), `backend/taskx/compile.go:243-252` (GetBranches)

**Interfaces:**
- Consumes: Updated `dagGraph` with branch subtask nodes (Task 1)
- Produces: `compiledDAG` that correctly includes branch subtask nodes in executable node calculation

- [ ] **Step 1: Update validateBranches for branch subtask nodes**

In `backend/taskx/compile.go`, update `validateBranches`:

```go
func validateBranches(g *dagGraph) error {
	for nodeKey, branches := range g.branches {
		// nodeKey is now the branch subtask node key, which must exist in the graph
		if _, exists := g.nodes[nodeKey]; !exists {
			return fmt.Errorf("branch subtask node %s not found in graph", nodeKey)
		}
		for _, branch := range branches {
			for endNode := range branch.EndNodes {
				if _, exists := g.nodes[endNode]; !exists {
					return fmt.Errorf("branch from node %s has invalid end node: %s", nodeKey, endNode)
				}
			}
		}
	}
	return nil
}
```

- [ ] **Step 2: Verify GetExecutableNodes includes branch subtasks**

The branch subtask is a regular `dagNode` in `g.nodes` with `ControlEdge` from its parent. When the parent completes (state `NodeSucceeded`), the control edge is satisfied, and `GetExecutableNodes` will naturally include the branch subtask. No code changes needed — just verify by reading the existing logic.

- [ ] **Step 3: Commit**

```bash
git add backend/taskx/compile.go
git commit -m "feat: validate branch subtask nodes in DAG compilation"
```

---

### Task 4: Distributed Branch Execution in Dispatch (dispatch.go)

**Files:**
- Modify: `backend/taskx/dispatch.go:510-608` (analysisTask, processBranches)

**Interfaces:**
- Consumes: Branch subtasks in DB (Task 1), branchExecutor (Task 2), compiled DAG with branch nodes (Task 3)
- Produces: Branch subtasks dispatched to workers via `allocateWorker`; post-execution branch result handling

- [ ] **Step 1: Guard processBranches for backward compat only**

In `backend/taskx/dispatch.go`, modify `analysisTask` to only call `processBranches` for legacy tasks:

```go
// In analysisTask(), after the rollback check and before "Get the next executable subtasks":

// Handle branch selection for LEGACY tasks only
// New tasks have dedicated branch subtask rows that flow through normal execution
if task.hasLegacyBranches() {
    t.processBranches(ctx, task)
}
```

- [ ] **Step 2: Add hasLegacyBranches helper to Task**

In `backend/taskx/task.go`:

```go
// hasLegacyBranches returns true if the task uses the old branch model
// (BranchConfig attached to parent subtask, no dedicated branch subtask rows).
func (t *Task) hasLegacyBranches() bool {
	for _, branches := range t.dag.branches {
		for _, br := range branches {
			for endKey := range br.EndNodes {
				if _, exists := t.subtaskMap[endKey]; exists {
					// Check if there's a corresponding branch subtask row
					// If not, this is legacy
					return true
				}
			}
		}
	}
	return false
}
```

- [ ] **Step 3: Implement post-execution branch result handling**

Add a new method in `backend/taskx/dispatch.go`:

```go
// handleBranchResult processes the result of a completed branch subtask.
// It reads the selected key from the branch output and skips unselected end nodes.
func (t *taskDispatcher) handleBranchResult(ctx context.Context, task *Task, branchSubtask *Subtask) {
	var settings SubtaskSettings
	if err := tools.Unmarshal([]byte(branchSubtask.subtask.Settings), &settings); err != nil {
		logger.Error("[handleBranchResult] failed to parse branch settings: %v", err)
		return
	}
	if settings.BranchConfig == nil {
		return
	}

	// Parse the branch output to get the selected key
	var output Output
	if err := tools.Unmarshal([]byte(branchSubtask.subtask.Output), &output); err != nil {
		logger.Error("[handleBranchResult] failed to parse branch output: %v", err)
		return
	}

	selectedKey := output.Output // The selected end node name or ID

	compiled := task.getCompiled()
	if compiled == nil {
		return
	}

	// Skip unselected end nodes
	for _, endNodeName := range settings.BranchConfig.EndNodes {
		if endNodeName == selectedKey {
			continue // This is the selected path, keep it active
		}
		// Resolve name to ID and skip
		for _, s := range task.subtaskMap {
			if s.GetName() == endNodeName && s.subtask.State == string(TaskPending) {
				_ = task.SkipSubtask(s.GetID())
				if err := t.SubtaskDao.SetOutputAndState(ctx, s.GetID(), "", string(TaskSkipped)); err != nil {
					logger.Error("[handleBranchResult] failed to skip subtask %s: %v", s.GetID(), err)
				}
				logger.Debug("[handleBranchResult] skipped unselected branch target %s (selected: %s)", s.GetID(), selectedKey)
				break
			}
		}
	}
}
```

- [ ] **Step 4: Call handleBranchResult in analysisTask after branch subtask completion**

In `backend/taskx/dispatch.go`, in `analysisTask()`, after the `sync task status to db` section, add:

```go
// Process completed branch subtask results
for _, subtask := range subtaskMap {
    if subtask.GetState() == string(TaskSucceeded) {
        var settings SubtaskSettings
        if err := tools.Unmarshal([]byte(subtask.subtask.Settings), &settings); err == nil && settings.BranchConfig != nil {
            t.handleBranchResult(ctx, task, subtask)
        }
    }
}
```

- [ ] **Step 5: Commit**

```bash
git add backend/taskx/dispatch.go backend/taskx/task.go
git commit -m "feat: distributed branch execution with backward-compatible legacy path"
```

---

### Task 5: Receiver Branch Routing (receiver.go)

**Files:**
- Modify: `backend/taskx/receiver.go:261-295` (execSubtask)

**Interfaces:**
- Consumes: branchExecutor (Task 2), Settings.BranchConfig detection
- Produces: Branch subtasks routed to branchExecutor instead of regular provider lookup

- [ ] **Step 1: Route branch subtasks to branchExecutor**

In `backend/taskx/receiver.go`, modify `execSubtask` to detect branch subtasks:

```go
func (t *taskReceiver) execSubtask(bag *SubtaskBag) {
	defer t.subtaskInflight.DeleteString(bag.subtask.ID)

	golocalv1.PutTraceID(bag.task.RequestID)
	defer golocalv1.Clean()
	ctx := golocalv1.GetContext()

	subtaskID := bag.subtask.ID
	taskID := bag.task.ID

	if !t.prepareSubtaskRun(ctx, bag, false, subtaskID, taskID) {
		return
	}

	// Check if this is a branch subtask
	var settings SubtaskSettings
	isBranch := false
	if bag.subtask.Settings != "" {
		if err := tools.Unmarshal([]byte(bag.subtask.Settings), &settings); err == nil && settings.BranchConfig != nil {
			isBranch = true
		}
	}

	if isBranch {
		// Execute branch condition
		selectedKey, err := executeBranchCondition(
			bag.task.TaskName,
			bag.subtask.ID,
			bag.subtask.Settings,
			bag.subtask.Input, // Input contains the parent node's output
		)
		if err != nil {
			t.persistSubtaskOutcome(ctx, bag, "", err)
		} else {
			t.persistSubtaskOutcome(ctx, bag, selectedKey, nil)
		}
		t.notifyLeader(taskID)
		return
	}

	// Regular execution path
	provider := getProvider(bag.task.TaskName, bag.subtask.TaskName)
	if provider == nil {
		t.handleExecutorMissing(ctx, bag.task, bag.subtask)
		return
	}

	execCtx, cancel := withSubtaskTimeout(ctx, bag.subtask)
	output, err := t.runExecutor(execCtx, provider,
		bag.task.TaskName, bag.subtask.TaskName,
		taskID, subtaskID, bag.task.RequestID, bag.subtask.Input)
	cancel()

	t.persistSubtaskOutcome(ctx, bag, output, err)
	t.notifyLeader(taskID)
}
```

- [ ] **Step 2: Ensure tools import is present**

Verify that `github.com/caiflower/common-tools/pkg/tools` is imported in `receiver.go`. If not, add it.

- [ ] **Step 3: Commit**

```bash
git add backend/taskx/receiver.go
git commit -m "feat: receiver routes branch subtasks to built-in branchExecutor"
```

---

### Task 6: Backward Compatibility and Tests

**Files:**
- Modify: `backend/taskx/dag_test.go`
- Modify: `backend/taskx/dispatch_test.go`
- Modify: `backend/taskx/integration_test.go`

**Interfaces:**
- Consumes: All previous tasks
- Produces: Test coverage for new and legacy branch behavior

- [ ] **Step 1: Add test for branch subtask creation in DAG**

In `backend/taskx/dag_test.go`, add:

```go
func TestAddBranchCreatesSubtaskNode(t *testing.T) {
	g := newDAGGraph()
	g.AddNode("A")
	g.AddNode("B")
	g.AddNode("C")

	branch := NewBranchFunc(
		func(ctx interface{}, input any) (string, error) { return "B", nil },
		map[string]bool{"B": true, "C": true},
	)

	err := g.AddBranch("A", branch)
	assert.NoError(t, err)

	// Verify branch subtask node exists
	branchKey := "branch_A_0"
	assert.Contains(t, g.nodes, branchKey)
	assert.Equal(t, NodePending, g.nodes[branchKey].state)

	// Verify control edges
	assert.Contains(t, g.controlAdj["A"], branchKey)
	assert.Contains(t, g.controlAdj[branchKey], "B")
	assert.Contains(t, g.controlAdj[branchKey], "C")
}
```

- [ ] **Step 2: Add test for branchExecutor error cases**

In `backend/taskx/dispatch_test.go`, add:

```go
func TestBranchExecutorMissingProvider(t *testing.T) {
	// Clear registries first
	ClearProviders("__builtin__")

	settings := SubtaskSettings{
		BranchConfig: &BranchConfig{
			EndNodes:          []string{"B", "C"},
			ConditionProvider: "nonexistent",
		},
	}
	settingsJSON := tools.ToJson(settings)

	_, err := executeBranchCondition("testTask", "branch_A_0", settingsJSON, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "condition provider not found")
}

func TestBranchExecutorInvalidSelectedKey(t *testing.T) {
	// Register a provider that returns a key not in EndNodes
	// ... test setup ...
}

func TestBranchExecutorNonStringResult(t *testing.T) {
	// Register a provider that returns a non-string
	// ... test setup ...
}
```

- [ ] **Step 3: Add integration test for end-to-end branch flow**

In `backend/taskx/integration_test.go`, add a test that creates a task with branches, submits it, and verifies:
- Branch subtask rows exist in DB
- Branch condition executes and selects the correct path
- Unselected targets are skipped
- Selected target executes successfully

- [ ] **Step 4: Run existing tests to ensure no regressions**

```bash
cd backend/taskx && go test ./... -v -count=1
```

- [ ] **Step 5: Commit**

```bash
git add backend/taskx/dag_test.go backend/taskx/dispatch_test.go backend/taskx/integration_test.go
git commit -m "test: branch subtask creation, executor errors, and integration flow"
```

---

### Task 7: Run Full Test Suite and Final Verification

- [ ] **Step 1: Run all tests**

```bash
cd backend/taskx && go test ./... -v -count=1 -timeout 120s
```

Expected: All tests pass, including new branch tests and existing regression tests.

- [ ] **Step 2: Run go build to verify compilation**

```bash
cd backend && go build ./...
```

Expected: Clean build with no errors.

- [ ] **Step 3: Commit final state**

```bash
git add -A
git commit -m "chore: final verification - all tests pass, clean build"
```
