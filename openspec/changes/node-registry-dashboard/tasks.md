## 1. Proto Definition

- [x] 1.1 Add `ListNodes` and `GetNode` RPCs with request/response messages to `remote_executor.proto`
- [x] 1.2 Regenerate Go code from proto (`protoc`)

## 2. Backend — Unified Node Storage Refactoring

- [x] 2.1 Add `LastHeartbeat int64` field to `NodeInfo` struct, remove separate heartbeat key logic
- [x] 2.2 Refactor `Register` to store `lastHeartbeat` in value, use nodeTTL (5min)
- [x] 2.3 Refactor `Heartbeat` to update `lastHeartbeat` field and refresh TTL
- [x] 2.4 Define constants: `heartbeatThreshold = 30s`, `nodeTTL = 5min`
- [x] 2.5 Update `GetNodesForFunc` to work with unified key and filter by `lastHeartbeat` freshness

## 3. Backend — List/Get API

- [x] 3.1 Implement `ListNodes` method on `NodeRegistry` (scan Redis, compute status from lastHeartbeat)
- [x] 3.2 Implement `GetNode` method on `NodeRegistry` (get single node by ID with status)
- [x] 3.3 Write unit tests for unified storage, `ListNodes`, and `GetNode` in `registry_test.go`

## 4. Backend — HTTP Routes

- [x] 4.1 Add HTTP routes `GET /api/v1/nodes` and `GET /api/v1/nodes/:id` in `router.go`
- [x] 4.2 Generate gRPC handler functions for the new NodeRegistry methods

## 5. Frontend — API Client

- [x] 5.1 Add `getNodes()` and `getNode(id)` functions to frontend API client

## 6. Frontend — Node Registry Page

- [x] 6.1 Create `NodeRegistryPage` component with table displaying node ID, address, functions, status, lastHeartbeat
- [x] 6.2 Add search/filter input to filter nodes by ID or function name
- [x] 6.3 Add auto-refresh toggle with configurable interval and manual refresh button
- [x] 6.4 Add route `/nodes` and sidebar navigation entry "节点管理"

## 7. Verification

- [x] 7.1 Run backend tests (`go test ./internal/node_registry/... ./internal/protocol/...`)
- [x] 7.2 Run backend build (`go build ./backend/cmd/server/...`)
- [x] 7.3 Run frontend build (`cd frontend && npm run build`)
