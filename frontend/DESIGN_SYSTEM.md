# simpleTranslate 设计规范（Design System）

> 设计语言：**Aurora** — 现代、沉静、专注
> 单一事实来源：`frontend/src/style.css` 中的 CSS 自定义属性（设计令牌）
> 本文档与令牌定义保持同步；如需调整，先改令牌、再同步本文。

---

## 1. 设计原则

| 原则 | 说明 |
| --- | --- |
| 令牌驱动 | 所有颜色/字号/间距/圆角/阴影均消费 CSS 变量，禁止在组件内硬编码 |
| 暗色优先 | 桌面应用默认暗色主题，浅色为 `.light-mode` 覆盖 |
| 4px 栅格 | 间距以 4px 为基础单位，保证视觉节奏一致 |
| 层级分明 | 通过背景层级（base→sidebar→surface→elevated）与阴影建立纵深 |
| 克制动效 | 过渡 150–300ms，缓动 `cubic-bezier(0.4,0,0.2,1)`；尊重 `prefers-reduced-motion` |
| 可访问性 | 键盘聚焦环、3:1 以上对比、语义化图标 + 文字标签 |

---

## 2. 色彩系统

### 2.1 品牌主色（靛蓝 / 紫罗兰）

| 令牌 | 暗色 | 浅色 | 用途 |
| --- | --- | --- | --- |
| `--primary` | `#6366f1` | `#4f46e5` | 主交互色、聚焦环、激活态 |
| `--primary-hover` | `#4f46e5` | `#4338ca` | 悬停加深 |
| `--primary-active` | `#4338ca` | `#3730a3` | 按下态 |
| `--primary-soft` | `rgba(99,102,241,.12)` | `rgba(79,70,229,.08)` | 选中/激活底色 |
| `--accent-glow` | `rgba(99,102,241,.28)` | `rgba(79,70,229,.18)` | 主色光晕阴影 |
| `--accent-grad` | `linear-gradient(135deg,#6366f1,#8b5cf6)` | 同左 | 主按钮/品牌徽标渐变 |

### 2.2 语义色

| 令牌 | 暗色 | 浅色 | 语义 |
| --- | --- | --- | --- |
| `--success` | `#10b981` | `#059669` | 完成、就绪、复制成功 |
| `--success-soft` | `rgba(16,185,129,.12)` | `rgba(5,150,105,.10)` | 成功底色 |
| `--warning` | `#f59e0b` | `#d97706` | 处理中、API 缺失提示 |
| `--warning-soft` | `rgba(245,158,11,.12)` | `rgba(217,119,6,.10)` | 警告底色 |
| `--danger` | `#ef4444` | `#dc2626` | 错误、失败 |
| `--danger-soft` | `rgba(239,68,68,.12)` | `rgba(220,38,38,.08)` | 错误底色 |
| `--info` | `#06b6d4` | `#0891b2` | 信息提示 |
| `--info-soft` | `rgba(6,182,212,.12)` | `rgba(8,145,178,.10)` | 信息底色 |

### 2.3 表面与文本层级

**背景层级（由远及近）：**

| 令牌 | 暗色 | 浅色 | 用途 |
| --- | --- | --- | --- |
| `--bg-base` | `#0d0f14` | `#f7f8fc` | 画布最底层、结果区 |
| `--bg-sidebar` | `#11141b` | `#ffffff` | 侧边栏、状态栏 |
| `--bg-surface` | `#161a23` | `#ffffff` | 主内容、卡片表面 |
| `--bg-elevated` | `#1c2130` | `#ffffff` | 弹窗、下拉选项 |
| `--bg-input` | `#0b0d12` | `#f3f5fa` | 输入框、kbd |
| `--bg-hover` | `#232838` | `#eef1f7` | 悬停反馈 |
| `--bg-overlay` | `rgba(0,0,0,.55)` | `rgba(15,23,42,.35)` | 模态遮罩 |

**文本：**

| 令牌 | 暗色 | 浅色 | 用途 |
| --- | --- | --- | --- |
| `--text-main` | `#e8ebf0` | `#16192a` | 主要文本 |
| `--text-sec` | `#98a2b3` | `#5b6478` | 次要文本、标签 |
| `--text-muted` | `#6b7280` | `#8b94a7` | 弱化文本、快捷键提示 |
| `--text-inverse` | `#ffffff` | `#ffffff` | 主色按钮上的文字 |

**边框：**

