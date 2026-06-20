# design-system

设计令牌系统，为 DAGFlow 建立统一的视觉基础。

## ADDED Requirements

### 主题系统
- 支持浅色（light）和深色（dark）两种主题
- 使用 CSS custom properties (variables) 定义所有颜色值
- 通过 `data-theme` 属性在 `<html>` 元素上切换主题
- 主题偏好持久化到 `localStorage`
- 默认跟随系统偏好（`prefers-color-scheme`），用户手动选择后覆盖系统偏好

### 设计令牌

#### 颜色
- 语义化命名：`--bg-primary`, `--bg-secondary`, `--bg-elevated`, `--text-primary`, `--text-secondary`, `--text-muted`, `--border-default`, `--border-subtle`, `--accent-primary`, `--accent-hover`
- 浅色主题：白/浅灰背景 + 深色文字 + 蓝色强调
- 深色主题：深灰/炭黑背景 + 浅色文字 + 蓝色强调

#### 字体
- 主字体：Inter（通过 `@fontsource/inter` 或 Google Fonts CDN）
- Fallback：`-apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif`
- 代码字体：`JetBrains Mono, 'SF Mono', monospace`

#### 间距
- 基准单位 4px，使用 4/8/12/16/24/32/48/64 阶梯

#### 圆角
- sm: 6px, md: 8px, lg: 12px, xl: 16px, full: 9999px

#### 阴影
- 浅色模式：分层柔和阴影（sm/md/lg）
- 深色模式：微妙的边框替代阴影，辅以极低透明度发光

### 图标系统
- 使用 `lucide-react` 图标库
- 统一尺寸：16px（内联）、20px（导航）、24px（标题）
- stroke-width: 1.5px（纤细现代风格）
