# Design: UI Modernization

## 架构决策

### 主题实现方案
**选定：CSS Custom Properties + data-theme 属性**

不使用 Tailwind 的 `dark:` 前缀（需要大量 `dark:bg-xxx` 重复），改用 CSS variables 驱动：

```css
:root, [data-theme="light"] {
  --bg-primary: #ffffff;
  --text-primary: #1d1d1f;
  ...
}
[data-theme="dark"] {
  --bg-primary: #1c1c1e;
  --text-primary: #f5f5f7;
  ...
}
```

Tailwind 配置中映射 variables：`bg-primary: 'var(--bg-primary)'`。这样组件代码保持简洁（`className="bg-primary text-primary"`），主题切换只需修改一个 attribute。

### 图标集成方案
**选定：lucide-react tree-shaking 导入**

```tsx
import { GitBranch, Play, Settings } from 'lucide-react';
```

Vite 自动 tree-shake，只打包使用的图标。统一创建 `<IconWrapper>` 组件处理尺寸/颜色一致性。

### 字体加载
**选定：Google Fonts CDN（Inter + JetBrains Mono）**

在 `index.html` 的 `<head>` 中 preconnect + preload，避免 FOIT。不引入 npm 字体包以保持构建轻量。

### 主题状态管理
**选定：轻量 React context + localStorage**

```
ThemeProvider (context)
  ├── useTheme() hook → { theme, setTheme, toggleTheme }
  └── localStorage key: 'dagflow-theme'
```

不需要 Zustand store（主题状态足够简单，不跨页面共享复杂状态）。

## 文件变更策略

```
新增文件：
  src/styles/theme.css          — CSS variables 定义（light + dark）
  src/styles/globals.css        — 全局基础样式 + Tailwind 指令
  src/components/ThemeProvider.tsx — 主题 context + provider
  src/components/ui/Button.tsx  — 统一按钮组件（4 变体）
  src/components/ui/Modal.tsx   — 统一弹窗组件

重写文件：
  src/components/layout/Layout.tsx  — 顶栏 + 侧边栏重构
  src/pages/FlowListPage.tsx        — 视觉重设计
  src/pages/FlowEditorPage.tsx      — 面板样式重设计
  src/pages/ExecutionPage.tsx       — 视觉重设计
  src/pages/ProtocolPage.tsx        — 视觉重设计
  src/components/dag/nodes/BaseNode.tsx — 配色融入设计系统
  src/components/dag/DAGViewer.tsx     — 画布背景/控件样式
  src/utils/stateColor.ts            — 状态色映射更新

修改文件：
  index.html              — 字体加载
  tailwind.config.ts      — 扩展 CSS variables 映射
  src/index.css            — 导入 theme.css
```

## 配色方案

### 浅色主题
| Token | 值 | 用途 |
|-------|------|------|
| --bg-primary | #ffffff | 主背景 |
| --bg-secondary | #f5f5f7 | 次级背景/卡片 |
| --bg-elevated | #ffffff | 浮层/弹窗 |
| --text-primary | #1d1d1f | 主文字 |
| --text-secondary | #6e6e73 | 次级文字 |
| --text-muted | #86868b | 占位/禁用文字 |
| --border-default | #d2d2d7 | 默认边框 |
| --border-subtle | #e8e8ed | 微妙分隔 |
| --accent-primary | #0071e3 | 蓝色强调（Apple blue） |
| --accent-hover | #0077ed | 蓝色 hover |

### 深色主题
| Token | 值 | 用途 |
|-------|------|------|
| --bg-primary | #000000 | 主背景 |
| --bg-secondary | #1c1c1e | 次级背景/卡片 |
| --bg-elevated | #2c2c2e | 浮层/弹窗 |
| --text-primary | #f5f5f7 | 主文字 |
| --text-secondary | #a1a1a6 | 次级文字 |
| --text-muted | #636366 | 占位/禁用文字 |
| --border-default | #38383a | 默认边框 |
| --border-subtle | #2c2c2e | 微妙分隔 |
| --accent-primary | #2997ff | 蓝色强调（dark mode blue） |
| --accent-hover | #40a0ff | 蓝色 hover |

## 风险

1. **ReactFlow 画布兼容性**：ReactFlow 内部样式可能与 CSS variables 冲突，需要在 `style.ts` 中逐一覆盖
2. **Tailwind v4 兼容性**：当前项目使用 `@import "tailwindcss"` (v4)，CSS variables 映射方式可能与 v3 不同
3. **浏览器兼容性**：`backdrop-filter` 在旧版 Firefox 中需要前缀，但工具应用不需要支持旧浏览器
