---
change: dag-editor-ux
design-doc: docs/superpowers/specs/2026-06-20-dag-editor-ux-design.md
base-ref: eb4595479655878b53cc60363259e640f4b3a669
archived-with: 2026-06-20-dag-editor-ux
---

# DAG Editor UX Improvements 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** 实现前后端错误透传、Toast 通知系统、DAG 节点 UI 优化四大 UX 改进

**Architecture:** 后端业务错误使用 `e.NewApiError()` 透传 4xx 状态码；前端 `unwrap()` 检测 `error` 字段抛出 `ApiError`；页面 catch 块统一显示 Toast；Toast 基于 Zustand store + Portal 组件实现；节点 UI 在现有设计语言上做渐进增强

**Tech Stack:** Go (common-tools e package), TypeScript/React (Axios, Zustand, ReactFlow, Lucide)

## 全局约束

- 业务错误使用 `github.com/caiflower/common-tools/web/common/e` 包的 `e.NewApiError()`，DB/基础设施错误保持 `fmt.Errorf` → 500
- Toast 消息统一使用英文
- 节点 UI 保持 Apple 风格设计语言，仅做渐进增强
- 所有修改保持最小化，不改变现有接口签名
- 前端使用 Zustand 管理状态，React 函数组件 + Hooks

archived-with: 2026-06-20-dag-editor-ux
---

### Task 1: 后端 flow_service.go 错误透传

**Files:**
- Modify: `backend/internal/service/flow_service.go:44-119`

**Interfaces:**
- Consumes: `github.com/caiflower/common-tools/web/common/e` (已存在于项目依赖)
- Produces: `Create`/`Get`/`Update` 返回 `e.NewApiError()` 替代 `fmt.Errorf` 用于业务错误

**目标：** 将 `flow_service.go` 中的业务错误（验证失败、资源不存在）从 `fmt.Errorf` 改为 `e.NewApiError()`，使前端能收到正确的 4xx HTTP 状态码。DAO 插入/更新失败的 DB 错误保持 `fmt.Errorf` → 500。

- [x] **Step 1: 添加 import**

在 `flow_service.go` 顶部 import 块中添加：

```go
import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/caiflower/common-tools/pkg/basic"
	"github.com/caiflower/common-tools/pkg/bean"
	"github.com/caiflower/common-tools/web/common/e"  // 新增

	"github.com/caiflower/dagflow/internal/converter"
	"github.com/caiflower/dagflow/internal/model"
	"github.com/caiflower/dagflow/internal/model/dao"
)
```

- [x] **Step 2: 修改 `Create` 方法验证错误（第 51 行附近）**

将：
```go
if err := converter.ValidateFlow(req.Nodes, req.Edges); err != nil {
    return nil, fmt.Errorf("invalid flow: %w", err)
}
```

改为：
```go
if err := converter.ValidateFlow(req.Nodes, req.Edges); err != nil {
    return nil, e.NewApiError(e.InvalidArgument, err.Error(), err)
}
```

**注意：** DAO Insert 失败（`s.FlowDAO.Insert`）的错误保持为 `fmt.Errorf("insert flow: %w", err)`，不修改。

- [x] **Step 3: 修改 `Get` 方法错误（第 72 行附近）**

将：
```go
if err != nil {
    return nil, fmt.Errorf("get flow %d: %w", id, err)
}
```

改为：
```go
if err != nil {
    return nil, e.NewApiError(e.NotFound, fmt.Sprintf("flow %d not found", id), err)
}
```

- [x] **Step 4: 修改 `Update` 方法错误（第 91-97 行附近）**

将：
```go
existing, err := s.FlowDAO.GetByID(ctx, req.ID)
if err != nil {
    return nil, fmt.Errorf("flow not found: %w", err)
}
```

改为：
```go
existing, err := s.FlowDAO.GetByID(ctx, req.ID)
if err != nil {
    return nil, e.NewApiError(e.NotFound, fmt.Sprintf("flow %d not found", req.ID), err)
}
```

将：
```go
if req.Nodes != nil {
    if err := converter.ValidateFlow(req.Nodes, req.Edges); err != nil {
        return nil, fmt.Errorf("invalid flow: %w", err)
    }
```

改为：
```go
if req.Nodes != nil {
    if err := converter.ValidateFlow(req.Nodes, req.Edges); err != nil {
        return nil, e.NewApiError(e.InvalidArgument, err.Error(), err)
    }
```

**注意：** DAO Update 失败（`s.FlowDAO.Update`）的错误保持为 `fmt.Errorf("update flow: %w", err)`，不修改。

- [x] **Step 5: 验证编译通过**

```bash
cd backend && go build ./...
```

预期：编译通过，无错误

- [x] **Step 6: 提交**

