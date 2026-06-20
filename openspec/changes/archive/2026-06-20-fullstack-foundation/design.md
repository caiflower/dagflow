## Architecture Overview

DAGFlow 采用前后端分离架构，后端基于 common-tools 框架构建 REST API，前端使用 React + ReactFlow 提供可视化界面。

```
┌─────────────────────────────────────────────────────────────────┐
│  Frontend (React + Vite + ReactFlow)                            │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐          │
│  │ Flow 列表 │ │DAG 编辑器 │ │ 执行监控  │ │模板/历史  │          │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘          │
│                         │ Axios /api/v1                          │
└─────────────────────────┼───────────────────────────────────────┘
                          │
┌─────────────────────────┼───────────────────────────────────────┐
│  Backend (Go + common-tools)                                    │
│  ┌──────────────────────────────────────────┐                   │
│  │  API Layer (web.Engine + RouterGroup)     │                   │
│  │  /api/v1/flows  /api/v1/runs  /api/v1/protocols             │
│  └──────────────────┬───────────────────────┘                   │
│  ┌──────────────────┴───────────────────────┐                   │
│  │  Service Layer (bean.Autowired 注入)       │                   │
│  │  FlowService  RunService                  │                   │
│  └──────────────────┬───────────────────────┘                   │
│  ┌──────────────────┴───────────────────────┐                   │
│  │  Converter + Protocol Registry            │                   │
│  │  Flow ↔ taskx DAG 转换                    │                   │
│  └──────────────────┬───────────────────────┘                   │
│  ┌──────────────────┴───────────────────────┐                   │
│  │  taskx Engine (不修改)                     │                   │
│  │  DAG 调度 / 重试 / 回滚 / 集群分发        │                   │
│  └──────────────────────────────────────────┘                   │
│  ┌──────────────────────────────────────────┐                   │
│  │  DAO Layer (db/v1 bun ORM)                │                   │
│  │  FlowDAO + taskx DAO (task/subtask/edge)  │                   │
│  └──────────────────────────────────────────┘                   │
└─────────────────────────────────────────────────────────────────┘
```

## Backend Design

### 启动流程 (cmd/server/main.go)

遵循 common-tools 标准启动顺序：
1. `constants.InitConfig()` — 加载 etc/config.yaml + etc/default.yaml
2. `logger.InitLogger()` — 初始化日志
3. `initBean()` — 注册 Service、DAO、ProtocolRegistry 等 Bean
4. `web.Init()` — 创建 web.Engine、注册路由组、添加为 Daemon
5. `dao.Init()` — 初始化 DB 连接、Flow 表迁移
6. `bean.Ioc()` — 自动装配所有 autowired 依赖
7. `global.DefaultResourceManger.Signal()` — 优雅退出

### 后端分层架构

```
api/ (Handler 层)
  ├── flow_handler.go    → app.RequestContext 参数解析 → 调用 Service
  ├── run_handler.go     → 执行触发/状态查询
  └── middleware.go      → CORS、日志、错误恢复

service/ (业务逻辑层)
  ├── flow_service.go    → Flow CRUD + 校验 + 转换
  └── run_service.go     → 执行生命周期管理

converter/ (转换层)
  └── flow_converter.go  → Flow JSON ↔ taskx.Task/Subtask/DAG

protocol/ (协议注册中心)
  ├── registry.go        → 注册/查找 ProtocolFactory
  ├── http_protocol.go   → HTTP ExecutorProvider 工厂
  ├── grpc_protocol.go   → gRPC ExecutorProvider 工厂
  ├── local_protocol.go  → Local ExecutorProvider 工厂
  └── mcp_protocol.go    → MCP ExecutorProvider 工厂

model/ (数据模型)
  ├── flow.go            → Flow bun model（含 nodes/edges JSON 列）
  └── dao/               → FlowDAO (db/v1)
```

### 数据模型设计

**flow 表** — 存储工作流定义：

