# 租户域名具体组合示例

> 配套设计：[tenant-custom-domain-redesign.md](tenant-custom-domain-redesign.md)（P0 只做 D；S/branded 标注为 P1/P2）；细则以 [p0-host-binding-contract.md](p0-host-binding-contract.md) 等三份契约为准
> 状态：配套 P0 冻结版（spike 输入）｜ 最后更新：2026-11-27
>
> 本文只给**具体域名组合**，用确定主机名说明「登录在哪里、`iss` 是什么、cookie 落在哪、哪些应用共享登录」。默认 **D 用 host-only cookie**；只有拆分 `login.<domain>` 与 `auth.<domain>` 时才用父域 cookie，并需安全评审。
>
> P0 消费 `usage=login/both` 的 primary/alias；`usage=email` 行可 active，但 **P0 无公开发现、不路由、不 302、不签发**，P1 router 才消费。示例表中的 `target` 是 API/展示用 host，内部存 `target_domain_id`。

## 0. 示例与阶段

| 示例 | 阶段 | 说明 |
|---|---|---|
| Acme：primary + alias + 多应用不同主域 | **P0** | 新建 D；多应用不同主域；1 个 issuer |
| Initech：一个租户多应用不同主域 | **P0** | 同一 D 登录域，三个 callback |
| Globex：primary + branded | **P1/P2** | branded 不在 P0；品牌间不共享 SSO |
| Umbrella：S tier | **P1** | P0 不改 S；中央登录/发现 |
| A+B hybrid | **P1** | 依赖 S/router |
| S→D 迁移 | **P1/P0.5** | P0 不做迁移；客户必须时单列 P0.5 |
| 域名退役 | **P0（alias + offboard primary）/ P1（换主域）** | alias 只停 302；offboard primary 退役；换主域是 P1 |
| 每应用一个 issuer | 非目标 | 不做；用 alias 或独立 tenant |

## 1. 单品牌 + 别名 + 多应用不同主域（Acme）

**租户**：Acme ｜ `isolation_mode=dedicated`

| 项 | 值 |
|---|---|
| primary（active 行） | `login.acme.com` |
| `issuer` | `https://login.acme.com/oidc` |
| 登录页 | `https://login.acme.com` |
| SSO cookie | **host-only `login.acme.com`**（默认）；只有拆 `auth.acme.com` 才用 `.acme.com` |

**域名行**

| domain | role | usage | target_domain | issuer | 行为 |
|---|---|---|---|---|---|
| `login.acme.com` | primary | login | — | `https://login.acme.com/oidc` | 登录/JWKS/discovery |
| `sso.acme.com` | alias | login | `login.acme.com` | — | 302 到 `login.acme.com` |
| `id.acme.com` | alias | login | `login.acme.com` | — | 302 到 `login.acme.com` |
| `acme.com` | alias | email | `login.acme.com` | — | P0 可 active 但不消费；P1 router |
| `acme.io` | alias | email | `login.acme.com` | — | P0 可 active 但不消费 |

**应用与回调**

| 应用 | 应用域 | callback | issuer |
|---|---|---|---|
| CRM | `crm.acme.com` | `https://crm.acme.com/auth/callback` | `https://login.acme.com/oidc` |
| HR | `hr.acme.io` | `https://hr.acme.io/auth/callback` | 同上 |
| Billing | `billing.vendor.com` | `https://billing.vendor.com/oidc/callback` | 同上 |

**实际发生什么**

1. 应用 302 到 `https://login.acme.com/oidc/authorize?client_id=crm&redirect_uri=https://crm.acme.com/auth/callback...`。
2. 用户在 `login.acme.com` 登录；cookie host-only 落 `login.acme.com`。
3. 回跳 `crm.acme.com`；应用建自己的会话；token `iss=https://login.acme.com/oidc`；token 端点 `/oidc/oauth/token`。
4. 用户再访问 `https://hr.acme.io` → 顶层跳 `login.acme.com`；OP 已有 cookie，不再输密码；HR 会话在 `hr.acme.io`，与 CRM 独立。
5. `billing.vendor.com` 不同注册域，不会收到 OP cookie；同样顶层跳 `login.acme.com`，回跳后用自己的会话；静默续期用 refresh token。
6. `sso.acme.com` 只 302，不签 token、不增加 issuer；它不是 issuer，只是浏览器入口；alias 不需要 console binding；仍需该域 TLS。

**结果**：1 个 issuer、3 个应用不同主域、共享 OP 登录；应用会话独立；别名不增加 issuer。

