# examples/rp-gin —— 外部项目集成 ark-iam/sdk 的最小样例

演示**其它 Go 项目**如何仅凭一条 `go get` 就接入 ark-iam 的 RP 侧本地验签能力，
无需引入 IAM 服务端、无需共享它的数据库/Redis。

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

返回当前身份（`user_id` 审计主键 / `tenant_id` / `client_id` / `machine` / `scopes`）。

## 关键点（见 main.go，全部约 80 行）

1. **启动时构建一次** `rp.NewKeys(ctx, issuer)` + `rp.NewVerifier(keys, WithIssuer, WithAudiences, WithLeeway)`，进程内复用。
2. **中间件里本地验签**：`verifier.Verify(ctx, raw)` → `*rp.Identity`；失败一律 401（fail-closed）。
3. **身份注入**：`rp.WithIdentity(ctx, identity)` 放入 request context，handler 用 `rp.IdentityFrom(ctx)` 取回。
4. 业务请求路径**绝不访问 OP**：JWKS 已在后台预取并本地 kid 查表。

## 能力

`github.com/morehao/ark-iam/sdk` 除本地验签（`rp.Verifier`）外还提供：
client_credentials（M2M 令牌自动续期）、introspect（RFC 7662）、revoke（RFC 7009）、
userinfo、back-channel logout 接收端、只读 directory。详见 [`sdk/README.md`](../../sdk/README.md)。
