# P0 Issuer Context / 登出契约

> 状态：**P0 冻结契约（spike 输入）** ｜ 最后更新：2026-11-27
>
> 配套：[tenant-custom-domain-redesign.md](tenant-custom-domain-redesign.md)、[p0-host-binding-contract.md](p0-host-binding-contract.md)
>
> 本文只解决 P0 的 issuer 来源、Gin/OP 上下文、refresh/session 列、登出/end_session 撤销范围；不涉及 S/router/branded/迁移。

## 1. 目标

- 任何 authorize/token/refresh/logout/session 校验都必须绑定**同一个可信 issuer + tenant**。
- OP（chi）与 Gin（`/v1/auth/*`、`/v1/tenant/*`、`/config.json`、`end_session`）不能各自解析 Host 后得到不同 issuer。
- A issuer 的 session/refresh 在 B issuer 必须拒绝；D 不使用 person 兜底。

## 2. Issuer 来源

| 链路 | 来源 |
|---|---|
| `/oidc/*`（provider/chi） | `op.IssuerFromContext(ctx)`；由自定义 `IssuerFromRequest` 写入 |
| `/v1/auth/*`、`/v1/tenant/*`、`/config.json`、Gin `/oidc/*` | `pkg/oidcctx.IssuerFromContext(ctx)`；入口中间件从 Host binding 填一次 |
| `objidentity.Operator` | 可选增加 `Issuer` 作为审计副本；**不参与验签/签发**，唯一真源是 `pkg/oidcctx` |
| `pkg/sso.CreateSession` | 增加 issuer 参数；Redis key `issuer+sid` |
| logout worker | job 带 issuer；不再用启动期静态 issuer |

新增纯 ctx 包 `pkg/oidcctx`（不依赖 gin/identity）：

```go
func WithIssuer(ctx context.Context, issuer string) context.Context
func IssuerFromContext(ctx context.Context) (string, bool)
```

- `pkg/oidcctx` 只做上下文传递，不做鉴权；鉴权在入口中间件。
- **单一真源**：入口中间件解析 Host binding 一次，写入 `*http.Request` context 的 `pkg/oidcctx`；Gin controller 把该 ctx 交给 service；chi 的自定义 `IssuerFromRequest` 从 `r.Context()` 读同一值，不复查库；`op.IssuerFromContext` 只用于 provider 内部。
- `Operator.Issuer` 只是审计副本，不允许参与 token 校验或 issuer 选择。
- Gin 直挂的 `/oidc/login`、`/login/selectTenant`、`/sso-login`、`/logged-out`、`/login/changePassword`、`/registerPerson`、`/createTenant` 也必须读同一 ctx。
- 缺失 issuer → fail-closed，不猜中央。

## 2.1 授权事务与最终签发门禁

- `AuthRequest` 固化 `Issuer`+`TenantID`；选租户/完成/回调/兑换都读它；D 的 Host tenant 是硬边界，不允许“登录后再选其他 tenant”。
- code 消费必须原子：按 issuer 命名空间隔离 code/spent key，或消费前原子比较 issuer；错误 issuer 返回 `invalid_grant`，不消费正确 code。
- 授权完成/发 code、token 兑换、refresh（含轮换宽限）、M2M/API Key 在最终写入前检查 `domain_epoch` + tenant/domain 状态；与 offboard/disable 的权威事务串行，或拒绝旧版本结果。
- 停签验收分两层：**DB 提交后不再签发**（不变量）与**副本观察 ≤5s p99**（可观测性）；不能把异步通知完成当停签。
- `op.IssuerFromContext` 只用于 OP 内部；Gin 入口通过 `pkg/oidcctx` 传同一 issuer；两者不得各自解析 Host。

## 3. 数据载体

| 表/对象 | 字段 | 索引/说明 |
|---|---|---|
| `iam_refresh_token` | `issuer varchar(256)` | 列名按代码实际 `token`；索引 `(issuer, token)` |
| `iam_session`（审计表） | `issuer varchar(256)` | 原 `session_id` 单列唯一改为 `(issuer, session_id)`；删旧 tag，旧库删库重建 |
| `LogoutJob` | `issuer`、`tenant_id` | worker 从 job 取 issuer |
| `LogoutRegistration` | `issuer`、`tenant_id` | SLO 按 issuer 分组 |
| Redis SSO session | key `issuer+sid` | 与审计表分开；不用审计表当会话事实源 |

**旧数据**：D 的存量 session/refresh 无 issuer，一律 fail-closed；接受 D 用户重登。S/平台按现有行为迁移；P0 不改 S。

## 4. 登出入口的范围

| 入口 | Host | 撤销范围 |
|---|---|---|
| `/v1/auth/logout` | D Host | 当前 issuer + 当前 sid/tenant 的 refresh/session；不按 person 全量 |
| `/v1/auth/logout` | 平台 Host | 平台 issuer 内当前用户；保持现有平台语义 |
| `/v1/auth/logoutAll` | D Host | **410**，P0 不做 D 跨 issuer 全局登出 |
| `/v1/auth/logoutAll` | 平台 Host | 平台 issuer 内全租户；不撤 D |
| `/oidc/end_session` | D/平台 Host | 当前 issuer + sid/hint；不跨 issuer |
| offboard | D Host | `(tenant, issuer)` 下全部 refresh/session；异步清理可重试 |

**撤销枚举（P0 实现口径）**：

