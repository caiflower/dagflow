## Why

The current gRPC setup has three issues: (1) `node_registry_port` is misleading — it's the general gRPC server port, not specific to node registry; (2) Flow/Protocol/Execution gRPC service implementations exist but only serve HTTP via `web.GRPC()` handler, not registered on the actual gRPC server; (3) the gRPC server is started in a raw goroutine without graceful shutdown support via `global.DefaultResourceManger`.

## What Changes

- Rename `node_registry_port` → `port` in `GRPCConfig` (breaking config change)
- Register Flow/Protocol/Execution gRPC services on the gRPC server alongside NodeRegistry
- Add SDK client support for calling Flow/Protocol/Execution gRPC services
- Wrap gRPC server as a `global.DaemonResource` for graceful shutdown via `global.DefaultResourceManger`

## Capabilities

### New Capabilities
- `grpc-server-unified`: Unified gRPC server hosting all services (NodeRegistry + Flow + Protocol + Execution) with graceful shutdown
- `grpc-sdk-client`: Go SDK support for calling DAGFlow gRPC services (Flow, Protocol, Execution)

### Modified Capabilities
<!-- None — existing HTTP API behavior unchanged -->

## Impact

- `constants/config.go` — rename `NodeRegistryPort` → `Port`
- `cmd/server/main.go` — refactor `startNodeRegistryServer` → unified gRPC daemon
- `internal/api/` — export gRPC service registrations for server setup
- `clients/go/sdk.go` — add Flow/Protocol/Execution gRPC client methods
- `etc/config.yaml` / `etc/default.yaml` — config key rename
