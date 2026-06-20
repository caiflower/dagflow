# layout-shell

应用框架重设计，建立现代化的导航和布局结构。

## ADDED Requirements

### 顶栏（替代当前侧边栏导航头部）
- 高度 48-56px，`backdrop-filter: blur(12px)` 毛玻璃效果
- 左侧：DAGFlow logo（文字，Inter 粗体）+ 应用描述
- 中部/左侧：页面导航 tabs（Flow 列表 / 执行记录 / 协议管理），使用 Lucide 图标 + 文字
- 右侧：主题切换按钮（Sun/Moon 图标）
- 底部细线分隔（`border-bottom: 1px solid var(--border-subtle)`）

### 侧边栏（仅编辑器页）
- Flow 编辑器页保留左侧面板（节点模板列表），重设计样式
- 可折叠（collapse/expand），折叠后仅显示图标
- 使用设计令牌的颜色和阴影

### 导航方式变更
- 当前：侧边栏垂直导航 → 改为：顶栏水平 tabs
- 当前 tab 激活态：`bg-blue-600 text-white` → 改为：底部蓝色指示条 + 文字高亮
- 页面切换添加 fade 过渡动画（100-200ms）

### 主题切换器
- Sun/Moon 图标按钮，点击切换主题
- 切换时添加旋转动画（图标旋转 180°，300ms）
- 状态：自动（跟随系统）/ 浅色 / 深色 三态循环

### 响应式
- 最小宽度 1024px（工具应用，不需要移动端适配）
- 侧边栏在 <1280px 时自动折叠
