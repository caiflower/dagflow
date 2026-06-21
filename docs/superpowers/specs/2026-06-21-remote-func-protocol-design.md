---
comet_change: remote-func-protocol
role: technical-design
canonical_spec: openspec
archived-with: 2026-06-21-remote-func-protocol
status: final
---

# RemoteFunc Protocol — Technical Design

## 1. Overview

Add `remoteFunc` protocol to DAGFlow, enabling third-party applications to connect as worker nodes via gRPC and execute tasks dispatched by the engine.

### Architecture

```
┌──────────────────────────────────────────────────────┐
│                   DAGFlow Engine                     │
│                                                      │
│  ┌──────────────────┐    ┌──────────────────────┐    │
│  │  NodeRegistry    │    │  RemoteExecutor      │    │
│  │  (gRPC server)   │    │  Provider            │    │
│  │                  │    │                      │    │
│  │  Register  ◀─────│────│── Execute ──────────▶│    │
│  │  Heartbeat ◀─────│    │  (gRPC client)       │    │
│  └────────┬─────────┘    └──────────┬───────────┘    │
│           │                         │                │
│      ┌────▼────┐               ┌───▼────┐           │
│      │  Redis  │               │ random │           │
│      │ dagflow:│               │  pick  │           │
│      │  node:* │               └───┬────┘           │
│      └─────────┘                   │                │
└─────────────────────────────────────┼────────────────┘
                                      │ Execute(func, input)
                  ┌───────────────────┼──────────────────┐
                  │                   ▼                  │
                  │  ┌───────────────────────────┐       │
                  │  │     Third-party App        │       │
                  │  │  ┌─────────────────────┐   │       │
                  │  │  │    Go SDK (clients/go)   │   │       │
                  │  │  │  Register[In,Out]()  │   │       │
                  │  │  │  Start()             │   │       │
                  │  │  └─────────────────────┘   │       │
                  │  └───────────────────────────┘       │
                  └──────────────────────────────────────┘
```

## 2. Project Structure

```
dagflow/
├── backend/internal/
│   ├── proto/remote_executor/
│   │   ├── remote_executor.proto  ← shared proto definition
│   │   ├── *.pb.go                ← generated
│   │   └── *_grpc.pb.go           ← generated
│   ├── protocol/
│   │   └── remote_func.go         ← RemoteFuncProtocol (ProtocolFactory)
│   ├── node_registry/
│   │   └── registry.go            ← Register/Heartbeat gRPC server + Redis ops
│   ├── remote_executor/
│   │   ├── client.go              ← gRPC connection pool
│   │   └── provider.go            ← ExecutorProvider + Rand-Robin scheduling
│   └── service/
│       └── providers.go           ← createProvider: add remoteFunc case
├── clients/go/                        ← Go SDK (independent go module)
│   ├── go.mod
│   ├── sdk.go                     ← New / Register[In,Out] / Start
│   └── examples/demo/main.go      ← integration example
└── frontend/src/
    ├── utils/stateColor.ts        ← remoteFunc color mapping
    └── components/dag/nodes/      ← icon + config form
```

## 3. Proto Definition

```protobuf
syntax = "proto3";
package dagflow.remote_executor;
option go_package = "github.com/caiflower/dagflow/internal/proto/remote_executor";

service RemoteExecutor {
  rpc Execute(ExecuteRequest) returns (ExecuteResponse);
  rpc HealthCheck(HealthRequest) returns (HealthResponse);
}

message ExecuteRequest {
  string func_name = 1;
  bytes input = 2;
  string task_id = 3;
}

message ExecuteResponse {
  bytes output = 1;
  string error = 2;
}

message HealthRequest {}
message HealthResponse { bool ok = 1; }

service NodeRegistry {
  rpc Register(RegisterRequest) returns (RegisterResponse);
  rpc Heartbeat(HeartbeatRequest) returns (HeartbeatResponse);
}

message FunctionInfo {
  string name = 1;
  int32 timeout = 2;
}

message RegisterRequest {
  string node_id = 1;
  string addr = 2;
  repeated FunctionInfo functions = 3;
}

message RegisterResponse { bool ok = 1; }

message HeartbeatRequest { string node_id = 1; }
message HeartbeatResponse { bool ok = 1; }
```

## 4. Redis Design

```
Prefix: dagflow:node:

dagflow:node:heartbeat:{nodeId}  → SETEX 30s  → {"addr":"...","functions":[...]}
dagflow:node:funcs:{funcName}    → SADD {nodeId}

Lifecycle:
  Register: SETEX heartbeat + SADD funcs for each function
  Heartbeat: EXPIRE heartbeat (refresh TTL only)
  Offline:   heartbeat key expires → node removed from discovery
```

