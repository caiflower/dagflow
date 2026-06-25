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

