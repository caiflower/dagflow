---
comet_change: grpc-server-improvements
role: technical-design
canonical_spec: openspec
---

# gRPC Server Improvements

## Overview

Unify the gRPC server to host all services (NodeRegistry + Flow + Protocol + Execution), rename the config port, add SDK client support, and implement graceful shutdown via `global.DaemonResource`.

## Implementation

### 1. Config rename
`GRPCConfig.NodeRegistryPort int` → `GRPCConfig.Port int`. Update `etc/config.yaml` and all references.

### 2. GrpcServer daemon

```go
type GrpcServer struct {
    server        *grpc.Server
    lis           net.Listener
    nodeReg       *node_registry.NodeRegistry
    flowSvc       *FlowGrpcService
    protocolSvc   *ProtocolGrpcService
    executionSvc  *ExecutionGrpcService
}

func (g *GrpcServer) Name() string { return "GrpcServer" }

func (g *GrpcServer) Start() error {
    g.lis, _ = net.Listen("tcp", fmt.Sprintf(":%d", port))
    g.server = grpc.NewServer()
    // Register all 4 services
    pbNode.RegisterNodeRegistryServer(g.server, g.nodeReg)
    pbFlow.RegisterFlowServer(g.server, g.flowSvc)
    pbProto.RegisterProtocolServer(g.server, g.protocolSvc)
    pbExec.RegisterExecutionServer(g.server, g.executionSvc)
    return g.server.Serve(g.lis)
}

func (g *GrpcServer) Close() {
    g.server.GracefulStop()
}
```

Registered as: `global.DefaultResourceManger.AddDaemon(grpcServer)`

### 3. SDK client

```go
type Client struct {
    conn *grpc.ClientConn
    Flow flowpb.FlowClient
    Protocol protopb.ProtocolClient
    Execution execpb.ExecutionClient
}

func NewClient(addr string) (*Client, error) { ... }
```

### 4. Registry

Export service registration from `internal/api/` via setter pattern already in place.

## Files changed

| File | Change |
|------|--------|
| `constants/config.go` | `NodeRegistryPort` → `Port` |
| `cmd/server/main.go` | `startNodeRegistryServer` → `GrpcServer` daemon |
| `internal/api/` | Minor: expose service getters |
| `clients/go/sdk.go` | Add `Client` struct with gRPC methods |
| `etc/config.yaml` | Config key rename |