```bash
git add backend/internal/service/flow_service.go
git commit -m "feat(backend): flow_service 业务错误透传 ApiError"
```

archived-with: 2026-06-20-dag-editor-ux
---

### Task 2: 后端 execution_service.go 错误透传

**Files:**
- Modify: `backend/internal/service/execution_service.go:75-142`

**Interfaces:**
- Consumes: `github.com/caiflower/common-tools/web/common/e`（已在 Task 1 同一 import 路径）
- Produces: `Run`/`GetStatus` 返回 `e.NewApiError()` 替代 `fmt.Errorf` 用于业务错误

**目标：** 将 `execution_service.go` 中的业务错误从 `fmt.Errorf` 改为 `e.NewApiError()`。

- [x] **Step 1: 添加 import**

在 `execution_service.go` 顶部 import 块中添加：

```go
import (
	"context"
	"fmt"
	"time"

	"github.com/caiflower/common-tools/pkg/basic"
	"github.com/caiflower/common-tools/pkg/bean"
	"github.com/caiflower/common-tools/pkg/logger"
	"github.com/caiflower/common-tools/pkg/tools"
	"github.com/caiflower/common-tools/web/common/e"  // 新增

	"github.com/caiflower/dagflow/internal/converter"
	"github.com/caiflower/dagflow/internal/model/dao"
	"github.com/caiflower/dagflow/taskx"
	taskxDAO "github.com/caiflower/dagflow/taskx/dao"
	taskxModel "github.com/caiflower/dagflow/taskx/dao/model"
)
```

- [x] **Step 2: 修改 `Run` 方法错误（第 75-103 行附近）**

逐处修改 `Run` 方法中的错误返回：

```go
// flow not found → NotFound
flow, err := s.FlowDAO.GetByID(ctx, req.FlowID)
if err != nil {
    return nil, e.NewApiError(e.NotFound, fmt.Sprintf("flow %d not found", req.FlowID), err)
}

// parse flow fails → InvalidArgument
flowNodes, flowEdges, err := converter.ParseFlowJSON(flow)
if err != nil {
    return nil, e.NewApiError(e.InvalidArgument, err.Error(), err)
}

// build task fails → InvalidArgument
task, err := converter.FlowToTask(flow, createProvider, req.NodeInputs)
if err != nil {
    return nil, e.NewApiError(e.InvalidArgument, err.Error(), err)
}

// compile task fails → InvalidArgument
if _, err := task.Compile(); err != nil {
    return nil, e.NewApiError(e.InvalidArgument, err.Error(), err)
}

// submit task fails → Internal
if err := taskx.SubmitTask(ctx, task); err != nil {
    return nil, e.NewApiError(e.Internal, err.Error(), err)
}
```

- [x] **Step 3: 修改 `GetStatus` 方法错误（第 124-128 行附近）**

```go
// execution record not found → NotFound
record, err := s.ExecutionRecordDAO.GetByID(ctx, execID)
if err != nil {
    return nil, e.NewApiError(e.NotFound, fmt.Sprintf("execution %s not found", execID), err)
}
if record == nil || record.FlowID == 0 {
    return nil, e.NewApiError(e.NotFound, fmt.Sprintf("execution %s not found", execID))
}
```

- [x] **Step 4: 验证编译通过**

```bash
cd backend && go build ./...
```

预期：编译通过，无错误

- [x] **Step 5: 提交**

```bash
git add backend/internal/service/execution_service.go
git commit -m "feat(backend): execution_service 业务错误透传 ApiError"
```

archived-with: 2026-06-20-dag-editor-ux
---

### Task 3: 前端 ApiError 类 + unwrap() 修改

**Files:**
- Modify: `frontend/src/api/client.ts:1-19`

**Interfaces:**
- Produces: `export class ApiError extends Error { code: number; type: string; constructor(code: number, message: string, type: string) }`
- Produces: `unwrap()` 修改为检测 `r.data.error` 字段并抛出 `ApiError`

**目标：** 在 `api/client.ts` 中新增 `ApiError` 类，修改 `unwrap()` 以检测后端返回的 `error` 字段并抛出类型化异常。

- [x] **Step 1: 新增 `ApiError` 类 + 修改 `unwrap()`**

修改 `frontend/src/api/client.ts`，在 `unwrap` 函数定义前添加：

```typescript
import axios from 'axios';
import type { Flow, FlowNode, FlowEdge, Protocol, Execution, NodeInput, PageResult } from '../types';

const api = axios.create({
  baseURL: '/api/v1',
  timeout: 30000,
});

/** API 业务错误 */
export class ApiError extends Error {
  code: number;
  type: string;
  constructor(code: number, message: string, type: string) {
    super(message);
    this.name = 'ApiError';
    this.code = code;
    this.type = type;
  }
}

/** 解包 GRPC 响应: {requestID, data: <payload>} → payload */
function unwrap<T>(r: { data: { data?: T; error?: { code: number; message: string; type: string } } }): T {
  if (r.data.error) {
    throw new ApiError(r.data.error.code, r.data.error.message, r.data.error.type);
  }
  return r.data.data as T;
}
```