| 令牌 | 暗色 | 浅色 | 用途 |
| --- | --- | --- | --- |
| `--border` | `#262c3d` | `#e4e8f0` | 默认分隔线、卡片边 |
| `--border-soft` | `#1f2433` | `#eef1f7` | 轻量分隔 |
| `--border-strong` | `#353c52` | `#d1d8e4` | 滚动条、强分隔 |

---

## 3. 排版系统

**字体栈：** `var(--font-sans)` = `-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", "Nunito", "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", sans-serif`
**等宽：** `var(--font-mono)` = `ui-monospace, "SF Mono", "JetBrains Mono", Menlo, Consolas, monospace`

### 字号尺度（Major Third 1.250）

| 令牌 | 值 | 典型用途 |
| --- | --- | --- |
| `--fs-xs` | 11px | 状态栏、标签说明、kbd |
| `--fs-sm` | 12px | 次要按钮、计数、提示 |
| `--fs-base` | 13px | 正文基准、Toast |
| `--fs-md` | 14px | 导航项、输入、主按钮 |
| `--fs-lg` | 16px | 品牌名、窄屏正文 |
| `--fs-xl` | 18px | 编辑器正文、弹窗标题 |
| `--fs-2xl` | 22px | 大标题 |
| `--fs-3xl` | 28px | 展示标题 |

### 字重 / 行高 / 字间距

| 字重 | 行高 | 字间距 |
| --- | --- | --- |
| `--fw-regular` 400 | `--lh-tight` 1.25 | `--tracking-tight` -0.01em |
| `--fw-medium` 500 | `--lh-snug` 1.40 | `--tracking-normal` 0 |
| `--fw-semibold` 600 | `--lh-normal` 1.50 | `--tracking-wide` 0.02em |
| `--fw-bold` 700 | `--lh-relaxed` 1.60 | `--tracking-wider` 0.05em / `--tracking-widest` 0.08em |

数字计数使用 `font-variant-numeric: tabular-nums`（字符计数等）。

---

## 4. 间距系统（4px 栅格）

| 令牌 | 值 | 令牌 | 值 |
| --- | --- | --- | --- |
| `--sp-0` | 0 | `--sp-5` | 20px |
| `--sp-1` | 4px | `--sp-6` | 24px |
| `--sp-2` | 8px | `--sp-8` | 32px |
| `--sp-3` | 12px | `--sp-10` | 40px |
| `--sp-4` | 16px | `--sp-12` | 48px |

**使用规范：** 组件内边距优先用 `--sp-3`/`--sp-4`；弹窗内边距 `--sp-6`/`--sp-8`；间距用 `--sp-2`/`--sp-3`。

---

## 5. 圆角系统

| 令牌 | 值 | 用途 |
| --- | --- | --- |
| `--radius-xs` | 4px | 引擎分段按钮内部 |
| `--radius-sm` | 6px | 小按钮、聚焦环、下拉 |
| `--radius-md` | 10px | 卡片、输入框、主按钮内部元素 |
| `--radius-lg` | 14px | 设置卡片、弹窗徽标 |
| `--radius-xl` | 20px | 大块容器 |
| `--radius-2xl` | 24px | 弹窗整体 |
| `--radius-full` | 9999px | 胶囊按钮、状态点、交换按钮 |

---

## 6. 阴影系统

| 令牌 | 用途 |
| --- | --- |
| `--shadow-sm` | 轻微凸起（分段选中态） |
| `--shadow-md` | 卡片、下拉 |
| `--shadow-lg` | 抽屉、Toast |
| `--shadow-xl` | 模态弹窗 |
| `--shadow-glow` | 主色按钮 / 品牌徽标光晕（基于 `--accent-glow`） |

---

## 7. 动效系统

| 令牌 | 值 | 用途 |
| --- | --- | --- |
| `--t-fast` | 0.15s | 按下、即时反馈 |
| `--t-base` | 0.2s | 悬停、颜色切换（默认） |
| `--t-slow` | 0.3s | 侧边栏宽度、面板展开 |
| `--ease-standard` | `cubic-bezier(0.4,0,0.2,1)` | 通用缓动 |
| `--ease-spring` | `cubic-bezier(0.34,1.56,0.64,1)` | 弹性（图标旋转） |
| `--ease-out` | `cubic-bezier(0,0,0.2,1)` | 入场 |

**约定：** 所有交互过渡都须附带缓动函数；用户开启系统级「减少动效」时，全局降级至 0.01ms（见 `style.css` 末尾 `@media (prefers-reduced-motion)`）。

---

## 8. 层级（z-index）

