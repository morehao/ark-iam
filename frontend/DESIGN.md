---
version: 1.0
name: ark-iam-frontend-design
description: "Ark IAM 前端（登录门户 / 平台管理控制台 / 租户自服务控制台）统一视觉语言。方向为「冷白工程台」收敛（方法论受 Stripe / Vercel 工程控制台启发）：靛蓝 #4f6ef7 是唯一强调色，仅用于主按钮 / 链接 / 选中态 / 登录与 Logo 品牌区；深蓝黑 #0f172a 侧栏；页面表面一律中性灰阶（禁用靛蓝淡底 tint）；卡片与内容区用冷调 hairline 边框分层、不使用卡片阴影（阴影仅留给 Modal 等浮层）；文字为三级 hex 灰阶并全站开启 tabular-nums（ID/时间/数字等宽）。语义状态一律用 success/default/error/warning 语义色渲染为 Tag，禁止在业务代码散落硬编码色值。代码事实源：packages/ui/src/theme.ts 的 tokens 对象（本文件与之一一对应，改动需同步两处）。"

colors:
  brand-primary: "#4f6ef7"
  brand-primary-hover: "#6b86ff"
  brand-primary-active: "#3a55d6"
  brand-purple: "#7a5af8"
  brand-gradient: "linear-gradient(135deg, #4f6ef7 0%, #7a5af8 55%, #a855f7 100%)"
  brand-gradient-soft: "linear-gradient(135deg, #eef2ff 0%, #f5f0ff 100%)"
  surface-layout: "#f6f7f9"
  surface-card: "#ffffff"
  surface-sidebar: "#0f172a"
  surface-table-header: "#f7f8fa"
  surface-table-row-hover: "#f3f4f6"
  surface-soft-fill: "#f1f3f5"
  surface-selected-bg: "rgba(79, 110, 247, 0.14)"
  surface-code: "#fafafa"
  border-hairline: "#e6e9ef"
  border-strong: "#d3d8e0"
  text-primary: "#1f2430"
  text-secondary: "#64748d"
  text-placeholder: "#94a3b8"
  semantic-success: "#22c55e"
  semantic-warning: "#f59e0b"
  semantic-error: "#ef4444"
  semantic-warning-bg: "#fffbe6"
  semantic-warning-border: "#ffe58f"

typography:
  family: "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', sans-serif"
  body-size: 14px
  mono: "ui-monospace, SFMono-Regular, Menlo, monospace"

rounded:
  control: 8px
  card: 12px
  modal: 14px
  login-card: 16px
  stat-icon: 14px

spacing:
  base: 4px
  page-content-margin: 20px
  content-padding: 20px
  search-control-width: 240px

components:
  main-sider:
    backgroundColor: "{colors.surface-sidebar}"
    width: 232px
  main-header:
    backgroundColor: "{colors.surface-card}"
    height: 56px
  page-card:
    backgroundColor: "{colors.surface-card}"
    border: "1px solid {colors.border-hairline}"
    rounded: "{rounded.card}"
    body-padding: "{spacing.content-padding}"
  status-tag-success: "{colors.semantic-success}"
  status-tag-error: "{colors.semantic-error}"
  status-tag-warning: "{colors.semantic-warning}"
  status-tag-default: "停用（无彩色底）"
---

# Ark IAM 前端 DESIGN.md

> 供 AI 编码代理与开发者共同遵循的前端「统一风格约定」。
> **代码唯一事实源是 `packages/ui/src/theme.ts` 的 `tokens` 对象**：改任何色值/圆角/间距必须先改 tokens，再同步本文件 front matter（二者必须一致）。业务页面与组件一律引用 tokens / antd token，禁止出现 `#4f6ef7`、`#f0f0f0`、`#94a3b8` 之类的裸色值。

## 1. 视觉主题总览

