# 编码风格对齐方案（gocli 模板 vs 当前代码）

> 状态：✅ **已执行完毕**（2026-08 完成，§四～§七 全部落地，AGENTS.md 已固化新约定）
> 背景：项目最初基于 `gocli/cmd/generate/template/module` 的编码风格，演进过程中出现了两类"偏离"：
> ① 因架构演进（单体拆分为 auth/platformadmin/tenantadmin/gateway 四应用）产生的**结构性变化**；
> ② 演进过程中产生的**内部自相矛盾**。本文只针对 ② 给出统一方案，① 属于合理演进不回退。

## 一、结论

| 类别 | 是否需要调整 | 说明 |
|---|---|---|
| 与模板一致的骨架（分层/接口/日志/错误码/DAO） | ❌ 不动 | 保留得很好，见 §二 |
| 结构性演进（pkg/iam 抽取、领域模块、错误码分段、路由应用标识段、Repository 模式） | ❌ 不动 | 合理演进且已写入 AGENTS.md，见 §三 |
| **内部不一致（DTO 命名 / JSON ID 大小写 / ID 类型 / DAO 注入风格）** | ✅ **需要统一** | 见 §四～§七 |
| 模板自身缺陷（如响应带 binding） | ⚠️ 若继续用 gocli 生成则需同步升级模板 | 见 §八 |

## 二、与模板一致、保持不变的部分（骨架）

- 分层：`internal/controller/ctrXxx`、`internal/service/svcXxx`、`internal/dto/dtoXxx`、`internal/router`
- Service：接口 + `xxxSvc struct{}` + `var _ XxxSvc = (*xxxSvc)(nil)` + `NewXxxSvc()`
- Controller：统一 `ctx.ShouldBindJSON/ShouldBindQuery` + `gincontext.Fail/Success`
- 日志：`glog.Errorf(ctx, "[svcxxx.Method] ... err:%v, req:%s", err, gutil.ToJsonString(req))`
- 错误：`code.GetError(code.XxxError)`
- Model：嵌入 `gorm.Model` + `TableName()` + `XxxEntityList` + `ToMap()`
- DAO：`gormdao.Dao[...]` + `XxxCond` + `BuildCondition`

## 三、结构性演进、不回退的部分

1. **model/dao/object 上移 `pkg/iam`**：四应用共享数据层（AGENTS.md 明文规定）。
2. **按业务领域而非单表划分模块**：`svcpermission`/`ctrpermission` 内含 role/menu/scope/resource；`svcuser` 内含 user/user_identity/user_login_log/user_department。
3. **错误码集中 `pkg/code` 并按领域分段**：tenant(1001XX-1004XX)、user(1005XX-1008XX)、permission(1006XX-1009XX)、auth(1010XX-1011XX)，`registerError` 防重。
4. **路由带应用标识段**：`/v1/platform/user/create`（auth→/v1/auth、tenant→/v1/tenant）。
5. **Repository 模式**（`var newXxxRepo = func()` + adapter）：23 个 service 文件在用，测试通过替换函数变量注入 fake（如 `user_object_scope_test.go`），是**为可测性做的有意增强**，保留。

## 四、待统一 ①：DTO 命名（优先级：高，纯后端改动，无契约影响）

### 现状：三种风格并存

| 风格 | 数量 | 代表 |
|---|---|---|
| 模板风格 `<名词><动词>Req` | 85 | `UserCreateReq`、`TenantCreateReq`、`RoleCreateReq` |
| 动词前置 `Create<名词>Req` | 7 | `CreateDomainReq`、`CreateApiKeyReq`、`CreateSecretReq`… |
| 裸名 `CreateReq` | 5 | `dtoapplication`、`dtotenantapplication`、`dtoapplicationclient` 的 CRUD |

### 目标规范

统一为模板风格：`<业务名词><动词>Req/Resp`，如 `UserCreateReq`。DTO 类型名与所在包名呼应。

### 改动清单（文件 → 新类型名）

**动词前置 → 模板风格：**

