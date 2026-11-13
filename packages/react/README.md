# @morehao/ark-iam-react

GoArk IAM 前端门面包（React + antd 5）。**对外只发布这一个包**，聚合了内部 `@ark-iam/{types,api,auth,ui}` 的能力，代码与类型均已打进自身 `dist`，安装本包即可使用全部能力。

提供四类能力（按 ESM 子路径暴露，可按需引入）：

| 子路径 | 作用 | 关键导出 |
|---|---|---|
| `@morehao/ark-iam-react/types` | 纯类型与错误码枚举（**零重依赖**，不含 antd/react） | `BizCode`、各类 `*Req`/`*Resp` 类型 |
| `@morehao/ark-iam-react/api` | 业务 API 客户端（axios 实例 + 拦截器） | `createApiClient`、默认 `request`、`setUserProvider`、`setSessionExpiredHandler` |
| `@morehao/ark-iam-react/auth` | OIDC 登录（PKCE）+ 租户切换 / 登出钩子 | `createAuthProvider`、`fetchRuntimeConfig`、`useTenantSwitching`、`useLogout`、`useAuthGuard` |
| `@morehao/ark-iam-react/ui` | 设计令牌 + 列表页列工厂 + 骨架组件 | `themeConfig`、`tokens`、`actionColumn`、`tableScrollX`、`AppShell`、`MainLayout` 等 |
| `@morehao/ark-iam-react` | 上面四层全部 re-export（一把梭） | 全部 |

## 安装

```bash
pnpm add @morehao/ark-iam-react
# 以下为 peer，由宿主提供（只引 @morehao/ark-iam-react/types 时可不必装 antd）：
pnpm add antd@^5.22.0 react@^18.3.1 react-dom@^18.3.1 react-router-dom@^7.15.0
```

> 追求体积请用子路径（如只 `import { BizCode } from '@morehao/ark-iam-react/types'` 不会拉入 antd）；图省事才从根入口 `@morehao/ark-iam-react` 全量引入。

## 主题

```tsx
import { ConfigProvider } from 'antd'
import zhCN from 'antd/locale/zh_CN'
import { themeConfig } from '@morehao/ark-iam-react/ui'

<ConfigProvider locale={zhCN} theme={themeConfig}>
  <App />
</ConfigProvider>
```

或直接用 `AppShell`（内部已包好 `ConfigProvider` + `themeConfig` + antd `App` 上下文）。

## 登录

```tsx
import { createAuthProvider } from '@morehao/ark-iam-react/auth'

// 工厂：返回一个 AuthProvider 组件。运行期会先拉 {base}/config.json 覆盖默认值。
const AuthProvider = createAuthProvider({ clientID: 'my_app_web' })

<AuthProvider><App /></AuthProvider>
```

`createAuthProvider` 参数：`clientID / clientSecret / issuer / redirectPath / postLogoutRedirectPath / getExtraQueryParams / useRuntimeConfig / runtimeConfigBase`。配置优先级：**props > `/config.json` > `VITE_*` > 相对默认值**。

## 请求

```ts
import { createApiClient } from '@morehao/ark-iam-react/api'
const api = createApiClient({ baseURL: 'https://iam.example.com/v1', timeout: 15000 })
```

`createApiClient({ baseURL, timeout, headers })` 返回 axios 实例，内置 401 自动跳登录、业务码、`silent` 抑制提示等拦截逻辑；默认单例 `request` 供同源 `/v1` 场景使用。

## 列表页操作列（最小示例）

```tsx
import { actionColumn, nameColumn, tableScrollX } from '@morehao/ark-iam-react/ui'
import { Table } from 'antd'

const columns = [
  nameColumn({ dataIndex: 'name', onClick: (r) => openDetail(r) }),
  actionColumn({
    max: 2,
    actions: (r) => [
      { key: 'edit', label: '编辑', onClick: () => handleEdit(r) },
      { key: 'del', label: '删除', danger: true, confirm: '确认删除？', onClick: () => handleDelete(r) },
    ],
  }),
]
<Table tableLayout="fixed" scroll={{ x: tableScrollX(columns) }} columns={columns} dataSource={rows} rowKey="id" />
```

完整接入说明见 `docs/design/application-integration-guide.md` §12「前端组件接入」；视觉规范见仓库根 `DESIGN.md`。

## peerDependencies 口径

UI 框架层（`react` / `react-dom` / `antd`(`^5.22`) / `@ant-design/icons`(`^5.4`) / `react-router-dom`(`^7.15`)）作为 peer 由宿主统一版本，避免重复安装导致主题/Context 断裂；纯工具依赖（`axios` 等）由本包 dependencies 带入。
