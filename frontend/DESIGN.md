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
- **左右分栏**（固定宽左栏 + 弹性右栏，如部门架构「左部门树 + 右子部门表」）用原生 flex 容器：`display:flex; alignItems:flex-start; gap:16`，左栏 `flexShrink:0` 定宽、右栏 `flex:1` 撑满剩余宽度。**禁止用 antd `<Space>` 包裹左右卡片并依赖内层 `flex:1` 撑满**：`Space` 会把子项再包一层不参与 grow 的 `.ant-space-item`，右卡片不会拉伸、页面右侧出现大片空白。
- 布局内边距：`Content` 外边距 20（MainLayout），卡片间距 16–20。
- 搜索区：`Input.Search allowClear prefix={<SearchOutlined/>}`，宽度统一 240–260（可按筛选项 180）。
- Table：`rowKey`、`loading`、分页 `showSizeChanger + showTotal: (t) => \`共 ${t} 条\``；操作列放最右，统一用共享组件 `RowActions` 渲染（≤3 横排，>3 收「更多」纵向下拉，见 §7.3 操作列硬规则）。
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
- **antd「结构级」微修正统一收口在 `AppShell` 的内联 `<style>`**：此类修正针对 antd 内部 DOM / portal 渲染的下拉等，无法用组件 inline style 或 token 表达（如 TreeSelect 下拉树去掉顶层左侧空白：`.ant-select-dropdown .ant-select-tree .ant-select-tree-switcher { width: 0; overflow: hidden; }`，子级缩进由 `.ant-select-tree-indent-unit` 独立控制、不受影响）。**禁止各 app 新增 css 文件、禁止在页面散落重复的全局 `<style>`**；页面级单例 tweak（如部门架构页 `#dept-tree-card` 的树）可内联保留。
- 骨架相关：`TenantSwitcher` 胶囊条 `border-strong` 边 + `table-header` 中性底 + 主色换租户图标。

### 7.2 状态 Tag 字典（唯一规范）

所有状态展示走 `packages/ui` 共享组件，禁止页面内联 `red/green/orange` 传统色名。

| 字段语义 | 组件 | 值 → 文案(Tag 色) |
|---|---|---|
| 启用/停用（status） | `EnableTag` | `enable` → 启用（`success`）；`disable` → 停用（`default`）；其余原样回显。**只认这一套取值**，不做数字/历史值兼容 |
| 挂起（isSuspended / status） | `SuspendedTag` | 1 / true / `suspended` → 挂起（`error`）；0 / false / `active` → 正常（`success`）；兼容布尔字段与 status 枚举两种后端形态 |
| 验证（isVerified） | `VerifiedTag` | 1 → 已验证（`success`）；0 → 未验证（`warning`） |
| 会话（isActive） | 内联语义色 | true → 活跃（`success`）；false → 已失效（`default`） |
| API Key 有效 | 内联语义色 | 有效（`success`）；已吊销（`error`） |

**分类标识（非状态，用固定分类色，不用语义组件）**：类型/来源/可见性等保持固定传统 Tag 色但收敛出处——租户类型用共享 `TypeTag`（platform→geekblue、customer→cyan）；应用/客户端来源与角色来源共用 `SourceTag`（builtin→gold「内置」、first_party→blue「第一方」、third_party→orange「第三方」、custom→blue「自定义」）；页面私有分类如 public/private、菜单类型、超管等允许内联但遵循「分类色」规则，禁止出现三态以上随意取色。

### 7.3 列表页模板（List 骨架）

按以下顺序书写，保证 18 个列表页观感一致：

```
PageContainer(title, description, extra=刷新 + 主操作[type=primary])
  ├─ 搜索区（Input.Search 240 / Select 筛选）
  ├─ Table(rowKey, loading, tableLayout="fixed", scroll={tableScrollX(columns)}, pagination.showTotal)
  │    ├─ ID 列 → idColumn()（IDCell：等宽 + 首 8 尾 4 + Tooltip 复制）
  │    ├─ 名称列 → nameColumn()（NameLink：详情/下级入口，主色可点击链接）
  │    ├─ 文本列 → textColumn()（EllipsisCell：超长省略号截断 + 悬浮全文，列宽固定不随内容膨胀）
  │    ├─ 编码/key/域名列 → textColumn({ monospace: true })
  │    ├─ 状态列 → 7.2 语义组件（列宽 STATUS_COL_WIDTH）
  │    ├─ 时间列 → timeColumn()（列宽 TIME_COL_WIDTH=180 + TimeCell 强制单行；秒级时间戳一律 'YYYY-MM-DD HH:mm:ss'）
  │    └─ 操作列 → actionColumn()（最右，fixed:'right'，RowActions：≤max 横排；>max 收「更多」纵向下拉）
  └─ 新建/编辑 Modal（layout=vertical + confirmLoading）
     详情 Drawer（Descriptions bordered size=small）
```

