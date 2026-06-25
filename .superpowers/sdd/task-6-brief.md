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

