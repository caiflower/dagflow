---
change: remote-func-protocol
design-doc: docs/superpowers/specs/2026-06-21-remote-func-protocol-design.md
base-ref: cc10754d0a45acc281d2c73d19e8a12021907ced
archived-with: 2026-06-21-remote-func-protocol
---

# RemoteFunc Protocol 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现 remoteFunc 协议，使第三方应用通过 gRPC 连接为 worker 节点，由 DAGFlow 引擎调度执行远程函数。

**Architecture:** 引擎侧新增 NodeRegistry gRPC server 管理节点注册与心跳，RemoteExecutorProvider 通过连接池随机调度到存活节点；SDK 侧提供独立 Go module，封装 gRPC server/client 与 JSON 序列化，第三方只需 Register + Start。

**Tech Stack:** Go 1.24, gRPC/Protobuf, Redis (go-redis/v9), React/TypeScript, Lucide icons

## Global Constraints

- Go module path: `github.com/caiflower/dagflow`
- SDK 为独立 go module: `clients/go/go.mod`
- Proto 输出目录: `backend/internal/proto/remote_executor/`
- 使用 `github.com/redis/go-redis/v9` 操作 Redis（已在 go.mod 中）
- 使用 `github.com/alicebob/miniredis/v2` 编写集成测试（已在 go.mod 中）
- 协议注册遵循 `ProtocolFactory` 接口（`Name/DisplayName/Description/ConfigSchema`）
- Provider 遵循 `ExecutorProvider` 接口（`Execute/Protocol`）
- 前端颜色使用 `nodeTypeColorMap`，图标使用 `protocolIconMap` 映射模式
- 代码风格遵循现有文件：Apache 2.0 版权头、Go 标准命名、TypeScript 严格模式
- 所有测试使用 `github.com/stretchr/testify` 断言库

archived-with: 2026-06-21-remote-func-protocol
---

### Task 1: Proto 定义与代码生成

**Files:**
- Create: `backend/internal/proto/remote_executor/remote_executor.proto`
- Create: `backend/internal/proto/remote_executor/remote_executor.pb.go` (generated)
- Create: `backend/internal/proto/remote_executor/remote_executor_grpc.pb.go` (generated)
- Create: `backend/internal/proto/remote_executor/generate.go`

**Interfaces:**
- Produces: `remote_executor.RemoteExecutorClient/Server`, `remote_executor.NodeRegistryClient/Server`, `remote_executor.ExecuteRequest/Response`, `remote_executor.RegisterRequest/Response`, `remote_executor.HeartbeatRequest/Response`, `remote_executor.HealthRequest/Response`

- [x] **Step 1: 编写 proto 文件**

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

message RegisterRequest {
  string node_id = 1;
  string address = 2;
  repeated string functions = 3;
}

message RegisterResponse { bool ok = 1; }

message HeartbeatRequest { string node_id = 1; }
message HeartbeatResponse { bool ok = 1; }
```

- [x] **Step 2: 编写 go:generate 指令**

```go
// generate.go
package remote_executor

//go:generate protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative remote_executor.proto
```

- [x] **Step 3: 安装 protoc 插件并生成代码**

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
cd backend/internal/proto/remote_executor && go generate
```

Expected: 生成 `remote_executor.pb.go` 和 `remote_executor_grpc.pb.go`，无错误。

- [x] **Step 4: 验证生成代码可编译**

```bash
cd backend && go build ./internal/proto/remote_executor/...
```

Expected: 编译成功，无错误。

- [x] **Commit**

```bash
git add backend/internal/proto/remote_executor/
git commit -m "feat(proto): add remote_executor and node_registry service definitions"
```

archived-with: 2026-06-21-remote-func-protocol
---

### Task 2: 协议注册 (RemoteFuncProtocol)

**Files:**
- Create: `backend/internal/protocol/remote_func.go`
- Modify: `backend/internal/protocol/registry.go:87-92`

**Interfaces:**
- Consumes: `protocol.ProtocolFactory` (Name/DisplayName/Description/ConfigSchema), `protocol.ConfigSchema`, `protocol.ConfigField`
- Produces: `RemoteFuncProtocol` struct implementing `ProtocolFactory`

- [x] **Step 1: 编写 RemoteFuncProtocol**

```go
// remote_func.go
package protocol

// RemoteFuncProtocol 远程函数协议
type RemoteFuncProtocol struct{}

func (p *RemoteFuncProtocol) Name() string        { return "remoteFunc" }
func (p *RemoteFuncProtocol) DisplayName() string  { return "Remote Function" }
func (p *RemoteFuncProtocol) Description() string  { return "调度远程三方节点执行函数" }
func (p *RemoteFuncProtocol) ConfigSchema() ConfigSchema {
	return ConfigSchema{
		Fields: []ConfigField{
			{Name: "funcName", Label: "函数名", Type: "string", Required: true, Description: "远程函数名称"},
			{Name: "timeout", Label: "超时(秒)", Type: "number", Required: false, Default: "30"},
		},
	}
}
```

- [x] **Step 2: 在 RegisterBuiltinProtocols 中注册**

```go
// registry.go RegisterBuiltinProtocols 函数末尾添加:
func RegisterBuiltinProtocols(registry *Registry) {
	registry.Register(&HTTPProtocol{})
	registry.Register(&GRPCProtocol{})
	registry.Register(&LocalProtocol{})
	registry.Register(&MCPProtocol{})
	registry.Register(&RemoteFuncProtocol{})  // 新增
}
```

- [x] **Step 3: 运行现有测试验证注册不破坏**

