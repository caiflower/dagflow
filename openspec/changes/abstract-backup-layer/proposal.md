## Why

`backupTask()` 当前直接使用 bun ORM 绕过 DAO 层，仅支持 MySQL，无法利用已有的 `StorageBackend` 双协议（sql/redis）机制。需要将其抽象为统一的 Backup 接口，使 Redis 和 MySQL 各自实现差异化的「备份」行为，同时为上层业务提供统一的 `GetTasks` 查询接口。

## What Changes

- **Backup 接口抽象**：定义 `BackupManager` 接口，包含 `BackupTasks(ctx, age)` 方法，由 MySQL 和 Redis 分别实现
- **MySQL 实现**：查终态+超龄任务 → 写入 `task_archive`/`subtask_archive`/`task_edge_archive` 冷表 → 删除原表数据
- **Redis 实现**：查终态+超龄任务 → 直接删除 key（无冷表，备份 age 默认 1 天（区别于 MySQL 的 7 天））
- **`GetTasks` 业务接口**：新建 `TaskQueryService` 层，提供 `GetTasks(taskIDs []string) ([]TaskDetail, error)`，MySQL 从主表+archive 表联合查询，Redis 仅查主表
- **冷表重命名**：`task_bak` → `task_archive`，`subtask_bak` → `subtask_archive`，新增 `task_edge_archive`
- **可配置化**：冷数据判定条件（终态列表 + 超龄天数）通过 Config 配置，默认终态（failed/succeeded），MySQL 超龄 7 天，Redis 超龄 1 天
- **`ExecutionService` 改造**：用 `GetTasks` 替换直接调用 `TaskDAO`/`SubtaskDAO` 的逻辑

## Capabilities

### New Capabilities
- `backup-manager`: 抽象的 Backup 接口，支持 MySQL（写冷表+删原表）和 Redis（直接删除）双协议
- `task-query-service`: 业务层任务查询接口 `GetTasks`，统一从主表/冷表获取 Task + Subtask + TaskEdge
- `cold-data-criteria`: 可配置的冷数据判定规则（终态+超龄），MySQL 默认 7 天，Redis 默认 1 天

### Modified Capabilities
<!-- No existing spec changes required -->

## Impact

- `backend/taskx/backup.go` — 重构，方法体改为调用 BackupManager 接口
- `backend/taskx/dao/model/` — TaskBak/SubtaskBak 重命名 bun table tag，新增 TaskEdgeBak 模型
- `backend/taskx/dao/` — TaskBakDAO/SubtaskBakDAO 接口扩展，新增 TaskEdgeBakDAO
- `backend/taskx/dao/sqld/` — SQL DAO 实现适配新表名，新增 TaskEdgeBak DAO
- `backend/taskx/dao/redisd/` — Redis DAO 实现适配（backup 即为删除）
- `backend/taskx/dispatch.go` — Config 新增冷数据判定配置项
- `backend/internal/service/execution_service.go` — 用 GetTasks 替换直接 DAO 调用
