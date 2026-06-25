---
comet_change: distributed-provider-registry
role: technical-design
canonical_spec: openspec
---

# Distributed Provider Registry — Technical Design

## Problem

Provider registration happens in-memory during `AddSubtask()` (called during `Run()`). In multi-instance deployment, `Run` on instance A registers providers in A's memory. When the receiver picks up the subtask on instance B, `getProvider()` returns nil.

## Approach

`SetProviderFallback` single callback + protocol metadata persistence. Lazy reconstruction on cache miss.

## Core Types

### Fallback Registration

```go
// taskx/executor.go

var _providerFallback func(protocol string, settings map[string]any) executor.ExecutorProvider
var _processorFallback func(protocol string, settings map[string]any) Processor

func SetProviderFallback(fn func(protocol string, settings map[string]any) executor.ExecutorProvider) {
    _providerFallback = fn
}
func SetProcessorFallback(fn func(protocol string, settings map[string]any) Processor) {
    _processorFallback = fn
}
```

### SubtaskSettings Extension

```go
type SubtaskSettings struct {
    Protocol       string         `json:"protocol,omitempty"`
    ProtocolConfig map[string]any `json:"protocolConfig,omitempty"`
    // ... existing fields
}
```

## Data Flow

```
AddSubtask (Run instance)
  └── persist protocol metadata to Settings → DB
      (no memory registration via this path)

getProvider (any instance)
  ├── 1. memory hit → return
  └── 2. memory miss
       ├── read DB subtask Settings
       ├── _providerFallback(protocol, config) → provider
       ├── registerProvider() → cache in memory
       └── return provider
```

## What Gets Cleaned Up

Removed from `initByBean`:
- Batch `registerProvider` / `registerProviders` calls
- Branch condition provider memory recovery logic
- All duplicate registration paths

## Registry Coverage

| Registry | Fallback | Lookup Function |
|----------|----------|-----------------|
| `_providerRegistry` | `_providerFallback` | `getProvider` |
| `_rollbackRegistry` | `_providerFallback` | `getRollbackProvider` |
| `_preProcessorRegistry` | `_processorFallback` | `getPreProcessor` |
| `_postProcessorRegistry` | `_processorFallback` | `getPostProcessor` |
| `_taskExecutorRegistry` | N/A | dispatcher-side only |

## Application Integration

```go
// cmd/server bootstrap
taskx.SetProviderFallback(func(protocol string, config map[string]any) executor.ExecutorProvider {
    switch protocol {
    case "remoteFunc":
        return remote_executor.NewProvider(
            config["funcName"].(string),
            time.Duration(config["timeout"].(float64)) * time.Second,
        )
    }
    return nil
})
```

## Files Changed

| File | Change |
|------|--------|
| `taskx/executor.go` | Add `SetProviderFallback`, `SetProcessorFallback`, fallback vars; modify `getProvider`, `getRollbackProvider`, `getPreProcessor`, `getPostProcessor` |
| `taskx/task.go` | `AddSubtask`: persist protocol metadata; `initByBean`: remove batch registration |
| `taskx/dao/model/` | `SubtaskSettings`: add `Protocol`, `ProtocolConfig` fields |
| `cmd/server/main.go` | Call `SetProviderFallback` at bootstrap |
| `internal/protocol/remote_executor/` | Expose `NewProvider` constructor if needed |

## Concurrency Safety

- Memory miss + reconstruction + re-registration is protected by `registerProvider`'s existing `_providerRegistry.Lock()`
- `_providerFallback` is set once at startup (happens-before all reads)

## Testing Strategy

- Unit: mock fallback, verify memory miss → fallback call → cached
- Integration: simulate cross-instance scenario, B rebuilds provider from DB metadata
- Regression: all existing tests must pass