- offboard 在清 `issuer` 的同一事务保存旧 `(tenant, issuer)` 到可重试撤销记录；
- 用 `iam_refresh_token (tenant_id, issuer, session_id)` 反查 sid 集合，**与创建时登记的 `(tenant, issuer)` 会话索引取并集**；逐 sid `RevokeSession` + 按 `(tenant, issuer)` 撤销 refresh；无 refresh grant 的客户端也必须覆盖；
- SSO Redis session 需补 `tenant/issuer` 维度（key `issuer+sid`、索引 `tenant/issuer`），不能只用 `sso_user_sessions:<person>`；
- `RevokeMemberSessions` 的 person 级 `CrossTenantContext` 全撤**不得**用于 D offboard；
- 清理失败进入重试队列；release 前核验撤销记录完成。

**登出入口范围**：

| 入口 | Host | 范围 |
|---|---|---|
| `/v1/auth/logout` | D | 当前 `(issuer, tenant, sid)`；不按 person 全量 |
| `/v1/auth/logout` | 平台 | 平台 issuer + 当前 sid；D 不受影响 |
| `/v1/auth/logoutAll` | D | 410 |
| `/v1/auth/logoutAll` | 平台 | 平台 issuer 内全租户；D 不受影响；这是一次行为变更（不再 person 级跨 issuer） |
| `/oidc/end_session` | D/平台 | 当前 issuer + 验证后的 sid/hint；无有效 sid 拒绝，不回退 person |
| offboard | 平台动作 | `(tenant, issuer)` 全量；异步可重试 |

**完成判据**：

- 签发门禁：DB 提交后立即停止新签发（authorize/token/refresh 检查 `domain_epoch`/状态）。
- 撤销：DB refresh 标记 + Redis session/登记/通知清理；异步失败可重试；offboard 的“完成”分“已停签”和“撤销最终完成”，不能把异步完成当同步。
- release 前必须确认旧 client/binding disable、refresh/session 清理完成、旧 token 排空、RP 撤信留证；未完成保持 `retired`。

## 5. 关键实现

- `TerminateSession` 从 `pkg/oidcctx`/`op.IssuerFromContext` 取 issuer；签名不接受“只有 clientID”的旧路径。
- `apps/auth/service/svcauth.Logout` 当前是 person 级全局撤销，必须改为当前 issuer；`LogoutAll` 单独实现并区分 D 410/平台全局。
- `pkg/sso.ValidateTokenSession` 无 sid 时不得回退 `HasActiveSession(person)`；D fail-closed。
- `pkg/sso` 的 SLO 登记、`RevokeMemberSessions` 不能直接用于 D offboard（person 级跨租户）。
- `TokenRequestByRefreshToken`、`RevokeToken`、`GetRefreshTokenInfo` 按 `(issuer, token)`/issuer 上下文定位。
- `provider.BuildAuthCallbackURL` 使用请求 issuer，不用启动期静态 issuer。

## 6. 代码落点

| 文件 | 改动 |
|---|---|
| `pkg/oidcctx`（新增） | `WithIssuer`/`IssuerFromContext` |
| `pkg/object/objidentity/operator.go` | `Operator` 增 `Issuer` |
| `apps/auth/router/router.go`、`router/oidc.go`、`middleware/auth_options.go`、`pkg/appassembly` | 入口中间件解析一次写 `pkg/oidcctx`；Gin→chi 桥接；D 路由校验 |
| `apps/auth/service/svcauth/auth.go`、`router/auth.go` | `/logout` 当前 issuer；D `/logoutAll` 410；平台保留全局 |
| `apps/auth/core/oidcop/storage.go`、`persistent_store.go`、`logout_worker.go` | `TerminateSession`、refresh 查询、worker 按 issuer |
| `pkg/sso/sso.go`、`pkg/sso/slo.go`、`pkg/dao/refresh_token.go` | `SSOSessionStore` 的 ValidateSession/SessionAMR/SessionAuthTime/RevokeSession/RevokeSessionsByPersonID/HasActiveSession 增 issuer 入参；Redis key `issuer+sid`；SLO/refresh 按 issuer；关闭 person 回退 |
| `apps/auth/service/svcoidc/provider.go` 的 `RedirectURIVerifier` | 仍按 code-only 查 client；改为请求 issuer + `(issuer, code)`；注释里的“client_id 全局唯一”删除 |
| `apps/auth/service/svcoidc/provider.go`、`controller/ctroidc/oidc.go` | 请求 issuer；callback/end_session |
| `pkg/model/refresh_token.go`、`pkg/model/session.go`、`pkg/model/automigrate.go` | `issuer` 列、`(issuer, token)`、`(issuer, session_id)`；删旧唯一 tag、新库重建 |

## 7. 验收

| ID | 场景 | 通过标准 |
|---|---|---|
| I-1 | 跨 issuer refresh | A issuer refresh 在 B issuer token 端点拒绝；不续签 |
| I-2 | 跨 issuer session | A issuer sid 在 B issuer 拒绝；D 无 sid 不回退 person |
| I-3 | D logout | 只撤当前 issuer/sid；A 登出不影响 B |
| I-4 | D logoutAll | 410；不产生跨 issuer 撤销 |
| I-5 | 平台 logoutAll | 只撤平台 issuer；不撤 D |
| I-6 | offboard | 停签 ≤5s p99；撤销可重试；release 前清理未完成则拒绝 |
| I-7 | end_session | 当前 issuer；不跨 issuer |

## 8. 未决

- `pkg/oidcctx` 与现有 `gincontext`/`identity` 的边界需在实现前做一次 design-guard。
- 平台 Host 的 `/logoutAll` 是否继续覆盖全部平台租户，需产品确认；P0 默认只撤平台 issuer。
- Redis 清理失败的补偿 worker 频率/告警阈值。
