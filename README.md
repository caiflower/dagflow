# DAGFlow — 可视化工作流平台

基于 DAG（有向无环图）的可视化工作流编排平台，支持通过拖拽节点构建任务流水线。

## 项目结构

```
dagflow/
├── backend/          # Go 后端
│   ├── cmd/server/   # 入口 main.go
│   ├── constants/    # 配置结构体
│   ├── etc/          # 配置文件 (config.yaml / default.yaml)
│   └── internal/     # 业务代码
│       ├── api/      # gRPC 服务 + 路由注册
│       ├── converter/# Flow ↔ taskx 转换
│       ├── model/    # 数据模型 + DAO
│       ├── proto/    # Protobuf 定义 + 生成代码
│       ├── protocol/ # 协议注册中心
│       └── service/  # 业务逻辑层
├── frontend/         # React 前端
│   └── src/
│       ├── api/      # Axios API 客户端
│       ├── components/dag/  # DAG 可视化组件
│       ├── pages/    # 页面组件
│       ├── store/    # Zustand 状态管理
│       ├── types.ts  # TypeScript 类型定义
│       └── utils/    # 工具函数
└── openspec/         # 需求规格文档
```

## 技术栈

| 层 | 技术 |
|---|---|
| 后端框架 | common-tools (web + db/v1 + bean + logger) |
| API 定义 | Protobuf + gRPC handler (`engine.GRPC()`) |
| ORM | bun (MySQL) |
| 前端框架 | React 19 + TypeScript |
| 构建工具 | Vite 8 |
| DAG 可视化 | @xyflow/react (ReactFlow) v12 |
| 状态管理 | Zustand |
| CSS | Tailwind CSS v4 |

---

## 快速开始

### 1. 启动后端

**前置条件：** Go 1.24+、MySQL（或使用 SQLite 本地开发）

```bash
# 进入后端目录
cd backend

# 下载依赖
go mod tidy

# 检查数据库配置
# 编辑 etc/default.yaml，确认 MySQL 连接信息：
#   database:
#     - name: dagflow
#       dialect: mysql          # 可改为 sqlite
#       url: "127.0.0.1:3306"
#       dbName: dagflow
#       user: root
#       password: "你的密码"

# 启动后端服务（监听 :8080）
go run cmd/server/main.go
```

后端启动成功后，API 地址为 `http://localhost:8080`。

**API 端点列表（RESTful + gRPC handler）：**

| 方法 | 路径 | 说明 | 参数绑定 |
|------|------|------|----------|
| POST | `/api/v1/flows` | 创建 Flow | JSON body |
| GET | `/api/v1/flows` | Flow 列表 | query: page, pageSize, name |
| GET | `/api/v1/flows/:id` | 获取 Flow 详情 | path: id |
| PUT | `/api/v1/flows/:id` | 更新 Flow | path: id + JSON body |
| DELETE | `/api/v1/flows/:id` | 删除 Flow | path: id |
| POST | `/api/v1/flows/validate` | 校验 Flow DAG | JSON body |
| GET | `/api/v1/protocols` | 协议列表 | — |
| GET | `/api/v1/protocols/:name` | 协议详情 | path: name |
| POST | `/api/v1/executions/run` | 执行 Flow | JSON body |
| GET | `/api/v1/executions/:id` | 执行状态查询 | path: id |
| GET | `/api/v1/executions` | 执行记录列表 | — |
| GET | `/api/v1/health` | 健康检查 | — |

### 2. 启动前端

**前置条件：** Node.js 18+

> 前端使用 `npm` 管理依赖（类似 Go 的 `go mod`），`node_modules/` 类似 `vendor/`。

```bash
# 进入前端目录
cd frontend

# 安装依赖（只需首次执行，类似 go mod download）
npm install

# 启动开发服务器（监听 :3000，自动代理 /api → :8080）
npm run dev
```

前端启动后，浏览器打开 `http://localhost:3000` 即可访问。

**前端常用命令：**

```bash
# 开发模式（热更新，改代码自动刷新浏览器）
npm run dev

# 生产构建（输出到 dist/ 目录）
npm run build

# 预览生产构建结果
npm run preview

# 代码检查
npm run lint
```

### 3. 联调验证

确保两个服务同时运行：

```
后端 :8080  ←→  前端 dev server :3000（自动代理 /api 请求到后端）
```

验证步骤：
1. 打开 `http://localhost:3000/flows`，应看到 Flow 列表页
2. 点击「新建 Flow」创建一个 Flow
3. 点击 Flow 卡片进入 DAG 编辑器，可拖拽节点、连线
4. 点击「保存」持久化 DAG 布局

---

## 前端开发备忘（后端工程师向）

### 类比关系

| Go 后端 | React 前端 | 说明 |
|---------|-----------|------|
| `go.mod` | `package.json` | 依赖声明 |
| `go mod tidy` | `npm install` | 安装/更新依赖 |
| `go build` | `npm run build` | 编译/构建 |
| `go run main.go` | `npm run dev` | 运行开发服务器 |
| `internal/service/*.go` | `src/store/index.ts` | 业务逻辑/状态管理 |
| `internal/api/router.go` | `src/api/client.ts` | API 调用封装 |
| `internal/model/*.go` | `src/types.ts` | 数据结构定义 |
| `struct` | `interface` (TypeScript) | 类型定义 |

### 关键概念

- **组件（Component）**：前端的核心构建单元，类似 Go 的 `func` + `struct`。每个 `.tsx` 文件导出一个组件函数。
- **Props**：组件的输入参数，类似 Go 函数参数。
- **State**：组件的内部可变状态，类似 Go struct 的字段（但有响应式更新机制）。
- **Hook（`useEffect`、`useState`）**：React 的状态/副作用管理，类似 Go 的 `init()` + goroutine。
- **Vite**：前端构建工具，类似 Go 的编译器。`npm run dev` 启动的 dev server 支持 HMR（热模块替换），改代码后浏览器自动刷新。

### 修改前端代码后

```bash
# 如果 dev server 在运行，改代码后浏览器会自动刷新，无需手动操作

# 如果要检查类型错误（类似 go vet）
cd frontend && npx tsc --noEmit

# 如果要构建生产包
cd frontend && npm run build
```

### 目录约定

```
src/
├── pages/          # 页面（每个路由对应一个 page 组件）
├── components/     # 可复用 UI 组件
│   ├── dag/        # DAG 可视化相关
│   └── layout/     # 全局布局（侧边栏、顶栏）
├── store/          # 全局状态管理（Zustand store）
├── api/            # HTTP 请求封装（Axios client）
├── utils/          # 工具函数
├── types.ts        # 全局 TypeScript 类型
└── App.tsx         # 路由配置入口
```

---

## 配置说明

### 后端配置

| 文件 | 作用 |
|------|------|
| `etc/default.yaml` | 基础设施配置（Web 端口、数据库连接、日志级别） |
| `etc/config.yaml` | 业务配置 |

### 前端配置

| 文件 | 作用 |
|------|------|
| `vite.config.ts` | 构建配置（开发端口 3000、API 代理到 :8080） |
| `tsconfig.json` | TypeScript 编译选项 |
| `package.json` | 依赖 + scripts |

### 切换数据库为 SQLite（本地开发）

编辑 `backend/etc/default.yaml`：

```yaml
database:
  - name: dagflow
    dialect: sqlite
    url: "./dagflow.db"
    dbName: dagflow
```

这样无需安装 MySQL，适合本地开发调试。
