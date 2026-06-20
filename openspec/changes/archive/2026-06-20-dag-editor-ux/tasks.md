## 1. 后端错误透传

- [x] 1.1 `flow_service.go`：Create/Get/Update/Delete 中业务错误改用 `e.NewApiError()`（NotFound、InvalidArgument）
- [x] 1.2 `execution_service.go`：Run/GetStatus 中业务错误改用 `e.NewApiError()`（NotFound、InvalidArgument）

## 2. 前端 API 错误透传

- [x] 2.1 `api/client.ts`：新增 `ApiError` 类，修改 `unwrap()` 检测 error 字段并抛出
- [x] 2.2 `pages/FlowListPage.tsx`：catch 块添加 toast 错误提示
- [x] 2.3 `pages/FlowEditorPage.tsx`：catch 块添加 toast 错误提示（加载、保存）
- [x] 2.4 `pages/ExecutionPage.tsx`：catch 块添加 toast 错误提示
- [x] 2.5 `pages/ProtocolPage.tsx`：catch 块添加 toast 错误提示

## 3. Toast 通知系统

- [x] 3.1 创建 `components/ui/Toast.tsx` Toast 组件（success/error/info + 自动消失 + 关闭按钮 + Portal）
- [x] 3.2 创建 `store/toastStore.ts` zustand store 管理 toast 队列
- [x] 3.3 在 `App.tsx` 中挂载 ToastProvider，导出便捷方法 `toast.success/error/info`

## 4. 保存通知

- [x] 4.1 `FlowEditorPage.tsx` handleSave 成功后调用 `toast.success('Flow saved')`
- [x] 4.2 handleSave catch 中调用 `toast.error()`

## 5. DAG 节点 UI 优化

- [x] 5.1 `BaseNode.tsx`：accent 条纹→渐变、hover 缩放+阴影、Handle 圆形+hover 放大、圆角增大
- [x] 5.2 `nodes/index.tsx`：协议图标前置到 icon 区域、config 展示优化为 chip 样式
- [x] 5.3 添加 running 状态脉冲动画（CSS keyframes）
- [x] 5.4 `stateColor.ts`：调整颜色饱和度与对比度