**注意：** 文件其余部分不变（`toProtoNode`、`toProtoEdge`、`flowApi`、`protocolApi`、`executionApi` 等均保持不变）。

- [x] **Step 2: 运行前端类型检查**

```bash
cd frontend && npx tsc --noEmit
```

预期：无类型错误

- [x] **Step 3: 提交**

```bash
git add frontend/src/api/client.ts
git commit -m "feat(frontend): 新增 ApiError 类和 unwrap 错误检测"
```

archived-with: 2026-06-20-dag-editor-ux
---

### Task 4: Toast Store 创建

**Files:**
- Create: `frontend/src/store/toastStore.ts`

**Interfaces:**
- Produces: `export interface Toast { id: string; type: 'success' | 'error' | 'info'; message: string; }`
- Produces: `export const useToastStore = create<ToastStore>(...)`
- Produces: `export const toast = { success(msg), error(msg), info(msg) }`

**目标：** 创建基于 Zustand 的 toast 状态管理 store，支持添加/移除 toast，并导出便捷方法。

- [x] **Step 1: 创建 toastStore.ts**

创建文件 `frontend/src/store/toastStore.ts`，完整内容：

```typescript
import { create } from 'zustand';

export interface Toast {
  id: string;
  type: 'success' | 'error' | 'info';
  message: string;
}

interface ToastStore {
  toasts: Toast[];
  add: (type: Toast['type'], message: string) => void;
  dismiss: (id: string) => void;
}

let toastId = 0;

export const useToastStore = create<ToastStore>((set) => ({
  toasts: [],

  add: (type, message) => {
    const id = `toast-${++toastId}`;
    set((s) => ({
      toasts: [...s.toasts, { id, type, message }],
    }));
    // 3 秒后自动消失
    setTimeout(() => {
      set((s) => ({
        toasts: s.toasts.filter((t) => t.id !== id),
      }));
    }, 3000);
  },

  dismiss: (id) => {
    set((s) => ({
      toasts: s.toasts.filter((t) => t.id !== id),
    }));
  },
}));

/** 便捷方法 */
export const toast = {
  success: (message: string) => useToastStore.getState().add('success', message),
  error: (message: string) => useToastStore.getState().add('error', message),
  info: (message: string) => useToastStore.getState().add('info', message),
};
```

- [x] **Step 2: 运行类型检查**

```bash
cd frontend && npx tsc --noEmit
```

预期：无类型错误

- [x] **Step 3: 提交**

```bash
git add frontend/src/store/toastStore.ts
git commit -m "feat(frontend): 创建 Toast Zustand store"
```

archived-with: 2026-06-20-dag-editor-ux
---

### Task 5: Toast UI 组件

**Files:**
- Create: `frontend/src/components/ui/Toast.tsx`
- Modify: `frontend/src/App.tsx:1-19`

**Interfaces:**
- Consumes: `useToastStore` from `../../store/toastStore`
- Produces: `export default function ToastProvider()` — 挂载到 App.tsx 顶层

**目标：** 创建 Toast UI 组件（success/error/info 三种样式），通过 React Portal 渲染到 `document.body`，支持自动消失、手动关闭、入场/退场动画。

- [x] **Step 1: 创建 Toast UI 组件**

创建文件 `frontend/src/components/ui/Toast.tsx`，完整内容：

