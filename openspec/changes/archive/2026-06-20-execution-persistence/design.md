# Design: Execution Persistence

## 架构决策

### 持久化策略
**选定：dagflow 层轻量映射表 + taskx 引擎层实时查询**

不在 dagflow 层维护状态字段（state/timing），仅存储 execID→taskID 的索引映射。所有状态和执行详情从 taskx 引擎层实时查询。

**理由**：
- 零状态同步 — 不存在双写一致性问题
- taskx 已有完整执行数据（Task/Subtask 的 State、Input、Output、LastRunTime）
- `GetTaskOutput` 已支持 TaskBak 回退查询（归档数据也能查）

### 映射表设计
```sql
CREATE TABLE execution_record (
  id          VARCHAR(64) PRIMARY KEY,   -- exec-{flowID}-{timestamp}
  flow_id     BIGINT NOT NULL,
  flow_name   VARCHAR(128) NOT NULL,
  task_id     VARCHAR(64) NOT NULL,      -- taskx Task ID (用于反查)
  created_at  DATETIME NOT NULL
);
```

仅 5 个字段，不存 state、start_time、end_time。

### 查询流程

**列表查询**：
```
SELECT * FROM execution_record ORDER BY created_at DESC
→ 对每条记录，用 task_id 调 TaskDAO.GetByID() 查状态
→ 如果 TaskDAO 找不到（已清理），标记 state=archived
```

**详情查询**：
```
1. 从 execution_record 查 task_id
2. 用 task_id 调 SubtaskDAO.GetByTaskID() → 获取每个节点的 Input、State、LastRunTime
3. 用 task_id 调 GetTaskOutput() → 获取每个节点的 Output（含 output + err）
4. 组装 NodeStatus（input/output/timing/nodeType/protocol）
```

### 节点时长计算
- `startTime`：从 Subtask.LastRunTime 获取（receiver 执行时设置）
- `endTime`：如果节点 state 为终态（succeeded/failed/skipped），用 Task.LastRunTime 近似
- `durationMs`：endTime - startTime（毫秒），如 endTime 不可用则为 0

### Proto 扩展
已在上一轮实现中完成（NodeStatus 新增 7 字段，Execution 新增 taskID），无需重做。

## 文件变更策略

```
新增文件：
  backend/internal/model/dao/execution_record.go  — ExecutionRecord 模型 + DDL
  backend/internal/dao/execution_record_dao.go    — DAO 接口 + bun 实现

重写文件：
  backend/internal/service/execution_service.go    — 移除内存 map，改用 DB 查询

修改文件：
  backend/taskx/dao/task_dao.go                   — 回滚 ListRecent 方法
  backend/taskx/dao/sqld/task_dao.go              — 回滚 ListRecent 实现
  backend/internal/api/execution_grpc_service.go   — 更新 execToProto 传递新字段
  frontend/src/types.ts                           — 扩展 NodeStatus 类型
  frontend/src/pages/ExecutionPage.tsx            — 展示节点详情

已修改（保留）：
  backend/internal/proto/dagflow.proto            — NodeStatus + Execution 扩展
  backend/internal/proto/dagflow.pb.go            — 已重新生成
```

## 风险

1. **N+1 查询**：列表页对每条记录单独查 taskx Task 状态，数据量大时有性能风险。缓解：映射表 LIMIT 50 + 批量 GetByIDs
2. **taskx 任务清理**：如果 taskx 有 TTL/清理策略，Task 不存在时需优雅降级为 archived 状态
3. **`GetTaskOutput` 可见性**：当前 `GetTaskOutput` 是 `taskDispatcher` 的内部方法，可能需要暴露为公开接口或复制查询逻辑
