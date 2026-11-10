# SDKS —— ark-iam 对外集成物索引

本页是**外部项目集成 ark-iam** 的入口。ark-iam 提供两套独立发布的「引依赖即用」集成物，
分别覆盖后端与前端，版本线各自独立：

| 集成物 | 语言/生态 | 安装 | 版本线 |
|---|---|---|---|
| **Go SDK** `github.com/morehao/ark-iam/sdk` | Go | `go get github.com/morehao/ark-iam/sdk@v0.4.0` | 子模块 tag `sdk/vX.Y.Z` |
| **前端门面包** `@morehao/ark-iam-react` | React / npm | `pnpm add @morehao/ark-iam-react@^0.2.0` | npm 版本 `X.Y.Z` |

> 两行、两条命令即可接入。两个包互不依赖，可单独使用，也可同时使用（前端登录 + 后端验签）。

### 后端两个可 `go get` 的 Go 模块（均在仓库根级）

| 模块 | 用途 | 安装 |
|---|---|---|
| `github.com/morehao/ark-iam/sdk` | **RP 侧 SDK**（`sdk/rp` + 独立契约 module `contract`）：本地验签 / M2M / introspect / revoke / userinfo / 登出 / 目录；另含 **`sdk/server` 门面**：`New(Domains)/Mount/MountConfig`，委托 `apps/*` 各域实现库（业务应用最常用） | `go get github.com/morehao/ark-iam/sdk@v0.4.0` |
| `github.com/morehao/ark-iam/contract` | **契约 module**（最底层，仅 stdlib + jwt）：claim/Identity/logout/目录 DTO/验签接口；由 `sdk` 传递引入，无需直接 get | （传递依赖，无需直接 get） |
| `github.com/morehao/ark-iam/pkg` | **公共层**（config/dbclient/middleware/model/dao/... ），随 `sdk/server`（经 `apps/*`）传递引入，一般无需直接依赖 | `go get github.com/morehao/ark-iam/pkg@v0.4.0` |

> **分层倒置**：`sdk/server` → `apps/*`（auth/platformadmin/tenantadmin 各域实现库）→ `pkg`。原独立 `iam` 模块已删除（其域实现并入 `apps/`，Go 侧不再托管前端静态资源）。`sdk/rp` 与独立 module `contract` 保持**框架无关**（只依赖标准库 + `jwt/v5`，由 `sdk/scripts/check-deps.sh` 强制）；`sdk/server` 是编排层，可引入 gin/gorm/redis。

---

## 后端：Go SDK（本地验签，框架无关）

### 安装

```bash
go get github.com/morehao/ark-iam/sdk@v0.4.0
```

```go
require github.com/morehao/ark-iam/sdk v0.4.0
```

SDK 完全**框架无关**，只依赖 Go 标准库 + `github.com/golang-jwt/jwt/v5`（测试另加 `testify`）。
不引入 gin/gorm/redis/oidc 等，也**不共享 IAM 服务的数据库与 Redis**——验签全程本地完成
（JWKS 预取 + 内存 kid 查表），业务请求路径上绝不访问 OP。由 `sdk/scripts/check-deps.sh` 强制约束依赖白名单。

### 能力

| 能力 | 包路径 | 说明 |
|---|---|---|
| 本地验签（唯一鉴权路径） | `sdk/rp`（`NewVerifier`） | JWT → `Identity`（person/machine、租户、scope、代操作人），fail-closed |
| 机器凭证 M2M | `sdk/rp`（`client_credentials`） | client_credentials 令牌，距 exp 60s 自动续期 |
| introspection | `sdk/rp/introspect` | RFC 7662 |
| 撤销 | `sdk/rp/revoke` | RFC 7009 |
| userinfo | `sdk/rp/userinfo` | 展示用途（勿放鉴权/审计关键路径） |
| 单点登出 | `sdk/rp/logout` | back-channel logout 接收端（jti 去重 + 会话撤销） |
| 只读目录 | `sdk/rp/directory` | TTL 缓存 / ETag 304 / 批量 / single-flight |
| claim 契约 | `contract`（独立 module） | 双端单一事实源、错误分类 |