- **定位**：企业级 IAM 管理后台，「冷白工程台」——浅色、数据密集、克制。
- **画布**：页面底色 `{colors.surface-layout}`（冷白）；内容承载于白色圆角卡片 `{colors.surface-card}`，卡片以 `1px solid {colors.border-hairline}` hairline 描边分层、**不使用卡片阴影**（阴影仅留给 Modal 等浮层）。
- **品牌**：靛蓝 → 紫。`{colors.brand-primary}` 靛蓝是唯一强调色（主按钮 / 链接 / 选中态 / 输入聚焦）；渐变仅出现在**登录品牌区与 Logo/头像**（全站唯一“hero 大气层”）。
- **导航**：左侧深色栏 `{colors.surface-sidebar}`（#0f172a，近黑蓝），菜单选中项用 `{colors.surface-selected-bg}` 柔和半透明高亮（非整块亮色）；顶部白 Header 56px。
- **基调**：以「表格 + 搜索 + 抽屉/弹窗」构成业务页；页面表面一律**中性灰阶**（禁用靛蓝淡底 tint），把主色当作稀有资源；全站开启 `tabular-nums` 让 ID/时间/数字等宽对齐。

> **参照锚点**：方法论受 **Stripe**（数据表格骨架、`tabular-nums`、ink-mute 灰阶文字）与 **Vercel**（精确灰阶分层、渐变只出现在登录/品牌区这一个“hero 大气层”）启发，纪律细节参考 **Linear / Supabase**（主色稀有、hairline 层级）。非照搬任何具体品牌配色，主色与数值为 Ark IAM 自定。

## 2. 色彩

### 品牌与强调

| 令牌 | 值 | 用途 | 稀缺级别 |
|---|---|---|---|
| `{colors.brand-primary}` | #4f6ef7 | 主按钮、链接、菜单选中、焦点环、品牌图标 | **稀有，只用这三五处** |
| `{colors.brand-primary-hover}` | #6b86ff | 主按钮 hover | 继承 |
| `{colors.brand-primary-active}` | #3a55d6 | 主按钮按下 | 继承 |
| `{colors.brand-purple}` | #7a5af8 | 分类图标强调（按钮类型、统计卡等） | 装饰点缀 |
| `{colors.brand-gradient}` | 靛→紫渐变 | 登录品牌区、Logo、头像 | 品牌专属 |
| `{colors.brand-gradient-soft}` | 淡靛渐变 | 登录品牌区浅淡底（页面内禁用） | 轻量 |

**禁止**：把主色/渐变大面积铺满页面、作任意卡片默认背景、作正文颜色；页面内不得出现主色淡底 tint（表头/行 hover/卡片一律中性）。仪表盘统计卡使用**单主色 + 中性底**（`{colors.surface-soft-fill}` 图标底 + `{colors.brand-primary}` 图标），不用多色统计。

### 表面 / 边框 / 文字

| 组 | 令牌 | 值 | 说明 |
|---|---|---|---|
| 表面 | `{colors.surface-layout}` | #f6f7f9 | 页面布局底色（antd colorBgLayout），冷白 |
| | `{colors.surface-card}` | #ffffff | 卡片 / 表头底（Header） |
| | `{colors.surface-sidebar}` | #0f172a | 侧栏深底 |
| | `{colors.surface-table-header}` | #f7f8fa | Table 表头（中性，禁止靛 tint） |
| | `{colors.surface-table-row-hover}` | #f3f4f6 | Table 行 hover（中性） |
| | `{colors.surface-soft-fill}` | #f1f3f5 | 中性填充底：统计卡图标底、应用固定条 |
| | `{colors.surface-code}` | #fafafa | 代码/内容块底 |
| 边框 | `{colors.border-hairline}` | #e6e9ef | 冷调 hairline，卡片/分隔（antd colorBorderSecondary 同源） |
| | `{colors.border-strong}` | #d3d8e0 | 更强的分割线（Header 内竖分割线等） |
| 文字 | `{colors.text-primary}` | #1f2430 | 冷近黑正文/标题（antd colorText） |
| | `{colors.text-secondary}` | #64748d | 次要说明（antd colorTextSecondary，借鉴 Stripe ink-mute） |
| | `{colors.text-placeholder}` | #94a3b8 | 空态、占位、弱辅助文字 |

### 语义色（状态 Tag 专用）

`{colors.semantic-success}` `#22c55e` / `{colors.semantic-warning}` `#f59e0b` / `{colors.semantic-error}` `#ef4444`，与 antd `colorSuccess/colorWarning/colorError` 对齐。

| 附加 | 令牌 | 值 |
|---|---|---|
| 轻告警底 | `{colors.semantic-warning-bg}` | #fffbe6 |
| 轻告警边 | `{colors.semantic-warning-border}` | #ffe58f |

