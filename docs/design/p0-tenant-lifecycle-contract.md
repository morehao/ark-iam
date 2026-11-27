# P0 Tenant / Domain 生命周期契约

> 状态：**P0 冻结契约（spike 输入）** ｜ 最后更新：2026-11-27
>
> 配套：[tenant-custom-domain-redesign.md](tenant-custom-domain-redesign.md)、[p0-host-binding-contract.md](p0-host-binding-contract.md)
>
> 本文只解决 P0 的租户状态、D onboarding/取消/offboard、域名状态机、D console client 收尾和 host 重登记；不涉及 S↔D/D→D 迁移。

## 1. 租户状态

| 状态 | 含义 | 可签发 |
|---|---|---|
| `provisioning` | D 已建租，尚未有 active primary | 否 |
| `active` | D 可签发，恰一个 active primary | 是 |
| `suspended` | offboard 后；primary disabled，issuer 已清空 | 否 |

**状态解析**（拆成解析 + 场景默认）：

- `ParseTenantStatus(raw string) (TenantStatus, bool, error)`：空返回 `(,“false”,nil)`；非空非法返回 `101212`；合法返回具体值。
- 普通 create：未指定 → `active`；dedicated create：未指定 → `provisioning`；update：未指定 → 保留原值。
- `PageList` 不调用解析；空筛选 = 不过滤。
- 白名单只保留一处；`packages/types`、`packages/ui`、`packages/console`、`glossary.md`、`system-design.md`、`api-reference.md` 同步三值。
- `tenant.status` 在 `seed_authority` 中仍是 `immutable`（通用控制台拒写）；activate/offboard 是系统动作，走服务内部状态转换 + 审计，不经通用 PUT。`isolation_mode` 是 create-only；P0 不允许 PUT 改。

## 2. D onboarding

1. `POST /v1/platform/tenants`：`{isolationMode:"dedicated", primaryDomain}`。
2. 同事务：tenant（`provisioning`，忽略/拒绝请求里的 `status`）+ primary（`draft`）+ binding（`draft`）。
3. primary：verify → tls_ready → activate；activate 同事务把 tenant 置 `active`，写 `issuer`、`domain_epoch+1`。
4. primary active + D console client 有效后，binding `enable`。
5. 未 active 前不解析、不签发、不启用 binding。

## 3. 取消/offboard/删除

**取消建租（provisioning 且无 active primary）**：

- 统一走 `POST /v1/platform/tenants/{tenantID}/offboard`，接受 `provisioning`；
- 同事务：tenant `suspended`；所有未上线域名 `draft/verified/tls_ready → released`；binding `disable`+软删；D console client `disable`+软删；清 issuer/epoch、撤会话；
- **D 的 `DELETE /v1/platform/tenants/{tenantID}` 一律 403**，P0 不提供直接删租户；suspended tenant 由运维归档。

**offboard（active D）**：

1. tenant `suspended`；
2. primary `active→disabled`；
3. 清空 `iam_tenant.issuer`，`domain_epoch+1`；
4. 停新签发；撤 session/refresh；
5. D console client、binding `disable` + 软删；指向 primary 的 active/disabled alias 同事务置 `disabled`，retired/released 历史行不改；
6. DB 提交后，primary 可 `disabled→retired`；满 90 天+审批后 `release`。

**release**：只校验 `issuer` 已为空（不重复清）；alias release 不动 tenant；primary release 要求 `tenant.suspended`、90 天+审批、旧 client/binding/refresh/session/RP 撤信完成。

## 4. 域名状态机

| 对象 | 动作 | from → to | 守卫 |
|---|---|---|---|
| primary | verify | draft → verified | DNS TXT/人工域控证明 |
| primary | tls-ready | verified → tls_ready | 证书就绪 |
| primary | activate | tls_ready → active | tenant provisioning/active；无其他 active primary；usage=login/both |
| primary | offboard | active → disabled | 仅 tenant offboard |
| primary | retire | disabled → retired | tenant suspended；reason/audit |
| primary | release | retired → released | 满 90 天（从 `retired_at` 起）且 `approvalTicket` |
| alias(login/both) | verify/tls/activate | tls_ready → active | target 同租户且 active |
| alias(email) D | verify/activate | verified → active | target 必填、同租户且 active；P0 不消费 |
| alias(email) S | verify/activate | verified → active | target 必须为空（中央）；P0 不消费 |
| alias | disable | active → disabled | — |
| alias | enable | disabled → active | target active；login/both 需 tls_ready |
| alias | retire | active/disabled → retired | — |
| alias | release | retired → released | 90 天+approvalTicket |
| 任意 | delete | draft/verified/tls_ready → released | 未 active；有未 released alias 引用时拒绝 |