### 最小用法（本地验签）

```go
keys, _ := rp.NewKeys(ctx, "https://iam.example.com")
v := rp.NewVerifier(keys, rp.WithIssuer("https://iam.example.com"), rp.WithAudiences("my-app"))
identity, err := v.Verify(ctx, rawToken) // 失败一律 401
```

详见 [`sdk/README.md`](./sdk/README.md)（含 Gin 接入约 20 行、M2M、目录、登出各节）。

### 样例

- [`examples/rp-gin/`](./examples/rp-gin/) —— 独立模块，`require github.com/morehao/ark-iam/sdk v0.4.0`（**不用 replace**），
  用 `rp.NewVerifier` 写 Gin 鉴权中间件的开箱照抄样例。

---

## 前端：`@morehao/ark-iam-react`（OIDC 登录门面包）

### 安装

```bash
pnpm add @morehao/ark-iam-react@^0.2.0
# peer（由宿主提供）：
pnpm add antd@^5.22.0 react@^18.3.1 react-dom@^18.3.1 react-router-dom@^7.15.0
```

按 ESM 子路径暴露，可按需引入（`types` / `api` / `auth` / `ui`），只引 `types` 不会拉入 antd。

### 能力

| 子路径 | 作用 | 关键导出 |
|---|---|---|
| `@morehao/ark-iam-react/types` | 纯类型与错误码（零重依赖） | `BizCode`、`*Req`/`*Resp` |
| `@morehao/ark-iam-react/api` | axios 客户端 + 拦截器 | `createApiClient`、`request` |
| `@morehao/ark-iam-react/auth` | OIDC 登录（PKCE）+ 租户切换/登出 | `createAuthProvider`、`useTenantSwitching`、`useLogout` |
| `@morehao/ark-iam-react/ui` | 设计令牌 + 列工厂 + 骨架组件 | `themeConfig`、`tokens`、`AppShell` |

### 最小用法

```tsx
import { createAuthProvider } from '@morehao/ark-iam-react/auth'
const AuthProvider = createAuthProvider({ clientID: 'my_app_web' })
<AuthProvider><App /></AuthProvider>
```

`createAuthProvider` 运行期会先拉 `{base}/config.json` 覆盖默认值（配置优先级：props > config.json > VITE_* > 默认）。

完整接入见 [`docs/integration/react-sdk.md`](./docs/integration/react-sdk.md)。

---

## 版本线

- **Go 模块**（sdk / pkg）：各自子模块 tag（`sdk/vX.Y.Z`、`pkg/vX.Y.Z`，Go 多模块标准，`go get .../<mod>@vX.Y.Z` 依赖前缀对齐目录）。
  当前 `sdk` 为 `v0.4.0`（分层倒置大改：新增 `sdk/server` 门面、废弃原 `iam` 模块）。与仓库级 tag（如 `v0.2.1`）并存、互不影响。
- **前端包**：`@morehao/ark-iam-react@X.Y.Z`（npm registry），当前 `0.2.0`。

两套版本线独立演进、独立发布，不要求同号。

---

## 参考样例（非 SDK 依赖路径）

- [`examples/rp-gin/`](./examples/rp-gin/) —— 独立模块，`require github.com/morehao/ark-iam/sdk v0.4.0`（**不用 replace**），
  演示「引 SDK 依赖接入已有 IAM」的 RP 侧验签接入。

> 原「内嵌 OP」示例 `backend/examples/embedded-iam-demo` 已随分层倒置重构删除：服务端域实现归 `apps/*`，
> 前端改由 npm 包 `@morehao/ark-iam-react` 提供，Go 侧不再托管任何前端静态资源。
