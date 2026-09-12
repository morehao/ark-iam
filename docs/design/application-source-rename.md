# 应用归属字段 type → source（同时下线 is_system）

> 状态：**已落地**（2026-09-12；代码/前端/文档/swagger 均已改，后端 5 模块编译通过、前端 typecheck + 42 测试通过；落地记录见 §12）
> 决策日期：2026-09-12
> 采纳口径：**S1b**——`application.type` → `application.source`（`builtin` / `first_party` / `third_party`）；删除 `application.is_system`，其「内置」语义并入 `source`
> 命名评审：**保持 `first_party` / `third_party` 不改**（结论与依据见 §2.6）
> 已定决策：控制台只能创建 `third_party`（§2.5）；`application_client` **同批同步**（§5）；错误码**常量名与文案改、编号不动**（§4）
> 影响一句话：`application` + `application_client` 各 **1 列改名 + 1 列下线**、2 个具名枚举；后端 21 文件、前端 7 文件、living docs 3 文件 + `frontend/DESIGN.md`、swagger 重生成
>
> 第三处冗余归属列 `application_client.is_third_party` 已**同批下线**（§13）——三处归属列最终收敛为一列 `source`。
> **归属再修正（§14）**：平台随产品交付的两个控制台应用（管理后台、租户自服务）**都归 `builtin`**；§2.3 把租户自服务留在 `first_party` 的判断已被取代，菜单归属改由「属于哪个控制台」判定，不再由内置性兼任。
> 本文取代 `organization-container-redesign.md:359` 中「`application.is_system` 属布尔字段改造成果」的表述——该列将在本次改造后下线。

---

## 0. 决策与范围

### 0.1 结论

1. `application.type`（`first_party` / `third_party`）更名为 **`application.source`**，并升级为**具名强类型枚举**（对齐 `model.MenuType` / `model.RoleSource` / `model.UserSource`）。
2. `application.is_system`（bool）**下线**，「是否内置」由 `source` 的首值 `builtin` 承担，禁删判据与控制台菜单判据改判 `source`。
3. 「应用来源」取值**必须让 `builtin` 成为其中一个值**——这是「参照项目其他 `source` 语义」的唯一落点（§1.3）。
4. **值域定稿（决策 ①）**：`builtin` / `first_party` / `third_party`（S1b），`first_party`、`third_party` 两个既有值不动（§2.6）。
5. **控制台创建口径（决策 ②）**：`Create` / `Update` 只接受 `third_party`，`source` 从入参移除或做白名单；`builtin` / `first_party` 仅由种子与运维产生（§2.5、§6）。
6. **客户端同批（决策 ③）**：`application_client.type` / `is_system` 与主表同批改造，新增独立具名类型 `ApplicationClientSource`（§3、§5）。
7. **错误码（决策 ④）**：`ApplicationSystemBuiltInErr` / `ApplicationClientSystemBuiltInErr` 常量名与文案同步改，**编号 `100746` / `100820` 不动**（§4）。

### 0.2 本次不做

- 不改 API 版本（仍 `/v1`）、不加兼容层、不双写、不做字段别名。
- 不改错误码**编号**（`100746` / `100820` 不动，前端错误码表无需改）；常量名与中文文案随语义同步改（§4）。
- 不动 `docs/superpowers/**` 的历史 plan/spec（12 文件 43 处 `first_party`/`third_party` 作为历史记录保留原文）。
- 不引入并行命名（不新增 `internal` / `external` 等价别名值，§2.6）。

### 0.3 与既有规范的关系

按 `AGENTS.md`「数据库 Schema 变更约定（按新项目处理）」：列改名与列下线**一律彻底删代码、不写迁移脚本、不加回填分支**；旧库残留列属预期，处置方式是开发/测试库删库重建。存量数据保全方案见 §8.3（一次性 SQL 交部署文档，不塞进启动流程）。

> **行号基准**：本文所有 `文件:行` 引用以 2026-09-12 的工作区为准——该工作区含一批**与本方案无关**的未提交改动（`application.visibility` 下线 + 租户应用订阅页），引用行号已按该状态校验；若这批改动被回退或再次编辑，请重新核对 §5 的行号。

---

## 1. 现状（证据）

### 1.1 两个字段各自承担什么

| 应用 | `code` | 现状 `type` | 现状 `is_system` | 删除保护 | 菜单归属 |
|---|---|---|---|---|---|
| 管理后台 | `platform-admin` | `first_party` | **`true`** | 禁删 | 平台控制台（租户控制台**排除**，`menu.go:111`） |
| 租户自服务 | `tenant-admin` | `first_party` | **`false`** | **可删** | **租户控制台**（`loadConsoleApps` 保留，`menu.go:166` / `:190`） |
| 第三方接入 | 控制台创建 | `third_party` | `false` | 可删 | 按租户订阅 |

证据：

- 两个种子应用的内置标志不同：`backend/pkg/seed/seed.go:74`（`true`）与 `:78`（`false`），两者 `Type` 同为 `model.AppTypeFirstParty`（`seed.go:248`）。
- 删除保护：`backend/apps/platformadmin/internal/service/svcapplication/application.go:93` → `code.ApplicationSystemBuiltInErr`（`backend/pkg/code/permission.go:31`，文案 `:97`）。
- 菜单范围：`backend/apps/tenantadmin/internal/service/svctenant/menu.go:104-112`（`loadConsoleApps` 在 `:111` 跳过 `IsSystem`），被 `buildTenantMenuTree`（`:166`）与 `buildMyMenuTree`（`:190`）消费。

**结论：同一张表里两个字段不是一回事——`type` 表归属方，`is_system` 表内置/保护/菜单归属。**

### 1.2 为什么 `type` 这个名字站不住

`docs/design/iam-design.md`（历史快照）里 `application.type` 的原语义是 `web/spa/native/machine`（客户端形态），第三方属性另由 `is_third_party` 布尔承担；两套枚举后续合并成现在的 `type`，于是留了一个既不是"类型"也不是"平台"的名字。今天它只表达一件事：**这个应用是我们自己的，还是外部接入的。**

### 1.3 为什么不能只换名字就完事：`source` 语义对标

| 模型 | 字段（Go 类型） | 值域 | 实际语义 |
|---|---|---|---|
| `user` | `Source`（`UserSource`，`model/user.go:31-36`） | `builtin` / `manual` | **产生方式**：随租户创建自动生成 / 控制台手工创建 |
| `role` | `Source`（`RoleSource`，`model/role.go:9-15`） | `builtin` / `custom` | **产生方式**：种子播种（禁删、禁改 `admin_type`）/ 租户创建 |
| `application` | `Type`（裸 `string`，`model/application.go:28`） | `first_party` / `third_party` | **归属方** |