```typescript
import { useEffect, useState } from 'react';
import { createPortal } from 'react-dom';
import { CheckCircle, XCircle, Info, X } from 'lucide-react';
import { useToastStore } from '../../store/toastStore';

const iconMap = {
  success: { Icon: CheckCircle, color: 'var(--success)' },
  error: { Icon: XCircle, color: 'var(--danger)' },
  info: { Icon: Info, color: 'var(--info)' },
};

const typeStyles: Record<string, { border: string; bg: string }> = {
  success: { border: 'var(--success)', bg: 'var(--success-subtle)' },
  error: { border: 'var(--danger)', bg: 'var(--danger-subtle)' },
  info: { border: 'var(--info)', bg: 'var(--info-subtle)' },
};

function ToastItem({ id, type, message }: { id: string; type: 'success' | 'error' | 'info'; message: string }) {
  const dismiss = useToastStore((s) => s.dismiss);
  const [visible, setVisible] = useState(false);

  useEffect(() => {
    // 入场动画
    requestAnimationFrame(() => setVisible(true));
  }, []);

  const handleDismiss = () => {
    setVisible(false);
    setTimeout(() => dismiss(id), 200);
  };

  const { Icon } = iconMap[type];
  const styles = typeStyles[type];

  return (
    <div
      role="alert"
      className={visible ? 'toast-enter-active' : 'toast-exit'}
      style={{
        display: 'flex',
        alignItems: 'flex-start',
        gap: 10,
        maxWidth: '360px',
        padding: '12px 16px',
        borderRadius: 'var(--radius-md)',
        border: `1px solid ${styles.border}`,
        background: 'var(--bg-elevated)',
        boxShadow: 'var(--shadow-lg)',
        backdropFilter: 'blur(12px)',
      }}
    >
      <Icon size={18} style={{ color: iconMap[type].color, flexShrink: 0, marginTop: 1 }} />
      <span style={{ flex: 1, fontSize: 13, color: 'var(--text-primary)', lineHeight: 1.4 }}>
        {message}
      </span>
      <button
        onClick={handleDismiss}
        style={{
          flexShrink: 0,
          padding: 2,
          cursor: 'pointer',
          color: 'var(--text-muted)',
          background: 'none',
          border: 'none',
          borderRadius: 'var(--radius-sm)',
        }}
        aria-label="Dismiss"
      >
        <X size={14} />
      </button>
    </div>
  );
}

export default function ToastProvider() {
  const toasts = useToastStore((s) => s.toasts);
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    setMounted(true);
  }, []);

  if (!mounted || toasts.length === 0) return null;

  return createPortal(
    <div
      style={{
        position: 'fixed',
        top: 16,
        right: 16,
        zIndex: 9999,
        display: 'flex',
        flexDirection: 'column',
        gap: 8,
      }}
    >
      <style>{`
        .toast-enter-active {
          animation: slideInRight 0.3s ease-out forwards;
        }
        .toast-exit {
          opacity: 0;
          transform: translateX(100%);
          transition: opacity 0.2s ease, transform 0.2s ease;
        }
        @keyframes slideInRight {
          from { opacity: 0; transform: translateX(100%); }
          to { opacity: 1; transform: translateX(0); }
        }
      `}</style>
      {toasts.map((t) => (
        <ToastItem key={t.id} id={t.id} type={t.type} message={t.message} />
      ))}
    </div>,
    document.body,
  );
}
```

- [x] **Step 2: 在 App.tsx 中挂载 ToastProvider**

修改 `frontend/src/App.tsx`，添加 import 和 `<ToastProvider />`：

```tsx
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { ThemeProvider } from './components/ThemeProvider';
import ToastProvider from './components/ui/Toast';  // 新增
import Layout from './components/layout/Layout';
import FlowListPage from './pages/FlowListPage';
import FlowEditorPage from './pages/FlowEditorPage';
import ExecutionPage from './pages/ExecutionPage';
import ProtocolPage from './pages/ProtocolPage';

export default function App() {
  return (
    <ThemeProvider>
      <BrowserRouter>
        <ToastProvider />  {/* 新增 */}
        <Routes>
          <Route element={<Layout />}>
            <Route path="/" element={<Navigate to="/flows" replace />} />
            <Route path="/flows" element={<FlowListPage />} />
            <Route path="/flows/:id" element={<FlowEditorPage />} />
            <Route path="/executions" element={<ExecutionPage />} />
            <Route path="/protocols" element={<ProtocolPage />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </ThemeProvider>
  );
}
```

- [x] **Step 3: 运行类型检查和构建验证**

```bash
cd frontend && npx tsc --noEmit && npx vite build
```

预期：无错误，构建成功

- [x] **Step 4: 提交**

```bash
git add frontend/src/components/ui/Toast.tsx frontend/src/App.tsx
git commit -m "feat(frontend): 创建 Toast UI 组件并挂载到 App"
```

archived-with: 2026-06-20-dag-editor-ux
---

### Task 6: 页面 Catch 块添加 Toast 错误提示（FlowListPage）

**Files:**
- Modify: `frontend/src/pages/FlowListPage.tsx`

**Interfaces:**
- Consumes: `ApiError` from `../api/client`, `toast` from `../store/toastStore`

**目标：** 修改 `FlowListPage.tsx` 中 store 操作的 catch 块，用 toast 显示具体错误信息。

- [x] **Step 1: 添加 import**

在 `FlowListPage.tsx` 顶部添加：

```tsx
import { toast } from '../store/toastStore';  // 在现有 import 中加入
```

无需单独 import `ApiError`，因为 `FlowListPage` 调用的是 store 的方法，store 内部调用 `flowApi`。但 store 的 `loadFlows`/`deleteFlow`/`createFlow` 抛出的错误会被传播上来。我们需要在 store 层面处理。**注意：** 当前 store 的 `loadFlows` 用 try/finally 吞掉了错误。所以需要同时修改 store。

