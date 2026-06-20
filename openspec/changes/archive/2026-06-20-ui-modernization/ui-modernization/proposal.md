## Why

DAGFlow 前端 UI 呈现 2010 年代的过时视觉风格——纯深色主题、emoji 图标、Tailwind 默认色平铺、紧凑无层次的布局。作为一个现代可视化工作流平台，UI 需要与产品定位匹配，达到 Linear/Vercel/Raycast 级别的精致工具美学。

## What Changes

- 引入双主题系统（浅色 + 深色），支持用户切换，使用 CSS variables + Tailwind dark mode
- 替换全部 emoji 图标为 Lucide React 线条图标
- 重设计 Layout 框架：毛玻璃导航栏、现代化侧边栏、响应式布局
- 重设计所有页面样式：Flow 列表（卡片 + 阴影层次）、编辑器（精致面板）、执行记录（现代表格/卡片）、协议管理
- 建立现代设计系统：Inter 字体、精致配色（含微妙渐变）、留白节奏、圆角规范、阴影层次
- 添加微动画：hover/focus transition、页面切换过渡

## Capabilities

### New Capabilities
- `design-system`: 设计令牌系统 — CSS variables 定义配色/字体/间距/阴影/圆角，浅色+深色双主题
- `layout-shell`: 应用框架重设计 — 毛玻璃顶栏、现代侧边栏（可折叠）、主题切换器、Lucide 图标集成
- `page-styles`: 页面级样式重设计 — Flow 列表/编辑器/执行记录/协议管理全部页面的视觉现代化

### Modified Capabilities

（无现有 spec 需要修改，本次变更为纯视觉改造，不涉及功能逻辑）

## Impact

- **代码**：`frontend/src/` 下所有组件文件和样式文件
- **依赖**：新增 `lucide-react`（图标）、可能需要 `@fontsource/inter`（字体）
- **API**：无后端变更
- **用户体验**：全新视觉体验，浅色/深色可切换，操作效率不降低
