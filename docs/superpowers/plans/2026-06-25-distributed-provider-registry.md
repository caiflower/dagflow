---
change: distributed-provider-registry
design-doc: docs/superpowers/specs/2026-06-25-distributed-provider-registry-design.md
base-ref: b78419e62d84c74e1df4f227b5eec3f18aa7a6d8
---

# Distributed Provider Registry — Implementation Plan

> **For agentic workers:** Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make providers discoverable on any cluster instance via `SetProviderFallback` + protocol metadata persistence.

**Architecture:** Single callback `SetProviderFallback` registered at app bootstrap. `getProvider` calls it on memory miss, reading protocol metadata from DB subtask settings. Provider is then cached in memory.

**Tech Stack:** Go, Redis (via miniredis for tests), gRPC/protobuf

## Global Constraints

- All existing tests must pass
- Backward compatible with single-instance deployments
- No new external dependencies

---

### Task 1: Add fallback vars and setters to executor.go

**Files:**
- Modify: `backend/taskx/executor.go`

**Interfaces:**
- Produces: `SetProviderFallback(fn)`, `SetProcessorFallback(fn)`, `_providerFallback`, `_processorFallback`

- [ ] **Step 1: Add fallback type and variables**

Add after the `_providerRegistry` definition in `executor.go`:

```go
// ProviderFallbackFn is called when a provider is not found in the in-memory registry.
// The callback should reconstruct the provider from protocol metadata stored in DB.
type ProviderFallbackFn func(protocol string, settings map[string]any) executor.ExecutorProvider

// ProcessorFallbackFn is called when a processor is not found in the in-memory registry.
type ProcessorFallbackFn func(protocol string, settings map[string]any) Processor

var (
	_providerFallback  ProviderFallbackFn
	_processorFallback ProcessorFallbackFn
	_fallbackMu        sync.RWMutex
)

// SetProviderFallback registers the fallback for provider reconstruction.
// Should be called once at application bootstrap.
func SetProviderFallback(fn ProviderFallbackFn) {
	_fallbackMu.Lock()
	defer _fallbackMu.Unlock()
	_providerFallback = fn
}

// SetProcessorFallback registers the fallback for processor reconstruction.
func SetProcessorFallback(fn ProcessorFallbackFn) {
	_fallbackMu.Lock()
	defer _fallbackMu.Unlock()
	_processorFallback = fn
}

func getProviderFallback() ProviderFallbackFn {
	_fallbackMu.RLock()
	defer _fallbackMu.RUnlock()
	return _providerFallback
}

func getProcessorFallback() ProcessorFallbackFn {
	_fallbackMu.RLock()
	defer _fallbackMu.RUnlock()
	return _processorFallback
}
```

- [ ] **Step 2: Commit**

```bash
git add backend/taskx/executor.go
git commit -m "feat(taskx): add SetProviderFallback and SetProcessorFallback"
```

---

### Task 2: Add Protocol/ProtocolConfig to SubtaskSettings

**Files:**
- Modify: `backend/taskx/dao/model/subtask.go` (or wherever `SubtaskSettings` is defined)

**Interfaces:**
- Produces: `SubtaskSettings.Protocol string`, `SubtaskSettings.ProtocolConfig map[string]any`

- [ ] **Step 1: Find SubtaskSettings definition**

Run: `grep -rn "type SubtaskSettings" backend/taskx/`

- [ ] **Step 2: Add fields**

```go
type SubtaskSettings struct {
	// ... existing fields ...
	Protocol       string         `json:"protocol,omitempty"`
	ProtocolConfig map[string]any `json:"protocolConfig,omitempty"`
}
```

- [ ] **Step 3: Commit**

```bash
git add backend/taskx/dao/model/
git commit -m "feat(taskx): add Protocol and ProtocolConfig to SubtaskSettings"
```

---

### Task 3: Persist protocol metadata in AddSubtask

**Files:**
- Modify: `backend/taskx/task.go` (AddSubtask method)

