## protocol-registry

可插拔协议注册中心，管理 ExecutorProvider 的注册和发现。

### 核心职责

- 维护协议名称到 ExecutorProvider 工厂函数的映射
- 提供协议元数据（名称、描述、配置 Schema）供前端渲染配置表单
- 支持运行时动态注册新协议

### 协议接口

```go
type ProtocolFactory struct {
    Name        string           // 协议标识：http, grpc, local, mcp
    DisplayName string           // 显示名称
    Description string           // 协议描述
    ConfigSchema json.RawMessage // JSON Schema，描述该协议的配置项
    Create      func(config json.RawMessage) (executor.ExecutorProvider, error)
}
```

### 内置协议

| 协议 | 说明 | ConfigSchema 关键字段 |
|------|------|---------------------|
| http | HTTP REST 调用 | url, method, headers, body, timeout |
| grpc | gRPC 远程调用 | host, port, service, method, body |
| local | 本地函数执行 | funcName, args |
| mcp | MCP 工具调用 | server, tool, args |

### API 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/protocols | 查询已注册协议列表（含 Schema） |
| GET | /api/v1/protocols/:name | 查询单个协议详情 |

### 注册机制

- 应用启动时自动注册内置协议（http/grpc/local/mcp）
- 通过 `bean.Autowired` 注入 ProtocolRegistry 到 Service 层
- 后续可扩展支持从配置或数据库加载自定义协议
