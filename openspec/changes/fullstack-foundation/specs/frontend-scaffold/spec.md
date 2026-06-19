## frontend-scaffold

前端项目脚手架，提供完整的基础设施支撑后续功能开发。

### 技术栈

| 技术 | 用途 |
|------|------|
| React 18 + TypeScript | UI 框架 |
| Vite | 构建工具 + dev server |
| ReactFlow | DAG 图渲染引擎 |
| Zustand | 状态管理 |
| Tailwind CSS | 样式框架 |
| React Router v6 | 路由 |
| Axios | HTTP 客户端 |

### 项目结构

```
frontend/src/
├── main.tsx              # 入口
├── App.tsx               # 根组件 + 路由配置
├── api/                  # API 通信层
│   ├── client.ts         # Axios 实例（baseURL, 拦截器, 错误处理）
│   ├── flow.ts           # Flow API
│   ├── run.ts            # 执行 API
│   ├── template.ts       # 模板 API（预留）
│   └── history.ts        # 历史 API（预留）
├── pages/                # 页面（占位）
│   ├── flow-list/
│   ├── editor/
│   ├── monitor/
│   ├── template/
│   └── history/
├── components/
│   ├── layout/           # 全局布局
│   │   ├── AppLayout.tsx # 侧边栏 + 顶栏 + 内容区
│   │   ├── Sidebar.tsx   # 导航侧边栏
│   │   └── Header.tsx    # 顶部栏
│   ├── dag/              # DAG 可视化组件
│   └── common/           # 通用组件
├── hooks/                # 自定义 Hooks
├── stores/               # Zustand stores
├── types/                # TypeScript 类型定义
├── utils/                # 工具函数
└── styles/               # 全局样式
```

### 全局布局

```
┌──────────────────────────────────────┐
│  Header（Logo + 面包屑）              │
├──────────┬───────────────────────────┤
│ Sidebar  │                           │
│ ─────── │      Content Area         │
│ Flow 列表│                           │
│ 模板库   │                           │
│ 历史     │                           │
│          │                           │
└──────────┴───────────────────────────┘
```

### 路由配置

| 路径 | 页面 | 说明 |
|------|------|------|
| / | FlowListPage | Flow 列表（默认首页） |
| /flows/:id/edit | EditorPage | DAG 编辑器（占位） |
| /flows/:id/monitor | MonitorPage | 执行监控（占位） |
| /templates | TemplateListPage | 模板库（占位） |
| /history | HistoryListPage | 历史列表（占位） |

### Vite 配置

- dev server proxy: `/api` → `http://localhost:8080`
- 路径别名: `@/` → `src/`

### API Client 规范

- 统一 baseURL: `/api/v1`
- 请求拦截器: 注入 X-Request-Id
- 响应拦截器: 统一错误提示、401 处理（预留）
- 所有 API 函数返回 `Promise<T>` 并使用 TypeScript 泛型
