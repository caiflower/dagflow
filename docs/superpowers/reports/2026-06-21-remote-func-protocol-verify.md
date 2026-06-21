# Verification Report: remote-func-protocol

**Date:** 2026-06-21  
**Change:** `openspec/changes/remote-func-protocol/`  
**Result:** PASS

## 1. Test Results (all verified locally)

| Test Suite | Tests | Result |
|-----------|-------|--------|
| NodeRegistry unit | 4 (Register/HB/TTL/NoMatch) | ✅ PASS |
| RemoteExecutor unit + integration | 12 (NoNodes/WrongFunc/Success/Error/Timeout/ConnReuse/EmptyInput + SingleNode/MultiNode/HBTimeout/Unregistered) | ✅ PASS |
| SDK unit | 3 (Dispatch/MissingFunc/MultiRegister) | ✅ PASS |
| **Total** | **19** | **ALL PASS** |

Run command:
```bash
cd backend && go test ./internal/node_registry/... ./internal/remote_executor/... -v -count=1
cd clients/go && go test ./... -v -count=1
```

## 2. Design Fidelity

| Design Decision | Implementation | Match |
|----------------|---------------|-------|
| Push mode + sync Execute | `RemoteFuncProvider.Execute` calls gRPC client synchronously | ✅ |
| Heartbeat via Redis TTL (30s) | `SETEX dagflow:node:heartbeat:{nodeId} 30` + `Expire` | ✅ |
| SDK Register[In,Out] generic | `sdk.Register[In, Out any](s, name, fn)` with JSON wrapper | ✅ |
| Random scheduling | `nodes[rand.Intn(len(nodes))]` | ✅ |
| Proto shared | `clients/go` imports `github.com/caiflower/dagflow/proto/remote_executor` | ✅ |
| Config: funcName + timeout | `RemoteFuncProtocol.ConfigSchema()` | ✅ |
| Redis via DefaultConfig | `GetRedisConfigByName("dagflow")` + `v2.NewRedisClient(*cfg)` | ✅ |
| gRPC port in config.yaml | `constants.Prop.GRPC.NodeRegistryPort` | ✅ |
| Embedded miniredis debug | `redis_embedded: true` → `miniredis.Run()` → `v2.NewRedisClient()` | ✅ |

### Implementation Divergence (justified)

1. **Proto location**: `backend/internal/proto/` → `backend/proto/` — Go `internal` packages cannot be imported by external modules
2. **ExecutorServer export**: Exported as `ExecutorServer` with `Sdk` field for testability
3. **Integration tests**: In `backend/internal/remote_executor/` using `bufconn`+`miniredis` (in-process, no external deps)
4. **NodeRegistry init**: In `initGrpcServices()` (after cluster ready) instead of `initBean()`
5. **SDK naming**: `clients/go/` (module `github.com/caiflower/dagflow/clients/go`) for multi-language extensibility
6. **Go workspace**: `go.work` at root + `replace` in clients/go.mod for local dev

## 3. File Inventory

| Category | Files |
|----------|-------|
| Proto | `backend/proto/remote_executor/` (4 files) |
| Protocol | `backend/internal/protocol/remote_func.go`, `registry.go` |
| NodeRegistry | `backend/internal/node_registry/registry.go`, `registry_test.go` |
| Provider | `backend/internal/remote_executor/` (4 files: client, provider, provider_test, integration_test) |
| Service | `backend/internal/service/providers.go` |
| Server | `backend/cmd/server/main.go` |
| SDK | `clients/go/` (5 files: go.mod, go.sum, sdk.go, sdk_test.go, examples/demo) |
| Config | `backend/etc/default.yaml`, `config.yaml`, `constants/config.go` |
| Frontend | `frontend/src/utils/stateColor.ts`, `nodes/index.tsx` |
| Workspace | `go.work` |

## 4. Build & Type Check

| Check | Result |
|-------|--------|
| `go build ./cmd/server/` | ✅ PASS |
| `npx tsc --noEmit` (frontend) | ✅ PASS |
| `gofmt -e .` | ✅ PASS |

## 5. Security Review

- No hardcoded credentials
- gRPC uses insecure (internal cluster)
- Redis keys use `dagflow:` prefix namespace
- `redis_embedded: true` for zero-dependency local dev

## 6. Recommendation

**PASS** — 19 tests pass, design matches spec, all config externalized. Ready to archive.