前端共享组件 `SourceTag`（`frontend/packages/ui/src/status.tsx:57-59`）据此定义：`builtin → 内置(gold)`，其余 → `自定义(blue)`；应用的 `first_party/third_party` 目前走的是另一个组件 `TypeTag`（`:43-51`）。

因此：**把 `first_party/third_party` 原样改名叫 `source`，并不满足"参照其他 source 语义"——只会让同一个包里 `source` 一词两义（user/role=产生方式，application=归属方），且 `SourceTag` 无法复用。真正对齐语义的做法是让「内置」成为 `source` 的取值，这恰好就是 `is_system` 让出来的槽位。**

### 1.4 顺带必须修的既有违规（同一批改掉）

| 违规 | 位置 |
|---|---|
| 常量是裸字符串，字段是裸 `string`，无具名类型（违反「强类型字符串枚举」） | `model/application.go:9-12`、`:28`；`model/application_client.go:9-12` |
| DTO 枚举字段用裸 `string` 而非具名类型 | `dtoapplication/request.go:9,21,40`、`response.go:15,28` |
| service 无非法值白名单校验，`req.Type` 直接写库（违反「非法值校验归 service」） | `svcapplication/application.go:40`、`:69` |
| 常量跨实体混用：把 `model.AppTypeFirstParty` 赋给 `ApplicationClientEntity.Type` | `backend/pkg/seed/seed.go:630` |
| 文档 ER 图残留已下线字段 `application_client.is_third_party` | `docs/design/system-design.md:283` |

---

## 2. 值域设计（已定稿：S1b）

### 2.1 三个候选

| 口径 | 取值 | 一句话 |
|---|---|---|
| **S1a** | `builtin` / `third_party` | 两值，内置 vs 第三方接入。**已否决**：会让租户控制台菜单变空（§2.2） |
| **S1b** ✅ | `builtin` / `first_party` / `third_party` | **定稿**：内置 / 平台自建 / 第三方接入，零行为变更 |
| **S1c** | `builtin` / `custom` | 对齐 user/role 字面，但丢失「第三方」这一产品维度。**已否决**（§2.4） |

### 2.2 S1a 不可直接落地：租户控制台菜单会变空

S1a 下「租户自服务」是种子播出的应用，按 `source` 语义只能是 `builtin`；而 `builtin` 即「内置」，于是：

1. 删除保护从"可删"变为"禁删"—— 这是可接受的收紧；
2. **`loadConsoleApps`（`menu.go:104-112`）会把它排除，而它正是租户控制台菜单的唯一载体**（`buildTenantMenuTree` / `buildMyMenuTree` 只遍历 `loadConsoleApps`）→ **租户控制台侧边栏整体为空**，属功能性破坏，不是文案级行为变更。

> 现有代码之所以没这个问题，是因为它用 `is_system=false` 给「租户自服务」开了口子；`is_system` 一删，这个区分必须由 `source` 里的第三个取值承担，否则就得改菜单判据逻辑（把「是否是内置应用」与「菜单归哪个控制台」拆开）。**这就是推荐 S1b 的全部理由。**

### 2.3 S1b（定稿）：零行为变更

| 应用 | 现状 (`type`, `is_system`) | S1b `source` | 行为差异 |
|---|---|---|---|
| 管理后台 | (`first_party`, `true`) | `builtin` | 无 |
| 租户自服务 | (`first_party`, `false`) | ~~`first_party`~~ → **`builtin`** | 见 §14（本批修正：内置性不再兼任菜单归属） |
| 第三方接入 | (`third_party`, `false`) | `third_party` | 无 |
| 控制台自建第一方（如存在） | (`first_party`, `false`) | `first_party` | 无 |

- 映射是**单射且可判定**：`is_system=true → builtin`；`type=third_party → third_party`；其余 → `first_party`。
- `builtin` 与 user/role 的 `builtin` 同名同义（系统播种、受保护），满足「参照其他 source 语义」；`first_party/third_party` 保住「第三方接入」这一 IAM 核心维度（OIDC 客户端接入的产品前提）。
- 语义读法自洽：**`source` = 这个应用从哪来**（平台内置 / 平台自建 / 第三方接入）。

### 2.4 S1c 不可取

`builtin` / `custom` 会让 `third_party` 这一维度从数据层消失，直接冲击：`docs/design/glossary.md:37-38` 的第一方/第三方应用定义、`application-integration-guide.md:44` 的接入决策表、前端「第三方」标签。**不建议。**

### 2.5 行为规则表（S1b，定稿）

| `source` | 禁止删除 | 控制台菜单归属 | 控制台可否创建 | 现有对应 |
|---|---|---|---|---|
| `builtin` | ✅ | 各自所属控制台（租户控制台排除**其它**控制台的内置应用） | ❌（仅种子/运维） | 管理后台、租户自服务（§14） |
| `first_party` | ❌ | 并入所在控制台 | ❌（仅种子/运维） | 无（仅运维自建，§14） |
| `third_party` | ❌ | 并入所在控制台 | ✅ | 第三方接入 |

> 决策 ②：`Create` / `Update` 只接受 `third_party`。落地方式二选一——把 `source` 从 DTO 入参移除、由 service 固定 `AppSourceThirdParty`（更彻底，推荐），或保留入参并在 service 内做白名单校验（非法值返回功能级错误码，符合 `AGENTS.md`「非法值校验归 service」）。`builtin` / `first_party` 仅由种子（`pkg/seed`）与运维产生。

### 2.6 命名评审：`first_party` / `third_party` 是否该换（结论：不换）

**起因**：`builtin` 与 `first_party` 语义重叠——管理后台既是「内置」也是「第一方」。候选与结论：

| 候选值集 | 评价 |
|---|---|
| **`builtin` / `first_party` / `third_party`** ✅ | **保持**。两个既有值零改动；`first_party`/`third_party` 是 OAuth 生态标准术语；`builtin` 与 user/role 同名同义 |
| `builtin` / `internal` / `external` | 否决。Google Workspace Marketplace 的 internal/external 指**组织内/外**，`builtin` 同样属于 internal，重叠分毫未解；反而丢掉协议标准术语，且两个既有值全变 |
| `builtin` / `platform` / `third_party` | 否决。`platform` 与 `builtin` 指向同一主体，对外部接入方（RP 开发者）无区分意义 |
| `system` / `custom` / `manual` 等 | 否决。`custom`/`manual` 与「租户自服务是种子播种」的事实矛盾；`system` 与 user/role 词汇脱钩，前端 `SourceTag` 无法复用 |

**关键论证：重叠不可消除，故不构成换名理由。**