**列宽与横向滚动硬规则（2026-09 横滚 + 操作列不可见问题后收敛）**

- **所有列都必须有显式列宽，优先取共享常量 / 工厂**：`ID_COL_WIDTH`(130) / `NAME_COL_WIDTH`(180) / `CODE_COL_WIDTH`(150) / `TAG_COL_WIDTH`(110) / `STATUS_COL_WIDTH`(100) / `COUNT_COL_WIDTH`(90) / `TEXT_COL_WIDTH`(200) / `LONG_TEXT_COL_WIDTH`(320)，时间列用 `timeColumn`。常量不合适时可在工厂调用处传显式 `width`（`scroll.x` 已由 `tableScrollX` 求和，不再有漂移风险），但**同类列在各页必须同宽**——禁止同一字段一处 150、一处 180，也禁止写与内容无关的整百"凑数宽度"。历史 bug 根因：租户列表 ID 150 + 租户名 180 + 编码 180 + 类型 120 + 标签 140 + 状态 100 + 双时间列 360 + 操作 240 = 1470px，远超内容区，一进页面就有横滚条。
- **列宽必须严格生效**：Table 一律 `tableLayout="fixed"`。auto 布局下列宽只是建议值，长文本会把列撑开，`EllipsisCell` / `TimeCell` 的省略号与不换行都会失效（表现为列宽失控 + 横滚加剧）。
- **`scroll.x` 由列宽求和得出**：写 `scroll={tableScrollX(columns)}`，**禁止手写 `scroll={{ x: 1490 }}`**。手写值会随列增删漂移（历史上 API Key 页实际合计 1580 却仍写 1520），且写大了会让本可放下的表格强制出现横滚条。总宽小于容器时表格按 `min-width: 100%` 铺满，既不留白也无滚动条。
- **操作列必须 `fixed: 'right'`**：一律用 `actionColumn()` 声明，横滚时钉在右侧可见。这是「操作列要横滑很久才看得到」的结构性修复——含两列 180px 时间列的宽表在窄窗口必然横滚，靠压列宽压不掉。
- **操作列宽度由 `actionColumnWidth(max)` 统一计算**：横排按钮文案保持简短（≤4 个汉字）；文案更长时降低 `max` 让其落入「更多」下拉，或显式传 `width`。页面禁止再手写 120/200/240。
- **文本列一律走 `textColumn()`**（内部 `EllipsisCell`）：既不让长文本撑列，也不让它换行把行高撑成两行；等宽语义（编码 / key / 域名 / IP）传 `monospace: true`。

**操作列与详情入口硬规则（2026-09 收敛）**

- **操作列最多横排 3 个操作**：≤max（默认 3）时全部横排（`Button type="link" size="small"`，危险操作 `danger`）；超过时保留前 `max-1` 个高频操作横排，其余收进「更多」下拉（菜单项纵向排列）。该形态与 Ant Design Table 官方「操作」示例的 `Delete + More actions` 一致，避免操作列被撑宽、按钮挤成一团。
- 操作列一律用共享的 `actionColumn()`（内部 `RowActions`，`@ark-iam/ui`）声明式配置，**禁止页面内手写 `Space + Button/Popconfirm` 拼装，也禁止给操作列另写 `title/key/width/fixed`**；`actions` 数组顺序即优先级，被收起的应是次要/危险操作。运行时隐藏用 `RowAction.hidden`（如已吊销密钥不再展示「吊销」）。
- `RowAction.confirm` 声明二次确认：横排操作走 `Popconfirm`，下拉菜单项走 `Modal.confirm`（下拉会先关闭，气泡无法稳定锚定）。
- **详情入口是名称，不是操作按钮**：列表不设「详情」操作，名称一律用 `nameColumn()`（内部 `NameLink`）渲染为主色可点击链接（hover 下划线 + 手型光标；过长省略号截断并悬浮展示全称，编码/日志键类名称传 `monospace`）。删除「详情」后操作列为空的表，直接移除操作列（`scroll.x` 由 `tableScrollX` 自动跟随收窄，无需手改）。

