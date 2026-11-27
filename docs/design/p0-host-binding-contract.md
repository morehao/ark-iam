# P0 Host / Binding 契约

> 状态：P0 冻结契约（spike 输入）｜ 最后更新：2026-11-27
>
> 配套：[tenant-custom-domain-redesign.md](tenant-custom-domain-redesign.md)、[p0-tenant-lifecycle-contract.md](p0-tenant-lifecycle-contract.md)、[p0-issuer-context-contract.md](p0-issuer-context-contract.md)
>
> 本文只解决 P0 的 Host 路由、`iam_console_binding`、控制台/API 入口、空库 install 和未知 Host 拒绝；不涉及 S/router/branded/迁移。

## 1. 目标与不变量

- 一个 auth/tenantadmin 后端承载多个 D Host；D 管理台是每个 D Host 的固定配置 SPA；platformadmin 在平台 Host。
- binding 是控制台/API 路由的事实源；一个 Host 可有多行 binding（`host + kind`），表达同源双控制台。
- `iam_console_binding`、平台代建 D client、`iam_domain` 三类跨租户行：**一律在具名 DAO 内显式作用域**（读 `CrossTenantContext`，写 `ExplicitTenantContext(目标租户)`）；binding **不登记** `tenantScopeSkipTables`，service 禁止裸用请求 ctx 直接读写。
- 空库必须能启动并进入 `/install`；未初始化时按部署配置的静态可信平台 Host 放行，不依赖 binding。
- 未知/disabled Host 的协议请求 400/404；不得回退中央 issuer。

## 2. Host × 路由矩阵

| Host kind | `/oidc/*` | `/v1/auth/*` | `/v1/tenant/*` | `/v1/platform/*` | `/v1/rp/*` | `/install/*` | `/config.json` | back-channel |
|---|---|---|---|---|---|---|---|---|
| D Host | 允许（D issuer） | 允许（D issuer） | 允许（D issuer/tenant） | **404** | **404** | **404** | D binding | `/oidc/bc-logout/tenant` |
| 平台 Host | 平台 issuer | 平台 issuer | 404 | 允许 | 允许（平台 issuer） | 仅首装 | 平台 binding | `/oidc/bc-logout/platform` |
| 合并 Host（gateway/dev） | 平台 issuer | 平台 issuer | 允许 | 允许 | 允许 | 仅首装 | 按 Origin/Referer 选 platform/tenant binding | 两个 receiver |
| 中央/tenantadmin Host（P1 现状） | 中央 issuer | 中央 issuer | 允许 | 404 | 404 | 404 | central binding | 中央 |

- 合并 Host 的 binding 是**多行**：`kind=platform` 和 `kind=tenant`（或 central），`host` 相同、`issuer` 相同、`client_id/audience` 不同；`UNIQUE(host, kind) WHERE deleted_at IS NULL`。
- `/config.json`：单 kind Host 直接取该行；合并 Host 按 `Origin/Referer` 匹配 binding 的 `redirect_uri` 来源，匹配不到按平台台处理。
- **alias Host**：矩阵外单独一段——不查 binding；active alias 只做 302。
- **P0 装配**：沿用 `ProcessAll`，对 `/v1/platform/*`、`/v1/rp/*`、`/install/*`、平台 back-channel 做 Host 白名单：只有 `kind=platform` 的 Host 可访问；D Host 404。D 的 `/oidc`、`/v1/auth`、`/v1/tenant` 由同一进程承载。
- **D SPA**：ingress/CDN 按 D Host 托管静态资源；Go 不托管静态页。`frontendLoginURL` 按 Host 派生（`<basePath>/login`）。

## 3. 空库 install 三态