**Interfaces:**
- Consumes: `SubtaskSettings.Protocol`, `SubtaskSettings.ProtocolConfig`
- Produces: protocol metadata persisted to DB via settings

- [ ] **Step 1: Add protocol extraction in AddSubtask**

In `AddSubtask`, after the provider registration block, add:

```go
// Persist protocol metadata so other instances can reconstruct the provider
if subtask.provider != nil {
	proto := subtask.provider.Protocol()
	subtask.subtask.Settings.Protocol = string(proto)
	subtask.subtask.Settings.ProtocolConfig = extractProviderConfig(subtask.provider)
}
```

- [ ] **Step 2: Add extractProviderConfig helper**

In `task.go`:

```go
func extractProviderConfig(p executor.ExecutorProvider) map[string]any {
	if p == nil {
		return nil
	}
	config := map[string]any{
		"protocol": string(p.Protocol()),
	}
	// If provider exposes its own config, merge it
	if cp, ok := p.(executor.ConfigurableProvider); ok {
		for k, v := range cp.ProviderConfig() {
			config[k] = v
		}
	}
	return config
}
```

- [ ] **Step 3: Check if ConfigurableProvider interface exists**

Run: `grep -rn "ConfigurableProvider" backend/taskx/executor/`

If not, add it to `executor/executor.go`:

```go
type ConfigurableProvider interface {
	ExecutorProvider
	ProviderConfig() map[string]any
}
```

And implement on `RemoteFuncProvider`:

```go
func (p *RemoteFuncProvider) ProviderConfig() map[string]any {
	return map[string]any{
		"funcName": p.FuncName,
		"timeout":  int(p.Timeout.Seconds()),
	}
}
```

- [ ] **Step 4: Commit**

```bash
git add backend/taskx/task.go backend/taskx/executor/executor.go backend/internal/protocol/remote_executor/provider.go
git commit -m "feat(taskx): persist protocol metadata in AddSubtask"
```

---

### Task 4: Modify getProvider with fallback path

**Files:**
- Modify: `backend/taskx/executor.go` (getProvider, getRollbackProvider)

**Interfaces:**
- Consumes: `getProviderFallback()`, `SubtaskSettings`
- Modifies: `getProvider`, `getRollbackProvider`

- [ ] **Step 1: Modify global getProvider**

```go
func getProvider(taskName, subTaskName string) executor.ExecutorProvider {
	_providerRegistry.RLock()
	if providers, ok := _providerRegistry.providers[taskName]; ok {
		if p := providers[subTaskName]; p != nil {
			_providerRegistry.RUnlock()
			return p
		}
	}
	_providerRegistry.RUnlock()

	// Fallback: reconstruct provider from DB metadata
	fn := getProviderFallback()
	if fn == nil {
		return nil
	}
	settings := loadSubtaskSettings(taskName, subTaskName)
	if settings == nil || settings.Protocol == "" {
		return nil
	}
	p := fn(settings.Protocol, settings.ProtocolConfig)
	if p != nil {
		registerProvider(taskName, subTaskName, p)
	}
	return p
}
```

- [ ] **Step 2: Modify getRollbackProvider similarly**

Same pattern, using `getProviderFallback()`.

- [ ] **Step 3: Add loadSubtaskSettings helper**

This reads subtask settings from DB. Need to inject dao access. Use existing `SubtaskDao`:

```go
// loadSubtaskSettings reads the subtask's settings from DB for fallback reconstruction.
// Returns nil if the subtask is not found or has no protocol metadata.
func loadSubtaskSettings(taskName, subTaskName string) *SubtaskSettings {
	// Find the subtask from the global task registry or DB
	task := getTaskByTaskName(taskName)
	if task == nil {
		return nil
	}
	subtask := task.GetSubtaskByName(subTaskName)
	if subtask == nil {
		return nil
	}
	return subtask.GetSettings()
}
```

Note: This approach uses existing task/subtask references. If the task isn't loaded yet, the subtask bean from DB can be read directly via SubtaskDao.

- [ ] **Step 4: Apply same fallback to getPreProcessor / getPostProcessor**

