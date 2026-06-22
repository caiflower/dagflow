## Approach

### 1. Config rename
`GRPCConfig.NodeRegistryPort` → `GRPCConfig.Port`. Update all references and config YAML files.

### 2. Unified gRPC server
Replace `startNodeRegistryServer()` with a `GrpcServer` daemon that:
- Creates a single `grpc.Server`
- Registers all 4 services: `NodeRegistryServer`, `FlowServer`, `ProtocolServer`, `ExecutionServer`
- Implements `global.DaemonResource` (Name, Start, Close)
- Registered via `global.DefaultResourceManger.AddDaemon()`

### 3. SDK client
Add methods to `clients/go/sdk.go`:
- `CreateFlow`, `ListFlows`, `GetFlow`, `UpdateFlow`, `DeleteFlow`, `ValidateFlow`
- `ListProtocols`, `GetProtocol`
- `RunExecution`, `GetExecution`, `ListExecutions`

Each dials the engine's gRPC port and calls the corresponding service.

### 4. Graceful shutdown
`GrpcServer.Close()` calls `s.GracefulStop()` then `s.Stop()` with timeout. Managed by `global.DefaultResourceManger.Signal()`.

## Risks

- Config key rename is breaking — update `etc/config.yaml` and `etc/default.yaml`
- gRPC server port must not conflict with HTTP port