| 文件 | 现状 | 目标 |
|---|---|---|
| `apps/platformadmin/internal/dto/dtoapikey/request.go` | `CreateApiKeyReq`、`DeleteApiKeyReq` | `ApiKeyCreateReq`、`ApiKeyDeleteReq` |
| `apps/platformadmin/internal/dto/dtoapikey/response.go` | `CreateApiKeyResp` | `ApiKeyCreateResp` |
| `apps/platformadmin/internal/dto/dtoapplicationclient/request.go` | `CreateSecretReq`、`DeleteSecretReq` | `SecretCreateReq`、`SecretDeleteReq` |
| `apps/platformadmin/internal/dto/dtoapplicationclient/response.go` | `CreateSecretResp` | `SecretCreateResp` |
| `apps/platformadmin/internal/dto/dtodomain/request.go` | `CreateDomainReq`、`UpdateDomainReq`、`DeleteDomainReq` | `DomainCreateReq`、`DomainUpdateReq`、`DomainDeleteReq` |

**裸名 → 加业务名词前缀：**

| 文件 | 现状 | 目标 |
|---|---|---|
| `dtoapplication/request.go` | `CreateReq`、`UpdateReq`、`DetailReq`、`DeleteReq`、`PageListReq` | `ApplicationCreateReq`… |
| `dtoapplication/response.go` | 对应 `Resp` | `Application*Resp` |
| `dtotenantapplication/request.go` | 同上 | `TenantApplicationCreateReq`… |
| `dtotenantapplication/response.go` | 对应 `Resp` | `TenantApplication*Resp` |
| `dtoapplicationclient/request.go` | 主 CRUD 的裸名 | `ApplicationClientCreateReq`… |
| `dtoapplicationclient/response.go` | 对应 `Resp` | `ApplicationClient*Resp` |

### 联动修改

- 对应 `svcXxx` 接口签名、`ctrXxx` 绑定、Swagger `@Param` 注解（`svcdomain`、`svcapikey`、`svcapplication`、`svcapplicationclient`、`svctenantapplication`）。
- 前端 TS 类型通常只消费 JSON 字段名，DTO 类型名不影响前端。

## 五、待统一 ②：JSON 字段 ID 大小写（优先级：最高，API 契约）

### 现状：同一代码库大小写混用

| JSON tag | 出现次数 | JSON tag | 出现次数 |
|---|---|---|---|
| `appId` | 15 | `appID` | 0（Go 字段为 `AppID` 的 json 用 `appID`? 需核对） |
| `connectorId` | 10 | `tenantAppId` | 5 |
| `applicationClientId` | 9 | `userId` | 2 |
| `roleId` | 7 | `sessionId` | 2 |
| `tenantId` | 6 | `userIds`/`appIds`/`secretId`/`factoryId`/`clientId` | 各 1 |

前端已同时出现 `userID`(100) / `userId`(37)、`roleID`(36) / `roleId`(9) 的适配，说明契约已产生摩擦（如 `frontend/apps/platform-admin-web/src/pages/role/index.tsx` 请求用 `roleID`，后端部分接口要求 `roleId`）。

### 目标规范

统一为 **ID 全大写**：`userID`、`roleID`、`tenantID`、`appID`、`connectorID`、`sessionID`、`applicationClientID`、`tenantAppID`、`secretID`、`factoryID`、`clientID`、`userIDs`、`appIDs`。

### 改动范围（后端）

1. **Go struct 字段名 + json tag 同步改**：涉及
   - `apps/auth/internal/dto/dtoconnector/*`、`dtoauth/*`、`dtouser/session.go`
   - `apps/platformadmin/internal/dto/dtouser/*`、`dtopermission/*`、`dtoapplication/*`、`dtoapplicationclient/*`、`dtotenantapplication/*`、`dtoapikey/*`、`dtodomain/*`
   - `apps/auth/internal/service/svcauth/connector_state_store.go`（`tenantID`/`connectorID` tag）
   - 含 Go 字段名错误写法 `AppId`（`dtoapplicationclient/request.go`）
2. **Swagger 注解与路由 path 参数名**：`@Param connectorId/sessionId/roleId/userId/secretId` 及 path `{connectorId}`/`{sessionId}`/`{roleId}/{userId}`/`{secretId}` 一并统一（若改 URL path，属破坏性变更，需与前端联调；path 参数名也可保持小写以减小改动面，但需在方案中明确取舍）。
3. **前端联动**：`frontend/` 中 `userId`→`userID`、`roleId`→`roleID` 等（约 37+9 处引用），e2e 用例同步。

### 验证

- 后端：`make build APP=gateway`、`go test ./...`
- 契约：跑 e2e（`e2e/` 目录）确保前后端字段对齐
- 用脚本断言：`grep -rnE "json:\"[a-zA-Z]*(Id|Ids)\"" apps --include="*.go"` 结果为空