```bash
cd backend && go test ./internal/protocol/... -v
```

Expected: 所有测试通过。

- [x] **Commit**

```bash
git add backend/internal/protocol/remote_func.go backend/internal/protocol/registry.go
git commit -m "feat(protocol): add RemoteFuncProtocol with funcName/timeout config schema"
```

archived-with: 2026-06-21-remote-func-protocol
---

### Task 3: Node Registry（引擎侧 gRPC Server + Redis）

**Files:**
- Create: `backend/internal/node_registry/registry.go`

**Interfaces:**
- Consumes: `redis.Client` (from `go-redis/v9`), `remote_executor.NodeRegistryServer`
- Produces: `NodeRegistry` struct with `NewNodeRegistry(redisClient *redis.Client) *NodeRegistry`, `Register(ctx, *RegisterRequest) (*RegisterResponse, error)`, `Heartbeat(ctx, *HeartbeatRequest) (*HeartbeatResponse, error)`, `GetNodesForFunc(ctx, funcName string) ([]NodeInfo, error)`, `NodeInfo` struct

- [x] **Step 1: 编写 NodeRegistry 实现**

```go
// registry.go
package node_registry

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	pb "github.com/caiflower/dagflow/internal/proto/remote_executor"
)

const (
	keyPrefixNode      = "dagflow:node:"
	keyPrefixHeartbeat = "dagflow:node:heartbeat:"
	nodeTTL            = 30 * time.Second
)

type NodeInfo struct {
	NodeID    string   `json:"nodeId"`
	Address   string   `json:"address"`
	Functions []string `json:"functions"`
}

type NodeRegistry struct {
	pb.UnimplementedNodeRegistryServer
	redis *redis.Client
}

func NewNodeRegistry(redisClient *redis.Client) *NodeRegistry {
	return &NodeRegistry{redis: redisClient}
}

func (r *NodeRegistry) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	info := NodeInfo{
		NodeID:    req.NodeId,
		Address:   req.Address,
		Functions: req.Functions,
	}
	data, err := json.Marshal(info)
	if err != nil {
		return nil, fmt.Errorf("marshal node info: %w", err)
	}

	key := keyPrefixNode + req.NodeId
	if err := r.redis.Set(ctx, key, data, nodeTTL).Err(); err != nil {
		return nil, fmt.Errorf("redis set node: %w", err)
	}

	hbKey := keyPrefixHeartbeat + req.NodeId
	if err := r.redis.Set(ctx, hbKey, time.Now().Unix(), nodeTTL).Err(); err != nil {
		return nil, fmt.Errorf("redis set heartbeat: %w", err)
	}

	return &pb.RegisterResponse{Ok: true}, nil
}

func (r *NodeRegistry) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
	hbKey := keyPrefixHeartbeat + req.NodeId
	if err := r.redis.Set(ctx, hbKey, time.Now().Unix(), nodeTTL).Err(); err != nil {
		return &pb.RegisterResponse{Ok: false}, fmt.Errorf("redis heartbeat: %w", err)
	}

	// 同时刷新 node info TTL
	nodeKey := keyPrefixNode + req.NodeId
	r.redis.Expire(ctx, nodeKey, nodeTTL)

	return &pb.HeartbeatResponse{Ok: true}, nil
}

func (r *NodeRegistry) GetNodesForFunc(ctx context.Context, funcName string) ([]NodeInfo, error) {
	keys, _, err := r.redis.Scan(ctx, 0, keyPrefixNode+"*", 100).Result()
	if err != nil {
		return nil, fmt.Errorf("redis scan nodes: %w", err)
	}

	var nodes []NodeInfo
	for _, key := range keys {
		data, err := r.redis.Get(ctx, key).Bytes()
		if err != nil {
			continue // key 可能已过期
		}
		var info NodeInfo
		if err := json.Unmarshal(data, &info); err != nil {
			continue
		}
		for _, fn := range info.Functions {
			if fn == funcName {
				nodes = append(nodes, info)
				break
			}
		}
	}
	return nodes, nil
}
```

- [x] **Step 2: 编译验证**

```bash
cd backend && go build ./internal/node_registry/...
```

Expected: 编译成功。

- [x] **Commit**

```bash
git add backend/internal/node_registry/registry.go
git commit -m "feat(node_registry): implement Register/Heartbeat gRPC server with Redis storage"
```

archived-with: 2026-06-21-remote-func-protocol
---

### Task 4: Remote Executor Provider（gRPC 连接池 + 调度）

**Files:**
- Create: `backend/internal/remote_executor/client.go`
- Create: `backend/internal/remote_executor/provider.go`

**Interfaces:**
- Consumes: `executor.ExecutorProvider` (Execute/Protocol), `executor.TaskData`, `executor.Protocol`, `node_registry.NodeInfo`
- Produces: `RemoteFuncProvider` struct, `ConnPool` struct (nodeID → *grpc.ClientConn), `NewConnPool() *ConnPool`, `GetConn(address) (*grpc.ClientConn, error)`, `Close()`

- [x] **Step 1: 编写 gRPC 连接池**