**可增长资源下拉硬规则（2026-09「只取前 100 条 → 选不到目标」后收敛）**

- 租户、应用等**可持续增长**的资源做下拉（列表筛选 / 表单选择）时，禁止一次性 `getXxxPageList({ page: 1, pageSize: 100 })` 全量铺选项——数据量上百后目标根本不在下拉里。一律用共享组件 **`RemoteSelect`**（单选）/ **`RemoteMultiSelect`**（多选，如角色授权，`@ark-iam/ui`）：选项由 `fetchOptions(keyword)` 按关键字从服务端取，内部 `filterOption={false}`（过滤交给服务端，否则只能搜到已加载的那一页），输入去抖后重新拉取。
- 两个组件保证同样两条不变量：**过期响应被丢弃**（快速输入时旧关键字的结果不会覆盖新结果）、**已选值始终显示名称**（`initialLabel` / `initialLabels` + 历史结果累积，搜索结果替换选项后不会回退成裸 UUID）。`onChange(value, option)` 的第二参是普通 `{ value, label }` 对象（不回传属性不可枚举的 antd 内部展平对象）；多选回传 `(values, options)` 数组。同一页面多个下拉可用 `labelCache` 共享「值 → 名称」，如筛选区选过的服务账号在弹窗表单里直接显示名称。
- **过滤条件必须能表达「空值」语义**：未归属应用的系统角色存储值是 `app_id = ''`，而空串查询参数在 DAO 里语义是「不过滤」，靠前端「全量拉一页再客户端过滤」实现等于把 100 条上限换个地方踩。此类维度加显式服务端开关（如租户角色列表的 `unassigned=true`），不要拿空串参数冒充过滤条件。
- 候选集天然有界且很小（状态枚举、令牌认证方式等）仍用普通 `Select` + 静态 `options`，不必包一层远程搜索。

**时间列硬规则（2026-09 折行问题后收敛）**

- 时间列一律用 `timeColumn<T>({ title, dataIndex })`（或 `TimeCell`），**禁止页面自写列宽**。历史 bug 根因：各页手写 150/160/170，而完整时间串 `2026-09-03 17:19:46` 在 14px + 全站 `tabular-nums` 下实测 146.06px，加 antd 单元格左右各 16px 内边距共需 **178.06px** → 会在唯一的空格处折成「日期 / 时间」两行。
- 宽度由常量给：绝对时间 `TIME_COL_WIDTH = 180`；相对时间 `TIME_COL_WIDTH_RELATIVE = 120`。
- `TimeCell` 的 `whiteSpace: nowrap` 与列宽是一对：nowrap 保证任何布局下都不折行，列宽保证不溢出串列，**两者必须同时生效**（故不要绕过组件直接 `fmtTime` + 自定宽度）。
- 空值语义交给 `placeholder`：`永不过期`（API Key / OAuth Secret 过期时间）、`未验证`（域名验证时间）、`从未使用`（最后使用 / 最近使用），其余默认 `-`。
- 次要时间字段（`最后使用` / `最近使用` / `登录时间`）用 `relative: true`：展示「3 天前」，悬浮 Tooltip 给完整时间；审计主字段（`创建时间`）一律绝对时间。
- **列表时间列必须成对**：所有列表都有「创建时间」列；记录可被编辑/状态流转的业务主体（租户、应用、OAuth 客户端、域名、租户应用、菜单、角色、成员、服务账号、部门、API Key）还必须有「更新时间」列，且后端列表 DTO 同步回传 `updatedAt`。纯追加型 / 不可变记录（审计日志、登录日志、OAuth Secret、第三方身份绑定）不设「更新时间」列——其 `updated_at` 恒等于 `created_at`；这类记录若已有事件时间列（登录日志的「登录时间」）即视为已表达创建语义，不重复加「创建时间」。两列均放状态列之后、操作列之前（`scroll.x` 由 `tableScrollX(columns)` 自动跟随，无需手改）。

### 7.4 登录页（login-web 凭证页 + ui LoginPage 引导页）

