# 种子身份键（seed_key）：把「定位」与「业务编码」解耦

> 状态：**已落地**（2026-09-12）
> 决策日期：2026-09-12
> 上游文档：[seed-initialization-redesign-20260912.md](seed-initialization-redesign-20260912.md)、[seed-authority-scope-revision-20260912.md](seed-authority-scope-revision-20260912.md)、[client-code-convention-20260912.md](client-code-convention-20260912.md)
> 一句话结论：`code` 一直被当成种子的"认行键"，导致**改编码就会让种子重建一行**（菜单重复、授权分叉）；引入不可见的**种子身份键 `seed_key`** 承担定位后，菜单/应用的 `code` 乃至归属应用都可以自由修改，而**唯一保留只读的是内置应用的 `code`（控制台菜单入口的定位值）与内置控制台客户端的 `client_id`**（网关 aud 信任边界）——改它们会当场切断执行改名的这条通道。

---

## 背景：为什么编码被锁

需求是「菜单编码可改、客户端编码也可改，除非有必须不可改的理由」。核查后，`code` 在系统里被四处按值引用：

| 引用点 | 位置 | 性质 |
|---|---|---|
| 种子认行 | `pkg/seed/seed.go`（`seedMenus` 按 `app_id + code`、`getOrCreateApplication` 按 `code`） | 内部 |
| **开通租户时按编码授权菜单** | `pkg/core/tenant/provision.go`（菜单 + 应用） | 内部 |
| 退役菜单清理 | `pkg/seed/retired_menu.go` | 内部 |
| **OIDC 每次授权/换令牌按 `client_id` 查客户端** | `apps/auth/internal/core/oidcop/persistent_store.go`（6 处） | **对外协议契约** |

前三处都是"本系统自己的记账方式"——**可以换成不变量**；最后一处是**协议契约**，改名的代价（RP 要改配置、旧令牌失效）是 OIDC 固有语义，不可消除。

## 决策

| 字段 | 决策 | 理由 |
|---|---|---|
| `menu.code` | ✅ 可改 | 纯内部标识；改由 `seed_key` 定位后无任何副作用 |
| `menu.app_id`（归属应用） | ✅ 可改 | 同上；把菜单移到别的应用是合理 IA 操作 |
| `application.code`（**自建应用**） | ✅ 可改 | 纯内部标识；菜单/订阅/角色都挂 `app_id`，按 `id` 关联，改编码不影响它们 |
| `application.code`（**内置两个控制台应用**） | ❌ **保持只读**（同日修订） | 控制台菜单入口按它定位，见下节 |
| `application_client.code`（**用户自建**） | ✅ 可改 | 无技术障碍；改名的后果是"该 RP 需同步 client_id + 旧令牌失效"，可预期、可恢复 |
| `application_client.code`（**内置两个控制台**） | ❌ **保持只读** | 见下节 |

### 为什么内置控制台的 `client_id` 必须只读

不是"种子认行"（那个已经用 `seed_key` 解决），而是两条更本质的：

1. **它就是网关的信任边界。** `platformadmin` / `tenantadmin` 靠 `aud` 区分"A 控制台令牌不得调 B 控制台接口"（`middleware.WithOIDCAudiences(model.SeedBuiltinClient*)`）。这个值就是那条边界。
2. **改它会当场切断"执行改名的这条通道"，且界面无法自救。** 在控制台改名 → 当前会话令牌 `aud` 立刻变旧 → 401；要重新登录，前端 `VITE_OIDC_CLIENT_ID`（**构建期**默认值）还在发旧 `client_id` → OP 报 `client not found`。必须改前端 env 重新构建才能恢复——这是自我指涉死锁。

它有零业务收益（改名不带来任何功能），只有"网关常量 + 前端 env + RP 注册"三处同步的运维负担。真要换（白标/迁移）属**版本级动作**，不该是控制台日常操作。

