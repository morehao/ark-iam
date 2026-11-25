# SDKS —— ark-iam 对外集成物索引

本页是**外部项目集成 ark-iam** 的入口。ark-iam 提供两套独立发布的「引依赖即用」集成物，
分别覆盖后端与前端，版本线各自独立：

| 集成物 | 语言/生态 | 安装 | 版本线 |
|---|---|---|---|
| **Go SDK** `github.com/morehao/ark-iam/sdk` | Go | `go get github.com/morehao/ark-iam/sdk@v0.4.0` | 子模块 tag `sdk/vX.Y.Z` |
| **前端门面包** `@morehao/ark-iam-react` | React / npm | `pnpm add @morehao/ark-iam-react@^0.2.0` | npm 版本 `X.Y.Z` |

> 两行、两条命令即可接入。两个包互不依赖，可单独使用，也可同时使用（前端登录 + 后端验签）。

### 职责边界表（对外集成面 vs 内部实现）

| 层次 | 归属 | 职责 | 是否对外发布 |
|---|---|---|---|
| **验签引擎** `sdk/rp` | `github.com/morehao/ark-iam/sdk`（Go SDK） | 框架无关的本地验签：JWT → `contract.Identity`；另含 M2M / introspect / revoke / userinfo / 登出 / 只读目录 | **是**（RP 侧接入的核心） |
| **gin 中间件 / 框架胶水** | **消费方自写** | 取 Bearer → `rp.Verifier.Verify` → `rp.WithIdentity` 注入 request ctx → 401；约 20 行，参考 [`examples/rp-gin`](./examples/rp-gin/) | 否（不放进 `sdk`，保持框架无关） |
| **服务端装配** | ark-iam 内部 `pkg/appassembly` | DB/Redis/setting 初始化、域依赖闭合、路由编排（`Mount` 等） | **否**（由原 `sdk/server` 内部化而来，不再作为对外 SDK 发布） |

> 一句话：ark-iam **只对外发验签引擎 `sdk/rp`**；gin 中间件由消费方自写；
> 服务端装配是 ark-iam 自身进程的实现细节（内部 `pkg/appassembly`），**不再有 `sdk/server` 这个对外门面**。

### 后端可 `go get` 的 Go 模块（均在仓库根级）

| 模块 | 用途 | 安装 |
|---|---|---|
| `github.com/morehao/ark-iam/sdk` | **RP 侧 SDK**（`sdk/rp` + 独立契约 module `contract`）：本地验签 / M2M / introspect / revoke / userinfo / 登出 / 目录。**只保留 `rp`**；服务端装配已内部化到 `pkg/appassembly`，不在本模块对外 | `go get github.com/morehao/ark-iam/sdk@v0.4.0` |
| `github.com/morehao/ark-iam/contract` | **契约 module**（最底层，仅 stdlib + jwt）：claim/Identity/logout/目录 DTO/验签接口；由 `sdk` 传递引入，无需直接 get | （传递依赖，无需直接 get） |
| `github.com/morehao/ark-iam/pkg` | **公共层**（config/dbclient/middleware/model/dao/... + 内部装配 `appassembly`）；ark-iam 自身各 app 使用，外部 RP 一般无需直接依赖 | `go get github.com/morehao/ark-iam/pkg@v0.4.0` |

> **分层倒置**：进程入口 → `pkg/appassembly`（内部装配，原 `sdk/server/internal`）→ `apps/*`（auth/platformadmin/tenantadmin 各域实现库）→ `pkg` → `contract`。
> `sdk/rp` 与独立 module `contract` 保持**框架无关**（只依赖标准库 + `jwt/v5`，由 `sdk/scripts/check-deps.sh` 强制）；
> `pkg/appassembly` 是内部装配层，可引入 gin/gorm/redis，但**不对外发布为 SDK**。

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
  当前 `sdk` 为 `v0.4.0`。其中 `sdk` 只保留 `rp`；服务端装配已内部化到 `pkg/appassembly`，**不再有 `sdk/server` 对外门面**。与仓库级 tag（如 `v0.2.1`）并存、互不影响。
- **前端包**：`@morehao/ark-iam-react@X.Y.Z`（npm registry），当前 `0.2.0`。

两套版本线独立演进、独立发布，不要求同号。

---

## 参考样例（非 SDK 依赖路径）

- [`examples/rp-gin/`](./examples/rp-gin/) —— **正式 RP 范本**。独立模块，`require github.com/morehao/ark-iam/sdk v0.4.0`（**不用 replace**），
  演示「引 SDK 依赖接入已有 IAM」的完整 RP 侧接入：`rp.NewVerifier` 本地验签 + 消费方自写 20 行 gin 中间件 +
  `rp.WithIdentity`/`IdentityFrom` 标准身份通道，并示范「令牌身份 → 本地业务身份」映射由消费方自加。

> 原「内嵌 OP」示例 `examples/embedded-iam-demo` 已删除：服务端装配内部化到 `pkg/appassembly`，不再对外发布；
> 前端改由 npm 包 `@morehao/ark-iam-react` 提供，Go 侧不再托管任何前端静态资源。
