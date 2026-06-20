## Why

DAGFlow 前端存在三个体验问题：(1) 后端业务错误（如"flow not found"、"invalid flow"）因服务层返回普通 `error` 而非 `ApiError`，被框架统一包装为 InternalError (500)，用户看不到具体错误原因；(2) DAG 编辑界面保存后无成功/失败反馈；(3) DAG 节点视觉设计单调。

## What Changes

- **后端错误透传**：服务层将业务错误（NotFound、InvalidArgument 等）改用 `e.NewApiError()` 返回，使框架正确透传 4xx 错误信息到前端；500 内部错误保持不变
- **前端 API 错误处理**：`unwrap()` 检测响应 `error` 字段，抛出可读 `ApiError`；各页面 catch 块展示 toast
- **Toast 通知系统**：构建轻量级 toast 组件，支持 success/error/info 三种类型
- **DAG 节点 UI 优化**：重构 BaseNode 视觉层次（渐变、阴影、动画）

## Capabilities

### New Capabilities

- `toast-notification`: 全局 toast 通知系统，支持 success/error/info，自动消失，可堆叠
- `dag-node-polish`: DAG 节点视觉优化，渐变 accent、hover 缩放+阴影、状态脉冲动画

### Modified Capabilities

<!-- 无已有 spec -->

## Impact

- 后端：`internal/service/flow_service.go`、`internal/service/execution_service.go` 错误返回方式
- 前端：`api/client.ts`、`pages/*`、`components/dag/nodes/*`、`components/ui/Toast.tsx`（新增）
- 无 API 变更，无 breaking changes
