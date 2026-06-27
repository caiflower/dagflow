---
comet_change: abstract-backup-layer
role: technical-design
canonical_spec: openspec
---

# Abstract Backup Layer — Technical Design

## Context

`backupTask()` 是 `taskDispatcher` 的方法，直接使用 `DBClient.GetDB()` + bun ORM 操作数据库，完全未利用已有的 `StorageBackend`（"sql"/"redis"）双协议机制。项目中已存在完整的 DAO 抽象层及其 sql/redis 双实现，但 backup 逻辑未遵循此模式。当前 `backupTask()` 在 `dispatch.go:254` 被注释掉，暂未启用。

## Goals / Non-Goals

**Goals:**
- 将 backup 逻辑抽象为 `BackupManager` 接口，遵循 `StorageBackend` 机制
- MySQL 实现：冷数据迁移到 archive 表并清理原表（含新建 `task_edge_archive`）
- Redis 实现：直接删除超龄 key
- 提供 `TaskQueryService.GetTasks` 业务接口，MySQL 下支持主表+archive 表联合查询
- 冷表命名改为 `task_archive` / `subtask_archive` / `task_edge_archive`
- 冷数据判定规则可配置化（终态 + age），MySQL 默认 7 天，Redis 默认 1 天
- 用 `GetTasks` 替换 `ExecutionService` 的直接 DAO 调用

**Non-Goals:**
- 不新增 gRPC 接口
- 不改动非 backup 相关的 DAO 接口方法
- 不改变 Redis key 命名规则
- 不引入新的外部依赖

## Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                      taskDispatcher                         │
│                                                              │
│  backupTask() ──▶ BackupManager.BackupTasks(cfg)            │
│                        │                                    │
│           ┌────────────┴────────────┐                       │
│           ▼                         ▼                       │
│   ┌──────────────┐          ┌──────────────┐                │
│   │sqlBackup     │          │redisBackup   │                │
│   │Manager       │          │Manager       │                │
│   │              │          │              │                │
│   │ Query tasks  │          │ Query tasks  │                │
│   │  via TaskDAO │          │  via TaskDAO │                │
│   │ Insert arch  │          │ DEL Redis    │                │
│   │  via BakDAO  │          │  keys        │                │
│   │ Delete orig  │          │              │                │
│   │  via TaskDAO │          │              │                │
│   └──────────────┘          └──────────────┘                │
│                                                              │
│  ┌───────────────────────────────────────────┐              │
│  │          TaskQueryService                 │              │
│  │  GetTasks(ids) → []TaskDetail             │              │
│  │                                           │              │
│  │  sql: main table → archive fallback       │              │
│  │  redis: main table only                  │              │
│  └───────────────────────────────────────────┘              │
└──────────────────────────────────────────────────────────────┘

ExecutionService
  ├─ GetStatus → TaskQueryService.GetTasks([taskID])
  └─ ListExecutions → TaskQueryService.GetTasks(taskIDs)
```

## Decisions

### 1. Interface Location

`BackupManager` 和 `TaskQueryService` 均放在 `backend/taskx/` 包内（新文件：`backup_manager.go`、`task_query_service.go`），与 `taskDispatcher` 同一包。

**Rationale**: 接口需要访问 `TaskDAO`、`SubtaskDAO`、`TaskEdgeDAO` 及 bak DAO，这些已在 `taskDispatcher` 中注入。同一包可直接访问，避免循环依赖和额外的包间耦合。

### 2. BackupManager Interface

```go
// backup_manager.go

type BackupConfig struct {
    Age         time.Duration // 超龄阈值：MySQL 默认 168h，Redis 默认 24h
    BatchSize   int           // 每批处理数量，默认 100
    FinalStates []string      // 终态列表，默认 ["failed", "succeeded"]
}

type BackupManager interface {
    BackupTasks(ctx context.Context, cfg BackupConfig) (int, error)
}
```

#### sqlBackupManager

```
1. TaskDAO.GetTodoTask(cfg.FinalStates, now - cfg.Age) → []model.Task
2. 提取 taskIDs
3. SubtaskDAO.GetByTaskID(taskID) → 转换 Subtask → SubtaskBak → batch insert
4. TaskEdgeDAO.GetByTaskID(taskID) → 转换 TaskEdge → TaskEdgeBak → batch insert
5. Task → TaskBak → batch insert
6. 批量事务 commit
7. 删除原表数据（task/subtask/task_edge）
```

#### redisBackupManager

```
1. TaskDAO.GetTodoTask(cfg.FinalStates, now - cfg.Age) → []model.Task
2. 提取 taskIDs
3. 对每个 taskID：
   - 删除 task key（RedisClient.Del）
   - 删除 subtask index + 各 subtask key
   - 删除 task edge key
4. 返回删除数量
```

**Redis 删除不扩展 TaskDAO 接口**，直接在 `redisBackupManager` 中使用 `RedisClient`。Redis 的 backup 就是删除，无需通过 DAO 抽象。

### 3. TaskQueryService Interface

```go
// task_query_service.go

type TaskDetail struct {
    Task     model.Task
    Subtasks []model.Subtask
    Edges    []model.TaskEdge
}