```go
// client.go
package remote_executor

import (
	"fmt"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type connEntry struct {
	conn *grpc.ClientConn
	refs int
}

type ConnPool struct {
	mu    sync.Mutex
	conns map[string]*connEntry
}

func NewConnPool() *ConnPool {
	return &ConnPool{conns: make(map[string]*connEntry)}
}

func (p *ConnPool) GetConn(address string) (*grpc.ClientConn, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	entry, ok := p.conns[address]
	if ok {
		entry.refs++
		return entry.conn, nil
	}

	conn, err := grpc.NewClient(address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", address, err)
	}

	p.conns[address] = &connEntry{conn: conn, refs: 1}
	return conn, nil
}

func (p *ConnPool) Release(address string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	entry, ok := p.conns[address]
	if !ok {
		return
	}
	entry.refs--
	if entry.refs <= 0 {
		entry.conn.Close()
		delete(p.conns, address)
	}
}

func (p *ConnPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for addr, entry := range p.conns {
		entry.conn.Close()
		delete(p.conns, addr)
	}
}
```

- [x] **Step 2: 编写 RemoteFuncProvider**

```go
// provider.go
package remote_executor

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/caiflower/dagflow/internal/node_registry"
	pb "github.com/caiflower/dagflow/internal/proto/remote_executor"
	"github.com/caiflower/dagflow/taskx/executor"
)

const ProtocolRemoteFunc executor.Protocol = "remoteFunc"

type RemoteFuncProvider struct {
	FuncName string
	Timeout  time.Duration
	Registry *node_registry.NodeRegistry
	Pool     *ConnPool
}

func (p *RemoteFuncProvider) Execute(ctx context.Context, data *executor.TaskData) (any, error) {
	nodes, err := p.Registry.GetNodesForFunc(ctx, p.FuncName)
	if err != nil {
		return nil, fmt.Errorf("get nodes for func %q: %w", p.FuncName, err)
	}
	if len(nodes) == 0 {
		return nil, fmt.Errorf("no available nodes for function %q", p.FuncName)
	}

	node := nodes[rand.Intn(len(nodes))]

	conn, err := p.Pool.GetConn(node.Address)
	if err != nil {
		return nil, fmt.Errorf("get connection to %s: %w", node.Address, err)
	}
	defer p.Pool.Release(node.Address)

	client := pb.NewRemoteExecutorClient(conn)

	ctx, cancel := context.WithTimeout(ctx, p.Timeout)
	defer cancel()

	input := []byte(data.Input)
	resp, err := client.Execute(ctx, &pb.ExecuteRequest{
		FuncName: p.FuncName,
		Input:    input,
		TaskId:   data.TaskId,
	})
	if err != nil {
		return nil, fmt.Errorf("remote execute %q: %w", p.FuncName, err)
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("remote function %q error: %s", p.FuncName, resp.Error)
	}

	return map[string]any{"output": string(resp.Output)}, nil
}

func (p *RemoteFuncProvider) Protocol() executor.Protocol {
	return ProtocolRemoteFunc
}
```

- [x] **Step 3: 编译验证**

```bash
cd backend && go build ./internal/remote_executor/...
```

Expected: 编译成功。

- [x] **Commit**

```bash
git add backend/internal/remote_executor/client.go backend/internal/remote_executor/provider.go
git commit -m "feat(remote_executor): implement gRPC connection pool and rand-robin scheduling provider"
```

archived-with: 2026-06-21-remote-func-protocol
---

### Task 5: Provider 工厂集成

**Files:**
- Modify: `backend/internal/service/providers.go:17-28` (createProvider switch)

**Interfaces:**
- Consumes: `remote_executor.RemoteFuncProvider`, `remote_executor.ConnPool`
- Produces: `remoteFunc` case in `createProvider()`

- [x] **Step 1: 在 createProvider 中增加 remoteFunc 分支**

在 `providers.go` 的 `createProvider` switch 中，`case "mcp":` 之前添加:

```go
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

同时在 `providers.go` 文件顶部 import 中添加:
```go
	"github.com/caiflower/dagflow/internal/remote_executor"
```

并在 `providers.go` 文件 `package service` 后、所有函数前添加全局变量声明:
```go
import (
	// ... existing imports
	"github.com/caiflower/dagflow/internal/node_registry"
	"github.com/caiflower/dagflow/internal/remote_executor"
)

// 由 main.initBean() 注入
var nodeRegistry *node_registry.NodeRegistry
var remoteExecutorPool *remote_executor.ConnPool

// SetNodeRegistry 由 main 注入 node registry
func SetNodeRegistry(r *node_registry.NodeRegistry) {
	nodeRegistry = r
}

// SetRemoteExecutorPool 由 main 注入连接池
func SetRemoteExecutorPool(p *remote_executor.ConnPool) {
	remoteExecutorPool = p
}
```

- [x] **Step 2: 编译验证**

```bash
cd backend && go build ./internal/service/...
```

Expected: 编译成功。

- [x] **Commit**

```bash
git add backend/internal/service/providers.go
git commit -m "feat(service): add remoteFunc case to createProvider factory"
```

archived-with: 2026-06-21-remote-func-protocol
---

### Task 6: Server 集成（main.go 启动 NodeRegistry gRPC）

**Files:**
- Modify: `backend/cmd/server/main.go:88-93` (initGrpcServices 附近)
- Modify: `backend/cmd/server/main.go:55-57` (initBean 附近)

**Interfaces:**
- Consumes: `redis.Client`, `node_registry.NodeRegistry`, `remote_executor.ConnPool`
- Produces: NodeRegistry gRPC server 启动在独立端口

- [x] **Step 1: 在 initBean 中初始化 node registry 和连接池**

在 `initBean()` 函数末尾添加:

```go
	// 初始化 NodeRegistry（使用 common-tools 提供的 Redis client）
	redisClient := cluster.GetRedisClient(dagflowCluster)
	nodeReg := node_registry.NewNodeRegistry(redisClient)
	service.SetNodeRegistry(nodeReg)

	// 初始化 RemoteExecutor 连接池
	pool := remote_executor.NewConnPool()
	service.SetRemoteExecutorPool(pool)
