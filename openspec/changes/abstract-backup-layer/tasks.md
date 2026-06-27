## 1. 模型层重命名与新增

- [x] 1.1 重命名 `TaskBak` bun table tag：`task_bak` → `task_archive`
- [x] 1.2 重命名 `SubtaskBak` bun table tag：`subtask_bak` → `subtask_archive`
- [x] 1.3 新增 `TaskEdgeArchive` 模型（`backend/taskx/dao/model/task_edge_archive.go`），bun table 为 `task_edge_archive`，字段与 `TaskEdge` 一致
- [x] 1.4 更新 `sqld.TableConfig` 默认表名：`TaskBak` → `task_archive`，`SubtaskBak` → `subtask_archive`，新增 `TaskEdgeArchive` → `task_edge_archive`

## 2. DAO 层扩展

- [x] 2.1 `TaskBakDAO` 接口新增 `GetByIDs` 和 `BatchInsert` 方法
- [x] 2.2 `SubtaskBakDAO` 接口新增 `BatchInsert` 方法
- [x] 2.3 新增 `TaskEdgeArchiveDAO` 接口：`BatchInsert` + `GetByTaskID`
- [x] 2.4 `sqld/task_bak_dao.go`：实现 `GetByIDs` 和 `BatchInsert`，表名适配 `task_archive`
- [x] 2.5 `sqld/subtask_bak_dao.go`：补 `BatchInsert`，表名适配 `subtask_archive`
- [x] 2.6 新增 `sqld/task_edge_archive.go`：SQL 实现 `TaskEdgeArchiveDAO`
- [x] 2.7 `redisd/task_bak_dao.go`：实现 `GetByIDs`（返回空）+ `BatchInsert`（空实现）
- [x] 2.8 `redisd/subtask_bak_dao.go`：同上
- [x] 2.9 新增 `redisd/task_edge_archive_dao.go`：Redis 实现（空实现）

## 3. BackupManager 接口与实现

- [x] 3.1 定义 `BackupManager` 接口和 `BackupConfig` 结构（`backend/taskx/backup_manager.go`）
- [x] 3.2 实现 `sqlBackupManager`：查询超龄任务 → 写入 archive 表 → 删除原表/边数据
- [x] 3.3 实现 `redisBackupManager`：查询超龄任务 → 直接删除 Redis key
- [x] 3.4 在 `InitTaskDispatcher` 中根据 `StorageBackend` 初始化对应的 `BackupManager`，注入到 `taskDispatcher`

## 4. 冷数据判定规则可配置化

- [x] 4.1 `Config` 新增 `BackupFinalStates []string` 字段；Redis 初始化时默认 age 设为 24h
- [x] 4.2 修改 `backupTask()` 方法，将硬编码的 `TaskFailed/TaskSucceeded` 替换为 `cfg.BackupFinalStates`
- [x] 4.3 重构 `backupTask()`：方法体改为调用 `BackupManager.BackupTasks()`

## 5. TaskQueryService 接口与实现

- [x] 5.1 定义 `TaskQueryService` 接口和 `TaskDetail` 结构（`backend/taskx/task_query_service.go`）
- [x] 5.2 实现 `sqlTaskQueryService`：`GetTasks` 先查主表 → 缺失 ID 从 archive 表补查 → 组装 TaskDetail
- [x] 5.3 实现 `redisTaskQueryService`：`GetTasks` 仅查主表
- [x] 5.4 在 `InitTaskDispatcher` 中根据 `StorageBackend` 初始化对应的 `TaskQueryService`
- [x] 5.5 注册 `TaskQueryService` 为 bean，供外部调用

## 6. ExecutionService 改造

- [x] 6.1 `ExecutionService` 注入 `TaskQueryService`（替换 `TaskDAO`/`SubtaskDAO` 直接注入）
- [x] 6.2 改造 `GetStatus`：用 `GetTasks` 获取 Task + Subtask 完整数据
- [x] 6.3 改造 `ListExecutions`：用 `GetTasks` 批量获取 Task 状态
- [x] 6.4 移除 `taskxDAO` 导入
