## Context

当前 `backupTask()` 是 `taskDispatcher` 的方法，直接使用 `DBClient.GetDB()` + bun ORM 操作数据库，完全未利用已有 `StorageBackend`（"sql"/"redis"）双协议机制。项目中已存在完整的 DAO 抽象层（`TaskDAO`、`TaskBakDAO`、`SubtaskDAO`、`SubtaskBakDAO`、`TaskEdgeDAO`）及其 sql/redis 双实现，但 backup 逻辑未遵循此模式。

## Goals / Non-Goals

**Goals:**
- 将 backup 逻辑抽象为 `BackupManager` 接口，遵循 `StorageBackend` 机制
- MySQL 实现：冷数据迁移到 archive 表并清理原表（含新建 `task_edge_archive`）
- Redis 实现：直接删除超龄 key
- 提供 `TaskQueryService.GetTasks` 业务接口，MySQL 下支持主表+archive 表联合查询
- 冷表命名改为 `task_archive` / `subtask_archive` / `task_edge_archive`
- 冷数据判定规则可配置化（终态 + age）
- 用 `GetTasks` 替换 `ExecutionService` 的直接 DAO 调用

**Non-Goals:**
- 不新增 gRPC 接口
- 不改动非 backup 相关的 DAO 接口方法
- 不改变 Redis key 命名规则
- 不引入新的外部依赖

## Decisions

### 1. BackupManager 接口定义

```go
type BackupManager interface {
    BackupTasks(ctx context.Context, cfg BackupConfig) (int, error)
}

type BackupConfig struct {
    Age         time.Duration // 超龄阈值（默认 7 天）
    BatchSize   int           // 每批处理数量（默认 100）
    FinalStates []string      // 终态列表（默认 failed, succeeded）
}
```

- SQL 实现通过 `TaskDAO.GetTodoTask` + `SubtaskDAO` + `TaskEdgeDAO` 查询待备份数据，然后写入 archive 表并删除原数据
- Redis 实现仅调用 `TaskDAO.GetTodoTask` 获取超龄任务 ID，然后逐个删除 key

### 2. TaskQueryService 接口定义

```go
type TaskQueryService interface {
    GetTasks(ctx context.Context, taskIDs []string) ([]TaskDetail, error)
}

type TaskDetail struct {
    Task    model.Task
    Subtasks []model.Subtask
    Edges    []model.TaskEdge
}
```

- MySQL：先查主表获取存在的 task，缺失的 ID 从 archive 表补查
- Redis：仅查主表（archive 不适用）
- `ExecutionService` 的 `GetStatus` / `ListExecutions` 改为通过此接口获取数据

### 3. 冷表命名变更

| 旧名 | 新名 | 说明 |
|------|------|------|
| `task_bak` | `task_archive` | 任务归档表 |
| `subtask_bak` | `subtask_archive` | 子任务归档表 |
| _(新建)_ | `task_edge_archive` | 任务边归档表 |

通过修改对应 model 的 `bun:"table:xxx"` tag 实现，同时更新 `TableConfig` 中的默认值。

### 4. 冷数据判定可配置化

现有 `BackupTaskAge` 已是 `time.Duration` 类型（默认 168h=7 天），保持不变。新增 `BackupFinalStates` 配置项：

```go
// Config 中新增
BackupFinalStates []string `yaml:"backupFinalStates" default:"failed,succeeded"`
```

判定逻辑：
```
state IN (BackupFinalStates) AND create_time <= NOW() - BackupTaskAge
```

### 5. Redis TTL

Redis 实现中，备份后直接删除 key，Redis 的 backup age 默认 1 天（24h），区别于 MySQL 的 7 天。两者均通过 `BackupTaskAge` 配置控制，默认值因 StorageBackend 不同而不同：sql=168h，redis=24h。

## Risks / Trade-offs

- [备份期间数据一致性] → backup 使用批量事务（MySQL），Redis 使用 pipeline，保持现有逻辑
- [冷表查询性能] → archive 表结构与原表一致，索引策略保持不变，查询性能无退化
- [命名变更兼容性] → 修改 model 的 `bun:table` tag 和 `TableConfig` 默认值，新部署自动生效；存量数据的表名迁移通过 DDL 变更脚本处理（不在本次代码范围内）
- [ExecutionService 改造] → `GetTasks` 返回完整 Task+Subtask+Edge，接口语义与现有 DAO 调用保持一致，行为无退化