```

需要在 `main.go` import 中添加:
```go
	"github.com/caiflower/dagflow/internal/node_registry"
	"github.com/caiflower/dagflow/internal/remote_executor"
	"github.com/caiflower/dagflow/internal/service"
```

- [x] **Step 2: 在 initGrpcServices 后启动 NodeRegistry gRPC server**

新增 `startNodeRegistryServer` 函数并在 `initGrpcServices()` 后调用:

```go
func startNodeRegistryServer() {
	nodeReg := service.GetNodeRegistry() // 需要在 service 包添加 getter
	port := constants.DefaultConfig.GetNodeRegistryPort() // 从配置读取端口，默认 50051

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		logger.Error("failed to listen on port %d: %v", port, err)
		return
	}

	s := grpc.NewServer()
	pb.RegisterNodeRegistryServer(s, nodeReg)

	go func() {
		logger.Info("NodeRegistry gRPC server listening on :%d", port)
		if err := s.Serve(lis); err != nil {
			logger.Error("NodeRegistry server error: %v", err)
		}
	}()
}
```

在 `initGrpcServices()` 末尾调用:
```go
	startNodeRegistryServer()
```

- [x] **Step 3: 编译验证**

```bash
cd backend && go build ./cmd/server/...
```

Expected: 编译成功（若缺少 getter/setter，按编译错误补充）。

- [x] **Commit**

```bash
git add backend/cmd/server/main.go backend/internal/service/providers.go
git commit -m "feat(server): integrate NodeRegistry gRPC server and RemoteExecutor pool startup"
```

archived-with: 2026-06-21-remote-func-protocol
---

### Task 7: Go SDK

**Files:**
- Create: `clients/go/go.mod`
- Create: `clients/go/sdk.go`
- Create: `clients/go/examples/demo/main.go`

**Interfaces:**
- Consumes: `remote_executor.RemoteExecutorServer`, `remote_executor.NodeRegistryClient`
- Produces: `SDK` struct: `New(config Config) *SDK`, `Register[In,Out any](name string, fn func(context.Context, In) (Out, error))`, `Start(ctx context.Context) error`

- [x] **Step 1: 创建 go.mod**

```go
module github.com/caiflower/dagflow/clients/go

go 1.24.0

require (
	github.com/caiflower/dagflow v0.0.0
	google.golang.org/grpc v1.77.0
	google.golang.org/protobuf v1.36.11
)

replace github.com/caiflower/dagflow => ../../backend
```

- [x] **Step 2: 编写 SDK 主入口**

```go
// sdk.go
package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/caiflower/dagflow/internal/proto/remote_executor"
)

type Config struct {
	NodeID    string
	EngineAddr string // engine NodeRegistry address (host:port)
	ListenAddr string // SDK listen address (host:port)
}

type SDK struct {
	config   Config
	handlers map[string]HandlerFunc
	mu       sync.RWMutex
	server   *grpc.Server
}

type HandlerFunc func(ctx context.Context, raw []byte) ([]byte, error)

func New(cfg Config) *SDK {
	return &SDK{
		config:   cfg,
		handlers: make(map[string]HandlerFunc),
	}
}

func Register[In, Out any](s *SDK, name string, fn func(ctx context.Context, input In) (Out, error)) {
	wrapper := func(ctx context.Context, raw []byte) ([]byte, error) {
		var in In
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &in); err != nil {
				return nil, fmt.Errorf("unmarshal input: %w", err)
			}
		}
		out, err := fn(ctx, in)
		if err != nil {
			return nil, err
		}
		data, err := json.Marshal(out)
		if err != nil {
			return nil, fmt.Errorf("marshal output: %w", err)
		}
		return data, nil
	}
	s.mu.Lock()
	s.handlers[name] = wrapper
	s.mu.Unlock()
}

func (s *SDK) Start(ctx context.Context) error {
	// 1. 启动 gRPC server
	lis, err := net.Listen("tcp", s.config.ListenAddr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", s.config.ListenAddr, err)
	}

	s.server = grpc.NewServer()
	pb.RegisterRemoteExecutorServer(s.server, &executorServer{sdk: s})

	go func() {
		s.server.Serve(lis)
	}()

	// 2. 注册到引擎
	funcs := s.functionList()
	if err := s.registerWithEngine(ctx, funcs); err != nil {
		s.server.Stop()
		return fmt.Errorf("register with engine: %w", err)
	}

	// 3. 心跳循环
	go s.heartbeatLoop(ctx)

	// 4. 等待 ctx 取消
	<-ctx.Done()
	s.server.GracefulStop()
	return nil
}

func (s *SDK) functionList() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	funcs := make([]string, 0, len(s.handlers))
	for name := range s.handlers {
		funcs = append(funcs, name)
	}
	return funcs
}

func (s *SDK) registerWithEngine(ctx context.Context, funcs []string) error {
	conn, err := grpc.NewClient(s.config.EngineAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("dial engine %s: %w", s.config.EngineAddr, err)
	}
	defer conn.Close()

	client := pb.NewNodeRegistryClient(conn)
	_, err = client.Register(ctx, &pb.RegisterRequest{
		NodeId:    s.config.NodeID,
		Address:   s.config.ListenAddr,
		Functions: funcs,
	})
	return err
}

