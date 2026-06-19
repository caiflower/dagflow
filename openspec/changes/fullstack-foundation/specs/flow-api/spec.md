## flow-api

Flow 工作流定义的 REST API，提供工作流的增删改查和校验能力。

### 数据模型

**Flow** — 工作流定义，包含节点和边的完整 DAG 描述：

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | Flow 唯一标识，自动生成 |
| name | string | Flow 名称 |
| description | string | 描述 |
| nodes | []Node | 节点列表 |
| edges | []Edge | 边列表 |
| settings | JSON | 全局配置（回滚策略、亲和性等） |
| status | int8 | 0=禁用, 1=启用 |
| createTime | time | 创建时间 |
| updateTime | time | 更新时间 |

**Node** — 节点定义：

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 节点唯一标识 |
| name | string | 节点显示名称 |
| protocol | string | 协议类型（http/grpc/local/mcp） |
| config | JSON | 协议相关配置（URL、方法、参数等） |
| triggerMode | string | all_predecessor / any_predecessor |
| priority | int | 优先级 |
| timeout | int | 超时秒数 |
| retry | int8 | 重试次数 |
| position | {x,y} | 画布坐标（前端使用） |

**Edge** — 边定义：

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 边唯一标识 |
| source | string | 源节点 ID |
| target | string | 目标节点 ID |
| type | string | control / data / control+data |
| mappings | []FieldMapping | 数据字段映射（仅 data 边） |

### API 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/flows | 查询 Flow 列表（分页、按名称/状态过滤） |
| GET | /api/v1/flows/:id | 查询 Flow 详情 |
| POST | /api/v1/flows | 创建 Flow |
| PUT | /api/v1/flows/:id | 更新 Flow |
| DELETE | /api/v1/flows/:id | 删除 Flow |
| POST | /api/v1/flows/:id/validate | 校验 Flow（环检测、类型兼容性、分支目标存在性） |

### 校验规则

- DAG 无环校验（复用 taskx compile 逻辑）
- 起始节点和终止节点存在性
- 分支目标节点存在性
- 节点协议类型必须在已注册协议中
- 必填字段校验（name、至少一个节点）

### Flow ↔ taskx 转换

- Flow.nodes → taskx.Subtask 列表（根据 protocol 查找 ExecutorProvider）
- Flow.edges → taskx.AddEdge / AddControlEdge / AddDataEdge
- Flow.settings → RollbackStrategy, AffinityType
- 转换结果用于调用 taskx.SubmitTask 提交执行