- 权威定义：IETF [OAuth 2.0 for First-Party Applications](https://datatracker.ietf.org/doc/draft-ietf-oauth-first-party-apps/) §1.1 —— "First-party applications are applications that are **controlled by the same entity as the authorization server** and that users understand as belonging to the same entity."。按此定义，管理后台与租户自服务**都是** first-party。
- 任何三值集合都会产生同构重叠：中间值换成 `internal`、`platform` 或别的词，`builtin` 依旧落在外延更大的那一类里。**问题源于「内置 ⊂ 平台自有」这一固有层级，而非取值字面。**
- `third_party` 是 [RFC 6749](https://www.rfc-editor.org/info/rfc6749/) §1 的原生词（"third-party applications"），行业与本文档读者（应用接入方）最熟悉，且已被本仓 `glossary.md:37-38`、`application-integration-guide.md:44`、前端 `TypeTag` 使用。

**消解方式（落地必做）**：把重叠写成显式规则，而不是靠改名回避——

1. 注释与文档统一表述：**`builtin` 是 `first_party` 中受保护的子集；枚举取最具体值**（`builtin` 应用同时也是第一方，但落 `builtin`）。
2. `AppSource` 补 `IsPlatformOwned()`（`builtin || first_party`），让「自有 vs 外部」的调用方无需枚举三值（§3）。
3. `glossary.md` 补 `builtin` 条目并写明优先级规则，避免后来者按 IETF 定义理解为「第一方包含 builtin」而产生困惑。

---

## 3. 模型层目标形态

```go
// backend/pkg/iam/model/application.go

// AppSource 应用来源。
type AppSource string

// 应用来源取值（禁止硬编码）。
const (
    AppSourceBuiltin    AppSource = "builtin"     // 平台内置：种子播种/随产品交付，禁删，菜单归各自专属控制台
    AppSourceFirstParty AppSource = "first_party" // 平台自建但非内置：可删，菜单并入所在控制台
    AppSourceThirdParty AppSource = "third_party" // 第三方接入：外部/租户接入的应用
)

// IsBuiltin 判断是否为平台内置应用（禁删 + 菜单归专属控制台）。
func (s AppSource) IsBuiltin() bool { return s == AppSourceBuiltin }

// IsPlatformOwned 判断是否平台自有（内置或自建），供「自有 vs 外部」口径使用，避免调用方枚举三值。
func (s AppSource) IsPlatformOwned() bool { return s == AppSourceBuiltin || s == AppSourceFirstParty }

type ApplicationEntity struct {
    gormdao.BaseEntity
    // ...
    Source AppSource `gorm:"column:source;type:varchar(32);not null;default:'third_party';comment:应用来源(builtin内置/first_party第一方/third_party第三方)" json:"source"`
    // IsSystem 删除
}
```

要点：

- 列类型沿用 `varchar(32)`（与现 `type` 列一致，避免 AutoMigrate 类型扰动）；**默认值改为 `third_party`**（控制台创建是主路径，`builtin` 只能由种子产生）。
- `IsBuiltin()` 方法名对齐 `UserSource.IsBuiltin()`（`model/user.go:40`）与 `RoleEntity.IsBuiltinAdmin()`（`model/role.go:34`）；`IsPlatformOwned()` 承担 §2.6-2 的「自有 vs 外部」口径。
- 旧常量 `AppTypeFirstParty` / `AppTypeThirdParty` **彻底删除**，全仓 `grep` 零残留。
- `application_client` **同批同步（决策 ③）**：新增独立具名类型 `ApplicationClientSource`（`builtin` / `first_party` / `third_party`），不复用 `AppSource`，避免跨实体混用常量（现存反例：`seed.go:630` 把 `AppTypeFirstParty` 赋给客户端实体）。

---

## 4. 行为判据替换清单

| 位置 | 现状 | 改为 |
|---|---|---|
| `apps/platformadmin/internal/service/svcapplication/application.go:93` | `if entity != nil && entity.IsSystem` | `if entity != nil && entity.Source.IsBuiltin()` |
| `apps/tenantadmin/internal/service/svctenant/menu.go:111` | `if app.IsSystem { continue }` | `if app.Source.IsBuiltin() { continue }` |
| `apps/tenantadmin/internal/service/svctenant/menu.go:57,101-103,186` | 注释「系统内置应用」 | 注释改为「内置应用（`source=builtin`）」，**口径不变** |
| `apps/platformadmin/internal/service/svcapplicationclient/application_client.go:110` | `if entity.IsSystem` | `entity.Source.IsBuiltin()`（决策 ③ 同批） |
| `backend/pkg/code/permission.go:31,97` + `:50,:108` | `ApplicationSystemBuiltInErr` / `ApplicationClientSystemBuiltInErr`，「为系统内置，不可删除」 | 常量名改 `ApplicationBuiltInErr` / `ApplicationClientBuiltInErr`，文案改「内置应用不可删除」；**编号 `100746` / `100820` 不动**（决策 ④） |

---

## 5. 全链路受影响文件清单

### 5.1 后端

| 文件 | 位置 | 改动 |
|---|---|---|
| `backend/pkg/iam/model/application.go` | `:9-12` | 常量组 → 具名类型 `AppSource` + 3 常量 + `IsBuiltin()` |
| 同上 | `:28` | `Type` → `Source`，列/json tag `type` → `source`，comment 更新 |
| 同上 | `:31` | 删除 `IsSystem` |
| `backend/pkg/iam/dao/application.go` | `:13`、`:28-29` | `Type string` → `Source model.AppSource`；`tableName+".source = ?"` |
| `backend/pkg/seed/seed.go` | `:74`、`:78`、`:248`、`:251` | 种子 `source` 赋值；`getOrCreateApplication` 签名 `isSystem bool` → `source model.AppSource` |
| `backend/pkg/seed/seed.go` | `:630` | 客户端常量改用 `ApplicationClientSourceFirstParty` |
| `backend/pkg/code/permission.go` | `:31`、`:97` | 见 §4 |
| `backend/apps/platformadmin/internal/dto/dtoapplication/request.go` | `:9`、`:21`、`:40` | `Type string` → `Source model.AppSource`（`json`/`form` tag 同步） |
| `backend/apps/platformadmin/internal/dto/dtoapplication/response.go` | `:15`、`:28` | 同上 |
| `backend/apps/platformadmin/internal/service/svcapplication/application.go` | `:40`、`:69`、`:117`、`:133`、`:148` | 字段映射；`updateMap` key `"type"` → `"source"` |
| 同上 | `:93` | 判据替换（§4） |
| `backend/pkg/iam/model/application_client.go` | `:9-12`、`:54` | （决策 ③ 同批）具名类型 + 删 `is_system` |
| `backend/pkg/iam/dao/application_client.go` | `:35-36` | （决策 ③ 同批）Cond/SQL 列名 |
| `backend/apps/platformadmin/internal/dto/dtoapplicationclient/*.go` | `request.go:6,26,56`、`response.go:26,42` | （决策 ③ 同批）字段改名 |
| `backend/apps/platformadmin/internal/service/svcapplicationclient/application_client.go` | `:79`、`:110`、`:135`、`:197`、`:212`、`:231`、`:285` | （决策 ③ 同批）字段/判据 |
| `backend/apps/tenantadmin/internal/service/svctenant/menu.go` | `:111`、注释 `:57,101-103,186` | 判据 + 注释（§4） |

### 5.2 前端

| 文件 | 位置 | 改动 |
|---|---|---|
| `frontend/packages/types/src/platform.ts` | `:11-25`、`:27-45` | `ApplicationItem.type` → `source`；`ApplicationCreateReq.type` / `ApplicationUpdateReq.type` → `source` |
| `frontend/packages/ui/src/status.tsx` | `:43-48` | `TYPE_META` 保留租户 `platform/customer`；`first_party/third_party` 迁入应用来源映射 |
| 同上 | `:57-59` | `SourceTag` 扩展应用域映射：`builtin→内置(gold)`、`first_party→第一方(blue)`、`third_party→第三方(orange)`，**保持分类色单一出处**（`frontend/DESIGN.md:181`） |
| `frontend/apps/platform-admin-web/src/pages/application/index.tsx` | `:45`、`:54`、`:113`、`:189-195`、`:234` | 表单默认值/回填、列 `dataIndex`、`render`、下拉选项、详情 `SourceTag` |
| `frontend/apps/platform-admin-web/src/pages/application/index.test.tsx` | `:28` | mock 字段改名 |
| `frontend/apps/platform-admin-web/src/pages/oauthClient/index.tsx` | `:198-203` | （决策 ③ 同批）客户端来源同步 |
| `frontend/apps/platform-admin-web/src/pages/oauthClient/Detail.tsx` | 类型标签处 | （决策 ③ 同批）同步 |
| `frontend/DESIGN.md` | `:181` | 分类标识段落：应用来源由 `SourceTag` 承担 |

### 5.3 文档

| 文件 | 位置 | 改动 |
|---|---|---|
| `docs/design/system-design.md` | `:259`、`:282`、`:528`、`:780` | ER 图/表说明/规则表 `type` → `source`；顺带删除 `:283` 过期字段 `is_third_party` |
| `docs/design/glossary.md` | `:37-38` | `application.type=first_party` → `application.source=first_party`；补 `builtin` 条目 |
| `docs/design/application-integration-guide.md` | `:44`、`:73` | 字段名与 JSON 示例 `"type": "first_party"` → `"source"` |
| `docs/design/organization-container-redesign.md` | `:359` | 标注 `application.is_system` 已下线并入 `source` |
| `docs/superpowers/**` | 12 文件 43 处 | **不动**（历史 spec/plan） |

### 5.4 测试

| 文件 | 改动 |
|---|---|
| `backend/apps/platformadmin/internal/service/svcapplication/application_delete_test.go:30,52` | `IsSystem: true/false` → `Source: builtin/first_party` |
| `backend/apps/platformadmin/internal/service/svcapplication/application_page_list_test.go` | 断言 `source` 往返 |
| `backend/apps/tenantadmin/internal/service/svctenant/menu_test.go:21,39` | `IsSystem` 播种 → `Source`；注释同步 |
| `backend/apps/platformadmin/internal/service/svcapplicationclient/application_client_delete_test.go:40` | （决策 ③ 同批）同步 |
| `backend/apps/auth/internal/core/oidcop/*_test.go`（`protocol_conformance_test.go:162,321`、`client_credentials_test.go:53,150,293`） | （决策 ③ 同批）`ApplicationClientTypeFirstParty` 常量改名 |

---

## 6. 前端改动细节

1. `SourceTag` 是**唯一**来源标签出口：不要新增页面内联 `Tag`。
2. 值映射集中一处（建议 `frontend/packages/ui/src/status.tsx` 内新增 `SOURCE_META`），`first_party/third_party` 从 `TYPE_META` 移除，避免同一值域两处维护。
3. 列表页「类型」列改名「来源」，`dataIndex: 'source'`，仍用 `TAG_COL_WIDTH`（`frontend/DESIGN.md:181` 的分类色规则不变）。
4. 新建表单（决策 ②已定）：**移除「来源」下拉**，由服务端固定 `third_party`；列表与详情只读展示 `SourceTag`。**来源不可编辑**——`builtin` / `first_party` 只能由种子与运维产生，若保留编辑入口就会出现"把内置应用改成第三方"的越权路径。

---

## 7. 测试要点

1. **枚举往返**：Create / Update / Detail / PageList 四个口子 `source` 正确落库并回传（含 json tag）。
2. **白名单校验**（新增）：`source = "foo"` 返回功能级错误码；`builtin` / `first_party` 的创建请求必须被拒（只有 `third_party` 可创建，§2.5 决策 ②）。
3. **删除保护**：`source=builtin` 拒删（`100746`）；`source=first_party` **仍可删**（§14 起 `first_party` 不再对应租户自服务——它已归 `builtin`，该分支由运维自建应用覆盖）。
4. **租户控制台菜单回归（关键）**：`tenant-admin` 的菜单仍出现在 `buildMyMenuTree` 结果中；`platform-admin`（`builtin`）仍被排除。**这是 S1a / S1b 的分水岭测试**，必须显式覆盖，否则重演 §2.2 的空菜单事故。§14 起判据为「内置且非本控制台应用才排除」，两个应用都是 `builtin`，该用例同步改用真实种子编码。
5. **seed 幂等**：二次运行不新增应用行、不覆盖既有 `source`（`getOrCreateApplication` 命中即返回）。
6. **前端**：`application/index.test.tsx` 断言「来源」列渲染与表单提交字段为 `source`。
7. **全仓零残留**：`grep -rn "AppTypeFirstParty\|AppTypeThirdParty\|ApplicationClientTypeFirstParty\|ApplicationClientTypeThirdParty" backend --exclude-dir=.tmp` 无结果；`grep -rn "\"type\"" backend/apps/platformadmin/internal/dto/dto{application,applicationclient}` 无结果；`grep -rn "IsSystem" backend/pkg/iam/model` 无结果。

---

## 8. 落地方式与兼容

### 8.1 API 破坏性

`type` → `source` 同时影响**请求与响应** JSON 字段（`docs/design/application-integration-guide.md:73` 是接入方示例），属破坏性变更，前后端与应用接入方必须同批发布；API 版本仍 `/v1`，不加兼容层（与 `department-container-rename.md` 的处置一致）。

### 8.2 库结构

按新项目处理：**不写迁移脚本、不加回填分支、不 `DROP COLUMN`**。AutoMigrate 只增不删，旧 `type` / `is_system` 列残留属预期（不再被读写），开发/测试库**删库重建**即得目标结构。`application` 与 `application_client` 两张表同批处理。

> **种子数据的 source 由种子自愈（不靠删库）**：旧库补出来的 `source` 取列默认值 `third_party`，而种子数据的两个应用**都不是第三方**（管理后台=内置、租户自服务=第一方）、种子 OAuth 客户端均为内置。故 `pkg/seed` 的两个 upsert 在命中既有行时**幂等回填 source**（只回填 source，不覆盖控制台可改的 name/description/sort/status），与 `getOrCreateTenant` 回填 `status=active` 同一惯例。详见 §12.1/§12.3-4。

### 8.3 存量数据回填（一次性 SQL，已落部署文档）

生产库确需保全数据时，把一次性 SQL 写进部署文档、由执行方在升级前执行，**不塞进启动流程**：

```sql
-- application：is_system(内置) 优先，其次 third_party，其余归 first_party
UPDATE application SET source = CASE
  WHEN is_system THEN 'builtin'
  WHEN type = 'third_party' THEN 'third_party'
  ELSE 'first_party'
END
WHERE source IS DISTINCT FROM (CASE
  WHEN is_system THEN 'builtin'
  WHEN type = 'third_party' THEN 'third_party'
  ELSE 'first_party'
END);

-- application_client：同规则（种子客户端为 is_system=true → builtin）
UPDATE application_client SET source = CASE
  WHEN is_system THEN 'builtin'
  WHEN type = 'third_party' THEN 'third_party'
  ELSE 'first_party'
END
WHERE source IS DISTINCT FROM (CASE
  WHEN is_system THEN 'builtin'
  WHEN type = 'third_party' THEN 'third_party'
  ELSE 'first_party'
END);
```

要点：

- `WHERE source IS DISTINCT FROM <目标值>` 让脚本**幂等**（重复执行 0 行受影响，可安全重跑）。
- **必须在部署新代码之前执行**：新代码不再写 `type`，升级后新建的行其 `type` 只剩列默认值 `'first_party'`，与「改造前就存在的一方应用」不可区分——此时再套用全量映射会把升级后建出来的 `third_party` 行误改成 `first_party`（纯标签错误，不影响 `builtin` 的删除保护）。升级后补做时只做 `is_system = true → builtin` 这一步。
- 该 SQL 与种子自愈（见 §8.2 提示）作用重叠但覆盖面更宽：种子的幂等回填只认自己播种的 2 个应用与种子客户端，**控制台/运维建出来的** `first_party` 行仍靠此 SQL（或删库重建）纠正。二者都不会在启动流程里探测旧列，符合「不引入 `information_schema`/`Migrator()` 判定」的约定。
- 已同步进部署文档 `docs/design/run-and-deploy.md` §2.3（含同一份 SQL 与执行时机说明）；本地开发库的执行记录见 §13.4。

### 8.4 建议执行顺序

`model`（两表枚举 + 删 `is_system`）+ `dao` → `pkg/seed` → `dto`（含 `source` 出参、移除创建入参）+ `service`（固定 `third_party` + 白名单校验 + 错误码改名）→ `tenantadmin` 判据 → 后端测试 → 前端 types/ui/页面 → `docs/design` 四篇 living doc + `frontend/DESIGN.md` → `make swag APP=platformadmin` 重生成 → 前端依赖 API 参考同步（`docs/design/api-reference.md` 若含字段说明）。

**不可拆分**：字段改名与 `is_system` 下线必须同一批，否则出现两个字段同时表达「内置」的中间态；`application` 与 `application_client` 也应同批，避免同值域两名并存。

---

## 9. 决策记录（已定稿）

| # | 决策项 | 结论 |
|---|---|---|
| ① | 值域 | **S1b** 三值 `builtin` / `first_party` / `third_party`（§2.3） |
| ①′ | `first_party` / `third_party` 是否换名 | **不换**：重叠不可消除，换名只损失标准术语与既有值（§2.6） |
| ② | 控制台创建口径 | 只接受 `third_party`；`source` 从 Create/Update 入参移除（推荐）或做白名单（§2.5） |
| ③ | `application_client` 是否同批 | **同批**，独立具名类型 `ApplicationClientSource`（§3） |
| ④ | 错误码 | 常量名 + 文案同步改，**编号 `100746` / `100820` 不动**（§4） |

---

## 10. 工作量

| 面 | 规模 | 说明 |
|---|---|---|
| 后端 | 14 文件 | 2 个枚举定义、3 处判据（含 client）、2 张表 DTO/DAO、seed、错误码改名、白名单校验 |
| 前端 | 8 文件 | types、`SourceTag` 映射、application 列表/表单/测试、oauthClient 页面 2 个 |
| 文档 | 4 篇 living doc | `system-design` / `glossary` / `application-integration-guide` / `organization-container-redesign` |
| 库 | 2 表各 2 列 | 各 1 列改名 + 1 列下线，无迁移脚本 |
| 风险 | 中 | 唯一高风险点是 §7-4 的租户控制台菜单回归 |

---

## 11. 落地前自检（pre-flight）

**触发条件**：工作区里那批与本方案无关的在途改动（`application.visibility` 下线 + 租户应用订阅页）已提交/隔离，`git status --short` 只剩本方案的改动预期。理由：那批改动与本方案**交叉编辑同一批文件**（`model/application.go`、`dtoapplication/*`、`menu.go`、`permission.go`），并发改写会让 §5 的行号漂移，甚至整体覆写导致改动被静默回退。

```bash
# 1) 工作区是否已干净（在途改动已落地）
git status --short

# 2) 行号基准复核：§5 引用的三个锚点
grep -n 'Type .*json:"type"' backend/pkg/iam/model/application.go \
  backend/apps/platformadmin/internal/dto/dtoapplication/request.go \
  backend/apps/platformadmin/internal/dto/dtoapplication/response.go
grep -n "IsSystem" backend/pkg/iam/model/application.go \
  backend/apps/platformadmin/internal/service/svcapplication/application.go
grep -n "app.IsSystem" backend/apps/tenantadmin/internal/service/svctenant/menu.go

# 3) 待下线常量/字段清点（改造后必须全部归零）
grep -rn "AppTypeFirstParty\|AppTypeThirdParty\|ApplicationClientTypeFirstParty\|ApplicationClientTypeThirdParty" \
  --include=*.go backend --exclude-dir=.tmp
grep -rn '"type"' backend/apps/platformadmin/internal/dto/dtoapplication backend/apps/platformadmin/internal/dto/dtoapplicationclient
```

**行号漂移时的处理**：以第 2、3 步的 `grep` 实测结果为准，就地修正 §4/§5 的行号后再动手；`§5` 的文件清单本身（文件名）不受漂移影响，可直接沿用作工作分解。改造完成后再跑一次第 3 步 + §7-7 的零残留检查。

---

## 12. 落地记录（2026-09-12）

### 12.1 实际改动

| 面 | 文件 | 要点 |
|---|---|---|
| model | `pkg/iam/model/application.go`、`application_client.go` | 新增 `AppSource` / `ApplicationClientSource` 具名类型与三值常量 + `IsBuiltin()`；字段 `Source`，列 `source varchar(32) not null default 'third_party'`；删除 `IsSystem` |
| dao | `pkg/iam/dao/application.go`、`application_client.go` | `Cond.Source` + `source = ?` 过滤 |
| 错误码 | `pkg/code/permission.go` | `ApplicationBuiltInErr` / `ApplicationClientBuiltInErr`（编号 `100746` / `100820` 不动），文案「内置应用不可删除」「内置OAuth客户端不可删除」 |
| seed | `pkg/seed/seed.go` | `getOrCreateApplication(..., source model.AppSource)`；管理后台与租户自服务都→`builtin`（§14 修正，初版曾把租户自服务落 `first_party`）；种子客户端→`ApplicationClientSourceBuiltin`（顺带修掉客户端误用 `AppStatusEnable` 的常量）；**两处 upsert 命中既有行时幂等回填 `source`**（§8.2 提示、§12.3-4） |
| DTO | `dtoapplication/{request,response}.go`、`dtoapplicationclient/{request,response}.go` | 入参删除 `type`/`source`；出参 `source`；分页过滤 `Source` 用具名类型 |
| service | `svcapplication/application.go`、`svcapplicationclient/application_client.go` | Create 固定 `third_party`；Update 移除 `type`；Delete 判 `Source.IsBuiltin()`；PageList 白名单校验非法 source；Detail 顺带按规范拆开 `err != nil` 与业务边界判断 |
| tenantadmin | `svctenant/menu.go` | `app.Source.IsBuiltin() && app.Code != tenantAdminAppCode`（§14 修正）+ 注释同步 |
| 测试 | 7 个后端测试文件 + 2 个前端测试文件 | 见 §12.2 |
| 前端 | `types/platform.ts`、`ui/status.tsx`、application 页面、oauthClient 页面 2 个 | 新增 `AppSource` 类型；`SourceTag` 提升为应用/客户端/角色来源共用映射；列表列「类型」→「来源」；表单移除下拉，编辑态只读展示来源 |
| 文档 | `system-design.md`、`glossary.md`、`application-integration-guide.md`、`frontend/DESIGN.md` | 表结构/术语/接入示例/分类色规范同步 |
| 生成物 | `apps/platformadmin/docs/platformadmin_docs.go` | `make swag APP=platformadmin` 重生成，`is_system`/`isSystem`/`isThirdParty` 零残留 |

### 12.2 验证

```bash
# 后端：5 模块编译 + 测试（工作区沙箱下需把 GOCACHE 指向仓内 .tmp/gocache）
cd backend && GOCACHE=$PWD/.tmp/gocache go build ./...   # pkg / auth / platformadmin / tenantadmin / gateway 全绿
go test ./...                                            # platformadmin / tenantadmin / auth / pkg 全绿
# 新增/改写的回归用例
#  - svcapplication：builtin 拒删、first_party 与 third_party 可删（回归：删 is_system 后一方应用不得被误判为内置）
#  - svcapplication：PageList 按 source 过滤命中唯一、非法 source 返回错误码、Create 强制 third_party
#  - svcapplicationclient：builtin 拒删、third_party 可删
#  - svctenant：订阅的 builtin 应用仍可作为角色归属；控制台菜单范围排除**其它控制台**的内置应用、但保留租户自服务自身（code=tenant-admin 是本控制台菜单载体，§14）
#  - pkg/seed：TestSeedIamBackfillsApplicationSource —— 把两应用与种子客户端的 source 降级成 third_party 后，
#    再跑种子必须原地回填为 builtin/builtin/builtin（两个应用 + 两个客户端，§14），且不新建记录、可反复幂等
cd frontend && pnpm run typecheck && pnpm run test        # typecheck 0 error；9 文件 42 测试通过
# 零残留
grep -rn "IsSystem\|AppTypeFirstParty\|ApplicationClientType\|SystemBuiltInErr\|is_system" --include=*.go backend  # 仅剩 1 处解释性注释
```

`docs/design/api-reference.md` 经检索不含 `application.type` / `is_system`，无需同步。

### 12.3 遗留与后续

1. ~~**`application_client.is_third_party`（bool）未下线**~~ → **已下线（见 §13）**：它是继 `type`（已改名 `source`）与 `is_system`（已删）之后的**第三处归属列**，与 `source=third_party` 语义重复，且写入链路已事实失效（前端没有任何表单项/调用点提交 `isThirdParty`，控制台路径上恒为 `false`），故单开一批全链路下线。
2. **控制台不再能创建 `first_party` 应用**（决策 ②的直接结果）：`Create` 固定 `third_party`；若将来需要「平台自有但非内置」的应用，应补运维/种子入口，而不是放开入参。
3. **`organization-container-redesign.md:359` 保留原文**：该文自述「本文其余内容保持发布时的原貌作为历史记录」，且其列举描述的是当时已执行的布尔化改造，故不回改（本方案取代其口径，见文首引言）。
4. **存量库的种子数据靠种子自愈，不强制删库**：`AutoMigrate` 只增不删，旧库补出的 `source` 取列默认值 `third_party`——若不纠正，管理后台会丢删除保护、其菜单还会串进租户控制台，租户自服务则被误标为第三方。`pkg/seed` 已按 §8.2 提示在命中既有行时幂等回填（只回填 `source`，不覆盖控制台可改字段），启动即收敛；覆盖不到的（运维自建的 `first_party` 行）用 §8.3 SQL 或删库重建。回归见 §12.2 的 `TestSeedIamBackfillsApplicationSource`。

---

## 13. 收尾批次：`is_third_party` 下线（2026-09-12）

承接 §12.3-1。`application_client` 上原本**三处**都在表达同一件事（`type`、`is_third_party`、`is_system`），本批把第三处也摘掉，最终只留 `source` 一列一事。

### 13.1 动的东西

| 面 | 文件 | 改动 |
|---|---|---|
| model | `pkg/iam/model/application_client.go` | 删除 `IsThirdParty` 字段（列 `is_third_party`） |
| DTO | `dtoapplicationclient/request.go`、`response.go` | Create/Update 入参删除 `isThirdParty`；Detail 与 PageListItem 出参删除 `isThirdParty` |
| service | `svcapplicationclient/application_client.go` | 删除 Create 赋值、UpdateMap 的 `is_third_party`、Detail/PageList/GetByClientID 的出参映射（共 5 处） |
| 前端 | `types/platform.ts`、`oauthClient/Detail.tsx` | `OAuthClientItem` 与两个 Req 类型删除 `isThirdParty`；详情抽屉删除「是否第三方」行（`source` 已表达同一信息） |
| 文档 | `system-design.md` | ER 图删除 `application_client.is_third_party`（§5 记为「应删的过期残留」，本批兑现） |
| 生成物 | `apps/platformadmin/docs/platformadmin_docs.go` | `make swag APP=platformadmin` 重生成，`isThirdParty` 计数归 0 |

**不动的**：`docs/superpowers/**` 的历史 plan/spec（历史记录保留原文，见 §0.2）；`organization-container-redesign.md:359`（自述为历史记录，见 §12.3-3）。

### 13.2 验证

```bash
grep -rn "IsThirdParty\|is_third_party\|isThirdParty" backend frontend --include=*.go --include=*.ts --include=*.tsx   # 代码零残留
grep -c isThirdParty backend/apps/platformadmin/docs/platformadmin_docs.go                                            # 生成物 0
# 后端 5 模块 go build ./... + go test ./... 全绿；前端 pnpm run typecheck 0 error、pnpm run test 9 文件 42 测试通过
```

### 13.3 兼容与数据

- **API 破坏性**：客户端 DTO 少一个 `isThirdParty` 字段（请求与响应均是）。该字段前端从来没有提交入口（写入链路恒为 `false`），故 **IDP/接入方无实际依赖**；与 `type`→`source` 同属一批破坏性变更，前后端仍须同批发布。
- **库结构**：按新项目处理，不写 `DROP COLUMN`；旧库的 `is_third_party` 残留列不再被读写，属预期（§8.2）。该列**无需回填**——信息已由 `source` 完整承载。
- **判据收敛完成**：客户端「是否内置」=`source.IsBuiltin()`、「是否第三方」=`source == third_party`，不再有第二个事实源。

### 13.4 既有数据回填执行记录（本地开发库）

`application-source-rename.md` §8.3 的一次性 SQL 已在本地开发库（`postgres://postgres@127.0.0.1:5432/iam`）执行并验证。

**执行前**（`source` 全是 AutoMigrate 补列的默认值 `third_party`，且旧列仍在）：

| 表 | code | 旧列 | source（前） | source（后） |
|---|---|---|---|---|
| `application` | `platform-admin`（管理后台） | `is_system=true` | `third_party` ❌ | `builtin` ✅ |
| `application` | `tenant-admin`（租户自服务） | `type=first_party` | `third_party` ❌ | `first_party` →（§14 再修正为）**`builtin`** ✅ |
| `application_client` | `platform-admin-web` | `is_system=true` | `third_party` ❌ | `builtin` ✅ |
| `application_client` | `tenant-admin-web` | `is_system=true` | `third_party` ❌ | `builtin` ✅ |

**结果**：`application` 回填 2 行、`application_client` 回填 2 行；幂等复跑影响 0 行。

**不改的**：旧列 `application.type` / `is_system` / `visibility`、`application_client.type` / `is_system` / `is_third_party` 仍留在库里（AutoMigrate 只增不删，不再被读写，属预期，§8.2）。要让库结构与 model 完全一致，按 §2.3 删库重建即可——本批**不做** `DROP COLUMN`。

**旁证**：该库执行前的状态正好验证了 §12.3-4 的判断——`source` 补列后若不回填，管理后台会以 `third_party` 落库（丢删除保护、菜单还会串进租户控制台）。

---

## 14. 归属再修正：两个种子应用都归 `builtin`（2026-09-12）

§2.3 的「零行为变更」映射把**租户自服务**留在 `first_party`，理由是绕开 `loadConsoleApps` 对 `builtin` 的一刀切排除（§2.2 的空菜单事故）。这个绕法本身没错，但它让**一个字段同时承担两件事**：租户自服务明明是平台随产品交付的控制台应用，却要靠「不是内置」来换取菜单可见性。

本批把两件事拆开：**内置性（`source`）只决定「是否受删除保护」，「菜单归哪个控制台」由应用归属判定。**

### 14.1 定稿口径

| 应用 | code | `source`（§2.3 旧） | `source`（现在） | 行为变化 |
|---|---|---|---|---|
| 管理后台 | `platform-admin` | `builtin` | `builtin` | 无 |
| 租户自服务 | `tenant-admin` | `first_party` ❌ | **`builtin`** ✅ | 由「可删」变**禁删**；菜单仍在租户控制台（不变） |

1. **`source` 收敛职责**：只表达归属与内置性，`builtin` = 平台随产品交付、禁删；删除保护对两个控制台应用一致（删掉 `tenant-admin` 等于所有租户的控制台凭空消失）。
2. **菜单范围改按控制台判定**：`svctenant.loadConsoleApps` 改为 `app.Source.IsBuiltin() && app.Code != tenantAdminAppCode` 才排除，即「内置应用的菜单归各自专属控制台，但**本控制台自己的应用必须留下**」。这与 platformadmin 侧用 `platformAdminAppCode = "platform-admin"` 锁定平台菜单（`svcpermission/menu.go:293`）**同源**：菜单归属看应用编码，不看内置性。
3. **`first_party` 目前无生产者**：种子不再产生它，控制台只能建 `third_party`；枚举值与白名单保留，供运维自建「平台自有但非内置」的应用（如为客户定制的内部系统）。`IsPlatformOwned()` 也正是为这类判断预留的。

### 14.2 与既往记录的关系（读旧章节时的口径）

- §2.3「零行为变更」不再成立：本批**有**行为变更（`tenant-admin` 变禁删），但**租户控制台菜单范围不变**——由 `TestConsoleMenuScopeExcludesBuiltInApp` 显式覆盖。
- §2.5 表中 `first_party → 现有对应：租户自服务` 改读为「无（仅运维自建）」；`builtin` 一行的「控制台菜单归属」改读为「各自所属控制台」。
- §7 的回归口径由「排除 builtin、保留 first_party」改读为「排除其它控制台的内置应用、保留租户自服务」。
- §8.3 / §13.4 的一次性 SQL **无需改动**：它只服务升级前的存量行，两个种子应用随后由 `pkg/seed` 收敛为 `builtin`（§14.3）。

### 14.3 改动与验证

| 面 | 改动 |
|---|---|
| `pkg/seed/seed.go` | 租户自服务 → `model.AppSourceBuiltin`（种子自愈的幂等回填同步生效） |
| `svctenant/menu.go` | 新增 `tenantAdminAppCode`；菜单范围判据改为「内置且非本控制台应用才排除」 |
| `model/application.go` | `AppSource` 注释改为「两个控制台应用都落 `builtin`；`first_party` 留给运维自建」 |
| 测试 | `TestSeedIamBackfillsApplicationSource` 断言两应用 + 两客户端都回填 `builtin`；`TestConsoleMenuScopeExcludesBuiltInApp` 用真实种子编码建两个 `builtin` 应用，断言菜单树只保留租户控制台菜单；`TestDeleteFirstPartyApplication` 与分页过滤用例的 fixture 由 `tenant-admin` 改为运维自建应用（`first_party` 仍须覆盖可删分支） |
| 文档 | 本节 + `glossary.md`（内置/第一方应用）+ `run-and-deploy.md` §2.3 + 前端 `types/platform.ts` 注释 |
| 存量库 | 本地开发库执行 `UPDATE application SET source='builtin' WHERE code='tenant-admin'`（幂等）：修正 1 行，复跑 0 行；两客户端本就是 `builtin` |

验证：后端 5 模块 `go build ./...` + `go test ./...` 全绿；前端 `pnpm run typecheck` 0 error、`pnpm run test` 9 文件 42 测试通过。

---

## 15. 应用编码规则统一为下划线连接（2026-09-12）

### 15.1 规则与映射

应用编码（`application.code`）统一为**下划线连接**，并固化为单一事实源 `model.AppCodePattern`：

```go
// 小写字母开头，仅含小写字母/数字/下划线
const AppCodePattern = `^[a-z][a-z0-9_]*$`
```

口径与自动生成的租户编码一致（`pkg/iam/tenant.GenerateCode` → `t_<12 位 hex>`，见 `pkg/iam/tenant/code.go`）。此前两个种子应用用连字符，既与租户编码风格不一，也还需要在 URL path / scope 串等位置额外转义。

| 应用 | 旧编码（连字符） | 新编码（下划线） |
|---|---|---|
| 管理后台 | `platform-admin` | `platform_admin` |
| 租户自服务 | `tenant-admin` | `tenant_admin` |

**范围**：本次只动 `application.code`。菜单编码（`grp-tenant` / `tenant-application` 等）与 OAuth `client_id`（`platform-admin-web` / `tenant-admin-web`）不在本批，保持原样。

### 15.2 代码改动

| 面 | 改动 |
|---|---|
| `pkg/seed/seed.go` | `appCodeAdmin` / `appCodeTenantAdmin` 改下划线；新增 `appCodeAdminLegacy` / `appCodeTenantAdminLegacy`；`getOrCreateApplication` 增加 `legacyCode` 入参，新编码缺席而旧编码命中时**原地改名**（保留主键） |
| `pkg/iam/tenant/provision.go` | `ProvisionAppCode` → `tenant_admin`（建租户链路的订阅/内置角色/菜单授权一并跟随） |
| `svcpermission/menu.go`、`svctenant/menu.go` | `platformAdminAppCode` / `tenantAdminAppCode` 同步改下划线 |
| `model/application.go` | 新增 `AppCodePattern` + `IsValidAppCode`（编码规则单一事实源） |
| `pkg/code/permission.go` | 新增 `ApplicationCodeInvalidError = 100748`（文案「应用编码格式不正确（以小写字母开头，仅含小写字母、数字与下划线）」） |
| `svcapplication.Create` | 入口按 `model.IsValidAppCode` 校验，非法编码返回 `ApplicationCodeInvalidError`，不落库 |
| `platform-admin-web` 应用表单 | `code` 规则加 `pattern`（与后端同口径），placeholder 改 `iam_web` |

### 15.3 既有数据（存量库）

编码是应用的**业务唯一键**，改名用原地 `UPDATE` 即可：菜单、租户订阅、角色全部以 `app_id` 关联，不随编码变化。

- **种子自愈（默认路径）**：`pkg/seed` 启动时对两个内置应用执行旧编码 → 新编码的原地改名（保留主键），随后照常回填 `source`；改名后再次启动走新编码分支，故改名前后都可反复幂等执行。
- **一次性 SQL**（生产库需保全数据、且必须在**升级前**执行，与 §8.3 同一约定）：

```sql
UPDATE application SET code = 'platform_admin' WHERE code = 'platform-admin';
UPDATE application SET code = 'tenant_admin'   WHERE code = 'tenant-admin';
```

- **时序要求**：先改名再上代码。若旧库直接跑新代码，种子会按新编码另建一套内置应用，而旧应用仍占着 `platform-admin` / `tenant-admin` 唯一键，导致菜单、订阅、删除保护全部落在两套应用上。
- **新旧编码并存时中断启动（fail closed）**：若库里同时存在 `platform_admin` 与 `platform-admin`（例如旧库曾被自建过同名应用），`getOrCreateApplication` 直接报错、不做静默择一——否则按新编码命中后回填 `source`，会把用户自建应用改写成内置应用（获得删除保护并接管菜单范围）。人工确认哪一行是内置应用、删除另一行后再启动。

### 15.4 验证

| 用例 | 断言 |
|---|---|
| `TestSeedIamMigratesLegacyApplicationCode` | 预置 `platform-admin` 应用及其名下菜单，种子跑两遍后：应用总数仍为 2、旧编码 0 行、新编码行主键与旧行一致、`source` 一并回填 `builtin`、菜单仍挂原 `app_id` |
| `TestSeedIamRejectsConflictingApplicationCode` | 新旧编码并存时 `SeedIam` 返回错误，且用户自建应用仍为 `third_party`、旧编码行未被改名/删除 |
| `TestApplicationCreateRejectsInvalidCode` | `my-app` / `My_App` / `1app` / `app.web` / `app web` / `_app` / `""` 均返回 `ApplicationCodeInvalidError` 且不落库 |
| `TestApplicationCreateAcceptsUnderscoreCode` | `my_app_2` 落库成功，`source` 恒为 `third_party` |
| `platform-admin-web/src/pages/application/index.test.tsx` | 连字符编码被表单拦截（不发创建请求），改 `my_app` 后提交成功 |

> **读旧章节的口径**：§12、§14 中出现的 `platform-admin` / `tenant-admin` 是那两批改造时点的编码，按 §15.1 映射换算为 `platform_admin` / `tenant_admin`；`client_id`（`*-web`）当时与现在都未变。