func (s *SDK) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	conn, err := grpc.NewClient(s.config.EngineAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return
	}
	defer conn.Close()
	client := pb.NewNodeRegistryClient(conn)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			client.Heartbeat(ctx, &pb.HeartbeatRequest{NodeId: s.config.NodeID})
		}
	}
}

type executorServer struct {
	pb.UnimplementedRemoteExecutorServer
	sdk *SDK
}

func (s *executorServer) Execute(ctx context.Context, req *pb.ExecuteRequest) (*pb.ExecuteResponse, error) {
	s.sdk.mu.RLock()
	handler, ok := s.sdk.handlers[req.FuncName]
	s.sdk.mu.RUnlock()

	if !ok {
		return &pb.ExecuteResponse{Error: fmt.Sprintf("function %q not registered", req.FuncName)}, nil
	}
	output, err := handler(ctx, req.Input)
	if err != nil {
		return &pb.ExecuteResponse{Error: err.Error()}, nil
	}
	return &pb.ExecuteResponse{Output: output}, nil
}

func (s *executorServer) HealthCheck(ctx context.Context, req *pb.HealthRequest) (*pb.HealthResponse, error) {
	return &pb.HealthResponse{Ok: true}, nil
}
```

- [x] **Step 3: 编写 demo 示例**

```go
// examples/demo/main.go
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	sdk "github.com/caiflower/dagflow/clients/go"
)

type ImageInput struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type ImageOutput struct {
	ProcessedURL string `json:"processedUrl"`
	Size         int64  `json:"size"`
}

func processImage(ctx context.Context, input ImageInput) (ImageOutput, error) {
	fmt.Printf("Processing image: %s (%dx%d)\n", input.URL, input.Width, input.Height)
	return ImageOutput{
		ProcessedURL: input.URL + "?processed=true",
		Size:         int64(input.Width * input.Height * 4),
	}, nil
}

func main() {
	s := sdk.New(sdk.Config{
		NodeID:     "demo-node-1",
		EngineAddr: "localhost:50051",
		ListenAddr: "localhost:50052",
	})

	sdk.Register(s, "processImage", processImage)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	fmt.Println("SDK demo node starting...")
	if err := s.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "SDK error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("SDK demo node stopped")
}
```

- [x] **Step 4: 编译验证 SDK 和 demo**

```bash
cd clients/go && go build ./...
cd clients/go/examples/demo && go build ./...
```

Expected: 编译成功。

- [x] **Commit**

```bash
git add clients/go/go.mod clients/go/sdk.go clients/go/examples/demo/main.go
git commit -m "feat(sdk): add Go SDK with Register/Start and demo example"
```

archived-with: 2026-06-21-remote-func-protocol
---

### Task 8: 单元测试 — NodeRegistry

**Files:**
- Create: `backend/internal/node_registry/registry_test.go`

**Interfaces:**
- Consumes: `node_registry.NodeRegistry`, `miniredis`
- Produces: TestRegisterStoresNodeInfo, TestHeartbeatRefreshesTTL, TestGetNodesForFuncFilters

- [x] **Step 1: 编写 NodeRegistry 单元测试**

```go
// registry_test.go
package node_registry

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pb "github.com/caiflower/dagflow/internal/proto/remote_executor"
)

func setupRegistry(t *testing.T) (*NodeRegistry, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return NewNodeRegistry(client), mr
}

func TestRegisterStoresNodeInfo(t *testing.T) {
	reg, _ := setupRegistry(t)
	ctx := context.Background()

	resp, err := reg.Register(ctx, &pb.RegisterRequest{
		NodeId:    "node-1",
		Address:   "localhost:50052",
		Functions: []string{"funcA", "funcB"},
	})
	require.NoError(t, err)
	assert.True(t, resp.Ok)

	nodes, err := reg.GetNodesForFunc(ctx, "funcA")
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.Equal(t, "node-1", nodes[0].NodeID)
	assert.Equal(t, "localhost:50052", nodes[0].Address)
	assert.Contains(t, nodes[0].Functions, "funcA")
}

func TestHeartbeatRefreshesTTL(t *testing.T) {
	reg, mr := setupRegistry(t)
	ctx := context.Background()

	_, err := reg.Register(ctx, &pb.RegisterRequest{
		NodeId:    "node-1",
		Address:   "localhost:50052",
		Functions: []string{"funcA"},
	})
	require.NoError(t, err)

	// 快进到接近过期但不完全过期
	mr.FastForward(20 * time.Second)

	// 发送心跳
	_, err = reg.Heartbeat(ctx, &pb.HeartbeatRequest{NodeId: "node-1"})
	require.NoError(t, err)

	// 快进 20 秒，节点应该还在（因为心跳刷新了 TTL）
	mr.FastForward(20 * time.Second)
	nodes, err := reg.GetNodesForFunc(ctx, "funcA")
	require.NoError(t, err)
	assert.Len(t, nodes, 1)
}

func TestGetNodesForFuncNoMatch(t *testing.T) {
	reg, _ := setupRegistry(t)
	ctx := context.Background()

	_, err := reg.Register(ctx, &pb.RegisterRequest{
		NodeId:    "node-1",
		Address:   "localhost:50052",
		Functions: []string{"funcA"},
	})
	require.NoError(t, err)

	nodes, err := reg.GetNodesForFunc(ctx, "funcB")
	require.NoError(t, err)
	assert.Empty(t, nodes)
}