| 列 | 类型 | 说明 |
|---|------|------|
| id | VARCHAR PK | Flow ID |
| name | VARCHAR | 名称 |
| description | TEXT | 描述 |
| nodes | TEXT (JSON) | 节点列表 JSON |
| edges | TEXT (JSON) | 边列表 JSON |
| settings | TEXT (JSON) | 全局配置 |
| status | INT8 | 0=禁用, 1=启用 |
| create_time | DATETIME | 创建时间 |
| update_time | DATETIME | 更新时间 |

**复用 taskx 表** — task、subtask、task_edge 表不做修改，用于运行时存储执行数据。

### Flow ↔ taskx 转换策略

```
Flow (前端 JSON)
  │
  ▼
FlowConverter.ToTask(flow) → taskx.Task
  ├── flow.nodes[i] → taskx.NewSubtask(name, protocolRegistry.Create(node.protocol, node.config))
  ├── flow.edges[i] → task.AddEdge / AddControlEdge / AddDataEdge
  ├── flow.settings.rollbackStrategy → task.SetRollbackStrategy()
  └── task.Compile() → 校验 DAG 合法性
  
FlowConverter.FromTask(task) → Flow JSON (用于查询)
```

### 协议注册中心设计

```go
type ProtocolRegistry struct {
    factories map[string]*ProtocolFactory
}

// 启动时注册
registry.Register(&ProtocolFactory{
    Name: "http",
    ConfigSchema: `{"type":"object","properties":{"url":{"type":"string"},"method":{"type":"string","enum":["GET","POST"]},...}}`,
    Create: func(config json.RawMessage) (executor.ExecutorProvider, error) {
        var cfg HTTPConfig
        json.Unmarshal(config, &cfg)
        return executor.NewHTTPProvider(cfg.URL, cfg.Method), nil
    },
})
```

## Frontend Design

### 技术栈选型理由

- **ReactFlow**: 成熟的 React DAG 可视化库，支持拖拽、连线、自定义节点/边，完美契合 DAG 编辑器需求
- **Zustand**: 轻量级状态管理，比 Redux 简洁，适合编辑器这类局部复杂状态
- **Tailwind**: 实用优先的 CSS 框架，快速构建一致 UI

### 状态管理设计

```typescript
// stores/appStore.ts — 全局状态
interface AppStore {
  sidebarCollapsed: boolean
  toggleSidebar: () => void
}

// stores/editorStore.ts — 编辑器状态（后续变更扩展）
interface EditorStore {
  nodes: Node[]
  edges: Edge[]
  selectedNodeId: string | null
  // ... 编辑操作
}
```

### DAGViewer 组件设计

DAGViewer 是核心复用组件，封装 ReactFlow 配置：

```typescript
// components/dag/DAGViewer.tsx
<DAGViewer
  nodes={flowNodes}        // 转换后的 ReactFlow 节点
  edges={flowEdges}        // 转换后的 ReactFlow 边
  nodeStates={runStates}   // 可选，运行时状态
  mode="status"            // preview | status
  onNodeClick={handleClick}
  fitView
/>
```

内部自动处理：
- Flow Node → ReactFlow Node 转换（位置、类型、样式）
- Flow Edge → ReactFlow Edge 转换（类型、标签、样式）
- 状态颜色叠加
- MiniMap、缩放控制

## Key Design Decisions

| 决策 | 选择 | 理由 |
|------|------|------|
| 后端框架 | common-tools/web | 项目统一框架，自带 bean 注入、路由、中间件 |
| DAG 引擎 | 复用 taskx | 成熟稳定，支持集群调度、重试、回滚 |
| Flow 存储 | 单表 + JSON 列 | nodes/edges 结构灵活，适合前端编辑场景 |
| 前端 DAG 渲染 | ReactFlow | 社区最成熟的 React DAG 库，API 丰富 |
| 状态轮询 | 前端定时轮询 | 简单可靠，后续可升级为 WebSocket |
| 协议扩展 | Registry 模式 | 工厂+注册，新增协议只需加文件+注册 |
