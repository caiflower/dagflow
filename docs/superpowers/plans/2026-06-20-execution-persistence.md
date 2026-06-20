---
change: execution-persistence
design-doc: docs/superpowers/specs/2026-06-20-execution-persistence-design.md
base-ref: 2640703b00c5ffafafedbe0c808b96e8f49126b8
archived-with: 2026-06-20-execution-persistence
---

# Implementation Plan: Execution Persistence

## Phase 1: 回滚 + 基础设施

### Task 1: 回滚 TaskDAO.ListRecent
- **文件**: `backend/taskx/dao/sqld/task_dao.go`
- **操作**: 删除 `ListRecent` 方法（L154-L168）
- **验证**: `grep -n ListRecent` 无结果

### Task 2: 创建 ExecutionRecord 模型
- **文件**: `backend/internal/model/dao/execution_record.go`（新建）
- **内容**: bun model 定义 + TableName tag
- **DDL**: 追加到 `backend/internal/model/dao/ddl/table.sql` 和 `table-sqlite.sql`

### Task 3: 创建 ExecutionRecordDAO
- **文件**: `backend/internal/dao/execution_record_dao.go`（新建）
- **方法**: Insert, List (LIMIT 50, DESC), GetByID, GetByIDs (批量)
- **注册**: bean.AddBean

### Task 4: 注册 DAO Bean
- **文件**: `backend/internal/model/dao/execution_record.go`
- **操作**: 添加 `InitExecutionRecord()` 函数

## Phase 2: ExecutionService 重写

### Task 5: 移除内存 map
- **文件**: `backend/internal/service/execution_service.go`
- **操作**: 删除 `executions map`, `mu sync.RWMutex`, `snapshot`, `snapshotUnsafe` 方法
- **新增**: `ExecutionRecordDAO` 字段

### Task 6: 重写 Run 方法
- **操作**: SubmitTask 后写入 execution_record 表（Insert）
- **返回**: 从 DB 构建 Execution 对象

### Task 7: 重写 GetStatus 方法
- **操作**: 从 execution_record 查 task_id → TaskDAO.GetByID + SubtaskDAO.GetByTaskID
- **组装**: NodeStatus 包含 Input/Output/timing（从 subtask.Output JSON 解析）
- **降级**: TaskDAO 查不到时返回 state=archived

### Task 8: 重写 ListExecutions 方法
- **操作**: ExecutionRecordDAO.List → 批量 GetByIDs 查 taskx 状态
- **降级**: 不存在的 Task 标记为 archived

## Phase 3: gRPC 层更新

### Task 9: 更新 execToProto
- **文件**: `backend/internal/api/execution_grpc_service.go`
- **操作**: 传递 NodeStatus 新字段（input/output/startTime/endTime/durationMs/nodeType/protocol）
- **操作**: 传递 Execution.TaskId

## Phase 4: 前端展示

### Task 10: 更新 types.ts
- **文件**: `frontend/src/types.ts`
- **操作**: 扩展 NodeStatus 类型（input/output/startTime/endTime/durationMs/nodeType/protocol）
- **操作**: Execution 新增 taskId

### Task 11: 重写 ExecutionPage.tsx
- **文件**: `frontend/src/pages/ExecutionPage.tsx`
- **操作**: 展示每个节点的输入、输出、执行时长（可折叠详情面板）

## Phase 5: 验证

### Task 12: 后端编译验证
- `cd backend && go build ./...`

### Task 13: 前端编译验证
- `cd frontend && npm run build`
