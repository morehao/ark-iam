# 租户级自定义域名（Tenant Custom Domain）实施方案

> 状态：待评审（B 档 · 租户级定制域名，设计稿未实施）
> 涉及：`auth`（登录/OIDC 入口按域名识别租户）、`pkg/iam`（`domain` 表语义重定义与消费）、`login-web`（按域名品牌化/跳租户选择）、`platformadmin`（域名管理页面保留并富化）；顺带移除 `system`（系统配置）冗余模块。
> 结论速览：**`domain` 模块重做保留**（作为被消费的租户域名源），**`system` 模块移除**。

## 1. 背景与目标

### 1.1 现状问题

现有 `domain`（域名管理）模块是**「只录入、不消费」**的纯 CRUD：`DomainEntity{TenantID, Domain, IsVerified}` 在平台管理「域名管理」页配置后，**登录、OIDC、SSO、前端路由均不读取它**。经排查：

- `DomainDao` / `DomainEntity` 的**唯一调用方是 `svcdomain` 自身**（5 个 CRUD 方法）。
- OIDC `redirect_uri` 白名单来自 `application_client.RedirectURIs`；issuer 来自 `config.OIDC.Issuer`；SSO cookie 域名来自 `config.OIDC.SSOCookieDomain()` —— **三者均与 `domain` 表无关**。
- 登录流程为「凭证直登 `/oidc/login` → 多租户则 `/selectTenant` 选租户」，租户提示来自 `?tenant` query（`TenantHint` / `oidcop.TenantHintKey`）。

### 1.2 目标（B 档 · 租户级）

每个租户用自己的域名登录、看到自己品牌，**域名即租户标识**：

1. **按域名识别租户**：登录/OIDC 授权入口根据 `Host` 反查租户域名 → 得到目标租户。
2. **预设租户、跳过选租户**：域名命中单租户时，登录后不再要求 `/selectTenant`，直接进入该租户（用户若不是该租户成员则需提示）。
3. **登录页品牌化**：`login-web` 按当前域名展示租户名/logo。
4. **回调安全前提**：租户域名仅用于**登录入口 + 租户预设**，**不改全局 issuer、不回写 redirect_uri**，因此不破坏现有 OIDC 回调白名单与令牌校验。

> 明确不做（C 档，如需另期）：应用级独立域名、多 issuer 多 OP、redirect_uri 按域名别名校验。

## 2. 设计决策

| 决策点 | 结论 | 理由 |
|---|---|---|
| `domain` 表去留 | **保留并重做** | 它是 B 档「按域名反查租户」的数据源，天然含 `TenantID` 字段，正合适 |
| `system` 表去留 | **移除** | 纯 CRUD、无消费者、未进 seed；误用风险已排除（真正生效的策略在 `application.TenantPolicy`） |
| 域名与 issuer 关系 | 域名 ≠ token `iss` | 一个 OP 单 issuer；域名只做登录层识别，避免波及 `oidcop`/回调 |
| 域名命中校验 | 用户在目标租户有成员身份才预设 | 防越权：杜绝「凭域名进入任意租户」 |
| 域名 DNS 归属 | 平台负责配置 | 平台管理「域名管理」页录入 → 运维把域名 CNAME/A 指向 auth 网关 |

## 3. 数据模型

### 3.1 `DomainEntity` 字段补充

当前（`pkg/iam/model/domain.go`）：

```go
type DomainEntity struct {
    gormdao.BaseEntity
    TenantID   string       // 租户id
    Domain     string       // 域名
    IsVerified bool         // 是否验证
    VerifiedAt sql.NullTime // 验证时间
    CreatedBy, UpdatedBy, DeletedBy string
}
```

新增必要字段（适配 B 档）：

```go
type DomainEntity struct {
    gormdao.BaseEntity
    TenantID   string       `comment:租户id`
    Domain     string       `comment:登录域名(host，不含协议)`
    IsPrimary  bool         `comment:是否该租户主域名(优先级最高的标识域名)`
    IsVerified bool         `comment:是否已验证`
    VerifiedAt sql.NullTime
    Status     string       `comment:enable/disable`          // 软停用不再对外识别
    CreatedBy, UpdatedBy, DeletedBy string
}
```

- `Domain` 建议**强制小写、不含 scheme/path**（统一存 host），录入时归一化。
- 校验标识：新增 `CheckDomainRecord` 能力（见 §5），`IsVerified` 仅作展示，不阻塞录入。

### 3.2 迁移

- `automigrate` 已有 `&DomainEntity{}`，字段新增走 GORM AutoMigrate 加列即可，**无需删表**。
- 已有域名记录 `IsPrimary=false`、`Status=enable` 兼容默认值。

## 4. 后端改造

### 4.1 新增 `pkg/iam/svcdomain` 或复用 dao —— 按域名反查租户

在 `pkg/iam/dao/domain.go` 补充查询（供 auth 侧使用，避免 auth 依赖 platformadmin 的 `svcdomain`）：

