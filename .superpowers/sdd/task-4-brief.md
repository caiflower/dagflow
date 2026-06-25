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

