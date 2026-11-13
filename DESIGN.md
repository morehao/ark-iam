---
version: alpha
name: Ark IAM — Ant Design Native
description: "Ark IAM 前端（登录门户 / 平台管理后台 / 租户管理后台）统一视觉语言 · Ant Design 原生风格（对齐 ark-go-starter）。以 antd 5.x 默认设计语言为基线：蓝色主色 #1677ff、浅色 Sider（白底 + 冷白画布）+ 浅色 Layout、6px 控件圆角 / 8px 卡片圆角、信息密度高、组件成熟。主色仅作功能性强调（主按钮 / 链接 / 选中态 / 聚焦环 / 分页当前页），页面表面一律中性灰阶、卡片以 hairline 边框分层、无默认阴影（阴影仅留给 Modal / Dropdown 等浮层）。语义状态一律用 success/warning/error/default 语义 Tag（浅底 + 浅边 + 深字三件套），禁止在业务代码散落硬编码色值。本文档是项目前端视觉规范的唯一事实源；代码事实源为 packages/ui/src/theme.ts 的 tokens（本文件与之一一对应，改动需两处同步）。遵循 Google DESIGN.md 官方格式规范（google-labs-code/design.md），可用 npx @google/design.md lint DESIGN.md 校验。"

colors:
  primary: "#1677ff"
  primary-strong: "#0b63d6"
  primary-hover: "#4096ff"
  primary-active: "#0958d9"
  primary-soft: "#e6f4ff"
  primary-soft-border: "#91caff"
  secondary: "#4096ff"
  tertiary: "#69b1ff"
  surface: "#ffffff"
  surface-layout: "#f5f5f5"
  surface-sider: "#ffffff"
  surface-sider-sub: "#fafafa"
  surface-table-header: "#fafafa"
  surface-row-hover: "#fafafa"
  surface-fill: "#f5f5f5"
  surface-selected-bg: "#e6f4ff"
  surface-code: "#fafafa"
  border: "#f0f0f0"
  border-strong: "#d9d9d9"
  on-surface: "rgba(0,0,0,0.88)"
  on-surface-secondary: "rgba(0,0,0,0.65)"
  on-surface-tertiary: "rgba(0,0,0,0.45)"
  on-surface-quaternary: "rgba(0,0,0,0.25)"
  on-sider: "rgba(0,0,0,0.88)"
  on-sider-secondary: "rgba(0,0,0,0.65)"
  success: "#52c41a"
  warning: "#faad14"
  error: "#ff4d4f"
  tag-success-bg: "#f6ffed"
  tag-success-border: "#b7eb8f"
  tag-success-text: "#237804"
  tag-warning-bg: "#fffbe6"
  tag-warning-border: "#ffe58f"
  tag-warning-text: "#874d00"
  tag-error-bg: "#fff2f0"
  tag-error-border: "#ffccc7"
  tag-error-text: "#a8071a"
  tag-default-bg: "#fafafa"
  tag-default-border: "#d9d9d9"

typography:
  page-title:
    fontFamily: "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', sans-serif"
    fontSize: 20px
    fontWeight: 600
    lineHeight: 1.4
  card-title:
    fontFamily: "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', sans-serif"
    fontSize: 16px
    fontWeight: 600
    lineHeight: 1.5
  statistic-value:
    fontFamily: "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', sans-serif"
    fontSize: 26px
    fontWeight: 600
    lineHeight: 1.2
    letterSpacing: -0.3px
  body:
    fontFamily: "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', sans-serif"
    fontSize: 14px
    fontWeight: 400
    lineHeight: 1.5715
  label:
    fontFamily: "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', sans-serif"
    fontSize: 14px
    fontWeight: 600
    lineHeight: 1.5
  small:
    fontFamily: "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', sans-serif"
    fontSize: 13px
    fontWeight: 400
    lineHeight: 1.5
  tag:
    fontFamily: "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', sans-serif"
    fontSize: 12px
    fontWeight: 400
    lineHeight: 20px
  mono:
    fontFamily: "ui-monospace, SFMono-Regular, Menlo, monospace"
    fontSize: 13px
    fontWeight: 400
    lineHeight: 1.5

rounded:
  none: 0px
  sm: 4px
  md: 6px
  lg: 8px
  modal: 8px
  full: 9999px

spacing:
  base: 4px
  content-padding: 20px
  card-padding: 20px
  grid-gap: 16px
  control-height: 32px
  header-height: 64px
  sider-width: 232px
  search-control-width: 240px
  cell-padding-y: 12px
  cell-padding-x: 16px

