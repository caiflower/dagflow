# Tasks: UI Modernization

## Phase 1: 设计系统基础

- [x] 安装依赖：`lucide-react`，配置 Google Fonts（Inter + JetBrains Mono）到 `index.html`
- [x] 创建 `src/styles/theme.css`：CSS variables 定义（light + dark 双主题全部 token）
- [x] 更新 `src/index.css`：导入 theme.css，配置 Tailwind v4 CSS variables 映射
- [x] 创建 `src/components/ThemeProvider.tsx`：主题 context + useTheme hook + localStorage 持久化 + 系统偏好跟随
- [x] 在 `App.tsx` 中包裹 `ThemeProvider`

## Phase 2: 通用 UI 组件

- [x] 创建 `src/components/ui/Button.tsx`：primary/secondary/ghost/danger 四变体按钮
- [x] 创建 `src/components/ui/Modal.tsx`：毛玻璃遮罩 + 居中卡片 + scale+fade 入场动画
- [x] 创建 `src/components/ui/Input.tsx`：统一输入框样式（focus ring）
- [x] 创建 `src/components/ui/Badge.tsx`：标签/徽章组件（语义色药丸样式）
- [x] 更新 `src/utils/stateColor.ts`：状态色映射与新设计系统对齐

## Phase 3: Layout 框架重设计

- [x] 重写 `src/components/layout/Layout.tsx`：毛玻璃顶栏 + 水平导航 tabs + 主题切换按钮
- [x] 替换所有 emoji 图标为 Lucide 图标（List / Play / Plug 等）
- [x] 添加页面切换 fade 过渡动画

## Phase 4: 页面样式重设计

- [x] 重写 `src/pages/FlowListPage.tsx`：现代卡片布局 + 搜索图标 + 空状态 + hover 抬升
- [x] 重写 `src/pages/FlowEditorPage.tsx`：精致顶栏 + 节点面板 + 属性面板 + 运行弹窗样式
- [x] 重写 `src/pages/ExecutionPage.tsx`：现代选择器 + 节点输入区 + 执行卡片样式
- [x] 重写 `src/pages/ProtocolPage.tsx`：卡片网格 + 配置展示样式

## Phase 5: DAG 组件微调

- [x] 更新 `src/components/dag/nodes/BaseNode.tsx`：配色融入设计系统 + 选中态样式
- [x] 更新 `src/components/dag/DAGViewer.tsx`：画布背景跟随主题 + 控件样式统一
- [x] 更新 ReactFlow 控件（缩放/小地图）样式融入新主题

## Phase 6: 验证与收尾

- [x] 浅色主题全页面视觉检查
- [x] 深色主题全页面视觉检查
- [x] 主题切换动画 + localStorage 持久化验证
- [x] TypeScript 编译 + 浏览器无报错验证
