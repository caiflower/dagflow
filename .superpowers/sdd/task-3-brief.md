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