## 3. 字体与排版

- 字体栈见 front matter `typography.family`（antd fontFamily 已全局注入），中文优先 PingFang SC / Microsoft YaHei。
- 字号沿用 antd 层级：正文 14px（默认）、标题 Title 4=20px/标题卡片等用 antd Typography、表内弱文字 12–13px、mono 13px。
- **等宽字体场景**：ID/UUID（`IDCell`）、Key/Secret、API Key、代码/日志 payload、菜单路径 `ui-monospace, SFMono-Regular, Menlo, monospace`。
- 表格中「@用户名 / 辅助说明」使用 `{colors.text-secondary}`；空态与弱说明（“加载中… / 未设置”等）使用 `{colors.text-placeholder}`。

## 4. 布局与间距

- 间距基数 4px；管理页由 `PageContainer` 统一骨架：页标题 + 描述 + 右上操作区（extra），下方白色内容卡（body padding 20、圆角 12、hairline 边）。
- **左右分栏**（固定宽左栏 + 弹性右栏，如组织架构「左部门树 + 右子部门表」）用原生 flex 容器：`display:flex; alignItems:flex-start; gap:16`，左栏 `flexShrink:0` 定宽、右栏 `flex:1` 撑满剩余宽度。**禁止用 antd `<Space>` 包裹左右卡片并依赖内层 `flex:1` 撑满**：`Space` 会把子项再包一层不参与 grow 的 `.ant-space-item`，右卡片不会拉伸、页面右侧出现大片空白。
- 布局内边距：`Content` 外边距 20（MainLayout），卡片间距 16–20。
- 搜索区：`Input.Search allowClear prefix={<SearchOutlined/>}`，宽度统一 240–260（可按筛选项 180）。
- Table：`rowKey`、`loading`、分页 `showSizeChanger + showTotal: (t) => \`共 ${t} 条\``；操作列放最右，用 `Button type="link" size="small"`（危险操作加 `danger`）。
- 弹窗表单：`Modal + Form layout="vertical"`、提交按钮 `confirmLoading`、`destroyOnClose`；详情用 `Drawer + Descriptions bordered size="small"`。
- 栅格/断点：业务控制台以桌面优先；卡片 `Row/Col` 用 `xs/sm/lg`（仪表盘 `xs=24 sm=12 lg=6`）。整体不做窄屏降级，最小宽度建议 ≥ 1100px。

## 5. 层级与阴影

- **卡片与内容区一律 hairline-only，无阴影**：层间靠 `surface` 灰阶 + 冷调 hairline 边框 + 留白分层（Linear / Supabase 哲学）。阴影只留给浮层：Modal / Dropdown / Drawer 使用 antd `boxShadowSecondary = 0 6px 24px rgba(15,23,42,0.08)`。
- 侧栏（Sider）紧贴画布靠色彩对比分层，**不加投影**；Header 用底部 hairline 分割。
- 唯一例外（品牌浮层）：登录卡大投影 `0 20px 60px rgba(15,23,42,0.12)`；主按钮投影带品牌色 `rgba(79,110,247,0.35)`。
- 深色底上的玻璃元素（半透明白 rgba 层）只存在于登录品牌区。

## 6. 圆角

| 场景 | 圆角 |
|---|---|
| 按钮 / 输入 / 单元格 Tag / Menu 项 | 8px（antd borderRadius） |
| 页面卡片 / 内容卡 / 详情 | 12px |
| Modal | 14px（antd borderRadiusLG 覆盖） |
| 登录卡 / 统计卡图标 | 16px / 14px |

**禁止**胶囊（pill）形按钮；Tag 可用默认小圆角。

## 7. 组件规格

### 7.1 全局骨架（MainLayout + AppShell）

