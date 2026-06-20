---
comet_change: dag-editor-ux
role: technical-design
canonical_spec: openspec
archived-with: 2026-06-20-dag-editor-ux
status: final
---

# DAG Editor UX Improvements — Technical Design

## Overview

Four UX improvements to the DAGFlow frontend + backend error handling:

1. **Backend error pass-through**: Business errors use `e.NewApiError()` so 4xx codes reach the frontend
2. **Frontend API error handling**: `unwrap()` detects `error` field in responses, throws typed `ApiError`
3. **Toast notification system**: Lightweight toast for success/error/info feedback
4. **DAG node UI polish**: Subtle enhancement of node visual design

archived-with: 2026-06-20-dag-editor-ux
status: final
---

## 1. Backend Error Pass-through

### Problem

```go
// flow_service.go — current
func (s *FlowService) Get(ctx context.Context, id int64) (*model.Flow, error) {
    flow, err := s.FlowDAO.GetByID(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("flow not found: %w", err)  // plain error
    }
}
```

In `doTargetMethod` (common-tools `handler.go:618`):

```go
case errors.As(err, &apiError):  // false — is a plain error
    return err.(e.ApiError)
default:
    st, ok := status.FromError(err)  // false
    ...
    return e.NewInternalError(err)   // always 500
```

All business errors become 500. The fix: return `e.ApiError` instead of plain `error`.

### Solution

Replace `fmt.Errorf` with `e.NewApiError()` for anticipated business errors. DB/infrastructure errors keep `fmt.Errorf` → 500.

**`internal/service/flow_service.go`**:

| Method | Condition | Before | After |
|--------|-----------|--------|-------|
| Create | ValidateFlow fails | `fmt.Errorf("invalid flow: %w", err)` | `e.NewApiError(e.InvalidArgument, err.Error(), err)` |
| Create | DAO Insert fails | `fmt.Errorf("insert flow: %w", err)` | *unchanged* (→ 500) |
| Get | DAO GetByID fails | `fmt.Errorf("get flow %d: %w", id, err)` | `e.NewApiError(e.NotFound, fmt.Sprintf("flow %d not found", id), err)` |
| Update | DAO GetByID fails | `fmt.Errorf("flow not found: %w", err)` | `e.NewApiError(e.NotFound, fmt.Sprintf("flow %d not found", req.ID), err)` |
| Update | ValidateFlow fails | `fmt.Errorf("invalid flow: %w", err)` | `e.NewApiError(e.InvalidArgument, err.Error(), err)` |
| Update | DAO Update fails | `fmt.Errorf("update flow: %w", err)` | *unchanged* (→ 500) |
| Delete | DAO Delete fails | DAO returns error directly | *unchanged* (→ 500) |

**`internal/service/execution_service.go`**:

| Method | Condition | Before | After |
|--------|-----------|--------|-------|
| Run | flow not found | `fmt.Errorf("flow not found: %w", err)` | `e.NewApiError(e.NotFound, fmt.Sprintf("flow %d not found", req.FlowID), err)` |
| Run | parse flow fails | `fmt.Errorf("parse flow: %w", err)` | `e.NewApiError(e.InvalidArgument, err.Error(), err)` |
| Run | build task fails | `fmt.Errorf("build task: %w", err)` | `e.NewApiError(e.InvalidArgument, err.Error(), err)` |
| Run | compile task fails | `fmt.Errorf("compile task: %w", err)` | `e.NewApiError(e.InvalidArgument, err.Error(), err)` |
| Run | submit task fails | `fmt.Errorf("submit task: %w", err)` | `e.NewApiError(e.Internal, err.Error(), err)` |
| GetStatus | not found | `fmt.Errorf("execution %s not found: %w", ...)` | `e.NewApiError(e.NotFound, fmt.Sprintf("execution %s not found", execID), err)` |

**Import addition**: `"github.com/caiflower/common-tools/web/common/e"`.

### HTTP Status Code Flow

GRPC routes have `IsRestful() = true` (no `?Action=` param), so `DefaultResultCallback` writes the correct HTTP status:

```
e.NewApiError(e.NotFound, "flow 1 not found")
  → doTargetMethod: errors.As → true → kept as-is
  → DefaultResultCallback: IsRestful()=true
  → HTTP 404 + {"requestID":"...", "error":{"code":404, "type":"NotFound", "message":"flow 1 not found"}}
```

archived-with: 2026-06-20-dag-editor-ux
status: final
---

## 2. Frontend API Error Handling

### `api/client.ts`

Add `ApiError` class before `unwrap()`:

```typescript
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
```

Modify `unwrap()`:

```typescript
function unwrap<T>(r: { data: { data?: T; error?: { code: number; message: string; type: string } } }): T {
  if (r.data.error) {
    throw new ApiError(r.data.error.code, r.data.error.message, r.data.error.type);
  }
  return r.data.data as T;
}
```

### Page Catch Blocks

Every page catch block displays error via toast:

```typescript
try { ... } catch (e) {
  const msg = e instanceof ApiError ? e.message : 'Network error, please retry';
  toast.error(msg);
}
```

Pages affected: `FlowListPage.tsx`, `FlowEditorPage.tsx`, `ExecutionPage.tsx`, `ProtocolPage.tsx`.

archived-with: 2026-06-20-dag-editor-ux
status: final
---

## 3. Toast Notification System

### Architecture

