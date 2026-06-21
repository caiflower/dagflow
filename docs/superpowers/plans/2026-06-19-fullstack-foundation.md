---
change: fullstack-foundation
design-doc: openspec/changes/fullstack-foundation/.comet/handoff/design-doc.md
base-ref: d790d4f0a6b34f4e9451b19c3e4ab1a229b4d506
---

# DAGFlow Fullstack Foundation — Implementation Plan

## 概述

基于 taskx DAG 引擎，构建前后端分离的可视化工作流平台基座。后端使用 common-tools/web 框架提供 REST API，前端使用 React + ReactFlow 提供可视化界面。

## 实施阶段

### Phase 1: 后端基础设施 (T1-T3)

#### T1: 后端项目骨架
**目标**: 创建应用入口和配置框架

**实现步骤**:
1. 创建 `backend/cmd/server/main.go`，实现标准启动流程
2. 创建 `backend/constants/config.go`，定义配置结构体
3. 创建 `backend/etc/config.yaml` 和 `backend/etc/default.yaml`
4. 实现 init() 函数：InitConfig → InitLogger → initBean → web.Init → dao.Init → bean.Ioc
5. 实现 main() 函数：调用 global.DefaultResourceManger.Signal()

**验收**: 应用可启动，输出启动日志

#### T2: Web 引擎初始化
**目标**: 创建 REST API 路由和中间件

**实现步骤**:
1. 创建 `backend/internal/api/router.go`
2. 实现 RegisterRoutes 函数，使用 engine.Group("/api/v1")
3. 创建 `backend/internal/api/middleware.go`
4. 实现 CORS 中间件（允许前端跨域）
5. 实现请求日志中间件（记录 method/path/status/duration）
6. 实现错误恢复中间件（捕获 panic，返回统一错误格式）

**验收**: 路由注册成功，中间件生效

#### T3: 数据库初始化 + Flow 表
**目标**: 创建 Flow 数据模型和 DAO

**实现步骤**:
1. 创建 `backend/internal/model/flow.go`，定义 Flow struct（bun model）
2. 创建 `backend/internal/model/dao/flow_dao.go`，实现 FlowDAO
3. 实现 CRUD 方法：Insert, GetByID, Update, Delete, List
4. 创建 `backend/dao/init.go`，实现 Init() 函数（DB 连接 + 建表）
5. 在 main.go 的 initBean() 中注册 FlowDAO bean

**验收**: Flow 表可自动创建，CRUD 操作正常

### Phase 2: 协议注册中心 (T4-T6)

#### T4: 协议注册中心核心
**目标**: 实现 ProtocolRegistry

**实现步骤**:
1. 创建 `backend/internal/protocol/registry.go`
2. 定义 ProtocolFactory struct（Name, DisplayName, ConfigSchema, Create func）
3. 实现 Register/Get/List/CreateProvider 方法
4. 使用 sync.RWMutex 保证并发安全
5. 在 initBean() 中注册 ProtocolRegistry bean

**验收**: 可注册和查找协议

#### T5: 内置协议实现
**目标**: 实现 4 个内置协议

**实现步骤**:
1. 创建 `backend/internal/protocol/http_protocol.go`
   - 定义 HTTPConfig（url, method, headers, body, timeout）
   - 实现 ConfigSchema JSON Schema
   - 实现 Create 函数，返回 executor.HTTPProvider
2. 创建 `backend/internal/protocol/grpc_protocol.go`
   - 定义 GRPCConfig（host, port, service, method, body）
   - 实现 Create 函数
3. 创建 `backend/internal/protocol/local_protocol.go`
   - 定义 LocalConfig（funcName, args）
   - 实现 Create 函数
4. 创建 `backend/internal/protocol/mcp_protocol.go`
   - 定义 MCPConfig（server, tool, args）
   - 实现 Create 函数
5. 在 initBean() 中调用注册函数，启动时自动注册