### 修订（2026-09-12 同日）：内置应用的 `code` 同样收敛为只读

初版把 `application.code` 对**所有**应用放开，理由是"菜单/订阅/角色都挂 `app_id`"。这漏掉了控制台**菜单入口**这一处：两条 `app_id` 定位链路至今仍以**编码**为查询条件——

| 位置 | 引用方式 |
|---|---|
| `apps/platformadmin/internal/service/svcpermission/menu.go`（`platformAdminAppCode = "platform_admin"`） | `MyTree` 按 `application.code` 查出平台控制台应用，再取它的菜单树 |
| `apps/tenantadmin/internal/service/svctenant/menu.go`（`tenantAdminAppCode = "tenant_admin"`） | `loadConsoleApps` 只保留编码为 `tenant_admin` 的内置应用，其余内置应用的菜单不并入租户侧边栏 |

因此改名同样是自我锁死：平台控制台改名后 `MyTree` 查不到应用（`MenuGetPageListError`），租户控制台改名后侧边栏全部菜单消失，而修复入口就在那个已经空掉的侧边栏里。**内置应用 `code` 改为 service 层拒改**（`ApplicationBuiltInCodeImmutableError = 100749`），前端表单对 `source=builtin` 置灰；自建应用不受影响（无任何代码按它们的编码定位）。若要彻底消除这层耦合，应把这两处改为按 `seed_key` 定位——那是独立改造，不在本次范围。

---

## 设计：`seed_key` 承担定位

### 数据模型

```go
// menu / application 各新增一列（application_client 不需要：内置 client_id 只读，种子按 code 认行依然成立）
SeedKey string `gorm:"column:seed_key;type:varchar(64);not null;default:'';...;
    uniqueIndex:uk_<t>_seed_key_active,where:deleted_at IS NULL AND seed_key <> ''"`