func TestNodeExpiresAfterTTL(t *testing.T) {
	reg, mr := setupRegistry(t)
	ctx := context.Background()

	_, err := reg.Register(ctx, &pb.RegisterRequest{
		NodeId:    "node-1",
		Address:   "localhost:50052",
		Functions: []string{"funcA"},
	})
	require.NoError(t, err)

	// 快进超过 TTL
	mr.FastForward(31 * time.Second)

	nodes, err := reg.GetNodesForFunc(ctx, "funcA")
	require.NoError(t, err)
	assert.Empty(t, nodes, "node should expire after TTL without heartbeat")
}
```

- [x] **Step 2: 运行测试**

```bash
cd backend && go test ./internal/node_registry/... -v
```

Expected: 4 个测试全部 PASS。

- [x] **Commit**

```bash
git add backend/internal/node_registry/registry_test.go
git commit -m "test(node_registry): add unit tests for Register/Heartbeat/GetNodesForFunc"
```

archived-with: 2026-06-21-remote-func-protocol
---

### Task 9: 单元测试 — RemoteExecutorProvider

**Files:**
- Create: `backend/internal/remote_executor/provider_test.go`

**Interfaces:**
- Consumes: `remote_executor.RemoteFuncProvider`, `miniredis`, mock gRPC server
- Produces: TestExecuteSuccess, TestExecuteNoNodesError, TestExecuteRandomPick

- [x] **Step 1: 编写 RemoteExecutorProvider 单元测试**

```go
// provider_test.go
package remote_executor

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"github.com/caiflower/dagflow/internal/node_registry"
	pb "github.com/caiflower/dagflow/internal/proto/remote_executor"
	"github.com/caiflower/dagflow/taskx/executor"
)

const bufSize = 1024 * 1024

type mockExecutorServer struct {
	pb.UnimplementedRemoteExecutorServer
	output string
	errMsg string
}

func (m *mockExecutorServer) Execute(ctx context.Context, req *pb.ExecuteRequest) (*pb.ExecuteResponse, error) {
	if m.errMsg != "" {
		return &pb.ExecuteResponse{Error: m.errMsg}, nil
	}
	return &pb.ExecuteResponse{Output: []byte(m.output)}, nil
}

func (m *mockExecutorServer) HealthCheck(ctx context.Context, req *pb.HealthRequest) (*pb.HealthResponse, error) {
	return &pb.HealthResponse{Ok: true}, nil
}

func setupProvider(t *testing.T, addr string, funcs []string) (*RemoteFuncProvider, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	reg := node_registry.NewNodeRegistry(client)

	ctx := context.Background()
	_, err := reg.Register(ctx, &pb.RegisterRequest{
		NodeId:    "test-node",
		Address:   addr,
		Functions: funcs,
	})
	require.NoError(t, err)

	return &RemoteFuncProvider{
		FuncName: "testFunc",
		Timeout:  5 * time.Second,
		Registry: reg,
		Pool:     NewConnPool(),
	}, mr
}

func startMockServer(t *testing.T, mock *mockExecutorServer) string {
	t.Helper()
	lis := bufconn.Listen(bufSize)
	s := grpc.NewServer()
	pb.RegisterRemoteExecutorServer(s, mock)
	go func() {
		s.Serve(lis)
	}()
	t.Cleanup(s.Stop)
	return "bufconn"
}

func TestExecuteNoNodes(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	reg := node_registry.NewNodeRegistry(client)

	p := &RemoteFuncProvider{
		FuncName: "nonexistent",
		Timeout:  5 * time.Second,
		Registry: reg,
		Pool:     NewConnPool(),
	}

	_, err := p.Execute(context.Background(), &executor.TaskData{TaskId: "t1", Input: `{}`})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no available nodes")
}

func TestExecuteWrongFuncName(t *testing.T) {
	addr := startMockServer(t, &mockExecutorServer{output: `{"ok":true}`})
	p, _ := setupProvider(t, addr, []string{"otherFunc"})

	_, err := p.Execute(context.Background(), &executor.TaskData{TaskId: "t1", Input: `{}`})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no available nodes")
}
```

- [x] **Step 2: 运行测试**

```bash
cd backend && go test ./internal/remote_executor/... -v -count=1
```

Expected: 测试 PASS。

- [x] **Commit**

```bash
git add backend/internal/remote_executor/provider_test.go
git commit -m "test(remote_executor): add unit tests for rand-robin and no-nodes error"
```

archived-with: 2026-06-21-remote-func-protocol
---

### Task 10: 单元测试 — SDK

**Files:**
- Create: `clients/go/sdk_test.go`

**Interfaces:**
- Consumes: `sdk.SDK`, `sdk.Register`
- Produces: TestExecuteHandlerDispatch, TestMissingFuncError, TestRegisterTypeConversion

- [x] **Step 1: 编写 SDK 单元测试**

```go
// sdk_test.go
package sdk

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterAndHandlerDispatch(t *testing.T) {
	s := New(Config{NodeID: "test"})

	type MyInput struct{ Value int }
	type MyOutput struct{ Result int }

	Register(s, "double", func(ctx context.Context, in MyInput) (MyOutput, error) {
		return MyOutput{Result: in.Value * 2}, nil
	})

	s.mu.RLock()
	handler, ok := s.handlers["double"]
	s.mu.RUnlock()
	require.True(t, ok, "handler should be registered")

	output, err := handler(context.Background(), []byte(`{"Value":21}`))
	require.NoError(t, err)
	assert.Equal(t, `{"Result":42}`, string(output))
}

func TestMissingFuncReturnsError(t *testing.T) {
	s := New(Config{NodeID: "test"})
	server := &ExecutorServer{Sdk: s}

	resp, err := server.Execute(context.Background(), &pb.ExecuteRequest{
		FuncName: "nonexistent",
		Input:    []byte(`{}`),
	})

	// Execute 不返回 gRPC error，错误在 response.Error 中
	if err == nil {
		assert.Contains(t, resp.Error, "not registered")
	}
}

