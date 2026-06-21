## Why

DAGFlow 当前仅支持同进程内执行任务（local/http），无法调度任务到远程第三方应用节点。需要新增 `remoteFunc` 协议，支持三方应用通过 gRPC 接入任务调度引擎，引擎作为调度中心将任务推送到三方节点执行。

## What Changes

- **remoteFunc 协议**：新增协议类型，前端 DAG 编辑器可选，配置函数名和超时时间
- **Node Registry**：引擎侧 gRPC server，接收三方节点注册（函数列表 + 心跳）
- **Remote Executor Provider**：引擎侧 gRPC client，调用三方节点执行函数
- **Redis 心跳**：节点心跳基于 Redis key TTL，超时自动判定下线
- **随机轮询调度**：多节点注册同一函数时，随机选择一个执行
- **Go SDK**：提供 Go SDK，包含函数注册映射、类型转换、gRPC server 启动
- **Proto 定义**：`RemoteExecutor` + `NodeRegistry` 两个 gRPC service

## Capabilities

### New Capabilities

- `remote-func-protocol`: remoteFunc 协议定义与实现，包括协议注册、配置 schema
- `node-registry`: 节点注册中心，gRPC服务 + Redis 心跳存储 + 节点发现
- `remote-executor-provider`: 远程函数执行器，gRPC client 调用 + 调度策略
- `dagflow-go-sdk`: Go SDK，函数注册、类型转换、节点接入

### Modified Capabilities

- `protocol-registry`: 新增 remoteFunc 协议注册

## Impact

- Proto: 新增 `remote_executor.proto`，定义两个 gRPC service
- 引擎: 新增 `node_registry` 包（gRPC server + Redis 管理）、`remote_executor` 包（gRPC client + 调度）
- 协议层: 新增 `RemoteFuncProtocol`，前端配置 schema 新增 `funcName` + `timeout` 字段
- SDK: 新建 `clients/go/` 目录，提供独立 SDK 模块
- 无 breaking changes，现有协议和行为不受影响
