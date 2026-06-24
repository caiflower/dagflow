## Context

Remote function nodes register with DAGFlow via the `NodeRegistry` gRPC service, storing `{nodeId, address, functions}` in Redis with a 30-second TTL. Heartbeats refresh the TTL. Currently, no API or UI exposes this data — operators must inspect Redis directly.

The frontend is a React + Vite + TypeScript app using `@xyflow/react`, `react-router-dom`, `zustand`, `axios`, and Tailwind CSS. Existing pages: FlowList, FlowEditor, ExecutionPage, ProtocolPage.

## Goals / Non-Goals

**Goals:**
- Expose registered node data (ID, address, functions, online status) via gRPC and HTTP APIs
- Display nodes in a frontend data-center-style table with online/offline indicators
- Support search/filter and auto-refresh

**Non-Goals:**
- Node management (deregister, edit) — read-only visualization only
- WebSocket/push-based updates — polling with configurable interval is sufficient
- Historical data or metrics — current state only

## Decisions

### 1. Extend existing NodeRegistry gRPC service (not separate service)

**Rationale**: The `NodeRegistry` service in `remote_executor.proto` already owns node data. Adding `ListNodes` and `GetNode` RPCs keeps related functionality together, avoids new service registration overhead, and follows the same pattern as `ProtocolService` (List + Get).

### 2. Use existing gRPC handler pattern for HTTP routes

**Rationale**: The project already uses `engine.GRPC()` to bridge gRPC handlers to HTTP routes (see `router.go`). Adding `/api/v1/nodes` and `/api/v1/nodes/:id` through the same pattern is consistent and requires no new controller layer.

**Alternative considered**: Creating a separate REST controller. Rejected because it would duplicate logic and break the established pattern.

### 3. Unified node key with `lastHeartbeat` field

**Problem**: Current implementation stores node info and heartbeat in two separate Redis keys with the same 30s TTL. When heartbeat stops, both keys expire simultaneously, making it impossible to distinguish "offline" from "deregistered".

**Solution**: Merge into a single key with longer TTL:

```
Key: dagflow:node:<nodeId>
Value: {nodeId, address, functions, lastHeartbeat (unix timestamp)}
TTL: 5 minutes (nodeTTL)

Register → SET key with lastHeartbeat=now, TTL=5min
Heartbeat → update lastHeartbeat field, refresh TTL to 5min

Status derivation:
  🟢 online:  now - lastHeartbeat < 30s  (heartbeatThreshold)
  🔴 offline: now - lastHeartbeat >= 30s (heartbeat stopped, data persists)
  ⬜ gone:    key expired (5min without heartbeat)
```

This requires:
- `NodeInfo` struct: add `LastHeartbeat int64` field
- `Register`: store `lastHeartbeat` in the value
- `Heartbeat`: update `lastHeartbeat` field, refresh TTL
- `ListNodes`/`GetNode`: compute status from `lastHeartbeat` vs threshold

### 4. Frontend table component with zustand store

**Rationale**: The project already uses zustand for state management. A `nodeRegistryStore` with fetch/auto-refresh logic keeps the pattern consistent. No new dependencies needed.

### 5. Proto message: `NodeInfo` with `status` field

The existing `RegisterRequest` has `node_id`, `address`, `functions`. The new response adds `status` and `last_heartbeat` fields (computed/read from Redis).

## Risks / Trade-offs

- **[Risk] Redis SCAN can be slow with many keys** → Mitigation: Pagination support; scan limit 100 per call. For large deployments, consider adding a Redis SET index for faster listing.
- **[Risk] `lastHeartbeat` update requires read-modify-write** → Mitigation: Redis SET is atomic; the heartbeat update path is low-concurrency (one heartbeat per node per ~15s)
- **[Risk] TTL discrepancy** → Mitigation: nodeTTL (5min) must always be > heartbeatThreshold (30s); enforced as constants