## 5. NodeRegistry Implementation

Located in `backend/internal/node_registry/registry.go`.

```go
type NodeRegistry struct {
    pb.UnimplementedNodeRegistryServer
    redis v2.RedisClient
}

func (r *NodeRegistry) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
    meta, _ := json.Marshal(NodeMeta{Addr: req.Addr, Functions: req.Functions})
    r.redis.SetEX(ctx, heartbeatKey(req.NodeId), meta, 30*time.Second)
    for _, fn := range req.Functions {
        r.redis.SAdd(ctx, funcsKey(fn.Name), req.NodeId)
    }
    return &pb.RegisterResponse{Ok: true}, nil
}

func (r *NodeRegistry) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
    r.redis.Expire(ctx, heartbeatKey(req.NodeId), 30*time.Second)
    return &pb.HeartbeatResponse{Ok: true}, nil
}

func (r *NodeRegistry) GetNodesForFunc(funcName string) ([]NodeMeta, error) {
    nodeIds := r.redis.SMembers(ctx, funcsKey(funcName))
    var alive []NodeMeta
    for _, nid := range nodeIds {
        meta, err := r.redis.Get(ctx, heartbeatKey(nid))
        if err == nil && meta != "" {
            var nm NodeMeta
            json.Unmarshal([]byte(meta), &nm)
            nm.NodeID = nid
            alive = append(alive, nm)
        }
    }
    return alive, nil
}
```

## 6. RemoteExecutor Provider

Located in `backend/internal/remote_executor/`.

```go
type RemoteFuncProvider struct {
    funcName string
    timeout  time.Duration
    registry *node_registry.NodeRegistry
    pool     *connPool
}

func (p *RemoteFuncProvider) Execute(ctx context.Context, data *executor.TaskData) (any, error) {
    nodes := p.registry.GetNodesForFunc(p.funcName)
    if len(nodes) == 0 {
        return nil, fmt.Errorf("no available node for function %q", p.funcName)
    }
    // Rand-Robin: randomly pick one node
    node := nodes[rand.Intn(len(nodes))]
    
    ctx, cancel := context.WithTimeout(ctx, p.timeout)
    defer cancel()
    
    resp, err := p.pool.GetClient(node.NodeID, node.Addr).Execute(ctx, &pb.ExecuteRequest{
        FuncName: p.funcName,
        Input:    []byte(data.Input),
        TaskId:   data.TaskId,
    })
    if err != nil { return nil, err }
    if resp.Error != "" { return nil, fmt.Errorf(resp.Error) }
    
    var result any
    json.Unmarshal(resp.Output, &result)
    return result, nil
}

func (p *RemoteFuncProvider) Protocol() executor.Protocol {
    return "remoteFunc"
}
```

### Connection Pool

```go
type connPool struct {
    conns map[string]*grpc.ClientConn
    mu    sync.RWMutex
}

func (p *connPool) GetClient(nodeId, addr string) (pb.RemoteExecutorClient, error) {
    p.mu.RLock()
    conn, ok := p.conns[nodeId]
    p.mu.RUnlock()
    if ok { return pb.NewRemoteExecutorClient(conn), nil }
    
    p.mu.Lock()
    defer p.mu.Unlock()
    // double-check after acquiring write lock
    if conn, ok = p.conns[nodeId]; ok { return pb.NewRemoteExecutorClient(conn), nil }
    
    conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(5*time.Second))
    if err != nil { return nil, err }
    p.conns[nodeId] = conn
    return pb.NewRemoteExecutorClient(conn), nil
}
```

## 7. Go SDK Design

Located in `clients/go/`.

### Core API

```go
type SDK struct {
    config   Config
    handlers map[string]handlerFunc
    server   *grpc.Server
    engine   pb.NodeRegistryClient
}

type Config struct {
    NodeID     string
    EngineAddr string
    ListenAddr string // empty = random port
}

func New(cfg Config) *SDK
```

### Generic Register

```go
func Register[In, Out any](s *SDK, name string, fn func(ctx context.Context, input In) (Out, error)) {
    wrapper := func(ctx context.Context, raw []byte) ([]byte, error) {
        var in In
        json.Unmarshal(raw, &in)
        out, err := fn(ctx, in)
        if err != nil { return nil, err }
        return json.Marshal(out)
    }
    s.handlers[name] = wrapper
}
```

### Start Flow

```
Start(ctx):
  1. net.Listen → gRPC server (RemoteExecutor)
  2. engine.Register(nodeId, addr, functions)
  3. go heartbeat loop (10s ticker → engine.Heartbeat)
  4. block until ctx.Done()
  5. server.GracefulStop()
```

### Execute Handler

