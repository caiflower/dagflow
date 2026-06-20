---
comet_change: execution-persistence
date: 2026-06-20
role: technical-design
canonical_spec: openspec
archived-with: 2026-06-20-execution-persistence
status: final
---

# Design: Execution Persistence

## 问题

`ExecutionService` 将执行记录存储在内存 `map[string]*Execution` 中，后端重启后所有历史执行记录丢失。前端仅展示节点 ID/Name/State，无法查看 input/output/时长。

## 架构决策

### 持久化策略
**dagflow 层轻量映射表 + taskx 引擎层实时查询**

不在 dagflow 层维护状态字段，仅存储 execID→taskID 的索引映射。所有状态和执行详情从 taskx 引擎层实时查询。

**理由**：
- 零状态同步 — 不存在双写一致性问题
- taskx 已有完整执行数据（Task/Subtask 的 State、Input、Output、LastRunTime）
- `GetTaskOutput` 已支持 TaskBak 回退查询（归档数据也能查）

### 映射表设计
```sql
CREATE TABLE execution_record (
  id          VARCHAR(64) PRIMARY KEY,
  flow_id     BIGINT NOT NULL,
  flow_name   VARCHAR(128) NOT NULL,
  task_id     VARCHAR(64) NOT NULL,
  created_at  DATETIME NOT NULL
);
```

### 查询流程
- **列表查询**：查 execution_record → 批量 GetByIDs 查 taskx Task 状态 → 不存在则 archived
- **详情查询**：查 execution_record → SubtaskDAO.GetByTaskID → 组装 NodeStatus（input/output/timing）

### Proto 扩展（已完成）
- NodeStatus：input/output/startTime/endTime/durationMs/nodeType/protocol
- Execution：taskID

## 文件变更策略

| 操作 | 文件 | 说明 |
|------|------|------|
| 回滚 | taskx/dao/sqld/task_dao.go | 移除 ListRecent 方法 |
| 新增 | internal/model/dao/execution_record.go | ExecutionRecord 模型 |
| 新增 | internal/model/dao/ddl/ 追加 | execution_record DDL |
| 新增 | internal/dao/execution_record_dao.go | DAO 接口 + bun 实现 |
| 重写 | internal/service/execution_service.go | 移除内存 map，改用 DB |
| 修改 | internal/api/execution_grpc_service.go | execToProto 传递新字段 |
| 修改 | frontend/src/types.ts | 扩展 NodeStatus 类型 |
| 修改 | frontend/src/pages/ExecutionPage.tsx | 展示节点详情 |

## 风险

1. N+1 查询：列表页 LIMIT 50 + 批量 GetByIDs 缓解
2. taskx 任务清理：Task 不存在时降级为 archived
3. GetTaskOutput 可见性：当前是 taskDispatcher 方法，需复制或暴露为公开接口