## 六、待统一 ③：ID 类型 uint vs uint64（优先级：中）

### 现状

所有 model 均嵌入 `gorm.Model`（ID 为 `uint`），但 DTO 层混入 `uint64`：

| 文件 | 字段 |
|---|---|
| `apps/auth/internal/dto/dtouser/session.go` | `SessionID`、`ID`、`AppID`、`TenantID` 为 `uint64` |
| `apps/auth/internal/dto/dtoconnector/request.go` | `ConnectorID uint64` |
| `apps/platformadmin/internal/dto/dtouser/request.go` | `RoleID uint64`(多处)、`UserID uint64`、`UserIDs []uint64`、`AppIDs []uint64` |
| `apps/platformadmin/internal/dto/dtouser/response.go` | `UserID`、`RoleID`、`AppID` 为 `uint64` |
| `apps/platformadmin/internal/dto/dtoapplicationclient/request.go` | `SecretID uint64` |
| `apps/platformadmin/internal/dto/dtoapplicationclient/response.go` | `ID`、`ApplicationClientID` 为 `uint64` |

### 目标规范

统一为 `uint`（与 `gorm.Model.ID` 一致），同时把 `RoleUserListReq`、`AssignRoleUsersReq`、`RemoveRoleUserReq` 等 role 域 DTO 从 `uint64` 拉回 `uint`。

### 风险

`uint64` 若曾暴露给前端大数场景（如雪花 ID）需先确认无超 `uint` 范围数据；当前自增 ID 场景无此问题。

## 七、待统一 ④：DAO 注入风格（优先级：低，改动有测试成本）

### 现状：同一文件内混用

- 直接调用：`dao.NewUserDao()`（svcuser.Create、PageListUserLoginLog 等）
- Repository 函数变量：`newUserObjectScopeRepo()`（svcuser.Delete/Update/Detail 等）
- 构造函数注入 struct 字段：svcauth 的 `connectorSvc`（复杂依赖，保留）

### 目标规范（建议）

1. **纯 CRUD、无单测**的服务：直接 `dao.NewXxxDao()`（模板风格，最简）。
2. **有单测**的服务：Repository 接口 + `var newXxxRepo = func()`（当前主流，测试可替换）。
3. **同一服务内禁止混用**：svcuser 需统一为"有单测 → 全部走 repository"（现有 5 个 repo 接口已覆盖 Create/Delete/Update/Detail/PageList 主链路，把 Create 里的 `dao.NewUserDao()`/`dao.NewPersonDao()` 也收口）。
4. svcauth 的 struct 字段注入作为复杂服务特例，在 AGENTS.md 中补充说明即可。

## 八、模板同步升级（若继续用 gocli 生成）

`gocli/cmd/generate/template/module/` 落后于实际主流风格，直接生成的代码会再次引入偏差：

| 模板文件 | 问题 | 建议 |
|---|---|---|
| `response.go.tpl` | `DetailResp`/`PageListItem` 带 `binding:"required"`（响应不该有 binding） | 去掉 binding |
| `service.go.tpl` | Update 生成**空 updateMap**；Delete 在 service 层取 `gincontext.GetUserID` 语义模糊 | 按 svcdomain 的实际写法升级 |
| `router.go.tpl` | 路由路径无应用标识段 `/v1/{AppName}/{module}/...` | 支持 `/v1/{service}/{module}/...` |
| `controller.go.tpl` | `@Router` 同样缺应用标识段 | 同上 |
| 模板整体 | model/dao/object 生成在 app 内，不支持 `pkg/iam` 共享层；错误码不支持领域分段 | 增加"共享层模式"开关 |

## 九、执行顺序建议

1. **§五 JSON ID 大小写**（契约收益最大，需前后端+e2e 联动）→ 单 PR
2. **§四 DTO 命名**（纯后端，机械替换，可用 IDE 重命名）→ 单 PR
3. **§六 uint64→uint**（配合 §五 一并处理 role 域契约）→ 可并入 1
4. **§七 DAO 注入收口**（改 svcuser 并补测试）→ 独立 PR
5. **§八 模板升级**（可选，与下次 gocli 生成需求绑定）

每步完成标准：`make build APP=gateway` 通过、`go test ./...` 通过、`make lint` 通过、e2e 全绿。