components:
  sider:
    backgroundColor: "{colors.surface-sider}"
    textColor: "{colors.on-sider}"
    width: "{spacing.sider-width}"
    height: "{spacing.header-height}"
  sider-logo:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.on-sider}"
    height: "{spacing.header-height}"
  menu-item:
    backgroundColor: "{colors.surface-sider}"
    textColor: "{colors.on-sider-secondary}"
    rounded: "{rounded.md}"
    height: "42px"
    padding: "0 16px"
  menu-item-active:
    backgroundColor: "{colors.surface-selected-bg}"
    textColor: "{colors.primary-active}"
  menu-item-hover:
    backgroundColor: "{colors.surface-fill}"
    textColor: "{colors.on-sider}"
  header:
    backgroundColor: "{colors.surface}"
    height: "{spacing.header-height}"
    padding: "0 {spacing.content-padding}"
  card:
    backgroundColor: "{colors.surface}"
    rounded: "{rounded.lg}"
    padding: "{spacing.card-padding}"
  content-card-border:
    backgroundColor: "{colors.surface}"
    rounded: "{rounded.lg}"
  statistic-card:
    backgroundColor: "{colors.surface}"
    rounded: "{rounded.lg}"
    padding: "{spacing.card-padding}"
  button-primary:
    backgroundColor: "{colors.primary-strong}"
    textColor: "{colors.surface}"
    rounded: "{rounded.md}"
    height: "{spacing.control-height}"
    padding: "0 15px"
  button-primary-hover:
    backgroundColor: "{colors.primary-active}"
    textColor: "{colors.surface}"
  button-primary-active:
    backgroundColor: "{colors.primary-active}"
    textColor: "{colors.surface}"
  button-default:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.on-surface}"
    rounded: "{rounded.md}"
    height: "{spacing.control-height}"
    padding: "0 15px"
  input:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.on-surface}"
    rounded: "{rounded.md}"
    height: "{spacing.control-height}"
  input-placeholder:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.on-surface-quaternary}"
    height: "{spacing.control-height}"
  input-focus-ring:
    backgroundColor: "{colors.primary-soft-border}"
    rounded: "{rounded.full}"
    size: "2px"
  input-focus:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.on-surface}"
    rounded: "{rounded.md}"
    height: "{spacing.control-height}"
  button-link-hover:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.primary-active}"
  table:
    backgroundColor: "{colors.surface}"
  table-header:
    backgroundColor: "{colors.surface-table-header}"
    textColor: "{colors.on-surface-secondary}"
    padding: "{spacing.cell-padding-y} {spacing.cell-padding-x}"
  table-cell:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.on-surface}"
    padding: "{spacing.cell-padding-y} {spacing.cell-padding-x}"
  table-row-hover:
    backgroundColor: "{colors.surface-row-hover}"
  tag-success:
    backgroundColor: "{colors.tag-success-bg}"
    textColor: "{colors.tag-success-text}"
    rounded: "{rounded.sm}"
  tag-warning:
    backgroundColor: "{colors.tag-warning-bg}"
    textColor: "{colors.tag-warning-text}"
    rounded: "{rounded.sm}"
  tag-error:
    backgroundColor: "{colors.tag-error-bg}"
    textColor: "{colors.tag-error-text}"
    rounded: "{rounded.sm}"
  tag-default:
    backgroundColor: "{colors.tag-default-bg}"
    textColor: "{colors.on-surface-secondary}"
    rounded: "{rounded.sm}"
  tag-success-border-ref:
    backgroundColor: "{colors.tag-success-border}"
    rounded: "{rounded.sm}"
    size: "1px"
  tag-warning-border-ref:
    backgroundColor: "{colors.tag-warning-border}"
    rounded: "{rounded.sm}"
    size: "1px"
  tag-error-border-ref:
    backgroundColor: "{colors.tag-error-border}"
    rounded: "{rounded.sm}"
    size: "1px"
  tag-default-border-ref:
    backgroundColor: "{colors.tag-default-border}"
    rounded: "{rounded.sm}"
    size: "1px"
  dropdown:
    backgroundColor: "{colors.surface}"
    rounded: "{rounded.lg}"
    padding: "4px"
  dropdown-item-hover:
    backgroundColor: "{colors.surface-fill}"
    textColor: "{colors.on-surface}"
  modal:
    backgroundColor: "{colors.surface}"
    rounded: "{rounded.modal}"
  pagination-item:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.on-surface}"
    rounded: "{rounded.md}"
    size: "{spacing.control-height}"
  pagination-item-active:
    backgroundColor: "{colors.primary-strong}"
    textColor: "{colors.surface}"
  divider:
    backgroundColor: "{colors.border}"
    height: "1px"
  outline-strong:
    backgroundColor: "{colors.border-strong}"
    height: "1px"
  page-canvas:
    backgroundColor: "{colors.surface-layout}"
  code-block:
    backgroundColor: "{colors.surface-code}"
    rounded: "{rounded.md}"
  filter-chip-selected:
    backgroundColor: "{colors.primary-soft}"
    textColor: "{colors.primary-active}"
    rounded: "{rounded.md}"
  link:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.primary-strong}"
  link-hover:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.primary-active}"
  sider-footer:
    backgroundColor: "{colors.surface-sider-sub}"
    textColor: "{colors.on-sider-secondary}"
  card-border-strong:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.on-surface-tertiary}"
    rounded: "{rounded.lg}"
  status-dot-success:
    backgroundColor: "{colors.success}"
    rounded: "{rounded.full}"
    size: "8px"
  status-dot-warning:
    backgroundColor: "{colors.warning}"
    rounded: "{rounded.full}"
    size: "8px"
  focus-glow:
    backgroundColor: "{colors.primary-hover}"
    rounded: "{rounded.full}"
    size: "2px"