```go
// GetPublicByDomain 精确匹配未停用且已验证(或未校验但启用)的域名，返回其租户id。
// 供 auth 登录入口中间件调用。
func (d *DomainDao) GetPublicByDomain(ctx context.Context, domain string) (*model.DomainEntity, error)
```

> 依赖方向注意：`auth` 属于 `pkg` 消费者，查询逻辑放 `pkg/iam` 而不是 `platformadmin/internal`，保证 auth 不反向依赖 platformadmin。

### 4.2 新增域名中间件 `apps/auth/internal/middleware/domain.go`

```go
// DomainTenantHint：根据 Host 反查租户域名，命中则写入 tenant hint。
// 幂等：已有 ?tenant 或已有过 hint 时跳过，显式 query 优先。
func DomainTenantHint() gin.HandlerFunc {
    return func(ctx *gin.Context) {
        host := ExtractHost(ctx.Request.Host)
        entity, err := dao.NewDomainDao().GetPublicByDomain(ctx, host)
        if err == nil && entity != nil {
            ctx.Set(ginKeyTenantHint, entity.TenantID) // 复用现有 hint 暂存键
        }
        ctx.Next()
    }
}
```

- 复用现有 `ginKeyTenantHint` + `CarryOIDCHints` 搬运机制（见 `internal/middleware/oidc.go`），**零改透明传管道**。
- 依赖 web 框架 Host 头可靠性：生产务必在网关层规范化 `Host`，并校验 Host 白名单防 Host 头注入（见 §6 安全）。

### 4.3 注册到路由（`apps/auth/internal/router/oidc.go`）

在 `/oidc/login` 与 `/oidc/authorize` 前挂载（两处均需域名识别）：

```go
oidcGroup.POST("/login", middleware.LoginRateLimit(), middleware.DomainTenantHint(), ctr.Login)
oidcGroup.GET("/authorize",
    middleware.DomainTenantHint(),   // 在 TenantHint 之前或之后按优先级实现
    middleware.TenantHint(),
    middleware.OIDCSilentAuth(...), oidcHandler)
```

**优先级约定**：显式 `?tenant=` 应覆盖 host 推断，故 `TenantHint` 设置在 `DomainTenantHint` **之后**（或单中间件内 `已有则跳过`），保证显式优先。

### 4.4 登录服务支持「域名预设租户」（`svcoidc/auth.go`）

现有 `CompleteLogin` 在「多租户且无合法 hint」时返回 `RequiresTenantSelection`。改造点：

- `authRequest.GetTenantID()`（来自 hint）在域名命中单租户时已有值 → 走现有 `resolvedTenant` 逻辑**自动完成**，无需额外改动；
- 唯一补充：命中域名但**用户不属于该租户**时，`AuthenticatePassword` 返回的 `tenants` 不含该租户 → 现有逻辑会回退到「多租户选租户」或「单租户用 user.TenantID」。
  - 建议新增一个空域名的明确结果标记：域名租户 != 用户任何所属租户时，返回专用错误（提示「该域名对应的租户与账号不匹配」），而非静默回退其他租户（防域名误配登录进错租户）。

### 4.5 platformadmin `domain` 管理富化（页面保留）

- 保留现有 CRUD（`svcdomain`/`ctrdomain`/`dtodomain`/`router/domain.go` 不改）。
- DTO 补齐 `isPrimary`/`status` 字段，供前端管理。
- **不做** 域名验证强流程：`IsVerified` 由运维手动置位即可（B 档够用）。

### 4.6 移除 `system` 模块

- 删文件：
  - `apps/platformadmin/internal/service/svctenant/system.go`
  - `apps/platformadmin/internal/controller/ctrtenant/system.go`
  - `apps/platformadmin/internal/dto/dtotenant/system_request.go`、`system_response.go`
  - `pkg/iam/{model/system.go, dao/system.go}`
  - 测试 `apps/platformadmin/internal/service/svctenant/tenant_scope_test.go` 的 `TestSystemDetailRejectsCrossTenantEntity`
- 删注册：`router/tenant.go` 的 `systemRouter(...)` 函数、`router/router.go` 的 `systemRouter(groups)` 调用。
- `automigrate`：删 `&SystemEntity{}`。
- 前端：删 `pages/system`、`App.tsx` 的 import/菜单/路由；`packages/api` 与 `packages/types` 中 system 相关导出。
- `svctenant`/`ctrtenant`/`dtotenant` 目录保留（tenant/log 模块仍在），**只删 system 相关文件**。

## 5. 域名归属校验（可选但推荐）

平台可开通「域名声明」接口，供租户管理员发起域名所有权验证：

- 新增 `POST /v1/platform/domains/{id}/verify`（动作子路径，R2 规范）。
- 校验方式：返回一段随机 token，要求 `Host` 或根路径返回该 token，平台探测比对。
- B 档 MVP 可**延迟到二期**，先用 `IsVerified` 手置 + admin 人工复核。

## 6. 安全考虑

