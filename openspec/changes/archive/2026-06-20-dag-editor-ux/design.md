## Context

当前错误流转链路：

```
Service: fmt.Errorf("flow not found") → 普通 error
  → doTargetMethod: errors.As → 不是 ApiError
  → status.FromError → 不是 gRPC status
  → e.NewInternalError(err) → 包装为 InternalError (500)
```

`common-tools` 框架的 `doTargetMethod` 按 `ApiError → gRPC status → InternalError` 优先级判断。当前 service 层用 `fmt.Errorf` 返回错误，全部落入 InternalError 分支，导致所有业务错误变成 500。

## Goals / Non-Goals

**Goals:**
- 后端 service 层对业务错误（NotFound、InvalidArgument 等）改用 `e.NewApiError()` 返回
- 500 内部错误（如 DB 异常）保持 `fmt.Errorf` 不变
- 前端 `unwrap()` 检测响应 error 字段并抛出可读异常
- Toast 通知系统 + 保存成功提示
- DAG 节点视觉优化

**Non-Goals:**
- 不修改 common-tools 框架
- 不引入第三方通知库
- 不重构整个前端 UI

## Decisions

### 1. 后端错误透传策略

**方案**：在 service 层将所有可预期的业务错误改为 `e.NewApiError()`

```go
// 之前
return nil, fmt.Errorf("flow not found: %w", err)

// 之后
return nil, e.NewApiError(e.NotFound, fmt.Sprintf("flow %d not found", id), err)
```

**错误码映射**：

| 业务场景 | 错误码 | 示例消息 |
|---------|--------|---------|
| 资源不存在 | NotFound (404) | "flow 1 not found" |
| 参数校验失败 | InvalidArgument (400) | "invalid flow: missing start node" |
| 编译/构建失败 | InvalidArgument (400) | "compile task: ..." |
| DB 异常 | 保持 `fmt.Errorf` → InternalError (500) | "insert flow: ..." |

**无需修改的场景**（本来就是对的）：
- `recoveryMiddleware` 的 panic recovery → 500，不变
- 框架内部错误 → 500，不变
- DAO 层返回的 DB 错误 → 保持 `fmt.Errorf`，框架包装为 500

### 2. 前端 API 错误处理

`unwrap()` 增加 error 检测：

```typescript
class ApiError extends Error {
  code: number;
  type: string;
  constructor(code: number, message: string, type: string) {
    super(message);
    this.code = code;
    this.type = type;
  }
}

function unwrap<T>(r: { data: { data?: T; error?: { code: number; message: string; type: string } } }): T {
  if (r.data.error) {
    throw new ApiError(r.data.error.code, r.data.error.message, r.data.error.type);
  }
  return r.data.data as T;
}
```

### 3. Toast 通知系统

- zustand store 管理 toast 队列
- ToastProvider 挂载 App 根组件（Portal 到 body）
- `toast.success(msg)`, `toast.error(msg)`, `toast.info(msg)`
- 自动消失 3s，top-right 固定定位，z-index 9999

### 4. DAG 节点 UI

- accent 条纹→渐变条
- hover 微缩放(1.02) + 阴影提升
- 状态脉冲动画（running 状态）
- Handle 圆形 + hover 放大
- 圆角 `var(--radius-lg)`

## Risks / Trade-offs

- **`e.NewApiError` 依赖 common-tools**：已有依赖，不是新增
- **前端 catch 块需全部修改**：逐页检查，确保每个 catch 调用 toast
PROPOSAL_EOF
echo "design.md updated"