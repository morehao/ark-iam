# @ark-iam/ui

Ark IAM 前端共享组件与设计令牌库（React + antd 5）。

提供三类能力：

1. **设计令牌**：`themeConfig` / `tokens` / `brand` —— 全站视觉唯一事实源（Ant Design 原生：浅色侧栏 + AntD 蓝主色 `#1677ff`，6px 控件圆角 / 8px 卡片圆角），配合 antd `ConfigProvider` 使用。
2. **列表页列工厂**：`actionColumn` / `nameColumn` / `textColumn` / `idColumn` / `timeColumn` / `tableScrollX` 及列宽常量 —— 收敛各页列宽、操作列与横向滚动。
3. **骨架与通用组件**：`AppShell` / `MainLayout` / `PageContainer` / `LoginPage` / `TenantSwitcher` / `ProfileCenter` / `RemoteSelect` / `RemoteMultiSelect` / 状态 Tag 等。

> 设计规范细节见仓库根 `DESIGN.md`（视觉规范唯一事实源，可用 `npx @google/design.md lint DESIGN.md` 校验格式）；接入完整说明见 `docs/design/application-integration-guide.md` §12「前端组件接入」。

> 说明：`@ark-iam/ui` 是**内部包**，不单独对外发布。外部业务项目请安装门面包 `@morehao/ark-iam-react`，从 `@morehao/ark-iam-react/ui` 引入本包的组件与令牌。见 `packages/react/README.md`。

## 安装（门面包）

```bash
pnpm add @morehao/ark-iam-react
# antd / react / react-dom / react-router-dom 作为 peer 由宿主提供
pnpm add antd@^5.22.0 react@^18.3.1 react-dom@^18.3.1 react-router-dom@^7.15.0
```

## 主题

```tsx
import { ConfigProvider } from 'antd'
import zhCN from 'antd/locale/zh_CN'
import { themeConfig } from '@morehao/ark-iam-react/ui'

<ConfigProvider locale={zhCN} theme={themeConfig}>
  <App />
</ConfigProvider>
```

或直接使用 `AppShell`（内部已包好 `ConfigProvider` + `themeConfig` + antd `App` 上下文）。

```tsx
import { AppShell } from '@morehao/ark-iam-react/ui'

<AppShell>
  <App />
</AppShell>
```

## 主要导出

### 主题令牌

| 导出 | 说明 |
|---|---|
| `themeConfig` | antd `ThemeConfig`，全局主题 |
| `tokens` | 设计令牌对象（颜色/表面/边框/文字/语义色），`theme.ts` 事实源 |
| `brand` | 历史兼容的品牌令牌子集，新代码请用 `tokens` |

### 列工厂（列表页专用）

| 导出 | 说明 |
|---|---|
| `actionColumn<T>({ actions, max?, width? })` | 操作列：≤`max`（默认 3）横排，超过收「更多」下拉；**永远 `fixed: 'right'`**；列宽由 `actionColumnWidth(max)` 自动给出 |
| `nameColumn<T>({ title, dataIndex, onClick?, monospace?, width? })` | 名称列（`NameLink` 主色链接，详情入口；过长省略号 + 悬浮全文） |
| `textColumn<T>({ title, dataIndex, width?, placeholder?, monospace? })` | 文本列（`EllipsisCell` 截断 + 悬浮全文） |
| `idColumn<T>({ dataIndex, title?, width? })` | ID 列（`IDCell` 等宽 + 首 8 尾 4 + Tooltip） |
| `timeColumn<T>({ title, dataIndex, relative?, placeholder? })` | 时间列（`TimeCell` 单行不折行） |
| `tableScrollX(columns)` | 由列宽求和得出 `scroll.x`，**禁止手写固定 `scroll.x`** |
| 列宽常量 | `ID_COL_WIDTH`(130) / `NAME_COL_WIDTH`(180) / `CODE_COL_WIDTH`(150) / `TAG_COL_WIDTH`(110) / `STATUS_COL_WIDTH`(100) / `COUNT_COL_WIDTH`(90) / `TEXT_COL_WIDTH`(200) / `LONG_TEXT_COL_WIDTH`(320) |

### 其它组件

`AppShell`、`MainLayout`、`PageContainer`、`LoginPage`、`TenantSwitcher`、`ProfileCenter`、`RemoteSelect` / `RemoteMultiSelect`、`EllipsisCell`、`IDCell`、`NameLink`、`RowActions`、`InitialPasswordModal`、状态 Tag（`EnableTag` / `SuspendedTag` / `VerifiedTag` / `TypeTag` / `SourceTag`）、时间格式化（`fmtTime` / `fmtRelativeTime` / `toMillis`）。

## 最小示例（`actionColumn` 表格）

```tsx
import { Table } from 'antd'
import { actionColumn, nameColumn, textColumn, tableScrollX } from '@morehao/ark-iam-react/ui'

interface Item {
  id: string
  name: string
  code: string
}

const columns = [
  nameColumn<Item>({ title: '名称', dataIndex: 'name', onClick: (r) => openDetail(r) }),
  textColumn<Item>({ title: '编码', dataIndex: 'code', monospace: true }),
  actionColumn<Item>({
    max: 2,
    actions: (r) => [
      { key: 'edit', label: '编辑', onClick: () => edit(r) },
      { key: 'delete', label: '删除', danger: true, confirm: '确认删除？', onClick: () => del(r) },
    ],
  }),
]

<Table
  rowKey="id"
  columns={columns}
  dataSource={rows}
  tableLayout="fixed"
  scroll={tableScrollX(columns)}
/>
```

`RowAction` 字段：`key`、`label`、`onClick?`、`danger?`、`disabled?`、`confirm?`(二次确认)、`icon?`、`hidden?`(运行时隐藏)。

## 规范要点（务必遵守）

- 操作列**一律用 `actionColumn`**，禁止页面内手写 `Space + Button/Popconfirm`，也不要另写 `title`/`key`/`width`/`fixed`。
- 详情入口是**名称**（`nameColumn`），列表不设独立「详情」按钮。
- Table 必须 `tableLayout="fixed"`，`scroll` 必须用 `tableScrollX(columns)`，禁止手写固定值。
- 所有列都有显式列宽，优先取列工厂/常量；同类列在各页必须同宽。
- 视觉只用 `tokens` / `themeConfig`，禁止硬编码色值或自造样式。

规范出处：`DESIGN.md` §7.3；接入说明：`docs/design/application-integration-guide.md` §12。

## peerDependencies 口径

`antd`(`^5.22.0`)、`@ant-design/icons`(`^5.4.0`)、`react`/`react-dom`(`^18.3.1`)、`react-router-dom`(`^7.15.0`) 应作为宿主提供的 peer 依赖，避免重复安装导致主题/Context 断裂。