| 状态 | `/install/*` | `/config.json` | 平台 binding |
|---|---|---|---|
| 未初始化 | 仅部署配置静态白名单里的平台 Host 放行 | 返回最小静态引导配置（canonical issuer/client/audience/redirect） | 不存在，允许 install |
| 已初始化 + binding 齐全 | 平台 Host 放行（若保留） | 按 binding | 正常 |
| 已初始化 + binding 缺失 | 拒绝；只读健康告警 | 拒绝/降级只读 | 不硬启动失败；给运维修复动作 |

- 平台 binding/平台 console client 由 install 同批 seed；host 取 `URL(oidc.issuer).Host`；分体部署按实际 host 建多行同 issuer 的 platform binding。
- 启动期只读校验，且以 `seed.IsInitialized` 为前置；未初始化跳过，不阻塞空库安装。

## 4. `iam_console_binding`

| 列 | 类型 | 说明 |
|---|---|---|
| `id` | varchar(36) | UUID v7 |
| `host` | varchar(256) | 归一化 Host |
| `tenant_id` | varchar(36) | 真实租户 ID；平台行=平台 seed 租户 ID |
| `issuer` | varchar(256) | 该 Host 的 issuer |
| `client_id` | varchar(64) | 控制台 client_id |
| `audience` | varchar(64) | 业务 API audience |
| `redirect_uri` | varchar(512) | 控制台回调 |
| `kind` | varchar(16) | `platform` / `tenant` / `central` |
| `status` | varchar(16) | `draft` / `enable` / `disable` |
| `deleted_at` | timestamptz | 软删；释放/重登记用 |

索引：`UNIQUE (host, kind) WHERE deleted_at IS NULL`，登记 `partialUniqueIndexes`。

**作用域**：保留 `tenant_id`，**不加入** `tenantScopeSkipTables`；读取 `CrossTenantContext`，写入 `ExplicitTenantContext(binding.tenant_id)`。

## 5. 生命周期

| 事件 | 动作 |
|---|---|
| 平台 binding | install 引导同批创建平台租户、平台 console client、platform binding，直接 `enable`；缺失只告警/修复，不硬启动失败 |
| D onboarding | `POST /v1/platform/tenants` 同事务建 tenant（provisioning）、primary（draft）、binding（draft） |
| D console client | primary activate 同事务创建，code 固定 `tenant_console`（系统保留名，通用 client 创建不可占用）；`issuer`=D issuer，`tenant_id`=D 租户，回填 `binding.client_id` |
| D 启用 | primary active + client 有效 + binding 字段齐全后，`POST /v1/platform/console-bindings/{bindingID}/enable`；同事务校验 tenant/issuer/client 一致 |
| D 停用 | offboard/release 同事务 `disable` + `deleted_at` 软删；不删历史行 |
| 手动 disable | 可恢复停用（`disable` 后允许 enable），但需审计；不是删除 |
| 同 Host 重登记 | 旧行已软删后**新建** binding 行；不 upsert 复活旧行 |
| 合并 Host | 同一 host 建 platform/tenant 两行；各自 client/audience；`/config.json` 按来源选 |

**D console client 作用域例外**：D console client 用内置 `tenant_admin` 应用，但写作用域必须 `ExplicitTenantContext(targetTenantID)`，`tenant_id == targetTenantID`；这是对现有“client.tenant_id == app.owner_tenant_id”的显式例外，审计/列表按 targetTenantID。

**binding 字段来源**：
- `client_id` = D console client code；
- `audience` = 该 client 的 audience（D/平台分别配置）；
- `redirect_uri` = `<issuer>/admin/callback`（或部署固定值，install/onboarding 配置）；
- `issuer` = 该 Host 的 canonical issuer。

## 6. Host 解析