**验收**: 4 个协议已注册，可创建 ExecutorProvider

#### T6: 协议 API
**目标**: 提供协议查询端点

**实现步骤**:
1. 创建 `backend/internal/api/protocol_handler.go`
2. 实现 List 方法：GET /api/v1/protocols，返回所有协议（含 Schema）
3. 实现 Get 方法：GET /api/v1/protocols/:name，返回单个协议详情
4. 在 router.go 中注册路由

**验收**: API 可查询协议列表和详情

### Phase 3: Flow CRUD (T7-T10)

#### T7: Flow Service
**目标**: 实现 Flow 业务逻辑

**实现步骤**:
1. 创建 `backend/internal/service/flow_service.go`
2. 注入 FlowDAO（autowired）
3. 实现 CreateFlow：校验必填字段 → 生成 ID → 插入 DB
4. 实现 GetFlow：查询 → 反序列化 nodes/edges JSON
5. 实现 UpdateFlow：校验 → 更新 DB
6. 实现 DeleteFlow：删除 DB
7. 实现 ListFlows：分页查询 + 过滤

**验收**: Flow CRUD 业务逻辑完整

#### T8: Flow API Handler
**目标**: 实现 Flow REST 端点

**实现步骤**:
1. 创建 `backend/internal/api/flow_handler.go`
2. 注入 FlowService（autowired）
3. 实现 List：GET /api/v1/flows（query params: page, pageSize, name, status）
4. 实现 Create：POST /api/v1/flows（JSON body）
5. 实现 Get：GET /api/v1/flows/:id
6. 实现 Update：PUT /api/v1/flows/:id
7. 实现 Delete：DELETE /api/v1/flows/:id
8. 使用 app.RequestContext 解析参数

**验收**: Flow REST API 可正常调用

#### T9: Flow 校验 API
**目标**: 实现 DAG 校验端点

**实现步骤**:
1. 在 flow_handler.go 中添加 Validate 方法
2. 实现 POST /api/v1/flows/:id/validate
3. 调用 FlowConverter.ToTask 构建 taskx DAG
4. 调用 task.Compile() 校验（环检测、分支目标、类型兼容性）
5. 返回校验结果（成功/失败 + 错误信息）

**验收**: 可校验 DAG 合法性

#### T10: FlowConverter
**目标**: 实现 Flow ↔ taskx 转换

**实现步骤**:
1. 创建 `backend/internal/converter/flow_converter.go`
2. 注入 ProtocolRegistry（autowired）
3. 实现 ToTask(flow *model.Flow) (*taskx.Task, error)
   - 解析 nodes JSON → 创建 Subtask（通过 ProtocolRegistry 获取 ExecutorProvider）
   - 解析 edges JSON → 添加 DAG 边（根据 type 调用不同方法）
   - 解析 settings JSON → 设置 Task 配置
   - 调用 task.Compile()
4. 实现 FromTask(task *taskx.Task) (*model.Flow, error)
   - 反向转换（用于查询）

**验收**: 转换逻辑正确，单元测试覆盖

### Phase 4: 执行 API (T11-T12)

#### T11: Run Service
**目标**: 实现执行生命周期管理

**实现步骤**:
1. 创建 `backend/internal/service/run_service.go`
2. 注入 FlowDao, TaskDao, SubtaskDao, Converter（autowired）
3. 实现 StartRun(flowID, params)
   - 加载 Flow → ToTask → taskx.SubmitTask → 返回 runId
4. 实现 GetRunStatus(runID)
   - 查询 task + subtasks → 组装 RunStatus
5. 实现 AbortRun(runID)
   - 调用 taskx 中止机制
6. 实现 RetrySubtask(runID, subtaskID)
   - 重置 subtask 状态 → 重新提交

**验收**: 执行触发和状态查询正常

#### T12: Run API Handler
**目标**: 实现执行 REST 端点