先修改 `frontend/src/store/index.ts` 中的 `loadFlows`、`loadFlow`、`loadProtocols`、`loadExecutions`，在 catch 中调用 toast：

**修改 `frontend/src/store/index.ts`：**

在文件头部添加 import：
```typescript
import { toast } from './toastStore';
```

然后在各个 store 方法中添加错误处理。不过这个设计更合理的方式是在页面层处理，因为每个页面的错误上下文不同。

因此，**暂时保持 store 层的 try/finally 不变**，在页面层对直接调用的 API 方法做 try/catch：

`FlowListPage.tsx` 中 `handleCreate` 添加 catch：

```tsx
import { ApiError } from '../api/client';
import { toast } from '../store/toastStore';
```

将 `handleCreate` 修改为：

```tsx
const handleCreate = async () => {
  if (!newName.trim()) return;
  const defaultNodes = [
    { id: 'start', name: '开始', type: 'start' as const, protocol: '', config: {}, position: { x: 250, y: 50 } },
    { id: 'end', name: '结束', type: 'end' as const, protocol: '', config: {}, position: { x: 250, y: 400 } },
  ];
  try {
    await createFlow({ name: newName, description: newDesc, nodes: defaultNodes, edges: [] });
    setShowCreate(false);
    setNewName('');
    setNewDesc('');
  } catch (e) {
    const msg = e instanceof ApiError ? e.message : 'Failed to create flow';
    toast.error(msg);
  }
};
```

同时，`deleteFlow` 的 store 方法也需要处理。修改 `frontend/src/store/index.ts` 中的 `deleteFlow`：

```typescript
deleteFlow: async (id: number) => {
  try {
    await flowApi.delete(id);
    set((s) => ({ flows: s.flows.filter((f) => f.id !== id) }));
  } catch (e) {
    const msg = e instanceof ApiError ? e.message : 'Failed to delete flow';
    toast.error(msg);
  }
},
```

**注意：** 需要在 `frontend/src/store/index.ts` 头部新增 import：
```typescript
import { ApiError } from '../api/client';
import { toast } from './toastStore';
```

- [x] **Step 2: 运行类型检查**

```bash
cd frontend && npx tsc --noEmit
```

预期：无类型错误

- [x] **Step 3: 提交**

```bash
git add frontend/src/pages/FlowListPage.tsx frontend/src/store/index.ts
git commit -m "feat(frontend): FlowListPage 添加 toast 错误提示"
```

archived-with: 2026-06-20-dag-editor-ux
---

### Task 7: 页面 Catch 块添加 Toast 错误提示（FlowEditorPage）

**Files:**
- Modify: `frontend/src/pages/FlowEditorPage.tsx`

**Interfaces:**
- Consumes: `ApiError` from `../api/client`, `toast` from `../store/toastStore`

**目标：** 修改 `FlowEditorPage.tsx` 中的 `handleSave` / `handleRun` 添加 try/catch + toast。

- [x] **Step 1: 添加 import**

在 `FlowEditorPage.tsx` 顶部添加：

```tsx
import { ApiError } from '../api/client';
import { toast } from '../store/toastStore';
```

- [x] **Step 2: 修改 handleSave 方法**

找到 `handleSave` 函数，添加 try/catch：

```tsx
const handleSave = useCallback(async () => {
  if (!id || !currentFlow) return;
  setSaving(true);
  try {
    const flowNodes = reactFlowToFlowNodes(nodes);
    const flowEdges = reactFlowToFlowEdges(edges);
    await flowApi.update({ id: currentFlow.id, name: currentFlow.name, nodes: flowNodes, edges: flowEdges });
    toast.success('Flow saved');
  } catch (e) {
    const msg = e instanceof ApiError ? e.message : 'Failed to save flow';
    toast.error(msg);
  } finally {
    setSaving(false);
  }
}, [id, currentFlow, nodes, edges]);
```

- [x] **Step 3: 修改 handleRun 方法**

找到 `handleRun`（或 `runFlow` 调用位置），添加 try/catch：

```tsx
const handleRun = useCallback(async () => {
  if (!id) return;
  setRunning(true);
  try {
    await runFlow(Number(id), nodeInputs);
    setShowRunModal(false);
  } catch (e) {
    const msg = e instanceof ApiError ? e.message : 'Failed to run flow';
    toast.error(msg);
  } finally {
    setRunning(false);
  }
}, [id, runFlow, nodeInputs]);
```

- [x] **Step 4: 修改 loadFlow catch 块**

在 `useEffect` 中 `loadFlow` 调用如果失败也需要处理，修改 store 的 `loadFlow` 方法（在 `frontend/src/store/index.ts` 中已处理，此步骤确认即可）：