- Sider `232px`、深底 `surface-sidebar` 且**无投影**；Menu 选中项为 `{colors.surface-selected-bg}` 柔和半透明高亮 + 白字（不用整块亮色填充），hover 用 `rgba(255,255,255,0.08)`；Logo 区 64px 主渐变底白字（品牌高光）；支持折叠。
- Header 白底 56px sticky、底部 hairline 分割，含折叠图标 / 租户切换器 / 用户头像（渐变底）+ 下拉。
- 全局主题由 `AppShell`（`ConfigProvider` zhCN + themeConfig）在每个 app 的 main.tsx 包裹；AppShell 根容器开启 `font-variant-numeric: tabular-nums`，**全站 ID/时间/数字等宽对齐**。
- **antd「结构级」微修正统一收口在 `AppShell` 的内联 `<style>`**：此类修正针对 antd 内部 DOM / portal 渲染的下拉等，无法用组件 inline style 或 token 表达（如 TreeSelect 下拉树去掉顶层左侧空白：`.ant-select-dropdown .ant-select-tree .ant-select-tree-switcher { width: 0; overflow: hidden; }`，子级缩进由 `.ant-select-tree-indent-unit` 独立控制、不受影响）。**禁止各 app 新增 css 文件、禁止在页面散落重复的全局 `<style>`**；页面级单例 tweak（如组织架构页 `#org-tree-card` 的树）可内联保留。
- 骨架相关：`TenantSwitcher` 胶囊条 `border-strong` 边 + `table-header` 中性底 + 主色换租户图标。

### 7.2 状态 Tag 字典（唯一规范）

所有状态展示走 `packages/ui` 共享组件，禁止页面内联 `red/green/orange` 传统色名。

| 字段语义 | 组件 | 值 → 文案(Tag 色) |
|---|---|---|
| 启用/停用（status） | `StatusTag` | enable / 1 / active → 启用（`success`）；disable / 0 / inactive → 停用（`default`）；suspended → 挂起（`error`） |
| 挂起（isSuspended） | `SuspendedTag` | 1 / true → 挂起（`error`）；0 / false → 正常（`success`） |
| 验证（isVerified） | `VerifiedTag` | 1 → 已验证（`success`）；0 → 未验证（`warning`） |
| 会话（isActive） | 内联语义色 | true → 活跃（`success`）；false → 已失效（`default`） |
| API Key 有效 | 内联语义色 | 有效（`success`）；已吊销（`error`） |

**分类标识（非状态，用固定分类色，不用语义组件）**：类型/来源/可见性等保持固定传统 Tag 色但收敛出处——租户/应用类型用共享 `TypeTag`（platform→geekblue、customer→cyan、first_party→blue、third_party→orange）；角色来源用共享 `SourceTag`（builtin→gold「内置」，其余→blue「自定义」）；页面私有分类如 public/private、菜单类型、超管等允许内联但遵循「分类色」规则，禁止出现三态以上随意取色。

### 7.3 列表页模板（List 骨架）

按以下顺序书写，保证 18 个列表页观感一致：

```
PageContainer(title, description, extra=刷新 + 主操作[type=primary])
  ├─ 搜索区（Input.Search 240 / Select 筛选）
  ├─ Table(rowKey, loading, scroll.x, pagination.showTotal)
  │    ├─ ID 列 → IDCell（等宽 + 首 8 尾 4 + Tooltip 复制）
  │    ├─ 长文本列 → EllipsisCell
  │    ├─ 状态列 → 7.2 语义组件
  │    ├─ 时间列 → fmtTime（秒级时间戳，一律 'YYYY-MM-DD HH:mm:ss'）
  │    └─ 操作列（最右，link+small；删除套 Popconfirm，危险按钮 danger）
  └─ 新建/编辑 Modal（layout=vertical + confirmLoading）
     详情 Drawer（Descriptions bordered size=small）
```

### 7.4 登录页（login-web 凭证页 + ui LoginPage 引导页）

- 左品牌区：`brand-gradient` 主渐变 + 白色玻璃圆/胶囊（半透明白层）+ 白字；右侧 `surface-layout` 底 + 白卡（宽 400、圆角 16、大投影）。
- 主按钮高 46、主渐变底、聚焦态输入框描边主色 + 3px 光晕；错误提示浅红底。
- `< 900px` 隐藏左品牌区（单卡居中）。

## 8. Do's and Don'ts

