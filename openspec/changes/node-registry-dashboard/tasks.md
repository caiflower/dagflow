## 1. Proto Definition

- [ ] 1.1 Add `ListNodes` and `GetNode` RPCs with request/response messages to `remote_executor.proto`
- [ ] 1.2 Regenerate Go code from proto (`protoc`)

## 2. Backend — Unified Node Storage Refactoring

- [ ] 2.1 Add `LastHeartbeat int64` field to `NodeInfo` struct, remove separate heartbeat key logic
- [ ] 2.2 Refactor `Register` to store `lastHeartbeat` in value, use nodeTTL (5min)
- [ ] 2.3 Refactor `Heartbeat` to update `lastHeartbeat` field and refresh TTL
- [ ] 2.4 Define constants: `heartbeatThreshold = 30s`, `nodeTTL = 5min`
- [ ] 2.5 Update `GetNodesForFunc` to work with unified key and filter by `lastHeartbeat` freshness

## 3. Backend — List/Get API

- [ ] 3.1 Implement `ListNodes` method on `NodeRegistry` (scan Redis, compute status from lastHeartbeat)
- [ ] 3.2 Implement `GetNode` method on `NodeRegistry` (get single node by ID with status)
- [ ] 3.3 Write unit tests for unified storage, `ListNodes`, and `GetNode` in `registry_test.go`

## 4. Backend — HTTP Routes

- [ ] 4.1 Add HTTP routes `GET /api/v1/nodes` and `GET /api/v1/nodes/:id` in `router.go`
- [ ] 4.2 Generate gRPC handler functions for the new NodeRegistry methods

## 5. Frontend — API Client

- [ ] 5.1 Add `getNodes()` and `getNode(id)` functions to frontend API client

## 6. Frontend — Node Registry Page

- [ ] 6.1 Create `NodeRegistryPage` component with table displaying node ID, address, functions, status, lastHeartbeat
- [ ] 6.2 Add search/filter input to filter nodes by ID or function name
- [ ] 6.3 Add auto-refresh toggle with configurable interval and manual refresh button
- [ ] 6.4 Add route `/nodes` and sidebar navigation entry "节点管理"

## 7. Verification

- [ ] 7.1 Run backend tests (`go test ./internal/node_registry/... ./internal/protocol/...`)
- [ ] 7.2 Run backend build (`go build ./backend/cmd/server/...`)
- [ ] 7.3 Run frontend build (`cd frontend && npm run build`)