```typescript
loadFlow: async (id: number) => {
  set({ loading: true });
  try {
    const flow = await flowApi.get(id);
    set({ currentFlow: flow });
  } catch (e) {
    const msg = e instanceof ApiError ? e.message : 'Failed to load flow';
    toast.error(msg);
  } finally {
    set({ loading: false });
  }
},
```

- [x] **Step 5: 运行类型检查**

```bash
cd frontend && npx tsc --noEmit
```

预期：无类型错误

- [x] **Step 6: 提交**

```bash
git add frontend/src/pages/FlowEditorPage.tsx frontend/src/store/index.ts
git commit -m "feat(frontend): FlowEditorPage 添加 toast 错误提示"
```

archived-with: 2026-06-20-dag-editor-ux
---

### Task 8: 页面 Catch 块添加 Toast 错误提示（ExecutionPage + ProtocolPage）

**Files:**
- Modify: `frontend/src/pages/ExecutionPage.tsx`
- Modify: `frontend/src/pages/ProtocolPage.tsx`
- Modify: `frontend/src/store/index.ts`

**Interfaces:**
- Consumes: `ApiError`, `toast`

**目标：** 修改 `ExecutionPage.tsx` 和 `ProtocolPage.tsx` 的 store 方法添加错误 toast。

- [x] **Step 1: 修改 store 的 `loadExecutions`**

在 `frontend/src/store/index.ts` 中修改：

```typescript
loadExecutions: async (page = 1, pageSize = 20, flowId?: number) => {
  set({ loading: true });
  try {
    const result = await executionApi.list(page, pageSize, flowId);
    set({ executions: result.items || [], total: result.total || 0 });
  } catch (e) {
    const msg = e instanceof ApiError ? e.message : 'Failed to load executions';
    toast.error(msg);
  } finally {
    set({ loading: false });
  }
},
```

- [x] **Step 2: 修改 store 的 `runFlow`**

在 `frontend/src/store/index.ts` 中修改 `runFlow`（不 catch，让调用方处理 — 已在 FlowEditorPage 的 handleRun 中处理）。保持原样。

- [x] **Step 3: 修改 store 的 `loadProtocols`**

```typescript
loadProtocols: async () => {
  try {
    const protocols = await protocolApi.list();
    set({ protocols });
  } catch (e) {
    const msg = e instanceof ApiError ? e.message : 'Failed to load protocols';
    toast.error(msg);
  }
},
```

- [x] **Step 4: 验证 store/index.ts 整体一致性**

确保 `frontend/src/store/index.ts` 中所有 catch 块都使用了 `ApiError` 和 `toast`，import 已在 Task 6 中添加。完整 import 应包含：

```typescript
import { create } from 'zustand';
import type { Flow, FlowNode, FlowEdge, Protocol, Execution, NodeInput } from '../types';
import { flowApi, protocolApi, executionApi, ApiError } from '../api/client';
import { toast } from './toastStore';
```

- [x] **Step 5: 运行类型检查**

```bash
cd frontend && npx tsc --noEmit
```

预期：无类型错误

- [x] **Step 6: 提交**

```bash
git add frontend/src/pages/ExecutionPage.tsx frontend/src/pages/ProtocolPage.tsx frontend/src/store/index.ts
git commit -m "feat(frontend): ExecutionPage/ProtocolPage 添加 toast 错误提示"
```

archived-with: 2026-06-20-dag-editor-ux
---

### Task 9: DAG 节点 BaseNode.tsx UI 增强

**Files:**
- Modify: `frontend/src/components/dag/nodes/BaseNode.tsx:1-98`
- Modify: `frontend/src/index.css`

**Interfaces:**
- Consumes: `getNodeStateColor` from `../../../utils/stateColor`
- Produces: 增强的 BaseNode（渐变 accent 条纹、hover 缩放+阴影、Handle 放大+悬停效果、运行态脉冲动画、圆角增大）

**目标：** 渐进增强 BaseNode 视觉：accent 条纹改用渐变、添加 hover 效果、增大 Handle 和圆角、运行态使用脉冲动画。

- [x] **Step 1: 修改 BaseNode 主体样式**

修改 `frontend/src/components/dag/nodes/BaseNode.tsx`：

**圆角增大：** `rounded-[var(--radius-md)]` → `rounded-[var(--radius-lg)]`

**阴影层：** 使用 composited shadow 叠加 `shadow-lg` 在 hover 时

**Accent 条纹 → 渐变：**

```tsx
{/* Accent stripe */}
<div
  className="h-0.5"
  style={{
    background: accentColor
      ? `linear-gradient(to right, ${accentColor}, ${accentColor}4D)`
      : 'linear-gradient(to right, var(--accent-primary), color-mix(in srgb, var(--accent-primary) 30%, transparent))',
  }}
/>
```