## 2. 多品牌：primary + branded（Globex，P1/P2）

**租户**：Globex ｜ `isolation_mode=dedicated`

| 项 | 值 |
|---|---|
| primary（active 行） | `login.globex.com`（集团品牌） |
| primary issuer | `https://login.globex.com/oidc` |
| branded | `login.globex-crm.com`、`login.globex-hr.io` |

**域名行**

| domain | role | usage | target_domain | issuer | cookie |
|---|---|---|---|---|---|
| `login.globex.com` | primary | login | — | `https://login.globex.com/oidc` | host-only `login.globex.com` |
| `sso.globex.com` | alias | login | `login.globex.com` | — | — |
| `login.globex-crm.com` | branded（P2） | login | — | `https://login.globex-crm.com/oidc` | host-only |
| `login.globex-hr.io` | branded（P2） | login | — | `https://login.globex-hr.io/oidc` | host-only |
| `globex.com` | alias | email | `login.globex.com` | — | — |
| `globex-crm.com` | alias | email | `login.globex-crm.com` | — | — |
| `globex-hr.io` | alias | email | `login.globex-hr.io` | — | — |

**应用**

| 应用 | 应用域 | callback | issuer |
|---|---|---|---|
| 集团门户 | `app.globex.com` | `https://app.globex.com/callback` | `https://login.globex.com/oidc` |
| CRM | `crm.globex-crm.com` | `https://crm.globex-crm.com/callback` | `https://login.globex-crm.com/oidc` |
| HR | `hr.globex-hr.io` | `https://hr.globex-hr.io/callback` | `https://login.globex-hr.io/oidc` |

**结果**

- 三个品牌各自登录、各自 cookie、各自 `iss`；**品牌之间不共享 SSO**。
- `carol@globex-crm.com` 的邮箱域行 `target_domain=login.globex-crm.com`；router 可明确路由到 CRM issuer，不靠猜测。
- 同一用户进 CRM 和 HR 要分别登录；要共享登录改用 `primary + alias`，所有应用指向 `login.globex.com`。
- 每个 branded 域需要自己的 TLS/密钥（P2）。

## 3. 一个租户多应用、不同主域、共享登录（Initech，你的场景）

**租户**：Initech ｜ `isolation_mode=dedicated`

| 项 | 值 |
|---|---|
| primary（active 行） | `login.initech.com` |
| `issuer` | `https://login.initech.com/oidc` |
| alias | `id.initech.com`（target `login.initech.com`） |
| email | `initech.com`（target `login.initech.com`；P0 可 active 但不消费） |
| 登录页 | `https://login.initech.com` |
| cookie | **host-only `login.initech.com`** |

**应用**

| 应用 | 应用域 | callback | issuer |
|---|---|---|---|
| CRM | `crm.initech.com` | `https://crm.initech.com/auth/callback` | `https://login.initech.com/oidc` |
| HR | `hr.initech.io` | `https://hr.initech.io/auth/callback` | 同上 |
| Payroll | `payroll.othervendor.com` | `https://payroll.othervendor.com/oidc/callback` | 同上 |

**结果**

- 三个应用主域分别是 `initech.com`、`initech.io`、`othervendor.com`，但 IAM 登录域只有一个 `login.initech.com`。
- 三个应用都注册 `redirect_uri`；都跳 `login.initech.com`；用户登录一次；各应用会话独立。
- 不存在 `app_id` 维度；`iam_domain` 只有 Initech 的 `login.initech.com`/`id.initech.com`/`initech.com`。
- 这是「一个租户订阅多个应用、每个应用不同主域」的推荐默认组合。

## 4. 长尾共享 tier（Umbrella，S，P1）

**租户**：Umbrella ｜ `isolation_mode=shared`

| 项 | 值 |
|---|---|
| primary（active 行） | 空（S） |
| issuer | `https://auth.platform.com/oidc`（平台） |
| 登录页 | `https://auth.platform.com/login` |
| router | `https://auth.platform.com/oidc/tenant-route` |
| cookie | **host-only `auth.platform.com`**（默认）；拆 `login.platform.com`/`auth.platform.com` 才用 `.platform.com` |
| email | `umbrella.com`（role alias、usage email、target 空=中央） |

**应用**

| 应用 | 应用域 | callback | issuer |
|---|---|---|---|
| 门户 | `app.umbrella.com` | `https://app.umbrella.com/callback` | `https://auth.platform.com/oidc` |
| 工单 | `tickets.umbrella.com` | `https://tickets.umbrella.com/callback` | 同上 |

