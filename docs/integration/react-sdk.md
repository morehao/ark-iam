# 前端接入：`@morehao/ark-iam-react`

本文档面向**其它 React 应用**，说明如何通过 npm 包 `@morehao/ark-iam-react` 接入 ark-iam 的
OIDC 登录（PKCE）与租户能力。**不搬目录**：npm 包身份由 `package.json` 的 `name` + registry 决定，
与仓库目录无关，消费者 `pnpm add` 即可。

> 包源在 ark-iam 仓库 `frontend/packages/react`，发布名 `@morehao/ark-iam-react`，当前 `0.2.0`。

## 1. 安装

```bash
pnpm add @morehao/ark-iam-react@^0.2.0
# peer 依赖由宿主提供：
pnpm add antd@^5.22.0 react@^18.3.1 react-dom@^18.3.1 react-router-dom@^7.15.0
```

按 ESM 子路径暴露，可按需引入：

| 子路径 | 作用 |
|---|---|
| `@morehao/ark-iam-react/types` | 纯类型与错误码（零重依赖，不含 antd/react） |
| `@morehao/ark-iam-react/api` | axios 客户端 + 拦截器 |
| `@morehao/ark-iam-react/auth` | OIDC 登录（PKCE）+ 租户切换/登出钩子 |
| `@morehao/ark-iam-react/ui` | 设计令牌 + 列表页列工厂 + 骨架组件 |

> 追求体积用子路径（`import { BizCode } from '@morehao/ark-iam-react/types'` 不会拉入 antd）；
> 图省事用根入口 `@morehao/ark-iam-react` 全量 re-export。

## 2. AuthProvider（登录门面）

```tsx
import { createAuthProvider } from '@morehao/ark-iam-react/auth'

// 工厂：返回一个 AuthProvider 组件。
const AuthProvider = createAuthProvider({ clientID: 'my_app_web' })

export default function Root() {
  return (
    <AuthProvider>
      <App />
    </AuthProvider>
  )
}
```

`createAuthProvider` 参数：`clientID / clientSecret / issuer / redirectPath / postLogoutRedirectPath /
getExtraQueryParams / useRuntimeConfig / runtimeConfigBase`。

**配置优先级：props > `/config.json` > `VITE_*` > 相对默认值。**
运行期默认先拉 `{base}/config.json` 覆盖默认值——这样同一份构建产物可部署到不同环境
（详见仓库测试报告 §9「登录页运行时 config.json 自适应 base」）。

## 3. 运行时 `config.json`

在应用 `public/`（或部署根目录）放一份运行期配置，供 `AuthProvider` 拉取覆盖：

```json
{
  "issuer": "https://iam.example.com",
  "clientID": "my_app_web",
  "redirectPath": "/auth/callback",
  "postLogoutRedirectPath": "/"
}
```

字段名与 `createAuthProvider` 参数对齐；缺省字段回落到 `VITE_*` 与默认值。

## 4. OIDC 回调 `redirect_uri`

- 前端登录回跳路径默认 `/auth/callback`（`redirectPath`），请在 IAM 的 client 注册里把
  `http://<your-host>/auth/callback` 加入允许的 `redirect_uri`。
- 登出回跳默认 `/`（`postLogoutRedirectPath`），同样需在白名单中。

## 5. 请求客户端与 token 注入

```ts
import { createApiClient } from '@morehao/ark-iam-react/api'

const api = createApiClient({
  baseURL: 'https://iam.example.com/v1', // 或同源 /v1 用默认单例 request
  timeout: 15000,
})
```

`createApiClient` 返回 axios 实例，内置 401 自动跳登录、业务码处理、`silent` 抑制提示等拦截逻辑，
自动完成 access_token 注入与过期处理。同源 `/v1` 场景可直接用默认单例 `request`。

## 6. 后端配合

前端拿到 access_token 后，后端用 ark-iam Go SDK 本地验签即可（不共享 IAM 库）：

```bash
go get github.com/morehao/ark-iam/sdk@v0.3.0
```

见 [`../../sdk/README.md`](../../sdk/README.md) 与样例 [`../../examples/rp-gin/`](../../examples/rp-gin/)，
以及根级索引 [`../../SDKS.md`](../../SDKS.md)。

## 7. 主题与组件（可选）

```tsx
import { ConfigProvider } from 'antd'
import zhCN from 'antd/locale/zh_CN'
import { themeConfig } from '@morehao/ark-iam-react/ui'

<ConfigProvider locale={zhCN} theme={themeConfig}><App /></ConfigProvider>
```

或用 `AppShell`（内部已包好 `ConfigProvider` + `themeConfig` + antd `App` 上下文）。
列表页可用 `actionColumn` / `nameColumn` / `tableScrollX` 等列工厂，保持与 ark-iam 一致的交互与视觉。