Using `getProcessorFallback()`.

- [ ] **Step 5: Commit**

```bash
git add backend/taskx/executor.go backend/taskx/task.go
git commit -m "feat(taskx): add fallback path in getProvider for cross-instance resolution"
```

---

### Task 5: Clean up initByBean batch registration

**Files:**
- Modify: `backend/taskx/task.go` (initByBean method)

- [ ] **Step 1: Remove batch provider registration from initByBean**

Find and remove:
- `registerProvider` / `registerProviders` calls in `initByBean`
- Branch condition provider memory recovery (the `getRegisteredBranches` → `registerProvider` block)
- Any `registerRollbackProvider`, `registerPreProcessor`, `registerPostProcessor` in initByBean

- [ ] **Step 2: Remove related memory structures if only used in initByBean**

Check `getRegisteredBranches` — if only used in initByBean, remove it.

- [ ] **Step 3: Commit**

```bash
git add backend/taskx/task.go
git commit -m "refactor(taskx): remove batch provider registration from initByBean"
```

---

### Task 6: Application-layer SetProviderFallback registration

**Files:**
- Modify: `backend/cmd/server/main.go` (or wherever server bootstrap runs)
- Modify: `backend/internal/protocol/remote_executor/provider.go`

- [ ] **Step 1: Expose constructor for RemoteFuncProvider**

In `remote_executor/provider.go`, add:

```go
func NewProvider(funcName string, timeout time.Duration) *RemoteFuncProvider {
	return &RemoteFuncProvider{
		FuncName: funcName,
		Timeout:  timeout,
	}
}
```

- [ ] **Step 2: Register fallback at bootstrap**

```go
taskx.SetProviderFallback(func(protocol string, config map[string]any) executor.ExecutorProvider {
	switch protocol {
	case "remoteFunc":
		funcName, _ := config["funcName"].(string)
		timeoutSec, _ := config["timeout"].(float64)
		return remote_executor.NewProvider(funcName, time.Duration(timeoutSec)*time.Second)
	}
	return nil
})
```

- [ ] **Step 3: Commit**

```bash
git add backend/cmd/server/main.go backend/internal/protocol/remote_executor/provider.go
git commit -m "feat: register SetProviderFallback at application bootstrap"
```

---

### Task 7: Run existing tests, fix regressions

**Files:**
- All modified files

- [ ] **Step 1: Run full test suite**

```bash
cd backend && go test ./... --count=1
```

- [ ] **Step 2: Fix any failing tests**

Expected issues:
- Tests that relied on initByBean registering providers may need adjustment
- Tests that directly manipulated `_providerRegistry` may need updates

- [ ] **Step 3: Commit any test fixes**

```bash
git add -A
git commit -m "test: fix test regressions from distributed provider registry changes"
```

---

### Task 8: Add unit test for fallback logic

**Files:**
- Create/Modify: `backend/taskx/executor_test.go`

- [ ] **Step 1: Write test for memory miss → fallback → cache**

```go
func TestGetProviderFallback(t *testing.T) {
	// Clear existing registry
	ClearAllProviders()

	// Set fallback
	called := false
	SetProviderFallback(func(protocol string, config map[string]any) executor.ExecutorProvider {
		called = true
		assert.Equal(t, "remoteFunc", protocol)
		return &mockProvider{protocol: "remoteFunc"}
	})

	// ... register a subtask with protocol metadata, then call getProvider
	// Verify fallback was called and provider is now cached
}
```

- [ ] **Step 2: Run test**

```bash
cd backend && go test ./taskx/... --count=1 -run TestGetProviderFallback -v
```

- [ ] **Step 3: Commit**

```bash
git add backend/taskx/executor_test.go
git commit -m "test: add unit test for provider fallback logic"
```

---

### Task 9: Final verification

- [ ] **Step 1: Run full test suite**

```bash
cd backend && go test ./... --count=1
```

- [ ] **Step 2: Build check**

```bash
cd backend && go build ./...
```

- [ ] **Step 3: Commit if any fixes needed**