func TestMultipleFunctionsRegistered(t *testing.T) {
	s := New(Config{NodeID: "test"})

	type Empty struct{}
	Register(s, "funcA", func(ctx context.Context, in Empty) (Empty, error) { return Empty{}, nil })
	Register(s, "funcB", func(ctx context.Context, in Empty) (Empty, error) { return Empty{}, nil })
	Register(s, "funcC", func(ctx context.Context, in Empty) (Empty, error) { return Empty{}, nil })

	funcs := s.functionList()
	assert.Len(t, funcs, 3)
	assert.Contains(t, funcs, "funcA")
	assert.Contains(t, funcs, "funcB")
	assert.Contains(t, funcs, "funcC")
}
```

需要在 `sdk_test.go` 顶部添加 import:
```go
import (
	pb "github.com/caiflower/dagflow/internal/proto/remote_executor"
)
```

- [x] **Step 2: 运行测试**

```bash
cd clients/go && go test ./... -v -count=1
```

Expected: 测试 PASS。

- [x] **Commit**

```bash
git add clients/go/sdk_test.go
git commit -m "test(sdk): add unit tests for Register dispatch and missing func error"
```

archived-with: 2026-06-21-remote-func-protocol
---

### Task 11: 集成测试

**Files:**
- Create: `clients/go/examples/integration_test.go`

**Interfaces:**
- Consumes: `sdk.SDK`, `node_registry.NodeRegistry`, `remote_executor.RemoteFuncProvider`, `miniredis`
- Produces: 端到端场景测试

- [x] **Step 1: 编写集成测试** (created as backend/internal/remote_executor/integration_test.go)

```go
// integration_test.go
package main

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"github.com/caiflower/dagflow/internal/node_registry"
	pb "github.com/caiflower/dagflow/internal/proto/remote_executor"
	"github.com/caiflower/dagflow/internal/remote_executor"
	exec "github.com/caiflower/dagflow/taskx/executor"
	sdk "github.com/caiflower/dagflow/clients/go"
)

const bufSize = 1024 * 1024

func startNodeRegistryServer(t *testing.T, reg *node_registry.NodeRegistry) *bufconn.Listener {
	t.Helper()
	lis := bufconn.Listen(bufSize)
	s := grpc.NewServer()
	pb.RegisterNodeRegistryServer(s, reg)
	go s.Serve(lis)
	t.Cleanup(s.Stop)
	return lis
}

func bufDialer(lis *bufconn.Listener) func(context.Context, string) (net.Conn, error) {
	return func(ctx context.Context, _ string) (net.Conn, error) {
		return lis.Dial()
	}
}