```
┌──────────────────────────────────────────────┐
│                  App.tsx                      │
│  ┌──────────────────────────────────────┐    │
│  │        <ToastProvider />              │    │
│  │  createPortal(                       │    │
│  │    <div class="toast-container">      │    │
│  │      <Toast /> × N                   │    │
│  │    </div>,                            │    │
│  │    document.body                      │    │
│  │  )                                    │    │
│  └──────────┬───────────────────────────┘    │
│             │ subscribe                       │
│  ┌──────────▼───────────────────────────┐    │
│  │         toastStore (zustand)          │    │
│  │  toasts: {id, type, message}[]        │    │
│  │  toast.success(msg) / error / info    │    │
│  └──────────────────────────────────────┘    │
└──────────────────────────────────────────────┘
```

### Files

| File | Purpose |
|------|---------|
| `components/ui/Toast.tsx` (new) | Toast component + ToastProvider |
| `store/toastStore.ts` (new) | Zustand store with add/dismiss |

### API

```typescript
import { toast } from '../store/toastStore';

toast.success('Flow saved');
toast.error('Flow not found');
toast.info('Loading...');
```

### Visual Specs

- **Position**: `fixed top-4 right-4`, z-index 9999
- **Enter**: `slideInRight` 300ms ease-out (CSS keyframes)
- **Exit**: `fadeOut` 200ms (CSS keyframes)
- **Auto-dismiss**: 3000ms
- **Dismiss button**: × icon (lucide-react `X`)
- **Stack**: newest below oldest (flex-col)
- **Icons**: `CheckCircle` (success), `XCircle` (error), `Info` (info) — all lucide-react
- **Width**: `max-w-sm`

### Toast Messages (all English)

| Context | Success | Error (specific) | Error (fallback) |
|---------|---------|------------------|------------------|
| FlowEditor save | `Flow saved` | `Failed to save flow` | `Network error, please retry` |
| FlowEditor load | — | `Failed to load flow` | `Network error, please retry` |
| FlowList delete | — | `Failed to delete flow` | `Network error, please retry` |
| Execution run | — | `Failed to run flow` | `Network error, please retry` |
| Execution list | — | `Failed to load executions` | `Network error, please retry` |
| Protocol list | — | `Failed to load protocols` | `Network error, please retry` |

archived-with: 2026-06-20-dag-editor-ux
status: final
---

## 4. DAG Node UI Polish

### Approach

Subtle enhancement preserving current Apple-style design language. All changes in `BaseNode.tsx` + `nodes/index.tsx` + `stateColor.ts`.

### `BaseNode.tsx` Changes

| Element | Current | After |
|---------|---------|-------|
| Accent stripe | `h-0.5` solid color | `h-0.5` → gradient (`color 80% → 30% opacity`) |
| Border-radius | `var(--radius-md)` (10px) | `var(--radius-lg)` (14px) |
| Hover | none | `transform: scale(1.02)` + `shadow-lg` (300ms transition) |
| Selected shadow | blue glow | blue glow (wider spread) + 2px border |
| Handle size | `w-2.5 h-2.5` | `w-3 h-3` |
| Handle hover | none | `scale(1.3)` + nodeType color fill |
| Running glow | static glow | `pulse-glow` animation |

### `nodes/index.tsx` Changes

- **TaskNode**: Move protocol icon into `BaseNode` icon prop (instead of rendering protocol badge text). Config preview chips use subtle background + border instead of plain text.
- **All nodes**: Name font-weight `medium` → `semibold`. Protocol badge add left margin.

### New CSS (inline or appended to `theme.css`)

```css
@keyframes pulse-glow {
  0%, 100% { box-shadow: 0 0 4px var(--glow-color, #2997ff80); }
  50% { box-shadow: 0 0 12px var(--glow-color, #2997ff80); }
}
.animate-pulse-glow {
  animation: pulse-glow 2s ease-in-out infinite;
}
```

### `stateColor.ts` Changes

- Dark mode `running` border: `#2997ff` → `#40a0ff` (brighter)
- Dark mode `failed` border: `#ff3b30` → `#ff453a` (brighter)
- Dark mode `succeeded` border: `#34c759` → `#30d158` (brighter)

archived-with: 2026-06-20-dag-editor-ux
status: final
---

## File Change Summary

| File | Action | Lines |
|------|--------|-------|
| `backend/internal/service/flow_service.go` | Modify | ~6 error returns |
| `backend/internal/service/execution_service.go` | Modify | ~5 error returns |
| `frontend/src/api/client.ts` | Modify | +15 lines (ApiError class + unwrap update) |
| `frontend/src/pages/FlowListPage.tsx` | Modify | catch block |
| `frontend/src/pages/FlowEditorPage.tsx` | Modify | handleSave + catch blocks |
| `frontend/src/pages/ExecutionPage.tsx` | Modify | catch block |
| `frontend/src/pages/ProtocolPage.tsx` | Modify | catch block |
| `frontend/src/store/toastStore.ts` | **New** | ~40 lines |
| `frontend/src/components/ui/Toast.tsx` | **New** | ~80 lines |
| `frontend/src/App.tsx` | Modify | +1 import + `<ToastProvider />` |
| `frontend/src/components/dag/nodes/BaseNode.tsx` | Modify | hover/animation/handle styles |
| `frontend/src/components/dag/nodes/index.tsx` | Modify | icon repositioning |
| `frontend/src/utils/stateColor.ts` | Modify | color adjustments |
| `frontend/src/styles/theme.css` | Modify | +pulse-glow keyframes |
