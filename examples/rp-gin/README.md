# examples/rp-gin —— 外部项目接入 ark-iam 的正式 RP 范本

演示**其它 Go 项目**如何仅凭一条 `go get` 就接入 ark-iam 的 RP 侧本地验签能力，
无需引入 IAM 服务端、无需共享它的数据库/Redis。

## 职责边界（先看这个）

| 层次 | 归属 | 职责 | 是否对外发布 |
|---|---|---|---|
| 验签引擎 `sdk/rp` | ark-iam 发布 | 本地验签：JWT → `rp.Identity`（框架无关，只依赖 stdlib + jwt） | **是** |
| gin 中间件（本文件 `authMiddleware`） | **消费方自写** | 取 Bearer → `Verifier.Verify` → `rp.WithIdentity` 注入 → Next；失败 401 | 否（约 20 行，照抄本范例） |
| 服务端装配 `pkg/appassembly` | ark-iam 内部 | 承载原 `sdk/server` 的装配（DB/Redis/路由编排） | **否**（不再是 `sdk/server`，不再对外发布） |

> 一句话：**ark-iam 只发验签引擎，不发 gin 中间件、也不再发 server 装配。**
> 中间件和“令牌身份 → 本地业务身份”的映射，都是消费方自己的事——本范例示范怎么做。

## 安装

```bash
go get github.com/morehao/ark-iam/sdk@v0.3.0
```

`go.mod`：

```go
require github.com/morehao/ark-iam/sdk v0.3.0
```

> 本目录的 `go.mod` 就**只用 require，不加 replace**——模拟真实外部项目。
> 本地无网络时可临时加 `replace github.com/morehao/ark-iam/sdk => ../../sdk` 后编译验证。

## 运行

```bash
export ARK_IAM_ISSUER="http://localhost:8080"   # 你的 ark-iam（OP）地址
export ARK_IAM_AUDIENCE="my-app"                # 本应用在 IAM 注册的 client_id
go run .
```

用有效 access_token 访问：

```bash
curl -H "Authorization: Bearer <access_token>" http://localhost:8098/api/whoami
```

返回当前身份（`subject_user_id`/`subject_tenant` 为令牌侧口径，`local_user_id`/`local_account`
为消费方映射出的本地业务身份）。

## 关键点（见 main.go）

1. **启动时构建一次** `rp.NewKeys(ctx, issuer)` + `rp.NewVerifier(keys, WithIssuer, WithAudiences, WithLeeway)`，进程内复用。
2. **中间件里本地验签**：`verifier.Verify(ctx, raw)` → `*rp.Identity`；失败一律 401（fail-closed）。
3. **身份注入**：`rp.WithIdentity(ctx, identity)` 放入 request context，handler 用 `rp.IdentityFrom(ctx)` 取回。
4. **业务映射由消费方自加**：`rp.Identity` 里是**令牌侧口径**（string 的 `UserID`/`TenantID`/`PersonID`）。
   把它映射到**本应用自己的本地数据**（本地 BIGINT 主键、本地账号、本应用角色/权限）是消费方的职责，
   SDK 不替你做。见 `main.go` 的 `localUserResolver` 与 `whoami` handler 示范。
5. 业务请求路径**绝不访问 OP**：JWKS 已在后台预取并本地 kid 查表。

## 能力

`github.com/morehao/ark-iam/sdk` 除本地验签（`rp.Verifier`）外还提供：
client_credentials（M2M 令牌自动续期）、introspect（RFC 7662）、revoke（RFC 7009）、
userinfo、back-channel logout 接收端、只读 directory。详见 [`sdk/README.md`](../../sdk/README.md)。