---

# Ark IAM 前端 DESIGN.md

> 供 AI 编码代理与开发者共同遵循的前端「统一风格约定」。
> **代码唯一事实源是 `packages/ui/src/theme.ts` 的 `tokens` 对象**：改任何色值/圆角/间距必须先改 tokens，再同步本文件 front matter（二者必须一致）。业务页面与组件一律引用 `tokens` / antd token，禁止出现 `#1677ff`、`#f0f0f0`、`rgba(0,0,0,0.25)` 之类的裸色值。
> 参考基准：同级仓库 **ark-go-starter**（Ant Design 原生中后台风格）。Ark IAM 与之一致采用 AntD 原生视觉语言，差异仅在业务域（状态 Tag 字典、列表页规范、登录门户）。

## 1. Overview

Ark IAM 是**企业级 IAM 管理后台**（登录门户 + 平台管理后台 + 租户管理后台），采用 **Ant Design 原生设计语言**——B 端中后台的事实标准。整体气质**克制、规整、信息密度高、交互可预期**：

- **色彩**：以中性灰阶为骨架，蓝色 `primary`（#1677ff）仅作为功能性强调色，用于可交互态与选中态（主按钮 / 链接 / 菜单选中 / 聚焦环 / 分页当前页），**不作大面积装饰**。
- **布局**：左侧**浅色导航 Sider（白底 232px）** + 右侧浅色工作区（白 Header 64px + 冷白 Content），三区经典后台骨架；网格以 4px 为间距基数。
- **层次**：**扁平描边 + 色阶对比**为主——卡片与内容区默认无阴影，靠 `border: 1px solid #f0f0f0` + 白底 vs `#f5f5f5` 画布分层；阴影只留给 Modal / Dropdown 等浮层。
- **密度**：业务页以「统计卡条 + 搜索 + 表格 + 抽屉/弹窗」构成，强调「扫视效率」与「操作一致性」。

> **与旧版差异（对齐 ark-go-starter 的改造）**：旧版为「冷白工程台」——深藏青侧栏（#0f172a）+ 靛蓝主色（#4f6ef7）+ 卡片 hairline 冷调边框。新版统一回到 **Ant Design 原生**：浅色侧栏、AntD 蓝主色、AntD 透明度文字体系、6px/8px 圆角。改造范围与执行计划见 `.dsh/docs/plans/` 下的改造计划文档。

## 2. Colors

调色板以中性灰阶为骨架，蓝色 `primary` 为唯一强调色。

### 品牌与强调