```

- 取值 = **种子定义时的 `code`**，创建时写入后**不再变化**；控制台不可见、不可写（`json:"-"`、DTO 里没有该字段）；
- 控制台自建行恒为空串；空串不进索引（**部分唯一索引**，与 `uk_*_active` 同款写法）；
- 顺带给 `menu` 补 **`(app_id, code)` 部分唯一索引**（原先没有，编码可编辑后重名会让开通/授权错乱）。

### 认行顺序（幂等，兼容存量库）

```
1) seed_key       —— 内置行的稳定身份：运营改过 code/归属后仍命中同一行
2) (app_id, code) —— 兜底：存量库首次升级时该行还没有 seed_key，按旧口径认领并回填
3) 都没有          —— 新建（同时写入 seed_key 与定义值）
```

存量库首次启动会把每个内置行认领并回填 `seed_key`（报告记为 `migrated`，启动日志可见），**不新建任何行**。

### 三处内部引用同步改造

| 位置 | 改造 |
|---|---|
| `seedMenus` / `getOrCreateApplication` | 按 `seed_key` 查行 + 回填（`findMenuBySeedKey` / `findApplicationBySeedKey`） |
| `pruneRetiredMenus` | 先按 `seed_key` 匹配；兜底 `seed_key = '' AND app_id = ? AND code = ?`（**必须限定应用**，否则运营自建的同名菜单会被误删） |
| `ProvisionTenantAdmin` | 应用与菜单都按 `seed_key` 定位（`ProvisionAppSeedKey`、`ProvisionMenuSeedKeys`）——运营改过编码后，新开通租户仍授权到同一批菜单 |

### 入口放开

| 层 | 改动 |
|---|---|
| DTO | `MenuUpdateReq` 复用既有 `MenuBaseInfo.Code`；`ApplicationUpdateReq.Code`、`ApplicationClientUpdateReq.Code` 新增（**留空表示不修改**） |
| service | 菜单：编码非空 + 唯一索引兜底；应用：`IsValidAppCode` 校验（**内置应用拒改**，同日修订见上节）；客户端：`IsValidClientCode` 校验 + **内置客户端拒改**（`ApplicationClientBuiltInCodeImmutableError = 100823`） |
| 菜单守卫 | 原「定位键拒写」整段删除，只剩**新增/删除**拒写（`MenuBuiltInCreateDeleteForbiddenError`，编号 100606 不变）。<br>**后续（2026-09-12）**：该守卫也已解除，见 [menu-console-crud-20260912.md](menu-console-crud-20260912.md)——`seedMenus` 改为识别软删"墓碑"，删除内置菜单不再被复活，100606 随之下线 |
| 前端 | 应用表单编码在编辑态可见（**内置应用置灰**，同日修订）；客户端表单编码在编辑态可见（内置置灰）；菜单表单编码解除只读 |

---

## 代价（有意接受的取舍）

**菜单既有行的跨版本结构变更不再自动生效。** 种子对菜单只剩三件事：按 `seed_key` 认行、缺失时创建、下线走 `retiredMenus`。因此：

- **新增**菜单：仍自动播种（创建不冲突）；
- **既有**菜单改名/改路径/改组件：由运维在控制台完成（这正是本次需求），或按"旧 `seed_key` 登记 `retiredMenus` + 新定义"两步走（会换 id，`role_menu` 授权由种子重建）；
- **开发者改 `seedMenus` 里的定义编码**：等价于换身份——必须同时把旧编码登记进 `retiredMenus`（既有约定），否则旧行成为孤儿。

---

## 影响面与升级路径

| 维度 | 结论 |
|---|---|
| 存量库 | 下次启动 `AutoMigrate` 加列/加索引 + 种子回填 `seed_key`；**不新建、不删除**任何行 |
| 升级前置检查 | `menu` 的 `(app_id, code)` 必须无重复（否则唯一索引创建失败、启动中断）。检查：`select app_id, code, count(*) from menu where deleted_at is null group by 1,2 having count(*) > 1;` |
| 接口契约 | 三个 Update 接口新增可选 `code` 字段（向后兼容：不传即不改） |
| 内置客户端 | 编码仍只读；网关 aud 白名单与前端默认值都无需变动 |
| 内置应用 | 编码只读（同日修订）：改它会失去控制台菜单入口；网关/前端无需变动 |
| 客户端改名（自建） | 使用该 client 的 RP 需同步 `client_id`，其旧令牌按新 `aud` 失效 → 重新登录 |

---

## 验收

| 编号 | 内容 | 用例 |
|------|------|------|
| V1 | 矩阵：菜单无 reconcile 字段；应用仅 `source` | `pkg/model`：`TestSeedReconcileFields`、`TestSeedOwnsFieldOnlyForReconcile` |
| V2 | 改内置菜单编码后重启种子**不重建行**，且旧编码不残留 | `pkg/seed`：`TestSeedIamKeepsMenuIdentityAfterCodeRename` |
| V3 | 改内置应用编码后重启种子**不重建应用**，菜单仍挂同一 `app_id` | `pkg/seed`：`TestSeedIamKeepsApplicationIdentityAfterCodeRename` |
| V4 | 存量库首次升级：按 `(app_id, code)` 认领并回填 `seed_key`，不新建行、二次执行幂等 | `pkg/seed`：`TestSeedIamBackfillsSeedKeyForExistingRows` |
| V5 | 菜单改名成退役清单里的编码**不被误删**（退役清理按 seed_key 认行） | `pkg/seed`：`TestSeedIamDoesNotPruneMenuRenamedToRetiredKey` |
| V6 | 开通链路按 `seed_key` 授权：菜单改名后新租户仍拿到全部菜单 | `pkg/core/tenant`：`TestProvisionTenantAdmin_ResolvesMenusBySeedKey` |
| V7 | 服务层：内置菜单字段全可改 + 空编码拒绝 | `svcpermission`：`TestUpdateBuiltInAppMenuAllowsAllFields` |
| V8 | 服务层：自建应用改编码放行 / 非法编码拒绝 / **内置应用拒改**（同日修订） | `svcapplication`：`TestUpdateApplicationAllowsCodeRename`、`TestUpdateApplicationRejectsInvalidCode`、`TestUpdateBuiltInApplicationRejectsCodeRename` |
| V9 | 服务层：自建客户端改编码放行 / 内置客户端拒改 / 非法编码拒绝 | `svcapplicationclient`：`TestUpdateClientAllowsCodeRenameForThirdParty`、`TestUpdateBuiltInClientRejectsCodeRename` |
| V10 | 前端：应用编码编辑态可见（内置置灰）；客户端编码编辑态可见且内置置灰；菜单编码可编辑 | platform-admin-web：四条用例 |
| V11 | 真实 Postgres：`AutoMigrate` 新列/部分唯一索引 + 种子幂等 | `go test -tags pg ./pkg/seed/ -run TestSeedIamAgainstPostgres` |

---

## 风险与回滚

| 风险 | 触发 | 应对 |
|------|------|------|
| R1 存量库有重复 `(app_id, code)` | 历史手改数据 | 加索引前会中断启动并报错；按上面 SQL 检出后去重（或删库重建） |
| R2 改名撞已有的编码 | 运营改成别的菜单/应用已占用的编码 | DB 唯一索引兜底 → 返回本领域更新错误码；不需要额外预检（预检有 TOCTOU 竞态） |
| R3 菜单跨版本结构不再自愈 | 新版本想改既有菜单路径/组件 | 登记 `retiredMenus` + 新定义，或由运维在控制台改 |
| 回滚 | 需回到"编码不可改" | 回退矩阵与三个 DTO/service 分支即可；`seed_key` 列保留无害（种子仍按它认行） |

---

## 落地记录（2026-09-12）

| 层 | 文件 | 内容 |
|----|------|------|
| 模型 | `pkg/model/menu.go`、`pkg/model/application.go` | 新增 `seed_key` + 部分唯一索引；`menu` 补 `(app_id, code)` 部分唯一索引；字段注释说明定位与编码分离 |
| DAO | `pkg/dao/menu.go`、`pkg/dao/application.go` | `MenuCond.SeedKey`、`ApplicationCond.SeedKey` |
| 种子 | `pkg/seed/seed.go`、`pkg/seed/retired_menu.go` | `seedMenus`/`getOrCreateApplication` 按 `seed_key` 认行 + 回填；`retiredMenu.menuSeedKey` 与兜底匹配 |
| 开通 | `pkg/core/tenant/provision.go` | `ProvisionAppSeedKey`、`ProvisionMenuSeedKeys`；应用/菜单按 `seed_key` 定位 |
| 矩阵 | `pkg/model/seed_authority.go` | 菜单 `app_id`/`code` 由 reconcile 改为 create_only（菜单已无收敛字段）；声明 `seed_key` |
| 服务 | `svcpermission/menu.go`、`svcapplication/application.go`、`svcapplicationclient/application_client.go` | 删除定位键守卫、放开三处编码、加校验与内置客户端拒改；同日修订：内置应用 `code` 也拒改 |
| 错误码 | `pkg/code/permission.go` | 100606 改名 `MenuBuiltInCreateDeleteForbiddenError`；新增 100823 `ApplicationClientBuiltInCodeImmutableError`、100749 `ApplicationBuiltInCodeImmutableError`（同日修订） |
| 前端 | application / menu / oauthClient 三页 + `packages/types` | 编码改为可编辑（内置客户端置灰）；更新入参类型补 `code`；同日修订：内置应用编码置灰 |
| 测试 | `pkg/model`、`pkg/seed`、`pkg/core/tenant`、三个 service、platform-admin-web | 新增/改写 11 组用例（含存量回填与退役误删防护） |