```go
func (s *SDK) Execute(ctx context.Context, req *pb.ExecuteRequest) (*pb.ExecuteResponse, error) {
    handler, ok := s.handlers[req.FuncName]
    if !ok {
        return &pb.ExecuteResponse{Error: fmt.Sprintf("function %q not registered", req.FuncName)}, nil
    }
    output, err := handler(ctx, req.Input)
    if err != nil {
        return &pb.ExecuteResponse{Error: err.Error()}, nil
    }
    return &pb.ExecuteResponse{Output: output}, nil
}
```

## 8. Protocol Registration

```go
// remote_func.go
type RemoteFuncProtocol struct{}

func (p *RemoteFuncProtocol) Name() string       { return "remoteFunc" }
func (p *RemoteFuncProtocol) DisplayName() string { return "Remote Function" }
func (p *RemoteFuncProtocol) Description() string {
    return "调度远程三方节点执行函数"
}
func (p *RemoteFuncProtocol) ConfigSchema() ConfigSchema {
    return ConfigSchema{Fields: []ConfigField{
        {Name: "funcName", Label: "函数名", Type: "string", Required: true, Description: "远程函数名称"},
        {Name: "timeout", Label: "超时(秒)", Type: "number", Required: false, Default: "30"},
    }}
}
```

Registered in `RegisterBuiltinProtocols()` alongside existing protocols.

## 9. Provider Factory Integration

```go
// providers.go: createProvider
case "remoteFunc":
    funcName, _ := config["funcName"].(string)
    timeout := 30 * time.Second
    if t, ok := config["timeout"].(float64); ok && t > 0 {
        timeout = time.Duration(t) * time.Second
    }
    return &remote_executor.RemoteFuncProvider{
        FuncName: funcName,
        Timeout:  timeout,
        Registry: nodeRegistry,
        Pool:     remoteExecutorPool,
    }, nil
```

## 10. Testing

### Unit Tests

| File | Scope |
|------|-------|
| `node_registry/registry_test.go` | Register writes Redis keys, Heartbeat refreshes TTL, GetNodesForFunc filters alive |
| `remote_executor/provider_test.go` | Rand-Robin selection, no-nodes error, Execute mock |
| `clients/go/sdk_test.go` | Generic Register type conversion, Execute handler dispatch, missing func error |

### Integration Tests

Single file `remote_executor/provider_integration_test.go` using `miniredis`:

| Scenario | Verification |
|----------|-------------|
| Single node Execute | SDK registers → provider calls Execute → correct output returned |
| Multi-node random pick | 2 nodes register same func → multiple calls hit both nodes |
| Node offline detection | Heartbeat expires → GetNodesForFunc returns empty |
| Function not found | Execute unregistered func → error message |
| Concurrent execution | Multiple goroutines Execute → no race conditions |

### Local Verification

```
1. go run backend/cmd/server/  (starts engine + NodeRegistry on :50051)
2. go run clients/go/examples/demo/  (starts SDK node, registers processImage)
3. Create DAG Flow via frontend: start → remoteFunc(processImage) → end
4. Execute Flow → verify SDK returns result
```

## 11. Frontend Changes

- `stateColor.ts`: add `remoteFunc: { bg, border, text }` color entry
- `nodes/index.tsx`: add `remoteFunc` → `Terminal` icon mapping
- Config form: renders `funcName` (text input) + `timeout` (number input) from `ConfigSchema`

## File Change Summary

| File | Action | Description |
|------|--------|-------------|
| `backend/internal/proto/remote_executor/*.proto` | New | Proto definition |
| `backend/internal/proto/remote_executor/*.pb.go` | Generated | gRPC stubs |
| `backend/internal/protocol/remote_func.go` | New | Protocol registration |
| `backend/internal/node_registry/registry.go` | New | NodeRegistry gRPC server |
| `backend/internal/remote_executor/client.go` | New | gRPC connection pool |
| `backend/internal/remote_executor/provider.go` | New | ExecutorProvider + scheduling |
| `backend/internal/service/providers.go` | Modify | Add remoteFunc case |
| `clients/go/go.mod` | New | SDK module |
| `clients/go/sdk.go` | New | SDK core |
| `clients/go/examples/demo/main.go` | New | Demo app |
| `backend/internal/node_registry/registry_test.go` | New | Unit tests |
| `backend/internal/remote_executor/provider_test.go` | New | Unit tests |
| `backend/internal/remote_executor/provider_integration_test.go` | New | Integration tests |
| `clients/go/sdk_test.go` | New | SDK tests |
| `frontend/src/utils/stateColor.ts` | Modify | remoteFunc color |
| `frontend/src/components/dag/nodes/index.tsx` | Modify | remoteFunc icon |