**实现步骤**:
1. 创建 `backend/internal/api/run_handler.go`
2. 注入 RunService（autowired）
3. 实现 Start：POST /api/v1/flows/:id/run（可选 params）
4. 实现 GetStatus：GET /api/v1/runs/:runId
5. 实现 GetSubtasks：GET /api/v1/runs/:runId/subtasks
6. 实现 Abort：POST /api/v1/runs/:runId/abort
7. 实现 RetrySubtask：POST /api/v1/runs/:runId/subtasks/:subtaskId/retry

**验收**: 执行 API 可正常调用

### Phase 5: 前端脚手架 (T13-T18)

#### T13: 前端项目初始化
**目标**: 创建 React + Vite 项目

**实现步骤**:
1. 在 `frontend/` 目录执行 `npm create vite@latest . -- --template react-ts`
2. 安装依赖：`npm install reactflow zustand axios react-router-dom`
3. 安装 Tailwind：`npm install -D tailwindcss postcss autoprefixer && npx tailwindcss init -p`
4. 配置 `vite.config.ts`：
   - proxy: `/api` → `http://localhost:8080`
   - alias: `@/` → `src/`
5. 配置 `tailwind.config.js`：content 路径
6. 在 `src/index.css` 引入 Tailwind

**验收**: `npm run dev` 可启动

#### T14: TypeScript 类型定义
**目标**: 定义前后端对齐的类型

**实现步骤**:
1. 创建 `frontend/src/types/flow.ts`
   - Flow, Node, Edge, FieldMapping, NodePosition
2. 创建 `frontend/src/types/run.ts`
   - RunStatus, SubtaskStatus, TaskState
3. 创建 `frontend/src/types/protocol.ts`
   - Protocol, ConfigSchema

**验收**: 类型定义完整

#### T15: API Client 层
**目标**: 封装 API 调用

**实现步骤**:
1. 创建 `frontend/src/api/client.ts`
   - axios.create({ baseURL: '/api/v1' })
   - 请求拦截器：注入 X-Request-Id
   - 响应拦截器：统一错误处理
2. 创建 `frontend/src/api/flow.ts`
   - getFlows, getFlow, createFlow, updateFlow, deleteFlow, validateFlow
3. 创建 `frontend/src/api/run.ts`
   - startRun, getRunStatus, getSubtasks, abortRun, retrySubtask
4. 创建 `frontend/src/api/protocol.ts`
   - getProtocols, getProtocol

**验收**: API 函数可调用

#### T16: 全局布局组件
**目标**: 创建应用布局

**实现步骤**:
1. 创建 `frontend/src/components/layout/AppLayout.tsx`
   - flex 布局：sidebar + main
2. 创建 `frontend/src/components/layout/Sidebar.tsx`
   - 导航菜单：Flow 列表、模板库、历史
   - 折叠/展开功能
3. 创建 `frontend/src/components/layout/Header.tsx`
   - Logo + 面包屑

**验收**: 布局渲染正确

#### T17: 路由配置
**目标**: 配置 React Router

**实现步骤**:
1. 在 `frontend/src/App.tsx` 配置路由
2. `/` → FlowListPage
3. `/flows/:id/edit` → EditorPage（占位）
4. `/flows/:id/monitor` → MonitorPage（占位）
5. `/templates` → TemplateListPage（占位）
6. `/history` → HistoryListPage（占位）
7. 创建占位页面组件

**验收**: 路由跳转正常

#### T18: 全局状态
**目标**: 创建 Zustand store

**实现步骤**:
1. 创建 `frontend/src/stores/appStore.ts`
   - sidebarCollapsed: boolean
   - toggleSidebar: () => void
2. 在 Sidebar 中使用 store

**验收**: 状态管理生效

### Phase 6: Flow 列表页 (T19)

#### T19: Flow 列表页
**目标**: 实现 Flow 列表展示