1. 入口只信任 ingress 的 Host（TrustedProxies）；自定义 `op.IssuerFromRequest` **只做 Host→issuer 映射**，忽略 `Forwarded`/`X-Forwarded-Host`。
2. Gin 中间件先规范化 Host；未初始化走 install 三态；已初始化查 binding/`iam_domain`。
3. **两段式**：先查 binding；未命中查 `iam_domain`：active alias → 302；active primary 无 binding → 拒绝（配置缺失）；其他 → 404。
4. **alias 302**：仅 GET/HEAD；保留 path+query；其他方法 404/405；alias 不返回 `/config.json`；不查 binding。
5. `basePath` = 挂载前缀（空或 `/iam`，不含 `/oidc`）；issuer = `https://<host><basePath>/oidc`；OP 端点 `<basePath>/oidc/*`；登录 SPA `<basePath>/login`；back-channel `<basePath>/oidc/bc-logout/*`。
6. 一个请求只解析一次，结果（host/tenantID/issuer/bindingID/kind/epoch）放进请求上下文；OP、Gin、back-channel 共同消费。
7. 缺失/不可读/版本落后 → fail-closed，不回退中央。

## 7. 代码落点

| 文件 | 改动 |
|---|---|
| `pkg/model/console_binding.go`（新增）、`pkg/dao/console_binding.go`（新增）、`pkg/model/automigrate.go` | 表/DAO/`(host, kind)` partial unique |
| `pkg/appassembly/server.go`、`middleware.go`、`config_endpoint.go` | 按 Host binding 解析 issuer/audience/client；多行 binding；`/v1/platform/*` 等 Host 白名单；预安装三态 |
| `pkg/oidckit/backchannel.go` | 静态 issuer/client 改为 resolver；`/oidc/bc-logout/{platform,tenant}` |
| `apps/auth/controller/ctroidc/oidc.go`、`apps/auth/service/svcoidc/provider.go` | 请求上下文 issuer；`BuildAuthCallbackURL` 用请求 issuer |
| `apps/platformadmin/service/svctenant`、`svcdomain`、`svcapplicationclient` | onboarding 同事务建 binding；D console client 例外；offboard/release 停用 |
| `pkg/seed` | 平台 binding、平台 console client 与平台租户/内置 client 同批引导 |

## 8. 验收

| ID | 场景 | 通过标准 |
|---|---|---|
| H-1 | 平台 Host | `/config.json` 返回平台 binding；platform API 可用；`/v1/tenant/*` 404 |
| H-2 | D Host | `/config.json` 返回 D binding；`/v1/auth/*`、`/v1/tenant/*` 可用；`/v1/platform/*`、`/v1/rp/*` 404 |
| H-3 | alias Host | active alias 无 binding，GET/HEAD 302 保留 path+query；POST 405；不返回 `/config.json` |
| H-4 | binding 生命周期 | draft 不可用；enable 后可用；disable 可恢复；offboard/release 后软删；同 Host 新租户新建行 |
| H-5 | 合并 Host | gateway/dev 同 host 双控制台：`/v1/platform/*` 与 `/v1/tenant/*` 都可用；`/config.json` 按 Origin/Referer 选对 client |
| H-6 | 空库 install | 未初始化静态平台 Host 可进 `/install`；install 后平台 binding/client 可用；缺 binding 只告警 |
| H-7 | back-channel | D Host 只收 D issuer 的 `/oidc/bc-logout/tenant`；平台 Host 只收平台 |

## 9. Design guards

1. **跨租户作用域 guard**：平台 scope 读/改 D binding、D client、`iam_domain` 必须命中；三类表不得进 skip。
2. **alias Host guard**：active alias 无 binding 时 GET/HEAD 302；active primary 无 binding 拒绝；未知 Host 404；alias 不查 binding。
3. **D console client code guard**：code 固定 `tenant_console`，系统保留；必须通过 `IsValidClientCode`；`(issuer, code)` 唯一；同 Host 重登记新建 client。

## 10. 未决（spike 验证）

- 合并 Host 的 `/config.json` Origin/Referer 匹配规则需 S1 实测。
- 平台 binding 的 install 静态字段（client/audience/redirect）需与 seed 配置对齐。
- 手动 disable 是否在 P0 UI 暴露；默认只开放 enable/disable 动作，不开放删除。
