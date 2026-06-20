# Tasks: Execution Persistence

## Phase 1: 回滚 + 基础设施

- [x] 回滚 `TaskDAO.ListRecent`：移除 taskx/dao/sqld/task_dao.go 中的 ListRecent 方法
- [x] 创建 `execution_record` 模型：`backend/internal/model/dao/execution_record.go`（bun model + DDL）
- [x] 创建 `ExecutionRecordDAO`：`backend/internal/model/dao/execution_record_dao.go`（Insert + List + GetByID + GetByIDs）
- [x] 注册 DAO Bean：在 main.go 中注册 ExecutionRecordDAO + 自动建表

## Phase 2: ExecutionService 重写

- [x] 移除内存 map：删除 `executions map[string]*Execution` 及相关锁和快照方法
- [x] Run 方法：submit taskx 后写入 execution_record 映射表
- [x] GetStatus 方法：从映射表查 task_id，再从 taskx 实时查询状态 + 节点详情（Input/Output/timing）
- [x] ListExecutions 方法：从映射表查列表，批量 GetByIDs 查 taskx 状态
- [x] archived 降级：taskx Task 不存在时返回 state=archived

## Phase 3: gRPC 层更新

- [x] 更新 `execToProto`：传递 NodeStatus 新字段（input/output/startTime/endTime/durationMs/nodeType/protocol）
- [x] 更新 `Execution.TaskId` 传递

## Phase 4: 前端展示

- [x] 更新 `types.ts`：扩展 NodeStatus 类型 + Execution 新增 taskId + archived 状态
- [x] 重写 `ExecutionPage.tsx`：展示每个节点的输入、输出、执行时长（可折叠详情面板）

## Phase 5: 验证

- [x] 后端编译通过 + 无错误
- [x] 前端编译通过 + 无警告