- 左品牌区：`brand-gradient` 主渐变 + 白色玻璃圆/胶囊（半透明白层）+ 白字；右侧 `surface-layout` 底 + 白卡（宽 400、圆角 16、大投影）。
- 主按钮高 46、主渐变底、聚焦态输入框描边主色 + 3px 光晕；错误提示浅红底。
- `< 900px` 隐藏左品牌区（单卡居中）。

## 8. Do's and Don'ts

### Do
- 改设计值：先改 `packages/ui/src/theme.ts` 的 `tokens`，再同步本文件 front matter（两处一致）。
- 页面/组件引用颜色一律用 `tokens.*`（`import { tokens } from '@ark-iam/ui'`）或 antd `theme.useToken()`；同包内既有 `brand.*` 为兼容别名，新代码优先 `tokens.*`。
- 状态用语义 Tag 组件；ID 用 `IDCell`；时间列用 `timeColumn()` / `TimeCell`（列宽取 `TIME_COL_WIDTH`），详情/描述区文本可用 `fmtTime`。
- 列宽只取共享常量 / 工厂（`idColumn` / `nameColumn` / `textColumn` / `timeColumn` / `actionColumn` 与 `*_COL_WIDTH`）；Table 一律 `tableLayout="fixed"` + `scroll={tableScrollX(columns)}`。
- 操作列用 `actionColumn()`（自动 `fixed: 'right'`，横滚也可见）；名称列用 `nameColumn()` 作为详情入口，不再设「详情」按钮（见 §7.3）。
- 主色仅用于主操作/链接/选中/焦点；页面表面保持中性灰阶 + 白卡（冷白工程台）。
- 卡片 hairline-only 无阴影；阴影只给 Modal/浮层；列表页骨架照 7.3 模板。
- 左右分栏用原生 flex 容器（见 §4）；antd 结构级微修正统一放 `AppShell` 内联 `<style>`（见 §7.1）。
- 删除无引用 import，避免 noUnusedLocals 报错。

### Don't
- 禁止在业务代码硬编码任何色值（示例：`#4f6ef7`、`#e6e9ef`、`#94a3b8`、`rgba(...)` 品牌投影等）。
- 禁止用传统色名 Tag（`color="green"/"red"/"orange"`）表示启用/停用/挂起等**状态**。
- 禁止 pill 形按钮；禁止把主色渐变当卡片默认背景；禁止同一字段在两端配色不一致。
- 禁止在页面表面使用主色淡底 tint（如 `#ece9ff`、`#fafbff`、`#f6f8ff` 一类表头/行 hover/卡片）——一律中性灰阶。
- 禁止绕过 `TimeCell` 直接给时间列写 `render: (v) => fmtTime(v)` 并自定列宽（会让时间折行或溢出串列）；禁止把 `fmtTime` 的输出再乘 1000（`fmtTime` 已按秒/毫秒自动识别）。
- 禁止页面自写操作列（`Space + Button/Popconfirm`）；禁止为列表新增独立「详情」按钮——详情一律走名称点击（见 §7.3）。
- 禁止在列表页手写列宽数字（`width: 150/180/240`）与 `scroll={{ x: 1490 }}`；禁止给操作列写裸 `RowActions`（必须走 `actionColumn()`，以保证 `fixed: 'right'` 与统一列宽）；禁止省略 `tableLayout="fixed"`（列宽会失效、长文本会撑列）。
- 不要在 auth/login-web 等**不依赖 ui 的层**复制渐变/颜色（依赖方向限制：下层包不能 import @ark-iam/ui）。
- 不要新增 css 文件承载后台样式；样式以 inline style + tokens 表达（登录页除外）。antd 结构级微修正统一放 `AppShell` 内联 `<style>`（见 §7.1），不新增 css 文件、不散落页面级 `<style>`。

## 9. 响应式

- 管理台桌面优先：页面级断点行为不单独处理，宽度不足时由 `tableScrollX(columns)` + `tableLayout="fixed"` 产生横向滚动；操作列 `fixed: 'right'` 始终可见，不需要用户横滑找操作。
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
3. 页面内少量**分类色**（如 role「超管」红色、application public/private、department 主部门 gold、menu 类型蓝/橙/紫）以 antd Tag 预设色内联，未全部收敛为共享组件；按 7.2「分类标识」规则允许，后续可逐步上提。
4. `theme.ts` 与 DESIGN.md 为人工同步，暂无 lint/test 强制一致。