- **Primary (#1677ff)**: 主按钮、链接、菜单选中、Tab/分段选中、分页当前页、输入聚焦环。仅用于可交互与选中态。
- **Primary strong (#0b63d6)**: 主色「承载白字」的无障碍变体（#1677ff 白字仅 4.10:1，此处 5.57:1 满足 WCAG AA）；主按钮、分页当前页底色用它。
- **Primary hover / active (#4096ff / #0958d9)**: 主色交互态。
- **Primary soft (#e6f4ff, 边 #91caff)**: 选中 Tag / 批量操作提示条 / 菜单选中项的浅蓝底。
- **Secondary / Tertiary (#4096ff / #69b1ff)**: AntD 蓝系延续，分类点缀与图表辅助色。

**禁止**：把主色/蓝色当大面积背景或正文色；页面表面使用蓝色淡底 tint（表头/行 hover/卡片一律中性）；用多色 KPI 装饰。统计卡用**单主色 + 中性底**。

### 表面 / 边框 / 文字

| 组 | 令牌 | 值 | 说明 |
|---|---|---|---|
| 表面 | `colors.surface-layout` | #f5f5f5 | 页面画布底色（antd colorBgLayout） |
| | `colors.surface` | #ffffff | 卡片 / Header / 表格底 |
| | `colors.surface-sider` | #ffffff | 侧栏底（**浅色**，与画布靠 hairline 分层） |
| | `colors.surface-sider-sub` | #fafafa | 侧栏次级/折叠底 |
| | `colors.surface-table-header` | #fafafa | Table 表头（中性） |
| | `colors.surface-row-hover` | #fafafa | Table 行 hover（中性） |
| | `colors.surface-fill` | #f5f5f5 | 中性填充底：菜单 hover、固定条 |
| | `colors.surface-selected-bg` | #e6f4ff | 侧栏菜单选中浅蓝底 |
| | `colors.surface-code` | #fafafa | 代码/内容块底 |
| 边框 | `colors.border` | #f0f0f0 | 卡片 / 分隔线（antd colorBorderSecondary） |
| | `colors.border-strong` | #d9d9d9 | 控件描边 / 更强分割线（antd colorBorder） |
| 文字 | `colors.on-surface` | rgba(0,0,0,0.88) | 正文 / 标题（antd colorText） |
| | `colors.on-surface-secondary` | rgba(0,0,0,0.65) | 次要说明（antd colorTextSecondary） |
| | `colors.on-surface-tertiary` | rgba(0,0,0,0.45) | 辅助文字（antd colorTextTertiary） |
| | `colors.on-surface-quaternary` | rgba(0,0,0,0.25) | 空态 / 占位（antd colorTextQuaternary） |
| | `colors.on-sider` / `on-sider-secondary` | 黑 88% / 65% | 浅色侧栏菜单文字 |

### 语义色（状态 Tag 专用）

`colors.success` #52c41a / `colors.warning` #faad14 / `colors.error` #ff4d4f，与 antd `colorSuccess/colorWarning/colorError` 对齐。

状态 Tag 一律「**浅底 + 浅边 + 深字**」三件套：

| 语义 | 底 `tag-*-bg` | 边 `tag-*-border` | 字 `tag-*-text` |
|---|---|---|---|
| success | #f6ffed | #b7eb8f | #237804 |
| warning | #fffbe6 | #ffe58f | #874d00 |
| error | #fff2f0 | #ffccc7 | #a8071a |
| default | #fafafa | #d9d9d9 | on-surface-secondary |

## 3. Typography

字体栈统一使用系统字体（`-apple-system, BlinkMacSystemFont, Segoe UI, Roboto, PingFang SC, Hiragino Sans GB, Microsoft YaHei`），中文优先 PingFang SC / Microsoft YaHei。正文基准 14px、行高 1.5715（antd 默认）。

| 令牌 | 字号 / 字重 / 行高 | 用途 |
|---|---|---|
| `typography.page-title` | 20px / 600 / 1.4 | 页面主标题（Typography.Title level=4） |
| `typography.card-title` | 16px / 600 / 1.5 | 卡片、表格工具栏标题 |
| `typography.statistic-value` | 26px / 600 / 1.2，字距 -0.3px | KPI 指标数值 |
| `typography.body` | 14px / 400 / 1.5715 | 正文与表格单元格 |
| `typography.label` | 14px / 600 / 1.5 | 表头、表单标签、导航菜单 |
| `typography.small` | 13px / 400 / 1.5 | 辅助说明、图例 |
| `typography.tag` | 12px / 400 / 20px | 状态标签、角标 |
| `typography.mono` | 13px，`ui-monospace, SFMono-Regular, Menlo, monospace` | ID/UUID、Key/Secret、API Key、代码/日志 payload、菜单路径 |

- 全站根容器开启 `font-variant-numeric: tabular-nums`，让 ID / 时间 / 数字等宽纵向对齐（由 `AppShell` 统一挂载）。

## 4. Layout

采用经典后台三区骨架：**左侧固定浅色 Sider（232px）+ 顶部白 Header（64px）+ 主内容 Content**。

- **间距基数 4px**，Content 外边距 20px，卡片间距 16px。
- **页面骨架**由 `PageContainer` 统一：页标题（20px/600）+ 描述 + 右上操作区（extra），下方白色内容卡（body padding 20、圆角 8、hairline 边、无阴影）。
- **概要区（summary）**：页标题与内容卡之间的指标卡条，4 列等宽栅格 `repeat(4,1fr)`（用 `StatRow`），窄屏（≤1100px）降为 2 列。
- **工具条（toolbar）**：内容卡顶部、与卡片共享边框的搜索/筛选/分段 Tab 区。
- **左右分栏**（固定宽左栏 + 弹性右栏，如部门架构「左部门树 + 右子部门表」）用原生 flex：`display:flex; alignItems:flex-start; gap:16`，左栏 `flexShrink:0` 定宽、右栏 `flex:1` 撑满。**禁止用 antd `<Space>` 包裹左右卡片并依赖内层 `flex:1`**（Space 会包一层不参与 grow 的 `.ant-space-item`，右卡片不拉伸、右侧留白）。
- **搜索区**：`Input.Search allowClear prefix={<SearchOutlined/>}`，宽度统一 240–260（按筛选项可 180）。
- **控件高度统一 32px**；整体桌面优先，最小推荐宽度 ≥ 1100px。
- **表格**：固定布局（`tableLayout="fixed"`）+ 显式列宽 + 横向 `scroll.x`，单元格内边距 `12px 16px`（详见 §7.3）。

## 5. Elevation & Depth

层次以**扁平描边 + 色阶对比**为主，除浮层外不使用投影：

- 卡片与内容区默认**无阴影**，靠 `border: 1px solid #f0f0f0` + 白底 vs `#f5f5f5` 画布分层。
- **浮层**（Modal / Dropdown / message / Tooltip）使用 antd 标准浮层阴影：`0 6px 16px rgba(0,0,0,0.08), 0 3px 6px -4px rgba(0,0,0,0.12), 0 9px 28px 8px rgba(0,0,0,0.05)`（antd `boxShadowSecondary`）。
- 输入聚焦：边框转主色 + 聚焦环 `0 0 0 2px rgba(5,145,255,0.1)`。
- 侧栏（Sider）浅色底，靠右侧 hairline 与工作区分层，**不加投影**；Header 用底部 hairline 分割。
- 指标卡（statistic-card）可在 hover 时轻阴影 `0 3px 12px rgba(0,0,0,0.06)`（可选）。

## 6. Shapes

形状语言为**方角 + 小圆角**，体现 antd 的实用工程感：

- 按钮 / 输入框 / 分段 / 分页器 / 菜单项：**6px**（`rounded.md`）。
- 卡片 / Statistic 卡 / 内容卡 / Dropdown / Modal：**8px**（`rounded.lg`）。
- 状态 Tag：**4px**（`rounded.sm`）。
- 头像（圆形）、开关（胶囊）、输入聚焦环：全圆 `rounded.full`。

**禁止**胶囊（pill）形按钮（头像、开关除外）；不使用大圆角。

## 7. Components

### 7.1 全局骨架（MainLayout + AppShell）

- **Sider**：`232px`、**浅色底 `colors.surface`**（与工作区靠右侧 `border` hairline 分层）、无投影，支持折叠。Logo 区 64px 白底 + **主色品牌图标**（`SafetyCertificateOutlined` 等）+ 标题（16px/600）+ 副标题（11px，`on-sider-secondary`）。
- **Menu**（浅色 inline）：菜单项高 42px、文字 `on-sider-secondary`；**选中项** `menu-item-active`——`surface-selected-bg`（#e6f4ff）浅蓝底 + `primary` 文字（不用整块亮色填充）；hover 用 `surface-fill`。
- **Header**：白底 64px sticky、底部 hairline 分割，含折叠图标 / 租户切换器 / 用户头像（主色底或图标）+ 下拉。
- **AppShell**：`ConfigProvider`（zhCN + `themeConfig`）在每个 app 的 main.tsx 包裹；根容器开启 `font-variant-numeric: tabular-nums`。
- **antd 结构级微修正**统一收口在 `AppShell` 的内联 `<style>`（针对 antd 内部 DOM / portal 下拉等无法用 token 表达的修正）；**禁止各 app 新增 css 文件、禁止在页面散落全局 `<style>`**。
- **租户切换器 `TenantSwitcher`**：`border-strong` 边 + `surface-table-header` 中性底 + 主色图标。

### 7.2 状态 Tag 字典（唯一规范）

所有状态展示走 `packages/ui` 共享组件，禁止页面内联 `red/green/orange` 传统色名。

| 字段语义 | 组件 | 值 → 文案(Tag 色) |
|---|---|---|
| 启用/停用（status） | `EnableTag` | `enable` → 启用（`success`）；`disable` → 停用（`default`）；其余原样回显。**只认这一套取值** |
| 挂起（status） | `SuspendedTag` | `suspended` → 挂起（`error`）；`active` → 正常（`success`）。**只认这一套取值**，不兼容历史 `isSuspended` |
| 验证（verificationStatus） | `VerifiedTag` | `verified` → 已验证（`success`）；`unverified` → 未验证（`warning`）。**只认这一套取值** |
| 会话（isActive） | 内联语义色 | true → 活跃（`success`）；false → 已失效（`default`） |
| API Key 有效 | 内联语义色 | 有效（`success`）；已吊销（`error`） |

**分类标识（非状态，用固定分类色，不用语义组件）**：租户类型用 `TypeTag`（platform→geekblue、customer→cyan）；应用/客户端来源与角色来源共用 `SourceTag`（builtin→gold「内置」、first_party→blue「第一方」、third_party→orange「第三方」、custom→blue「自定义」）；页面私有分类（public/private、菜单类型、超管等）允许内联但遵循「分类色」规则，禁止三态以上随意取色。

### 7.3 列表页模板（List 骨架）

按以下顺序书写，保证所有列表页观感一致：

```
PageContainer(title, description, extra = 刷新 + 主操作[type=primary])
  ├─ summary（可选）：StatRow + StatCard（如总数 / 本页数 / 当前筛选）
  ├─ toolbar：搜索区（Input.Search 240 / Select 筛选）+ 重置
  ├─ Table(rowKey, loading, tableLayout="fixed", scroll={tableScrollX(columns)}, pagination.showTotal)
  │    ├─ ID 列 → idColumn()（IDCell：等宽 + 首 8 尾 4 + Tooltip 复制）
  │    ├─ 名称列 → nameColumn()（NameLink：详情/下级入口，主色可点击链接）
  │    ├─ 文本列 → textColumn()（EllipsisCell：超长省略号截断 + 悬浮全文）
  │    ├─ 编码/key/域名列 → textColumn({ monospace: true })
  │    ├─ 状态列 → §7.2 语义组件（列宽 STATUS_COL_WIDTH）
  │    ├─ 时间列 → timeColumn()（TIME_COL_WIDTH=180 + TimeCell 强制单行）
  │    └─ 操作列 → actionColumn()（最右，fixed:'right'，RowActions：≤max 横排；>max 收「更多」纵向下拉）
  └─ 新建/编辑 Modal（layout=vertical + confirmLoading）
     详情 Drawer（Descriptions bordered size=small）
```

**列宽与横向滚动硬规则**

- **所有列都必须有显式列宽，优先取共享常量 / 工厂**：`ID_COL_WIDTH`(130) / `NAME_COL_WIDTH`(180) / `CODE_COL_WIDTH`(150) / `TAG_COL_WIDTH`(110) / `STATUS_COL_WIDTH`(100) / `COUNT_COL_WIDTH`(90) / `TEXT_COL_WIDTH`(200) / `LONG_TEXT_COL_WIDTH`(320)，时间列用 `timeColumn`。常量不合适时可传显式 `width`，但**同类列在各页必须同宽**，禁止与内容无关的整百「凑数宽度」。
- **Table 一律 `tableLayout="fixed"`**：auto 布局下长文本会撑开列，`EllipsisCell` / `TimeCell` 的省略号与 nowrap 会失效。
- **`scroll.x` 由列宽求和得出**：写 `scroll={tableScrollX(columns)}`，**禁止手写 `scroll={{ x: 1490 }}`**（手写值随列增删漂移，且写大了会强制出现横滚条；总宽小于容器时按 `min-width: 100%` 铺满）。
- **操作列必须 `fixed: 'right'`**：一律用 `actionColumn()` 声明，横滚时钉在右侧可见；列宽由 `actionColumnWidth(max)` 计算，禁止手写 120/200/240。
- **文本列一律走 `textColumn()`**（内部 `EllipsisCell`）；等宽语义（编码 / key / 域名 / IP）传 `monospace: true`。

**操作列与详情入口硬规则**

- **操作列最多横排 3 个操作**：≤max（默认 3）时全部横排（`Button type="link" size="small"`，危险操作 `danger`）；超过时保留前 `max-1` 个高频操作横排，其余收进「更多」下拉（纵向）。与 Ant Design Table 官方「操作」示例的 `Delete + More actions` 一致。
- 一律用共享 `actionColumn()`（内部 `RowActions`）声明式配置，**禁止页面内手写 `Space + Button/Popconfirm` 拼装，也禁止给操作列另写 `title/key/width/fixed`**；`actions` 数组顺序即优先级。运行时隐藏用 `RowAction.hidden`。
- `RowAction.confirm` 声明二次确认：横排操作走 `Popconfirm`，下拉菜单项走 `Modal.confirm`。
- **详情入口是名称，不是操作按钮**：列表不设「详情」操作，名称一律用 `nameColumn()`（内部 `NameLink`）渲染为主色可点击链接（hover 下划线 + 手型光标；过长省略号截断并悬浮展示全称，编码/日志键传 `monospace`）。删除「详情」后操作列为空的表，直接移除操作列。

**可增长资源下拉硬规则**

- 租户、应用等**可持续增长**的资源做下拉时，禁止一次性 `getXxxPageList({ page: 1, pageSize: 100 })` 全量铺选项。一律用共享组件 **`RemoteSelect`**（单选）/ **`RemoteMultiSelect`**（多选，如角色授权）：选项由 `fetchOptions(keyword)` 按关键字从服务端取，内部 `filterOption={false}`，输入去抖后重新拉取。
- 组件保证：**过期响应被丢弃**（快速输入时旧结果不覆盖新结果）、**已选值始终显示名称**（`initialLabel` / `initialLabels` + 历史结果累积）。`onChange(value, option)` 第二参为普通 `{ value, label }`；多选回传 `(values, options)`。同页多下拉可用 `labelCache` 共享「值 → 名称」。
- **过滤条件必须能表达「空值」语义**：未归属应用的系统角色存储值是 `app_id = ''`，此类维度加显式服务端开关（如 `unassigned=true`），不要拿空串参数冒充过滤条件。
- 候选集天然有界且很小（状态枚举、令牌认证方式）仍用普通 `Select` + 静态 `options`。

**时间列硬规则**

- 时间列一律用 `timeColumn<T>({ title, dataIndex })`（或 `TimeCell`），**禁止页面自写列宽**（完整时间串 `2026-09-03 17:19:46` 实测 ~146px + 单元格左右各 16px 内边距需 ~178px，写小了会在唯一空格处折行）。
- 宽度由常量给：绝对时间 `TIME_COL_WIDTH = 180`；相对时间 `TIME_COL_WIDTH_RELATIVE = 120`。`TimeCell` 的 `whiteSpace: nowrap` 与列宽必须同时生效。
- 空值语义交给 `placeholder`：`永不过期`（API Key / OAuth Secret 过期时间），其余默认 `-`。
- 次要时间字段（`登录时间`）用 `relative: true`（展示「3 天前」+ Tooltip 完整时间）；审计主字段（`创建时间`）用绝对时间。
- **列表时间列必须成对**：所有列表都有「创建时间」列；可被编辑/状态流转的业务主体（租户、应用、OAuth 客户端、域名、租户应用、菜单、角色、成员、服务账号、部门、API Key）还必须有「更新时间」列，且后端列表 DTO 同步回传 `updatedAt`。纯追加型/不可变记录（审计日志、登录日志、OAuth Secret、第三方身份绑定）不设「更新时间」。两列均放状态列之后、操作列之前。

### 7.4 登录页（login-web 凭证页 + ui SsoLoginPage）

- 左品牌区：主色渐变（#1677ff → #4096ff → #69b1ff）+ 白色玻璃圆/胶囊（半透明白层）+ 白字；右侧 `surface-layout` 底 + 白卡（宽 400、圆角 8–12、浮层阴影）。
- 主按钮高 40–46、主色底、聚焦态输入框描边主色 + 2px 光晕；错误提示浅红底（error 语义浅底）。
- `< 900px` 隐藏左品牌区（单卡居中）。

## 8. Do's and Don'ts

### Do
- 改设计值：先改 `packages/ui/src/theme.ts` 的 `tokens`，再同步本文件 front matter（两处一致）。
- 页面/组件引用颜色一律用 `tokens.*`（`import { tokens } from '@morehao/ark-iam-react/ui'`）或 antd `theme.useToken()`；新代码优先 `tokens.*`。
- 状态用语义 Tag 组件（`EnableTag` / `SuspendedTag` / `VerifiedTag` / `TypeTag` / `SourceTag`）；ID 用 `IDCell`；时间列用 `timeColumn()` / `TimeCell`（列宽取 `TIME_COL_WIDTH`），详情/描述区文本可用 `fmtTime`。
- 列宽只取共享常量 / 工厂（`idColumn` / `nameColumn` / `textColumn` / `timeColumn` / `actionColumn` 与 `*_COL_WIDTH`）；Table 一律 `tableLayout="fixed"` + `scroll={tableScrollX(columns)}`。
- 操作列用 `actionColumn()`（自动 `fixed: 'right'`）；名称列用 `nameColumn()` 作为详情入口，不再设「详情」按钮。
- 主色仅用于主操作/链接/选中/焦点；页面表面保持中性灰阶 + 白卡。
- 卡片无阴影（仅浮层/Modal 有阴影）；列表页骨架照 §7.3 模板。
- 左右分栏用原生 flex（§4）；antd 结构级微修正统一放 `AppShell` 内联 `<style>`（§7.1）。
- 状态 Tag 保持 WCAG AA 对比度（正文 ≥ 4.5:1）；删除无引用 import，避免 noUnusedLocals 报错。

### Don't
- 禁止在业务代码硬编码任何色值（如 `#1677ff`、`#f0f0f0`、`rgba(0,0,0,0.65)`、品牌投影）。
- 禁止用传统色名 Tag（`color="green"/"red"/"orange"`）表示启用/停用/挂起等**状态**（分类标识除外）。
- 禁止胶囊形按钮；禁止把主色/蓝色当卡片默认背景或正文色；禁止同一字段在两端配色不一致。
- 禁止在页面表面使用蓝色淡底 tint（表头/行 hover/卡片一律中性）。
- 禁止绕过 `TimeCell` 直接给时间列写 `render: (v) => fmtTime(v)` 并自定列宽；禁止把 `fmtTime` 输出再乘 1000（`fmtTime` 已自动识别秒/毫秒）。
- 禁止页面自写操作列（`Space + Button/Popconfirm`）；禁止为列表新增独立「详情」按钮（详情走名称点击）。
- 禁止手写列宽数字（`width: 150/180/240`）与 `scroll={{ x: 1490 }}`；禁止给操作列写裸 `RowActions`（必须走 `actionColumn()`）；禁止省略 `tableLayout="fixed"`。
- 不要在依赖方向下层（如 auth/login-web 无法 import `@ark-iam/ui` 时）复制渐变/色值而不加注释说明原因。
- 不要新增 css 文件承载后台样式（登录页除外）；antd 结构级微修正统一放 `AppShell` 内联 `<style>`。

## 9. Responsive

- 管理台桌面优先：页面级断点不单独处理，宽度不足时由 `tableScrollX(columns)` + `tableLayout="fixed"` 产生横向滚动；操作列 `fixed: 'right'` 始终可见。
- ≤1100px：KPI 指标条 4 列 → 2 列；左右分栏降为上下堆叠（可选）；Sider 可折叠为抽屉。
- 登录页 `< 900px` 折叠左品牌区；仪表盘卡片 `xs 24 / sm 12 / lg 6` 自动换行。
- 控件高度与字号在窄屏不缩小，保证可点性。

## 10. Token → antd 映射

| DESIGN 令牌 | 值 | antd token |
|---|---|---|
| `colors.primary` | #1677ff | `colorPrimary` / `colorInfo` / `colorLink` |
| `colors.primary-hover` / `primary-active` | #4096ff / #0958d9 | `colorPrimaryHover` / `colorPrimaryActive` |
| `colors.primary-strong` | #0b63d6 | 主按钮/分页当前页底色（`tokens.primaryStrong`） |
| `colors.surface-layout` | #f5f5f5 | `colorBgLayout` |
| `colors.surface` | #ffffff | `colorBgContainer` |
| `colors.surface-sider` | #ffffff | `Layout.Sider` 浅色背景（`siderBg`） |
| `colors.surface-table-header` | #fafafa | `Table.headerBg` |
| `colors.surface-row-hover` | #fafafa | `Table.rowHoverBg` |
| `colors.border` / `border-strong` | #f0f0f0 / #d9d9d9 | `colorBorderSecondary` / `colorBorder` |
| `colors.on-surface(-secondary/-tertiary/-quaternary)` | 黑 88/65/45/25% | `colorText` / `colorTextSecondary` / `colorTextTertiary` / `colorTextQuaternary` |
| `colors.success` / `warning` / `error` | #52c41a / #faad14 / #ff4d4f | `colorSuccess` / `colorWarning` / `colorError` |
| `rounded.sm` / `md` / `lg` | 4px / 6px / 8px | `tokens.radiusSm` / `borderRadius` / `borderRadiusLG` |
| `spacing.control-height` | 32px | `controlHeight` |
| `spacing.header-height` | 64px | `Layout.headerHeight` |
| `spacing.sider-width` | 232px | `Layout.Sider width` |
| `typography.body.fontSize` | 14px | `tokens.fontSizeBody` / `fontSize` |
| `typography.small.fontSize` | 13px | `tokens.fontSizeSmall` |
| `typography.tag.fontSize` | 12px | `tokens.fontSizeTag` |

> 代码侧 `tokens` 命名采用 camelCase（`primaryStrong` / `surfaceLayout` / `onSurfaceSecondary` 等），与本文件 kebab-case 令牌一一映射。

## Known Gaps

1. `theme.ts` 的 `brand` 是历史兼容别名（主色/背景/文字）；改造需同步去掉深藏青/靛蓝相关值，业务代码统一迁移到 `tokens.*`。
2. `packages/auth/src/guards.tsx`（FullPageSpinner）与 login-web 的独立 CSS 因依赖方向无法 import `@ark-iam/ui`，仍需字面 hex；长期方案：把 tokens 下沉到叶子共享包或由宿主注入 CSS 变量。
3. 页面内少量**分类色**（role「超管」、application public/private、department 主部门、menu 类型）以 antd Tag 预设色内联，按 §7.2「分类标识」规则允许，后续可逐步上提为共享组件。
4. `theme.ts` 与 DESIGN.md 为人工同步，暂无 lint/test 强制一致（可加 CI 步骤跑 `npx @google/design.md lint DESIGN.md` 做结构校验）。
