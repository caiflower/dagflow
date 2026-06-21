## 1. Proto 定义

- [x] 1.1 创建 `backend/internal/proto/remote_executor/remote_executor.proto`，定义 RemoteExecutor + NodeRegistry service
- [x] 1.2 生成 Go pb 代码（`protoc` 生成 .pb.go + _grpc.pb.go）
- [x] 1.3 编写 proto 编译脚本或 go:generate 指令

## 2. 协议注册

- [x] 2.1 新增 `RemoteFuncProtocol` 实现 ProtocolFactory 接口（config: funcName + timeout）
- [x] 2.2 在 `registry.go` 的 `RegisterBuiltinProtocols` 中注册

## 3. Node Registry（引擎侧 gRPC server）

- [x] 3.1 实现 Register RPC：接收节点信息 + 函数列表，写入 Redis
- [x] 3.2 实现 Heartbeat RPC：刷新 Redis key TTL（SETEX node:heartbeat:{nodeId} 30）
- [x] 3.3 节点发现：从 Redis SCAN 读取存活节点列表 + 注册函数映射
- [x] 3.4 启动 gRPC server，注册到引擎路由

## 4. Remote Executor Provider

- [x] 4.1 实现 gRPC client 连接池管理（nodeId → grpc.ClientConn）
- [x] 4.2 实现 ExecutorProvider 接口（Execute + Protocol）
- [x] 4.3 实现随机轮询调度：从 node_registry 获取可用节点列表，随机选一个
- [x] 4.4 降级处理：无可用节点返回明确错误

## 5. Provider 工厂集成

- [x] 5.1 在 `providers.go` 的 `createProvider` 中增加 `remoteFunc` case
- [x] 5.2 传递 funcName + timeout 配置到 RemoteFuncProvider

## 6. Go SDK

- [x] 6.1 创建 `clients/go/` 目录 + 独立 go module（go.mod）
- [x] 6.2 SDK 主入口：New(config) → Register(funcName, handler) → Start() 阻塞
- [x] 6.3 函数注册映射：map[string]HandlerFunc → 根据函数名查找 handler
- [x] 6.4 JSON 序列化/反序列化：引擎传 JSON bytes，SDK 按参数类型自动转换
- [x] 6.5 gRPC server：实现 RemoteExecutor service.Execute(ctx, req) → 查找 handler → 执行 → 返回
- [x] 6.6 NodeRegistry client：Register + Heartbeat 循环（10s 间隔）
- [x] 6.7 `examples/demo/main.go`：三方接入示例

## 7. 单元测试

- [x] 7.1 `node_registry/registry_test.go`：Register/Heartbeat RPC 单元测试
- [x] 7.2 `remote_executor/provider_test.go`：节点选择 + Execute mock 测试
- [x] 7.3 `clients/go/sdk_test.go`：函数注册 + JSON 类型转换 + gRPC server 测试

## 8. 集成测试

- [x] 8.1 编写集成测试脚本 (created as backend/internal/remote_executor/integration_test.go) `clients/go/examples/integration_test.go`
- [x] 8.2 测试场景：单节点注册 → 引擎调度 → 执行成功
- [x] 8.3 测试场景：多节点注册同一函数 → 随机选择一个执行
- [x] 8.4 测试场景：节点心跳超时 → 引擎不再调度到该节点
- [x] 8.5 测试场景：无可用节点 → 返回明确错误
- [x] 8.6 测试场景：函数名未注册 → 返回错误

## 9. 前端适配

- [x] 9.1 protocol 选择列表增加 remoteFunc 选项
- [x] 9.2 节点配置表单：funcName 输入 + timeout 数字
- [x] 9.3 remoteFunc 节点配色 + icon 映射
