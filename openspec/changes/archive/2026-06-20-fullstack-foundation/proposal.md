## Why

DAGFlow 需要一个前后端分离的可视化工作流平台，面向后端研发人员用于技术运维和问题排查。现有 taskx 模块提供了成熟的 DAG 调度引擎（支持集群调度、重试、回滚、条件分支等），但缺少 Web UI 和 REST API，无法通过可视化方式构建、监控和管理 DAG 工作流。本次变更构建全栈基座，为后续的 DAG 编辑器、执行监控、模板库等功能提供基础支撑。

## What Changes

### 后端
- 新增 REST API 层，基于 `common-tools/web` 框架，使用 `web.Engine` + `RouterGroup` 注册路由
- 实现 Flow CRUD API（创建/查询/更新/删除工作流定义）
- 实现执行触发 API（启动执行、状态轮询）
- 新增 Flow 数据模型和 DAO 层，使用 `common-tools/db/v1`（bun ORM）持久化 Flow 定义（节点+边 JSON）
- 实现 Flow ↔ taskx DAG 转换层（converter），将前端 JSON 格式的 Flow 转换为 taskx Task/Subtask/DAG 对象
- 实现协议注册中心（protocol registry），提供可插拔的 ExecutorProvider 工厂机制
- 应用入口 `cmd/server/main.go`，遵循 common-tools 启动顺序（constants → logger → bean → web → dao → bean.Ioc）

### 前端
- 使用 React + Vite + TypeScript 初始化前端项目
- 集成 ReactFlow 作为 DAG 图渲染引擎
- 集成 Zustand 状态管理、Tailwind CSS 样式框架
- 实现全局布局（AppLayout：侧边栏导航 + 顶部栏 + 内容区）
- 实现路由框架（React Router v6），预置页面路由：Flow 列表、编辑器、监控、模板、历史
- 封装 API Client 层（Axios 实例 + 拦截器 + 各模块 API 函数）
- 实现基础 DAG 图渲染组件（DAGViewer），可渲染只读 DAG 图（后续编辑器和监控页复用）
- Vite dev server 配置 proxy 代理后端 API

## Capabilities

### New Capabilities
- `flow-api`: Flow 工作流定义的 REST API（CRUD + 校验 + Flow↔taskx 转换）
- `run-api`: 执行触发和状态轮询 API（启动执行、轮询 Task/Subtask 状态）
- `protocol-registry`: 可插拔协议注册中心（注册/发现 ExecutorProvider，提供 Schema 描述）
- `frontend-scaffold`: 前端项目脚手架（React+Vite+ReactFlow+Zustand+Tailwind，全局布局、路由、API Client）
- `dag-viewer`: DAG 图只读渲染组件（基于 ReactFlow，节点/边状态可视化）

### Modified Capabilities
<!-- 无现有 capability 需要修改 -->

## Impact

- **新增代码**：`backend/cmd/server/`、`backend/internal/`（api/service/converter/protocol/model 全部新增）
- **新增前端**：`frontend/` 整个目录从零搭建
- **依赖新增**：后端新增 `common-tools/web` 相关依赖（已在 go.mod 中）；前端新增 React、ReactFlow、Zustand、Tailwind、Axios 等 npm 依赖
- **不修改**：现有 `taskx/` 核心引擎代码不做任何修改
- **数据库**：新增 Flow 表（存储工作流定义的 JSON），复用 taskx 已有的 task/subtask/task_edge 表
- **API 影响**：新增 `/api/v1/flows`、`/api/v1/runs` 等 RESTful 端点