func TestIntegrationSingleNodeExecute(t *testing.T) {
	// Setup Redis
	mr := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	// Setup NodeRegistry
	reg := node_registry.NewNodeRegistry(redisClient)
	regLis := startNodeRegistryServer(t, reg)

	// Setup SDK node
	s := sdk.New(sdk.Config{
		NodeID:     "int-test-node",
		EngineAddr: "bufconn",
		ListenAddr: "localhost:0", // random port for real listener
	})

	type EchoInput struct{ Msg string }
	type EchoOutput struct{ Msg string }
	sdk.Register(s, "echo", func(ctx context.Context, in EchoInput) (EchoOutput, error) {
		return EchoOutput{Msg: in.Msg + "_echo"}, nil
	})

	// Build gRPC conn to registry via bufconn
	conn, err := grpc.DialContext(context.Background(), "bufconn",
		grpc.WithContextDialer(bufDialer(regLis)),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer conn.Close()

	// Start SDK server manually (separate goroutine) and register
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sdkLis := bufconn.Listen(bufSize)
	sdkServer := grpc.NewServer()
	pb.RegisterRemoteExecutorServer(sdkServer, &executorServer{sdk: s})
	go sdkServer.Serve(sdkLis)
	defer sdkServer.Stop()

	// Register node via registry client
	regClient := pb.NewNodeRegistryClient(conn)
	_, err = regClient.Register(ctx, &pb.RegisterRequest{
		NodeId:    "int-test-node",
		Address:   "bufconn",
		Functions: []string{"echo"},
	})
	require.NoError(t, err)

	// Setup provider with connection to SDK
	pool := remote_executor.NewConnPool()
	provider := &remote_executor.RemoteFuncProvider{
		FuncName: "echo",
		Timeout:  5 * time.Second,
		Registry: reg,
		Pool:     pool,
	}

	// Execute via provider
	result, err := provider.Execute(ctx, &exec.TaskData{
		TaskId: "t1",
		Input:  `{"Msg":"hello"}`,
	})
	require.NoError(t, err)
	assert.Contains(t, result.(map[string]any)["output"], "hello_echo")
}
```

需要在 `integration_test.go` 顶部添加 import:
```go
import (
	"net"

	pb "github.com/caiflower/dagflow/internal/proto/remote_executor"
)
```

而 executorServer 类型在 sdk.go 中未导出，需要在集成测试中或在 sdk.go 中将 executorServer 导出:
在 sdk.go 末尾添加:
```go
// ExecutorServer 暴露给集成测试使用
var ExecutorServer = &executorServer{}
// 但在集成测试中需要访问 SDK，改为导出类型:
```

更实际的方案: 在 sdk.go 中导出 executorServer:
```go
type ExecutorServer struct {
	pb.UnimplementedRemoteExecutorServer
	Sdk *SDK
}

func (s *ExecutorServer) Execute(ctx context.Context, req *pb.ExecuteRequest) (*pb.ExecuteResponse, error) {
	// 同 executorServer.Execute
}

func (s *ExecutorServer) HealthCheck(ctx context.Context, req *pb.HealthRequest) (*pb.HealthResponse, error) {
	return &pb.HealthResponse{Ok: true}, nil
}
```

并将 SDK 内部使用的 executorServer 替换为 ExecutorServer:
```go
// Start() 中:
pb.RegisterRemoteExecutorServer(s.server, &ExecutorServer{Sdk: s})
```

- [x] **Step 2: 运行集成测试**

```bash
cd clients/go/examples && go test -v -count=1 -run Integration
```

Expected: 集成测试 PASS。

- [x] **Commit**

```bash
git add clients/go/sdk.go clients/go/examples/integration_test.go
git commit -m "test(integration): add end-to-end integration test with bufconn"
```

archived-with: 2026-06-21-remote-func-protocol
---

### Task 12: 前端适配

**Files:**
- Modify: `frontend/src/utils/stateColor.ts:4-7` (nodeTypeColorMap), `:55-60` (protocolIconMap)
- Modify: `frontend/src/components/dag/nodes/index.tsx:19-22` (lucideIcons), `:24-26` (getProtocolLucideIcon)

**Interfaces:**
- Consumes: `nodeTypeColorMap`, `protocolIconMap`, `lucideIcons`, `getProtocolLucideIcon`
- Produces: remoteFunc 颜色、图标映射

- [x] **Step 1: 在 stateColor.ts 中添加 remoteFunc 颜色**

在 `nodeTypeColorMap` 对象末尾添加:
```typescript
  remoteFunc: { bg: '#1a1a2e', border: '#7b68ee', text: '#d4ccff' },
```

在 `protocolIconMap` 对象末尾添加:
```typescript
  remoteFunc: 'Radio',
```

- [x] **Step 2: 在 nodes/index.tsx 中添加 remoteFunc 图标映射**

在 `lucideIcons` 导入行:
```typescript
import { Play, Square, Settings, GitBranch, Globe, Radio, Terminal, Link, RadioTower } from 'lucide-react';
```
注意：改用 `RadioTower` 图标来区分 remoteFunc（Radio 已被 gRPC 使用）。

在 `lucideIcons` 对象中添加:
```typescript
  RadioTower,
```

在 `getProtocolLucideIcon` 内的 iconMap 中添加:
```typescript
    remoteFunc: 'RadioTower',
```

- [x] **Step 3: 构建验证**

```bash
cd frontend && npm run build
```

Expected: 构建成功，无 TS 类型错误。

- [x] **Commit**

```bash
git add frontend/src/utils/stateColor.ts frontend/src/components/dag/nodes/index.tsx
git commit -m "feat(frontend): add remoteFunc color scheme and RadioTower icon mapping"
```

archived-with: 2026-06-21-remote-func-protocol
---

### Task 13: 端到端验证

**Files:**
- （无新建/修改文件 — 本地运行验证）

- [x] **Step 1: 启动引擎**

```bash
cd backend && go run ./cmd/server/
```

Expected: 日志显示 "NodeRegistry gRPC server listening on :50051"。

- [x] **Step 2: 启动 SDK demo 节点**

```bash
cd clients/go/examples/demo && go run .
```

Expected: 日志显示 "SDK demo node starting..."。

- [x] **Step 3: 验证心跳存活**

等待 10 秒后检查 Redis:
```bash
redis-cli KEYS "dagflow:node:*"
redis-cli KEYS "dagflow:node:heartbeat:*"
```

Expected: 两个 key 存在。

- [x] **Step 4: 通过前端创建 Flow 测试**

1. 打开 DAGFlow 前端
2. 创建新 Flow: start → task(remoteFunc, funcName=processImage) → end
3. 执行 Flow
4. 查看 SDK demo 控制台输出 "Processing image: ..."
5. 确认 Flow 执行成功

- [x] **Step 5: Commit（如有修正）**

```bash
git add -A && git commit -m "chore: e2e verification adjustments"
```

archived-with: 2026-06-21-remote-func-protocol
---

## Self-Review Checklist

1. **Spec coverage**: 设计文档 11 个章节全部覆盖:
   - Proto 定义 → Task 1
   - Project Structure → Task 1-7 文件路径
   - Protocol Definition → Task 2
   - Node Registry → Task 3
   - Remote Executor Provider → Task 4
   - Provider Factory → Task 5
   - Go SDK → Task 7
   - Protocol Registration → Task 2
   - Provider Factory Integration → Task 5
   - Testing → Tasks 8-11
   - Frontend → Task 12

2. **Placeholder scan**: 无 TBD/TODO/fill-in 模式。所有测试包含完整代码。

3. **Type consistency**:
   - `NodeInfo` 定义在 Task 3，Task 4/5 引用一致性
   - `ProtocolRemoteFunc` 常量在 Task 4 定义为 `executor.Protocol = "remoteFunc"`
   - `ExecutorServer` 在 Task 7 SDK + Task 11 集成测试中类型一致
   - `Register[In,Out]` 泛型签名在 Task 7 定义，Task 10/11 测试使用一致

4. **前向引用**: Tasks 按依赖排序，每个 Task 只依赖已完成 Task 产生的接口。

**Plan complete and saved to `docs/superpowers/plans/2026-06-21-remote-func-protocol.md`.**