**登录路径**

1. 用户打开 `https://auth.platform.com/login`，或从 `login.platform.com` 的 landing 进入。
2. 输入 `bob@umbrella.com`，或应用带 `organization=umbrella`/`domain_hint=umbrella.com`。
3. 中央登录；cookie host-only `auth.platform.com`；`iss=https://auth.platform.com/oidc`。
4. 回跳应用；应用会话独立。
5. 另一个 S 租户（如 `acme-lite.com`）的用户在同一中央登录，可以共享中央 SSO（同 issuer）。

> `login.platform.com` 只作为可选 landing/router 页；它不是 issuer，也不是协议端点。若产品要拆登录 UI 与协议 host，必须固定端点、CORS、cookie/CSRF 契约。

## 5. A+B 混合 + router（Acme/D + Umbrella/S，P1）

平台 router：`https://auth.platform.com/oidc/tenant-route`；`login.platform.com` 仅可选 landing。

| 用户输入 | router 返回 | 后续 | `iss` | cookie |
|---|---|---|---|---|
| `alice@acme.com` | `{mode:D, issuer:https://login.acme.com/oidc, loginUrl:...}` | 打开应用/租户入口，由 RP 发起 authorize | `https://login.acme.com/oidc` | host-only `login.acme.com` |
| `bob@umbrella.com` | `{mode:S, issuer:https://auth.platform.com/oidc, organization:umbrella}` | 中央登录，由 RP 发起 authorize | `https://auth.platform.com/oidc` | host-only `auth.platform.com` |
| `carol@globex-crm.com` | `{mode:D, issuer:https://login.globex-crm.com/oidc, loginUrl:...}`（P2 branded） | 打开 CRM 登录/应用入口 | `https://login.globex-crm.com/oidc` | host-only `login.globex-crm.com` |
| `unknown@nowhere.test` | `{matched:false}` | 通用入口/租户选择 | 中央 | — |

**结果**：router 只返回发现 JSON，**不 302 到 authorize、不拼 `redirect_uri`**；由 RP 创建带 state/nonce/PKCE 的登录事务。D 用户在租户 issuer 登录，中央不签 token；S 用户在中央登录。

## 6. alias 与 branded 的域名行对比（同一租户 Acme）

| domain | role | usage | target_domain | issuer | cookie | 登录行为 | 与 primary 共享 SSO |
|---|---|---|---|---|---|---|---|
| `login.acme.com` | primary | login | — | `https://login.acme.com/oidc` | host-only | 直接登录 | — |
| `id.acme.com` | alias | login | `login.acme.com` | — | — | 302 到 `login.acme.com` | 是 |
| `login.acme-crm.com` | branded（P2） | login | — | `https://login.acme-crm.com/oidc` | host-only | 自己登录 | **否** |

**区别**

- alias：不签 token、不建 issuer；只需要该域 TLS + 302；`iss`/cookie 仍是 primary。
- branded：自己的 issuer/cookie/session；需要自己的证书/密钥；与 primary 无跨域 SSO。
- 只想让 `id.acme.com` 也能登录 → alias；要让 CRM 品牌有独立登录/会话 → branded。

## 7. S→D 迁移（Umbrella，P1/P0.5）

**迁移前（S）**

| 项 | 值 |
|---|---|
| `isolation_mode` | `shared` |
| `issuer` | `https://auth.platform.com/oidc` |
| primary（active 行） | 空（S） |
| 域名行 | `umbrella.com`（role alias、usage email、target 空=中央） |
| 应用 callback | `https://app.umbrella.com/callback` |
| cookie | host-only `auth.platform.com` |

**迁移后（D）**

| 项 | 值 |
|---|---|
| `isolation_mode` | `dedicated` |
| `issuer` | `https://login.umbrella.com/oidc` |
| primary（active 行） | `login.umbrella.com` |
| 域名行 | `login.umbrella.com`（primary/login）、`umbrella.com`（alias/email、target `login.umbrella.com`） |
| 应用 callback | 不变 `https://app.umbrella.com/callback` |
| cookie | host-only `login.umbrella.com` |

**迁移影响（P1/P0.5 硬切）**

