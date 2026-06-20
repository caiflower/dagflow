# Proposal: Execution Persistence

## Why

当前 `ExecutionService` 将执行记录存储在内存 `map[string]*Execution` 中，后端重启后所有历史执行记录丢失。同时，前端执行记录仅展示节点 ID/Name/State 三个字段，无法查看每个节点的执行输入、输出和耗时，严重影响调试和运维效率。

## What Changes

- **新增 `execution_record` 映射表**：dagflow 层自建轻量映射表，仅存储 execID→taskID 的关联关系（5 个字段），不维护状态字段
- **重写 `ExecutionService`**：Run 时写入映射记录，查询时从 taskx 引擎层实时获取状态和节点详情
- **扩展 Proto `NodeStatus`**：新增 input/output/startTime/endTime/durationMs/nodeType/protocol 字段
- **扩展 Proto `Execution`**：新增 taskID 字段
- **前端执行详情展示**：ExecutionPage 展示每个节点的输入、输出和执行时长
- **回滚已有改动**：撤销之前对 `TaskDAO.ListRecent` 接口和 `Task.Description` 元数据注入的修改
- **必要时修改 taskx**：利用已有 `GetTaskOutput` 函数（支持 TaskBak 归档回退查询）辅助历史查询

## Capabilities

### New Capabilities
- `execution-mapping`: dagflow 层执行记录映射表的存储与查询，提供 execID↔taskID 关联
- `execution-detail`: 从 taskx 引擎层实时查询节点级执行详情（input/output/timing）

### Modified Capabilities

## Impact

- **后端**：dagflow 新增 DAO 层（execution_record 表 + DDL）、ExecutionService 重写、Proto 扩展、回滚 TaskDAO 改动
- **taskx**：可能需要调整 `GetTaskOutput` 可见性或添加辅助查询方法
- **前端**：types.ts、ExecutionPage.tsx 更新以展示节点详情
- **数据库**：新增 `execution_record` 表
