## run-api

执行触发和状态轮询 API，管理 Flow 的执行生命周期。

### API 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/v1/flows/:id/run | 触发 Flow 执行（可传入参数覆盖） |
| GET | /api/v1/runs/:runId | 查询执行详情（Task + 所有 Subtask 状态） |
| GET | /api/v1/runs/:runId/subtasks | 查询所有子任务状态列表 |
| POST | /api/v1/runs/:runId/abort | 中止执行 |
| POST | /api/v1/runs/:runId/subtasks/:subtaskId/retry | 重试指定子任务 |

### 执行触发流程

1. 接收请求，加载 Flow 定义
2. 通过 converter 将 Flow 转换为 taskx.Task（含 Subtask + DAG 边）
3. 通过 protocol-registry 查找每个节点的 ExecutorProvider
4. 调用 taskx.SubmitTask 提交执行
5. 返回 runId（即 taskx Task.ID）

### 状态轮询

- GET /api/v1/runs/:runId 返回完整的执行状态：
  - task 级别：state（pending/running/succeeded/failed）、创建时间、开始时间、结束时间
  - subtask 级别：每个节点的 state、input、output、error、worker、lastRunTime
- 前端通过定时轮询此接口实现状态刷新（默认 2 秒间隔）

### 响应格式

```json
{
  "runId": "t_xxx",
  "flowId": "f_xxx",
  "flowName": "服务健康检查",
  "state": "running",
  "createTime": "2024-01-01T00:00:00Z",
  "startTime": "2024-01-01T00:00:01Z",
  "subtasks": [
    {
      "id": "st_xxx",
      "name": "收集告警信息",
      "state": "succeeded",
      "input": "{...}",
      "output": "{...}",
      "worker": "node-1",
      "lastRunTime": "2024-01-01T00:00:02Z"
    }
  ]
}
```
