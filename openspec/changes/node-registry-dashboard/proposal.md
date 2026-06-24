## Why

Remote function protocol allows third-party nodes to register with DAGFlow, but operators have no visibility into what nodes are registered, what functions they provide, or their health status. All registration data is stored silently in Redis with no API or UI to access it. This makes debugging, monitoring, and managing the remote execution cluster impossible without direct Redis access.

## What Changes

- Add `ListNodes` and `GetNode` methods to the `NodeRegistry` gRPC service, returning registered node information with heartbeat-based health status
- Expose HTTP endpoints `GET /api/v1/nodes` and `GET /api/v1/nodes/:id` through the existing API router
- Add a `NodeRegistryPage` in the frontend displaying a data-center-style table of registered nodes with their ID, address, functions, and online/offline status
- Support auto-refresh and search/filter on the frontend page for real-time monitoring
- Add a sidebar navigation entry for the new page

## Capabilities

### New Capabilities
- `node-registry-api`: Backend API to list and query registered remote function nodes with health status
- `node-registry-dashboard`: Frontend page that visualizes registered nodes in a table, with online/offline status, search, and auto-refresh

### Modified Capabilities
<!-- None -->

## Impact

- **Backend**: New methods on `NodeRegistry` (proto + Go implementation), new HTTP routes in `router.go`, new gRPC service registration in `main.go`
- **Frontend**: New `NodeRegistryPage` component, new route in App.tsx, new sidebar entry in Layout.tsx
- **Proto**: Update `remote_executor.proto` with `ListNode`/`GetNode` RPCs
- **Redis**: Read-only access to existing `dagflow:node:*` and `dagflow:node:heartbeat:*` keys — no schema changes