**实现步骤**:
1. 创建 `frontend/src/pages/flow-list/FlowListPage.tsx`
2. 使用 useFlow hook 调用 getFlows
3. 实现列表展示（表格模式）
4. 实现搜索过滤（按名称）
5. 实现创建按钮（跳转编辑器）
6. 实现删除按钮（调用 deleteFlow）
7. 点击行跳转编辑器

**验收**: 列表页可展示、搜索、创建、删除 Flow

### Phase 7: DAG 可视化组件 (T20-T23)

#### T20: DAG 节点组件
**目标**: 实现自定义节点

**实现步骤**:
1. 创建 `frontend/src/components/dag/nodes/BaseNode.tsx`
   - 状态颜色、协议图标、名称
   - Handle（输入/输出连接点）
2. 创建 `frontend/src/components/dag/nodes/TaskNode.tsx`
   - 圆角矩形样式
3. 创建 `frontend/src/components/dag/nodes/BranchNode.tsx`
   - 菱形图标
4. 创建 `frontend/src/components/dag/nodes/StartNode.tsx`
   - 圆形，绿色
5. 创建 `frontend/src/components/dag/nodes/EndNode.tsx`
   - 圆形，红色

**验收**: 节点渲染正确

#### T21: DAG 边组件
**目标**: 实现自定义边

**实现步骤**:
1. 创建 `frontend/src/components/dag/edges/ControlEdge.tsx`
   - 实线箭头
2. 创建 `frontend/src/components/dag/edges/DataEdge.tsx`
   - 虚线箭头 + 数据流标注
3. 创建 `frontend/src/components/dag/edges/MixedEdge.tsx`
   - 实线箭头 + 数据流标注

**验收**: 边渲染正确

#### T22: DAGViewer 组件
**目标**: 封装 ReactFlow 容器

**实现步骤**:
1. 创建 `frontend/src/components/dag/DAGViewer.tsx`
2. Props: nodes, edges, nodeStates?, mode, onNodeClick?, fitView?
3. 内部转换：Flow Node → ReactFlow Node
4. 配置 MiniMap、缩放控制
5. 根据 mode 切换样式（preview/status）
6. status 模式下叠加状态颜色

**验收**: DAGViewer 可渲染 DAG 图

#### T23: DAG 工具函数
**目标**: 实现辅助函数

**实现步骤**:
1. 创建 `frontend/src/utils/dagLayout.ts`
   - 自动布局算法（拓扑排序 + 层级布局）
2. 创建 `frontend/src/utils/stateColor.ts`
   - state → color/icon 映射
3. 创建 `frontend/src/utils/flowSerializer.ts`
   - Flow ↔ ReactFlow nodes/edges 转换

**验收**: 工具函数可正常工作

### Phase 8: 集成验证 (T24)

#### T24: 前后端联调验证
**目标**: 验证完整流程

**实现步骤**:
1. 启动后端：`cd backend && go run cmd/server/main.go`
2. 启动前端：`cd frontend && npm run dev`
3. 验证场景：
   - Flow 列表页可拉取数据
   - 创建 Flow 后列表刷新
   - Flow 详情页可查看 DAG 图渲染
4. 记录测试结果

**验收**: 前后端联调成功

## 依赖关系

```
T1 → T2 → T3 → T4 → T5 → T6
                    ↓
              T7 → T8 → T9 → T10 → T11 → T12
                                           ↓
T13 → T14 → T15 → T16 → T17 → T18 → T19 → T20 → T21 → T22 → T23 → T24
```

## 测试策略

- **后端**: 每个 Phase 完成后运行 `go test ./...`
- **前端**: 组件创建后运行 `npm test`（如果配置了测试框架）
- **集成**: T24 进行端到端验证

## 风险与应对

1. **common-tools/web API 不明确**: 实现前阅读 docs/web.md 和源码
2. **FlowConverter 复杂度高**: 编写单元测试，逐步验证
3. **ReactFlow 学习曲线**: 参考官方文档和示例
