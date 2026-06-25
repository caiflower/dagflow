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