**Handle 尺寸 + hover 效果：**

将 `!w-2.5 !h-2.5` 改为 `!w-3 !h-3`，添加 hover 放大：

```tsx
{/* In Handle */}
{data.nodeType !== 'start' && (
  <Handle
    type="target"
    position={Position.Top}
    className="!w-3 !h-3 !border-2 !transition-transform hover:!scale-[1.3]"
    style={{
      background: 'var(--node-bg)',
      borderColor: accentColor || 'var(--border-default)',
    }}
  />
)}
```

```tsx
{/* Out Handle */}
{data.nodeType !== 'end' && (
  <Handle
    type="source"
    position={Position.Bottom}
    className="!w-3 !h-3 !border-2 !transition-transform hover:!scale-[1.3]"
    style={{
      background: 'var(--node-bg)',
      borderColor: accentColor || 'var(--border-default)',
    }}
  />
)}
```

**Hover 效果 + transition：**

在根 div 上添加 hover 缩放 + 阴影。修改 style 和 className：

```tsx
<div
  className="relative min-w-[160px] rounded-[var(--radius-lg)] overflow-hidden transition-all duration-300 cursor-pointer hover:scale-[1.02]"
  style={{
    background: 'var(--node-bg)',
    border: `1.5px solid ${isSelected ? 'var(--node-selected-border)' : 'var(--node-border)'}`,
    boxShadow: isSelected
      ? '0 0 0 3px color-mix(in srgb, var(--node-selected-border) 30%, transparent), var(--shadow-md)'
      : stateColor.glow !== 'none'
        ? `${stateColor.glow}${state === 'running' ? ', 0 0 12px var(--glow-color, #2997ff80)' : ''}`
        : 'var(--shadow-sm)',
  }}
  onMouseEnter={(e) => {
    if (!isSelected) {
      e.currentTarget.style.boxShadow = 'var(--shadow-lg)';
    }
  }}
  onMouseLeave={(e) => {
    if (!isSelected) {
      e.currentTarget.style.boxShadow = stateColor.glow !== 'none'
        ? `${stateColor.glow}${state === 'running' ? ', 0 0 12px var(--glow-color, #2997ff80)' : ''}`
        : 'var(--shadow-sm)';
    }
  }}
>
```

**运行态脉冲动画：** 当 `state === 'running'` 时，添加 CSS class `animate-pulse-glow`

在根 div 的 className 中添加条件：

```tsx
className={`relative min-w-[160px] rounded-[var(--radius-lg)] overflow-hidden transition-all duration-300 cursor-pointer hover:scale-[1.02]${state === 'running' ? ' animate-pulse-glow' : ''}`}
```

- [x] **Step 2: 在 index.css 中添加 pulse-glow 动画**

在 `frontend/src/index.css` 的动画区域（`/* ========== Animations ========== */` 块）添加：

```css
@keyframes pulseGlow {
  0%, 100% { box-shadow: 0 0 4px var(--glow-color, rgba(41, 151, 255, 0.5)); }
  50% { box-shadow: 0 0 14px var(--glow-color, rgba(41, 151, 255, 0.5)); }
}
.animate-pulse-glow {
  animation: pulseGlow 2s ease-in-out infinite;
}
```

- [x] **Step 3: 提交**

```bash
git add frontend/src/components/dag/nodes/BaseNode.tsx frontend/src/index.css
git commit -m "feat(frontend): BaseNode UI 增强 - 渐变条纹/hover/Handle/脉冲动画"
```

archived-with: 2026-06-20-dag-editor-ux
---

### Task 10: DAG 节点 nodes/index.tsx 协议图标 + config chip 优化

**Files:**
- Modify: `frontend/src/components/dag/nodes/index.tsx:1-96`

**Interfaces:**
- Consumes: `BaseNode`, `getNodeTypeColor`, `getProtocolLucideIcon`
- Produces: TaskNode 协议图标通过 icon prop 传入，config 展示使用 chip 样式

**目标：** TaskNode 的协议图标前置到 BaseNode icon 区域；config 展示改为 chip 样式（subtle 背景 + 边框）。

**说明：** 当前 `nodes/index.tsx` 中 TaskNode 已经通过 `icon` prop 将协议图标前置到 BaseNode 了，设计文档的要求基本已满足。需要修改的是 Config 预览部分（extra 区域）从普通文本改为 chip 样式。

- [x] **Step 1: 修改 TaskNode extra 区域的 config 展示**