- 应用必须把 issuer 从中央改为 `https://login.umbrella.com/oidc`，否则 token 校验失败。
- 停止旧签发/续期 → 撤销 refresh/session → 切 RP 信任 → 用户重登。
- 旧 access token 只能按 TTL 自然过期；离线 RP 不会因 DB 切 tier 立即拒绝，按 access TTL 排空。
- 中央 SSO 不再适用于 Umbrella；router 对 `@umbrella.com` 从「中央 S」改为返回 D 的 issuer/loginUrl。
- 账号映射按 `(iss, sub)` 保留，不按邮箱自动合并。
- 回滚：旧 issuer 只读/撤信窗口；双 issuer 兼容窗口另行定义，不是 P0/P1 默认。

## 7.1 域名退役（P0 alias + offboard primary；P1 换主域）

**alias 退役（P0，示例 old.acme.com）**

alias 不签 token、不建会话，退役只影响 302 和入口：

| 步骤 | 状态/动作 | 可见效果 |
|---|---|---|
| 1 | 先 `disable`（停止 302）；再从入口移除 | 不再有新登录从旧 alias 进入 |
| 2 | `status=retired` + `retired_at` + `retire_reason` | 行保留，host 唯一性仍占用；其他租户不能立即登记 |
| 3 | 旧链接 400/404；TLS/路由下线 | 不 302、不回退中央；不撤 primary 的 refresh/session |
| 4 | 隔离 90 天 **且** 审批后 `status=released` + `released_at` | 才允许重新登记；仍需 DNS 验证/平台审核 |

**primary 退役**

P0 只能走 **offboard**（不换主域）：

| 步骤 | 动作 |
|---|---|
| 1 | `POST /tenants/{id}/offboard`：tenant suspended、primary active→disabled、清空 issuer、撤销 session/refresh |
| 2 | 停新 code/refresh/M2M；旧 access 按 TTL 排空 |
| 3 | `POST /domains/{id}/retire`（primary disabled→retired） |
| 4 | 隔离 90 天+审批后 `release` |

P1 换主域（示例 login.umbrella.com）：

| 步骤 | 动作 |
|---|---|
| 1 | P1 primary migration：新 primary 在同一迁移事务里替换旧 primary；不允许两个 active primary 并行 |
| 2 | 租户级签发门禁：停新 code/refresh/M2M |
| 3 | 撤 refresh/session；旧 access 按 TTL 排空 |
| 4 | 切 RP 信任到新 issuer；账号按 `(iss, sub)` 绑定 |
| 5 | 旧 primary 置 retired/墓碑；隔离 90 天+审批后 released |

**注意**：P0 不换 primary；offboard 后 primary 才可退役；未知 Host/退役 Host 的协议请求一律拒绝，不动态构造 issuer。

## 8. 每应用一个 IAM 登录域（不推荐）

| 应用 | 要求的登录域 | 如果按应用做 issuer |
|---|---|---|
| CRM | `login.crm.acme.com` | `https://login.crm.acme.com/oidc` |
| HR | `login.hr.acme.com` | `https://login.hr.acme.com/oidc` |
| Billing | `login.billing.acme.com` | `https://login.billing.acme.com/oidc` |

**后果**

- 3 个 issuer、3 套 host-only cookie/session、3 张证书/密钥；
- 同一用户在 CRM/HR/Billing 之间不共享登录，要输 3 次密码；
- 证书/密钥/配置按应用数增长；
- `iam_domain` 仍不应加 `app_id`；这不是应用维度，是品牌/登录域维度。

**替代**

- 只想品牌化 → `crm-sso.acme.com`/`hr-sso.acme.com` 作为 alias 302 到 `login.acme.com`，仍是 1 个 issuer、共享登录。
- 确实要独立品牌会话 → branded（P2），按品牌而不是按应用。
- 应用只是在自己的域上 → 应用域只做 `redirect_uri`。

## 9. 组合选择速查

| 诉求 | 域名组合 | issuer 数 | 登录/SSO | 证书/密钥 |
|---|---|---|---|---|
| 一个登录域，多应用不同主域 | primary + app `redirect_uri` | 1 | 应用共享 OP 登录 | 1 |
| 一个登录域 + 多别名 | primary + alias（target=primary） | 1 | 共享 | 1 + alias 证书 |
| 多品牌，各登录域 | primary + branded | 1+N | 品牌内共享；跨品牌不共享 | 1+N |
| 长尾，无自定义域 | S：中央 issuer + `organization`/`domain_hint` | 1（中央） | 中央 SSO | 平台 |
| 每应用一个登录域 | 不建议；用 alias 或 branded/独立 tenant | N | 应用间不共享 | N |
| 应用在客户域 | 应用域 = `redirect_uri` | 不变 | 不变 | 不变 |