| 令牌 | 值 | 用途 |
| --- | --- | --- |
| `--z-base` | 1 | 主内容 |
| `--z-sidebar` | 10 | 侧边栏 |
| `--z-drawer` | 50 | 历史抽屉 + 遮罩 |
| `--z-banner` | 100 | API 缺失横幅 |
| `--z-modal` | 1000 | 设置/快捷键弹窗 |
| `--z-toast` | 2000 | 错误 Toast |

---

## 9. 组件状态定义

所有可交互组件须定义以下状态：

| 状态 | 视觉表现 | 实现 |
| --- | --- | --- |
| **默认 default** | 中性色（`--text-sec` / 透明背景 / `--border`） | 基础样式 |
| **悬停 hover** | `background: var(--bg-hover)`；图标转 `--text-main`；主色按钮 `brightness(1.08) + translateY(-1px)` | `:hover` |
| **激活/选中 active** | `color: var(--primary)` + `background: var(--primary-soft)`；或按下 `transform: scale(0.94~0.96)` | `.active` 类 / `:active` |
| **聚焦 focus** | `outline: 2px solid var(--primary); outline-offset: 2px` | 全局 `:focus-visible` |
| **禁用 disabled** | `opacity: 0.5~0.7` + `cursor: not-allowed` | `:disabled` |
| **成功 success** | `--success` 边框/文字 + `--success-soft` 底 | `.success` |
| **错误 error** | `--danger` 文字 / Toast 背景 `--danger` | `.error` / Toast |

### 按钮体系

- **主按钮（primary）**：`--accent-grad` 渐变背景 + `--text-inverse` 文字 + `--shadow-glow`；悬停提亮并上移。
- **次按钮（secondary）**：透明底 + `--border`；悬停 `--bg-hover`。
- **胶囊模式按钮（mode-btn/toggle-pill）**：激活态 `--primary` 文字 + `--primary-soft` 底。
- **图标按钮（tool-btn/icon-btn）**：透明；悬停 `--bg-hover` + `--border`；按下缩放。

### 输入控件

- 输入框：`--bg-input` 底 + `--border`；聚焦 `border-color: var(--primary)` + `box-shadow: 0 0 0 3px var(--accent-glow)`。
- 下拉：原生 `<select>` 去 appearance，自定义箭头（`::after` 旋转 45° 边框），悬停箭头转 `--primary`。

---

## 10. 响应式布局

桌面应用窗口可缩放，断点：

| 断点 | 行为 |
| --- | --- |
| `> 720px` | 默认布局：侧边栏 240px、头部两端对齐、编辑器 18px |
| `≤ 720px` | 头部 padding 收紧至 `--sp-3`；模式按钮隐藏图标仅留文字；编辑器 padding 收紧、字号降至 `--fs-lg`；状态栏快捷键提示文字隐藏 |

侧边栏可折叠（240px ↔ 72px），折叠时导航文字与引擎选择隐藏、工具按钮纵向居中排列。

---

## 11. 跨设备 / 跨浏览器一致性

- **字体渲染**：`-webkit-font-smoothing: antialiased; -moz-osx-font-smoothing: grayscale; text-rendering: optimizeLegibility;` 保证 macOS/Windows 文字清晰。
- **中文字体回退**：字体栈含 `PingFang SC` / `Hiragino Sans GB` / `Microsoft YaHei`，覆盖 macOS 与 Windows 中文显示。
- **毛玻璃**：`backdrop-filter` 同时写 `-webkit-backdrop-filter`，兼容 WebKit/Wails。
- **下拉箭头**：`appearance: none; -webkit-appearance: none;` 兼容 WebKit。
- **滚动条**：自定义 `::-webkit-scrollbar`，thumb 使用透明边距 + `background-clip: padding-box` 避免溢出。
- **选区**：`::selection` 使用 `--primary-soft`，两套主题一致。
- **动效降级**：`prefers-reduced-motion` 全局降级，兼顾无障碍。
- **盒模型**：全局 `box-sizing: border-box`（见 `style.css`）。
- **目标运行时**：Wails (Chromium WebView)，覆盖 macOS / Windows。

---

## 12. 令牌使用约定

1. 组件 `<style>` **只消费令牌**，不重复定义 `:root` / `.light-mode`（令牌集中于 `style.css`）。
2. 新增颜色前先评估能否复用语义色或层级色；确需新增时在此文档与 `style.css` 同步登记。
3. 禁止使用 `var(--token, #fallback)` 形式的硬编码回退——令牌已全局可用。
4. 装饰性光晕阴影（如 danger glow）允许直接写 `rgba()`，但颜色值须与对应语义色一致。