type TaskQueryService interface {
    GetTasks(ctx context.Context, taskIDs []string) ([]TaskDetail, error)
}
```

#### sqlTaskQueryService

```
1. TaskDAO.GetByIDs(taskIDs) → 命中主表的 tasks
2. 计算缺失的 taskIDs
3. 缺失 ID → TaskBakDAO.GetByIDs(missingIDs) → 从 archive 表补查
4. 将 TaskBak 转换为 Task 结构（字段映射一致）
5. 对每个 taskID → SubtaskDAO.GetByTaskID + TaskEdgeDAO.GetByTaskID
   （archived 的 task 也从 archive 表查 subtask/edge）
6. 组装 []TaskDetail
```

#### redisTaskQueryService

```
1. TaskDAO.GetByIDs(taskIDs) → 仅查 Redis 主表
2. 对每个 taskID → SubtaskDAO.GetByTaskID + TaskEdgeDAO.GetByTaskID
3. 组装 []TaskDetail（缺失的任务不在结果中）
```

### 4. TaskBak → Task 转换

`TaskBak` 和 `Task` 结构体字段完全一致（均不含 `bun.BaseModel` 以外的差异）。转换函数：

```go
func taskBakToTask(bak *model.TaskBak) *model.Task {
    return &model.Task{
        ID: bak.ID, RequestID: bak.RequestID, TaskName: bak.TaskName,
        Input: bak.Input, Output: bak.Output, Worker: bak.Worker,
        Retry: bak.Retry, RetryInterval: bak.RetryInterval,
        Urgent: bak.Urgent, State: bak.State, Description: bak.Description,
        CreateTime: bak.CreateTime, LastRunTime: bak.LastRunTime,
        ExecuteTime: bak.ExecuteTime, Status: bak.Status,
        AffinityType: bak.AffinityType, PrimaryWorker: bak.PrimaryWorker,
        RollbackStrategy: bak.RollbackStrategy,
    }
}
```

同理 `SubtaskBak → Subtask` 和 `TaskEdgeBak → TaskEdge`。

### 5. Cold Table Naming

| Old | New | Model | DAO Interface |
|-----|-----|-------|---------------|
| `task_bak` | `task_archive` | `TaskBak` (bun tag) | `TaskBakDAO` |
| `subtask_bak` | `subtask_archive` | `SubtaskBak` (bun tag) | `SubtaskBakDAO` |
| *(new)* | `task_edge_archive` | `TaskEdgeBak` | `TaskEdgeBakDAO` |

`TableConfig` 新增 `TaskEdgeBak string` 字段，默认 `task_edge_archive`。

### 6. Config Changes

```go
// dispatch.go - Config 新增
BackupFinalStates []string `yaml:"backupFinalStates" default:"failed,succeeded"`
```

初始化逻辑（`InitTaskDispatcher` 中）：
```go
// MySQL: BackupTaskAge 默认 168h（已有）
// Redis: BackupTaskAge 默认 24h
if cfg.StorageBackend == "redis" && cfg.BackupTaskAge == 0 {
    cfg.BackupTaskAge = 24 * time.Hour
}
```

### 7. ExecutionService Migration

```go
// 旧
type ExecutionService struct {
    TaskDAO    taskxDAO.TaskDAO    `autowired:""`
    SubtaskDAO taskxDAO.SubtaskDAO `autowired:""`
}

// 新
type ExecutionService struct {
    TaskQuery taskx.TaskQueryService `autowired:""`
}
```

`GetStatus` 改为：
```go
details, err := s.TaskQuery.GetTasks(ctx, []string{record.TaskID})
if len(details) > 0 {
    taskModel := &details[0].Task
    subtasks := details[0].Subtasks
    // buildNodeStatuses 使用 taskModel + subtasks
}
```

`ListExecutions` 改为：
```go
details, _ := s.TaskQuery.GetTasks(ctx, taskIDs)
for _, d := range details {
    taskMap[d.Task.ID] = &d.Task
}
```

### 8. Bean Registration

在 `InitTaskDispatcher` 中根据 `StorageBackend` 注册对应实现：

```go
if cfg.StorageBackend == "redis" {
    backupMgr := &redisBackupManager{client: cfg.RedisClient, taskDao: ..., ...}
    bean.AddBean(backupMgr)
    querySvc := &redisTaskQueryService{taskDao: ..., subtaskDao: ..., edgeDao: ...}
    bean.AddBean(querySvc)
} else {
    backupMgr := &sqlBackupManager{dbClient: ..., taskDao: ..., ...}
    bean.AddBean(backupMgr)
    querySvc := &sqlTaskQueryService{taskDao: ..., taskBakDao: ..., ...}
    bean.AddBean(querySvc)
}
```

`BackupManager` 和 `TaskQueryService` 均通过 `autowired:""` 注入 `taskDispatcher` 和 `ExecutionService`。

## Risks / Trade-offs

- **[backupTask 启用策略]** `backupTask()` 当前被注释掉。本次重构只改方法体，不改变调用方式。启用需单独决策。
- **[archive 表 DDL]** 存量的 `task_bak`/`subtask_bak` 表需要 DDL 重命名 + 新建 `task_edge_archive` 表。DDL 不属于本次代码变更范围。
- **[TaskBak→Task 转换性能]** 大批量查询时 archive 表补查会增加一次 SQL 查询。通常 `GetTasks` 以单条或少量 ID 调用（`GetStatus` 单条，`ListExecutions` 默认 20 条/页），影响可忽略。
- **[SubtaskBak→Subtask 类型适配]** `buildNodeStatuses` 接收 `[]model.Subtask`，archive 返回 `[]model.SubtaskBak`。通过转换函数统一为 `Subtask` 再传入，保持接口不变。