**规则**：

- `iam_tenant.issuer` 在 **offboard** 清一次；release 只校验为空，不重复清；alias release 不动 tenant；offboard 的 epoch 只 +1，取消未上线域名不重置 epoch。
- `released` 是终态；host 可重登记；`retired` 仍占用 host。
- P0 不创建 S email alias；若已有/未来创建，只允许 `alias + usage=email + target 空`，不参与 P0 路由。
- `target` 被 disable/retire 时，只把 **active/disabled** 的 alias 置 disabled；**不得改 retired/released 历史行**；保留 `target_domain_id`，不自动改指向。
- `DELETE` 已被 alias 引用的行时拒绝；先处理 alias。取消与正式 release 都用 `released`，但取消不填 `release_approval`。
- 取消（未上线）与正式 release（退役后 90 天+审批）分开审计；都使用 `released` 终态，但 `release_approval` 只在正式 release 填写。

## 5. D console client 收尾

- D console client code：固定 `tenant_console`（或只用小写字母的稳定值）；`issuer` 绑 D issuer；`tenant_id` = D 租户。
- primary activate 同事务创建 D console client 并回填 `binding.client_id`；offboard/release 时 client `disable` + 软删 + 审计。
- 同一 Host 重登记：旧 binding/client 已软删后，新租户新建 binding/client；不允许 upsert 复活旧行。
- AC-10 拆成：binding 清理、client 清理、同 Host 新租户激活并登录。

## 6. 前端/文档同步

- `TenantStatus` 增 `provisioning`；列表 tag、筛选、编辑表单三态；D 编辑不提交 `status`。
- D 的 `PUT /tenants` 只在 `status` 与原值不同时拒绝；前端 D 编辑不提交 status。
- `glossary.md`、`system-design.md`、`api-reference.md`、`packages/types`、swagger 同步；`101212` 进错误码表。

## 7. 代码落点

| 文件 | 改动 |
|---|---|
| `pkg/model/tenant.go`、`pkg/object/objtenant`、`pkg/core/tenant` | 三值枚举；Normalize 空/非法语义；dedicated create 强制 provisioning；D 的 PUT/DELETE 守卫；offboard 接受 provisioning |
| `pkg/model/domain.go`、`pkg/dao/domain.go`、`pkg/core/tenantdomain`、`pkg/model/automigrate.go` | 状态机；`released`/`release_approval`；target 级联；partial unique |
| `apps/platformadmin/service/svcdomain`、`svctenant` | 动作端点；offboard/release 副作用；D console client 清理 |
| `apps/platformadmin/service/svcapplicationclient`、`pkg/seed` | D console client 创建/停用/软删；code 规则；seed 平台 client |
| `packages/*`、`docs/design/*` | 三值状态、D 编辑不提交 status、动作端点 |

## 8. 验收

| ID | 场景 | 通过标准 |
|---|---|---|
| L-1 | D onboarding | provisioning + primary draft；activate 后 tenant active；binding enable |
| L-2 | 取消建租 | provisioning 可 offboard；D DELETE 403；无死租户、无死 host |
| L-3 | offboard | tenant suspended、primary disabled、issuer 空、session/refresh 撤销；之后 retire/release |
| L-4 | alias release | 只释放 alias；primary issuer/登录不受影响 |
| L-5 | primary release | retired 满 90 天+审批；旧 client/binding 已清理；同 Host 新租户可激活 |
| L-6 | 并发 | 两 activate 只一个成功；offboard 与 release 不产生非法态 |
| L-7 | 前端 | provisioning 显示“建租中”；D 编辑不提交 status；筛选三值可用 |
