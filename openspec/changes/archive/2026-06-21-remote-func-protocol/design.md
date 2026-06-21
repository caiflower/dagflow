## Context

DAGFlow 是一个任务调度引擎，支持 http/local/mcp 等协议接入。当前所有 executor 均在同一进程内执行，无法调度任务到远程第三方应用。需要新增 `remoteFunc` 协议实现远程 worker 节点调度。

### 项目结构

```
dagflow/
├── backend/internal/
│   ├── proto/remote_executor/
│   │   └── remote_executor.proto   ← proto 定义（引擎 + SDK 共享）
│   ├── protocol/
│   │   └── remote_func.go          ← RemoteFuncProtocol 注册
│   ├── node_registry/
│   │   └── registry.go             ← Register + Heartbeat gRPC server
│   ├── remote_executor/
│   │   ├── client.go               ← gRPC client 连接池
│   │   └── provider.go             ← ExecutorProvider 实现 + 调度
│   └── service/
│       └── providers.go            ← createProvider 新增 remoteFunc case
├── clients/go/                         ← Go SDK（独立 go module）
│   ├── go.mod
│   ├── sdk.go                      ← 主入口 New/Register/Start
│   ├── server.go                   ← gRPC server（RemoteExecutor）
│   ├── registry.go                 ← 函数映射 + JSON 类型转换
│   └── examples/demo/main.go       ← 三方接入示例
└── frontend/src/
    ├── utils/stateColor.ts         ← remoteFunc 配色
    └── components/dag/nodes/       ← icon 映射
```

## Goals / Non-Goals

**Goals:**
- 引擎通过 gRPC 推送任务到三方节点执行（Push 模式）
- 三方节点通过 gRPC 注册函数列表 + 心跳
- 基于 Redis key TTL 的心跳检测机制
- 多节点随机轮询调度
- Go SDK 提供开箱即用的接入能力
- 充分的测试用例 + 本地集成验证

**Non-Goals:**
- 不实现 Pull 模式轮询
- 不实现加权/最少连接等复杂调度策略（后续扩展）
- 不支持非 Go 语言的 SDK（架构预留）

## Decisions

### 1. Push 模式 + 同步 Execute

引擎作为 gRPC client 主动调用 SDK 的 Execute RPC，同步等待结果返回。
- 与现有 `ExecutorProvider.Execute(ctx, data) (any, error)` 完美契合
- 超时通过 gRPC deadline + context 控制

### 2. 心跳机制

基于 Redis key TTL，SDK 每 10s 调用 Heartbeat RPC 刷新 Redis key。

```
SDK Heartbeat → SETEX node:heartbeat:{nodeId} 30 → nodeId + metadata
引擎调度      → EXISTS node:heartbeat:{nodeId}   → 判定在线
```

### 3. SDK 函数注册 + 类型转换

SDK 端 map 映射函数名 → handler。引擎传 JSON bytes，SDK 端按注册函数的参数类型自动 JSON 反序列化。

```go
sdk.Register("processImage", func(ctx context.Context, input []byte) ([]byte, error) {...})
sdk.Register("sendNotification", func(ctx context.Context, input map[string]any) (any, error) {...})
```

### 4. 调度策略

默认随机轮询。从 Redis 读取存活节点列表，随机选择一个。

### 5. Proto 共享

proto 定义在 `backend/internal/proto/remote_executor/`，SDK 通过 go.mod require 引擎 module import。

## Testing Strategy

### 单元测试

| 模块 | 测试文件 | 覆盖 |
|------|---------|------|
| Node Registry | `node_registry/registry_test.go` | Register/Heartbeat RPC，Redis key 写入/过期 |
| Remote Executor | `remote_executor/provider_test.go` | 节点选择、连接池、Execute 调用 |
| SDK | `clients/go/sdk_test.go` | 函数注册、类型转换、gRPC server 启停 |

### 集成测试

本地集成验证场景：

```
1. 启动 DAGFlow Engine（含 NodeRegistry gRPC server + Redis）
2. 启动两个 SDK Demo 节点，分别注册不同函数
3. 创建 DAG：start → remoteFunc(processImage) → end
4. 执行 Flow，验证：
   a. 引擎选择注册了 processImage 的节点
   b. gRPC Execute 调用成功
   c. SDK 返回正确结果
   d. 心跳维持在线状态
5. 停掉一个节点，等待心跳超时
6. 再次执行，验证引擎只选择存活节点
```

### 边界场景

- 无可用节点时 Execute 返回明确错误
- 节点中途下线（心跳未超时），gRPC 连接断开 → 重试
- 并发 Execute 调用同一节点
- 函数名未注册 → 返回明确错误

## Risks / Trade-offs

- [心跳超时任务调度] → 重试间隔 > Redis TTL 缓解
- [单点引擎] → 当前架构即为单点，集群化后续考虑
- [SDK 版本兼容] → proto 向后兼容，SDK 独立版本号