### Do
- 改设计值：先改 `packages/ui/src/theme.ts` 的 `tokens`，再同步本文件 front matter（两处一致）。
- 页面/组件引用颜色一律用 `tokens.*`（`import { tokens } from '@ark-iam/ui'`）或 antd `theme.useToken()`；同包内既有 `brand.*` 为兼容别名，新代码优先 `tokens.*`。
- 状态用语义 Tag 组件；时间用 `fmtTime`；ID 用 `IDCell`。
- 主色仅用于主操作/链接/选中/焦点；页面表面保持中性灰阶 + 白卡（冷白工程台）。
- 卡片 hairline-only 无阴影；阴影只给 Modal/浮层；列表页骨架照 7.3 模板。
- 左右分栏用原生 flex 容器（见 §4）；antd 结构级微修正统一放 `AppShell` 内联 `<style>`（见 §7.1）。
- 删除无引用 import，避免 noUnusedLocals 报错。

### Don't
- 禁止在业务代码硬编码任何色值（示例：`#4f6ef7`、`#e6e9ef`、`#94a3b8`、`rgba(...)` 品牌投影等）。
- 禁止用传统色名 Tag（`color="green"/"red"/"orange"`）表示启用/停用/挂起等**状态**。
- 禁止 pill 形按钮；禁止把主色渐变当卡片默认背景；禁止同一字段在两端配色不一致。
- 禁止在页面表面使用主色淡底 tint（如 `#ece9ff`、`#fafbff`、`#f6f8ff` 一类表头/行 hover/卡片）——一律中性灰阶。
- 不要在 auth/login-web 等**不依赖 ui 的层**复制渐变/颜色（依赖方向限制：下层包不能 import @ark-iam/ui）。
- 不要新增 css 文件承载后台样式；样式以 inline style + tokens 表达（登录页除外）。antd 结构级微修正统一放 `AppShell` 内联 `<style>`（见 §7.1），不新增 css 文件、不散落页面级 `<style>`。

## 9. 响应式

- 管理台桌面优先：页面级断点行为不单独处理，宽度不足时 Table 用 `scroll={{x}}` 横向滚动。
- 登录页 `< 900px` 折叠品牌区；仪表盘卡片 `xs 24 / sm 12 / lg 6` 自动换行。

## 10. 令牌清单速查（与 theme.ts tokens 对齐）

| 代码 tokens | DESIGN 令牌 | 语义 |
|---|---|---|
| `tokens.primary` | brand-primary | 主强调色 |
| `tokens.purple` | brand-purple | 分类紫（统计/菜单按钮图标） |
| `tokens.gradient` / `tokens.gradientSoft` | brand-gradient(-soft) | 品牌渐变 |
| `tokens.bg` | surface-layout | 页面底 |
| `tokens.cardBg` / `tokens.headerBg` | surface-card | 卡/Header 底 |
| `tokens.sidebarBg` | surface-sidebar | 侧栏底 |
| `tokens.tableHeaderBg` / `tokens.rowHoverBg` | surface-table-header / row-hover | 表格（中性） |
| `tokens.softFill` | surface-soft-fill | 中性填充底（统计图标底/固定条） |
| `tokens.selectedBg` | surface-selected-bg | 深色导航选中柔和底 |
| `tokens.codeBg` | surface-code | 代码块底 |
| `tokens.border` / `tokens.borderStrong` | border-hairline / border-strong | 边框 |
| `tokens.text` / `textSecondary` / `textPlaceholder` | text-primary / secondary / placeholder | 文字三级 |
| `tokens.success` / `warning` / `error` | semantic-success / warning / error | 语义色 |
| `tokens.warningBg` / `warningBorder` | semantic-warning-bg / warning-border | 语义告警淡底 |

## Known Gaps

1. `packages/auth/src/guards.tsx`（FullPageSpinner）已中性化（`#f6f7f9` 底），但 auth 在 ui 依赖方向下层、无法 import @ark-iam/ui，仍需字面 hex；login-web 同理。长期方案：把 tokens 下沉到叶子共享包（如 @ark-iam/types 同级）或由宿主注入 CSS 变量。
2. `apps/login-web/src/LoginPage.css` 用独立 CSS（brand 渐变、focus、错误色）未走 tokens，且其渐变仍为“品牌高光”允许范围；重构需 login-web 依赖 ui 或 CSS 变量注入，本分支未做。
3. 页面内少量**分类色**（如 role「超管」红色、application public/private、organization 主部门 gold、menu 类型蓝/橙/紫）以 antd Tag 预设色内联，未全部收敛为共享组件；按 7.2「分类标识」规则允许，后续可逐步上提。
4. `theme.ts` 与 DESIGN.md 为人工同步，暂无 lint/test 强制一致。