将 TaskNode 的 `extra` 区域从当前：
```tsx
extra={
  data.config && Object.keys(data.config).length > 0 ? (
    <div className="text-[10px] space-y-0.5">
      {Object.entries(data.config).slice(0, 3).map(([k, v]) => (
        <div key={k} className="truncate">
          <span style={{ color: 'var(--text-muted)' }}>{k}:</span>{' '}
          <span style={{ color: 'var(--text-secondary)' }}>{String(v).slice(0, 20)}</span>
        </div>
      ))}
      {Object.keys(data.config).length > 3 && (
        <div style={{ color: 'var(--text-muted)' }}>+{Object.keys(data.config).length - 3} more</div>
      )}
    </div>
  ) : null
}
```

改为 chip 样式：
```tsx
extra={
  data.config && Object.keys(data.config).length > 0 ? (
    <div className="flex flex-wrap gap-1">
      {Object.entries(data.config).slice(0, 3).map(([k, v]) => (
        <span
          key={k}
          className="text-[10px] px-1.5 py-0.5 rounded-[var(--radius-sm)] truncate max-w-[120px]"
          style={{
            background: 'var(--bg-tertiary)',
            border: '1px solid var(--border-subtle)',
            color: 'var(--text-secondary)',
          }}
        >
          {k}: {String(v).slice(0, 12)}
        </span>
      ))}
      {Object.keys(data.config).length > 3 && (
        <span
          className="text-[10px] px-1.5 py-0.5 rounded-[var(--radius-sm)]"
          style={{ color: 'var(--text-muted)' }}
        >
          +{Object.keys(data.config).length - 3}
        </span>
      )}
    </div>
  ) : null
}
```

- [x] **Step 2: 所有节点 label font-weight medium → semibold**

在 `BaseNode.tsx` 中对应的 label 处已经使用 `font-medium`，改为 `font-semibold`：

```tsx
<div className="text-sm font-semibold truncate" style={{ color: 'var(--text-primary)' }}>
  {data.label}
</div>
```

- [x] **Step 3: 协议 badge 与节点类型名间距**

在 `BaseNode.tsx` 的 body 区域，协议 badge 已有 `mt-0.5` 间距。如需增加间距，在协议 badge 上添加 `ml-0.5`（在显示协议的行中，只影响 `data.protocol` 所在的 div）：

当前代码在 `BaseNode.tsx`:
```tsx
<div className="flex items-center gap-1.5 mt-0.5">
```

保持不变即可（gap-1.5 已提供间距）。

- [x] **Step 4: 运行类型检查**

```bash
cd frontend && npx tsc --noEmit
```

预期：无类型错误

- [x] **Step 5: 提交**

```bash
git add frontend/src/components/dag/nodes/index.tsx frontend/src/components/dag/nodes/BaseNode.tsx
git commit -m "feat(frontend): TaskNode config chip 样式 + label semibold"
```

archived-with: 2026-06-20-dag-editor-ux
---

### Task 11: stateColor.ts 颜色调整

**Files:**
- Modify: `frontend/src/utils/stateColor.ts:18-25`

**Interfaces:**
- Consumes: 无
- Produces: 更新后的 `nodeStateColorMap` 和 `nodeTypeColorMap` 暗色模式颜色值

**目标：** 调整暗色模式下的 state 颜色，使 running/succeeded/failed 的 border 颜色更亮。

- [x] **Step 1: 修改 `nodeStateColorMap` 暗色模式颜色**

修改 `frontend/src/utils/stateColor.ts` 中的 `nodeStateColorMap`：

```typescript
export const nodeStateColorMap: Record<string, { bg: string; border: string; glow: string }> = {
  pending:   { bg: '#1c1c1e', border: '#636366', glow: 'none' },
  running:   { bg: '#0a2a4a', border: '#40a0ff', glow: '0 0 8px #40a0ff80' },
  succeeded: { bg: '#0a3a1a', border: '#30d158', glow: '0 0 8px #30d15880' },
  failed:    { bg: '#3a1010', border: '#ff453a', glow: '0 0 8px #ff453a80' },
  skipped:   { bg: '#1c1c1e', border: '#545458', glow: 'none' },
};
```

变更细节：
- `running.border`: `#2997ff` → `#40a0ff`
- `running.glow`: `0 0 8px #2997ff80` → `0 0 8px #40a0ff80`
- `succeeded.border`: `#34c759` → `#30d158`
- `succeeded.glow`: `0 0 8px #34c75980` → `0 0 8px #30d15880`
- `failed.border`: `#ff3b30` → `#ff453a`
- `failed.glow`: `0 0 8px #ff3b3080` → `0 0 8px #ff453a80`

- [x] **Step 2: 运行类型检查**

```bash
cd frontend && npx tsc --noEmit
```

预期：无类型错误

- [x] **Step 3: 提交**

```bash
git add frontend/src/utils/stateColor.ts
git commit -m "feat(frontend): stateColor 暗色模式颜色增强"
```

archived-with: 2026-06-20-dag-editor-ux
---
