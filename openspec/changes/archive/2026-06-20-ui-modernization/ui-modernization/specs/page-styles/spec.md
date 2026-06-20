# page-styles

各页面视觉重设计，在保留功能逻辑的前提下全面升级视觉表现。

## ADDED Requirements

### Flow 列表页 (FlowListPage)
- 页面标题区：大标题 + 描述文字 + 新建按钮（右对齐），充足留白
- 搜索框：圆角 + 搜索图标（Lucide Search），无背景色，仅底部边框
- Flow 卡片：白色/深色背景 + 微妙阴影 + hover 抬升效果（shadow 加深 + translateY -1px）
- 状态标签：圆角药丸样式，颜色语义化
- 空状态：居中图标 + 引导文案

### Flow 编辑器页 (FlowEditorPage)
- 顶栏：返回箭头 + Flow 名称 + 版本号 + 运行/保存按钮（精致按钮样式）
- 左侧节点面板：精致卡片列表，hover 高亮，拖拽手柄
- 右侧属性面板：分组折叠，表单输入框统一样式（圆角、focus 蓝色发光）
- 运行弹窗：毛玻璃背景 + 居中卡片 + 阴影层次
- ReactFlow 画布：背景色跟随主题，控件样式统一

### 执行记录页 (ExecutionPage)
- Flow 选择区：现代化 select + 标签展示
- 节点输入区：精致 textarea（等宽字体、行号提示）
- 当前执行卡片：状态色带 + 节点状态药丸
- 执行历史列表：卡片样式（非表格），hover 交互

### 协议管理页 (ProtocolPage)
- 协议列表：卡片网格布局
- 协议详情：配置字段展示，code-like 样式

### 通用组件样式
- 按钮：primary/secondary/ghost/danger 四种变体，统一圆角和 hover 效果
- 输入框：统一 focus ring（蓝色发光 2px offset）
- 弹窗/模态框：毛玻璃遮罩 + 居中卡片 + 入场动画（scale + fade）
- 标签/徽章：药丸圆角 + 语义色

### DAG 节点样式（微调）
- BaseNode：配色融入新设计系统（不再使用 Tailwind 默认蓝/绿/红）
- 选中态：accent-primary 边框 + 微妙发光
- 状态色：与新设计系统状态色统一
