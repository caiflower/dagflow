## ADDED Requirements

### Requirement: dag-viewer

#### Scenario: Full

#### Scenario: Basic

## dag-viewer

DAG 图只读渲染组件，基于 ReactFlow 渲染工作流图，支持编辑器和监控场景复用。

### 核心组件

**DAGViewer** — 只读 DAG 渲染容器：

| Props | 类型 | 说明 |
|-------|------|------|
| nodes | Node[] | 节点列表（含 position） |
| edges | Edge[] | 边列表 |
| nodeStates? | Record<string, string> | 节点状态映射（监控模式） |
| mode | 'preview' \| 'status' | 预览模式 / 状态模式 |
| onNodeClick? | (nodeId) => void | 节点点击回调 |
| fitView? | boolean | 是否自适应视图 |

### 节点渲染

**BaseNode** — 基础节点组件：

- 显示节点名称、协议图标、状态颜色
- 状态颜色映射：
  - pending → 灰色 (#9CA3AF)
  - running → 蓝色 (#3B82F6) + 动画
  - succeeded → 绿色 (#10B981)
  - failed → 红色 (#EF4444)
  - skipped → 黄色 (#F59E0B)
- 预览模式下统一使用浅灰色边框

**节点类型**：
- TaskNode: 普通任务节点（圆角矩形）
- BranchNode: 分支节点（菱形图标）
- StartNode: 起始节点（圆形，绿色）
- EndNode: 结束节点（圆形，红色）

### 边渲染

- ControlEdge: 实线箭头
- DataEdge: 虚线箭头（带数据流标注）
- ControlAndDataEdge: 实线箭头 + 数据流标注
- 边上的标签显示字段映射信息

### 画布功能

- 缩放（Ctrl+滚轮）
- 平移（拖拽画布）
- 小地图（MiniMap）
- 自适应视图（fitView）
- 节点/边只读（不可拖拽编辑）

### 复用场景

- **Flow 列表页**: 小尺寸预览缩略图
- **DAG 编辑器**: 编辑模式的画布基础（扩展可编辑能力）
- **执行监控页**: 状态模式，节点带状态颜色和动画
- **历史回溯页**: 回放模式，可逐步查看执行过程
