## Tasks

### 后端 — 基础设施搭建

- [x] **T1: 后端项目骨架** — 创建 `cmd/server/main.go`，实现 common-tools 标准启动流程（config → logger → bean → web → dao → bean.Ioc → Signal）。创建 `etc/config.yaml` 和 `etc/default.yaml` 配置文件模板。创建 `constants/config.go` 定义配置结构体。
- [x] **T2: Web 引擎初始化** — 创建 `internal/api/router.go`，使用 `web.Engine.Group("/api/v1")` 注册路由组（已改为 gRPC handler 方式注册）。创建 `internal/api/middleware.go`，实现 CORS 中间件、请求日志中间件、统一错误恢复中间件。
- [x] **T3: 数据库初始化 + Flow 表迁移** — 创建 `internal/model/flow.go`（bun model），创建 `internal/model/dao/flow_dao.go`（FlowDAO，基于 db/v1）。实现 `dao.Init()` 自动建表。

### 后端 — 协议注册中心

- [x] **T4: 协议注册中心** — 创建 `internal/protocol/registry.go`，实现 ProtocolRegistry（注册/查找/列表 ProtocolFactory）。
- [x] **T5: 内置协议实现** — 创建 `internal/protocol/protocols.go`，实现 HTTP/gRPC/Local/MCP 四种协议的 ConfigSchema 定义和 ExecutorProvider 工厂函数，启动时自动注册。
- [x] **T6: 协议 API** — 实现 ProtocolService gRPC 服务（`internal/api/protocol_grpc_service.go`），提供 List/Get 两个 RPC 方法，通过 `engine.GRPC()` 注册到 `/api/v1/protocols/list`、`/api/v1/protocols/get`。

### 后端 — Flow CRUD

- [x] **T7: Flow Service** — 创建 `internal/service/flow_service.go`，实现 Flow CRUD 业务逻辑（Create/Get/Update/Delete/List），注入 FlowDAO。
- [x] **T8: Flow API Handler** — 实现 FlowService gRPC 服务（`internal/api/flow_grpc_service.go`），定义 Proto Service（Create/Get/List/Update/Delete），通过 `engine.GRPC()` 注册路由。
- [x] **T9: Flow 校验 API** — 实现 `Validate` RPC 方法，通过 converter 构建 taskx DAG 并调用 Compile 校验。

### 后端 — Flow ↔ taskx 转换

- [x] **T10: FlowConverter** — 创建 `internal/converter/flow_converter.go`，实现 `ToTask(flow) → taskx.Task` 和 `FromTask(task) → Flow JSON`。ToTask 负责将 Flow 节点/边 JSON 转换为 taskx Subtask 和 DAG 边，通过 ProtocolRegistry 查找 ExecutorProvider。

### 后端 — 执行 API

- [x] **T11: Run Service** — 创建 `internal/service/execution_service.go`，实现执行触发（FlowConverter.ToTask → taskx.SubmitTask）、状态查询（查询 taskx Task + Subtask 状态）、中止执行。
- [x] **T12: Run API Handler** — 实现 ExecutionService gRPC 服务（`internal/api/execution_grpc_service.go`），提供 Run/Get/List 三个 RPC 方法，通过 `engine.GRPC()` 注册路由。

### 前端 — 项目初始化

- [x] **T13: 前端项目脚手架** — 使用 Vite 初始化 React + TypeScript 项目。安装依赖：reactflow, zustand, tailwindcss, axios, react-router-dom。配置 vite.config.ts（proxy `/api` → `http://localhost:8080`，路径别名 `@/`）。配置 Tailwind。
- [x] **T14: TypeScript 类型定义** — 创建 `src/types.ts`，定义 Flow、Node、Edge、Run、Subtask、Protocol 等 TypeScript 类型，与后端 API 模型对齐。
- [x] **T15: API Client 层** — 创建 `src/api/client.ts`（Axios 实例 + 拦截器）。各模块 API 调用函数待拆分。

### 前端 — 全局布局与路由

- [x] **T16: 全局布局组件** — 创建 `src/components/layout/Layout.tsx`（侧边栏 + 顶栏 + 内容区）。
- [x] **T17: 路由配置** — 创建 `src/App.tsx`，配置 React Router v6 路由。
- [x] **T18: 全局状态** — 创建 `src/store/index.ts`（Zustand，侧边栏折叠等全局状态）。

### 前端 — Flow 列表页

- [x] **T19: Flow 列表页** — 创建 `src/pages/FlowListPage.tsx`，实现 Flow 列表展示（卡片/表格切换）、搜索过滤、创建/删除操作。调用 `api/client.ts` 获取数据。

### 前端 — DAG 可视化组件

- [x] **T20: DAG 节点组件** — 创建 `src/components/dag/nodes/` 目录，实现 BaseNode、TaskNode、BranchNode、StartNode、EndNode 组件。状态颜色映射、协议图标显示。
- [x] **T21: DAG 边组件** — 创建 `src/components/dag/edges/` 目录，实现 ControlEdge、DataEdge、MixedEdge 自定义边组件。
- [x] **T22: DAGViewer 组件** — 创建 `src/components/dag/DAGViewer.tsx`，封装 ReactFlow 容器（节点/边转换、MiniMap、缩放控制、只读模式）。支持 preview 和 status 两种模式。
- [x] **T23: DAG 工具函数** — 创建 `src/utils/dagLayout.ts`（自动布局算法）、`src/utils/stateColor.ts`（状态颜色映射）、`src/utils/flowSerializer.ts`（Flow ↔ ReactFlow 节点/边转换）。

### 集成验证

- [x] **T24: 前后端联调验证** — 后端 `go build ./...` 编译通过，前端 `tsc -b && vite build` 构建通过（244 modules）。FlowEditorPage 已重构使用 DAGViewer 组件 + 工具函数。