| 风险 | 缓解 |
|---|---|
| Host 头注入 → 错配租户 | 网关层校验 Host 白名单/规范 Host；中间件忽略 scheme 端口，仅比对归一化 host |
| 域名命中非用户所属租户 → 越权 | §4.4 明确回退策略：不静默落入其他租户，域名租户不符即失败并提示 |
| 域名停用/被夺 → 仍被识别 | 中间件仅取 `Status=enable` 的记录；删除域名即时生效（N+1 可加缓存并短 TTL 失效） |
| 明文 HTTP 登录 | 依赖部署层强制 HTTPS（`CookieSecure` true），域名仅作路由标识不改变该约束 |
| 性能 | 中间件按 host 查询可加本地短 TTL 缓存（如 60s）降频；`GetPublicByDomain` 加唯一索引 `(domain)` |

索引：`domain` 列加唯一索引（唯一、规范化存储前提下）。

## 7. 前端改造（login-web）

- 新增/复用接口：`GET /v1/auth/tenants/by-domain`（返回按当前域名的租户元信息 name/logo，可选）或复用 `me/tenants` 过滤。
- 登录页 `LoginPage.tsx`：
  - 启动时取当前 host → 请求域名租户信息；
  - 命中且用户是该租户成员 → 显式展示该租户品牌，登录后**跳过 `/selectTenant`**；
  - 未命中域名 → 保持现有「选租户下拉」流程向后兼容。
- 平台 `domain` 管理页 `pages/domain`：表单补 `isPrimary`/`status` 字段展示。

## 8. 涉及文件清单

### 后端 auth
| 文件 | 变更 |
|---|---|
| `internal/middleware/domain.go` | **新增** `DomainTenantHint` 中间件 |
| `internal/router/oidc.go` | `login`/`authorize` 挂载域名中间件 |
| `internal/service/svcoidc/auth.go` | 域名租户 → resolvedTenant 自动完成；域名不符错误回退 |
| `internal/service/svcoidc/*_test.go` | 补域名命中/不符单测 |

### 后端 pkg
| 文件 | 变更 |
|---|---|
| `pkg/iam/model/domain.go` | 增 `IsPrimary/Status` 字段 |
| `pkg/iam/dao/domain.go` | 增 `GetPublicByDomain` |
| `pkg/iam/model/automigrate.go` | 保留 `DomainEntity`；删 `SystemEntity` |
| `pkg/iam/model/system.go`、`pkg/iam/dao/system.go` | **删除** |

### 后端 platformadmin
| 文件 | 变更 |
|---|---|
| `internal/dto/dtodomain/*` | 补 `isPrimary/status` 字段 |
| `internal/dto/dtotenant/system_request.go`、`system_response.go` | **删除** |
| `internal/service/svctenant/system.go`、`internal/controller/ctrtenant/system.go` | **删除** |
| `internal/service/svctenant/tenant_scope_test.go` | 删 system 用例 |
| `internal/router/tenant.go`、`router/router.go` | 删 `systemRouter` |

### 前端
| 文件 | 变更 |
|---|---|
| `apps/login-web/src/pages/LoginPage.tsx` | 域名品牌化 + 跳选租户 |
| `apps/platform-admin-web/src/pages/domain/*` | 富化表单字段 |
| `apps/platform-admin-web/src/pages/system/*`、`App.tsx` | **删除** system 页面/导航 |
| `packages/api/src/resources/platform.ts`、`packages/types/src/platform.ts` | 删 system 导出；补 domain 新字段 |

## 9. 验证

1. `make build APP=auth` / `make build APP=platformadmin` 编译通过。
2. `go test ... ./pkg/iam/... ./apps/auth/internal/... ./apps/platformadmin/internal/...` 全绿。
3. `make swag APP=auth`、`make swag APP=platformadmin` 重新生成 API 文档。
4. `pnpm build` 前端通过。
5. 手动验收：
   - 配置租户 A 域名 `acme.portal.com` → `/oidc/login` 携带该 Host → 成功预设租户 A、登录后直达；
   - 用户不属于租户 A 域名 → 得到明确错误，不落入其他租户；
   - 全仓 `grep` 确认无 `SystemEntity`/`createSystemConfig`/`DomainItem` 之外残留 System 引用。

## 10. 里程碑拆期（推荐）

| 阶段 | 内容 |
|---|---|
| P0（本次） | 移除 `system`；`domain` 表加字段 + `GetPublicByDomain`；`DomainTenantHint` 中间件挂到 `login`；登录服务域名预设租户；login-web 品牌化 |
| P1 | `authorize` 也挂域名识别 + 域名不符错误回退完善 + 短 TTL 缓存 |
| P2（可选） | 域名所有权验证接口 `/verify`、多 issuer/应用级域名（C 档，需重估 oidcop 影响） |

## 11. 结论速览

- **`domain`：重做保留**——它是 B 档租户域名的数据源与登录识别入口，物尽其用。
- **`system`：移除**——纯摆设 CRUD，删掉不损失能力，降低维护面。
