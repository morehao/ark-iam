# 客户端编码（client_id）改为创建时必填 + 前后端双重校验

> 状态：**已落地**（2026-09-12）
> 决策日期：2026-09-12
> 场景/档位：**S1 约定收敛 + 入参校验** · 轻量档 · 数据约定 + 接口契约 + 迁移手法
> 一句话结论：`application_client.code`（= OIDC `client_id`）此前**不在创建入参里**——由服务端随机生成 `uuid`（含连字符），库里因此出现连字符 client_id，且没有任何一处校验它的合法性；改为**由创建方填写**、前端表单与后端 service 各校验一份（`^[a-z][a-z_]*$`：小写字母开头，仅小写字母与下划线），并把内置 client_id 收敛为 `pkg/model` 单一常量源。
> ⚠️ **可改性补充**（2026-09-12 同日后继改造）：本文只定了"创建必填 + 校验"；随后按"谁按 code 认行"重新划线——**用户自建客户端的 `code` 现在可改**（`ApplicationClientUpdateReq.Code`），**内置客户端仍只读**（网关 aud 信任边界 + 改名会自我锁死）。菜单与**自建**应用/客户端的 `code` 一并放开；**内置应用与内置客户端的 `code` 保持只读**。见 [seed-identity-key-20260912.md](seed-identity-key-20260912.md)。

---

## 背景与现状

### 现象

| 问题 | 现状 |
|------|------|
| 内置客户端编码含连字符 | `platform-admin-web` / `tenant-admin-web`（同一份种子里，应用编码却是下划线形态 `platform_admin`） |
| 创建接口不接收 `code` | `ApplicationClientCreateReq` 无 `Code` 字段；`generateClientCode()` 直接返回 `uuid.New().String()`——**必然含连字符** |
| 没有任何校验 | 应用创建有 `IsValidAppCode` + `ApplicationCodeInvalidError`；客户端这条链路**一处校验都没有**，规则只写在注释里 |
| 文档口径自相矛盾 | `application_client.go` 与 `sso-oidc-concepts.md` §3.2 都写着"客户端编码**不适用** `AppCodePattern`、**允许连字符**" |

### 为什么必须校验而不是只改数据

`client_id` 不只是标识符，它**同时是令牌 audience**：`platformadmin` / `tenantadmin` 的 aud 白名单、back-channel logout 的客户端识别、前端 `VITE_OIDC_CLIENT_ID` 都必须与它逐字一致。写错一个字符的后果不是"显示不好看"，而是**该客户端签发的令牌全部 401**。这类值必须在入口拦下，而不是落库后靠人工纠正。

---

## 决策

> **`application_client.code` 由创建方填写（必填），前端与后端各校验一份，规则为 `^[a-z][a-z_]*$`：小写字母开头，仅含小写字母与下划线（禁数字、禁连字符）。**
>
> 内置 client_id 收敛为 `pkg/model` 的种子身份常量，种子、网关、前端共用同一份声明；存量库的连字符编码由种子启动时原地改名。

### 与 `AppCodePattern` 的关系

两者是**同一种做法的两套规则**，刻意不同：

| | 应用编码 `application.code` | 客户端编码 `client_id` |
|---|---|---|
| 正则 | `^[a-z][a-z0-9_]*$`（`AppCodePattern`） | `^[a-z][a-z_]*$`（`ClientCodePattern`） |
| 数字 | 允许（`my_app_2`） | **不允许** |
| 生成方 | 创建方填写 | 创建方填写（本次改造前为服务端随机生成） |

客户端更严是有意的：它是协议标识符且进 aud 白名单，取值越可预测越好。**先严后宽是向后兼容的**，反之会弄脏既有数据。

```go
// pkg/model/application_client.go
const ClientCodePattern = `^[a-z][a-z_]*$`
func IsValidClientCode(code string) bool
```

---

## 详细设计

### 1. 入参（后端）

```go
// dtoapplicationclient/request.go
type ApplicationClientCreateReq struct {
    AppID string `json:"appID" binding:"required"`
    Code  string `json:"code" binding:"required"`   // 新增：= OIDC client_id
    Name  string `json:"name" binding:"required"`
    ...
}
```

`Update` 不接受 `code`（与 `application` 一致：编码是定位键，创建后不可改）。

### 2. 校验（service 入口，与 `svcapplication.Create` 同款）

```go
if !model.IsValidClientCode(req.Code) {
    glog.Errorf(ctx, "[svcapplicationclient.Create] 非法客户端编码, req:%s", gutil.ToJsonString(req))
    return nil, code.GetError(code.ApplicationClientCodeInvalidError)
}
```

- 新增错误码 `ApplicationClientCodeInvalidError = 100822`（`1008xx` 客户端号段；`100821` 上一批下线，编号不复用）；
- 校验归 service（`AGENTS.md` 硬规则 3）：DTO 绑定的是前端原始字符串，非法值在此拦截并返回功能级错误码。

### 3. 校验（前端表单，与后端同口径）

```tsx
// platform-admin-web/src/pages/oauthClient/index.tsx
const CLIENT_CODE_PATTERN = /^[a-z][a-z_]*$/

<Form.Item
  name="code"
  label="客户端编码"
  tooltip="即 OIDC 的 client_id，创建后不可修改"
  rules={[
    { required: true, message: '请输入客户端编码' },
    { pattern: CLIENT_CODE_PATTERN, message: '以小写字母开头，仅含小写字母与下划线' },
  ]}
>
  <Input placeholder="唯一编码，如 iam_client" />
</Form.Item>
```

- 仅在创建态渲染（与服务端契约一致：`Update` 不接受 `code`）；
- 正则无法在前后端之间共享（跨语言），因此**两处各写一份并在注释里互相指名**：改一处必须同步另一处，回归用例覆盖两侧。

### 4. 删除服务端生成

`clientCodePrefix` / `generateClientCode()` 一并删除（`Code` 全部来自入参）。这也顺带消除了"生成逻辑必然产出不合规值"的隐患：`uuid` 原文含连字符，而 `cli_` + 去连字符 hex 又含数字，两条路在 `^[a-z][a-z_]*$` 下都不合法——**随机生成与严格字符集天然冲突**，所以改为调用方填写是唯一自洽解。

### 5. 内置 client_id 的单一常量源（`pkg/model/seed_authority.go`）

```go
const (
    SeedBuiltinClientPlatformAdminWeb = "platform_admin_web"
    SeedBuiltinClientTenantAdminWeb   = "tenant_admin_web"
)
```

消费方（改完后零字面量）：

| 消费方 | 位置 | 用途 |
|--------|------|------|
| 种子 | `pkg/seed/seed.go` | 定位/创建/改名内置客户端（入口先按 `ClientCodePattern` 校验） |
| 平台管理网关 | `apps/platformadmin/app.go` | aud 白名单 + back-channel logout 客户端识别 |
| 租户管理网关 | `apps/tenantadmin/app.go` | 同上 |
| 前端默认值 | `apps/*-web/src/main.tsx` | `VITE_OIDC_CLIENT_ID` 未设置时的 `clientID`（跨语言，只能同值字面量） |

### 6. 存量库迁移（`pkg/seed`）

内置客户端编码是种子的查重键：不迁移就会"查不到旧行 → 新建两行"，库里出现 4 行、旧行上的令牌引用失联。按既有 legacy 模式（`tenantCodePlatformLegacy`、`appCodeAdminLegacy`）处理：

```go
switch {
case legacy == nil:      // 正常路径：全新库或已迁移过
case entity != nil:      // 新旧编码并存 → 中断启动，交人工确认（不静默择一）
default:                 // 原地改名：Update code，保留主键
}
```

保留主键是关键：`application_client_secret`、`refresh_token` 以 `application_client.id` 为外键。

---

## 影响面与兼容

| 维度 | 结论 |
|------|------|
| 接口契约 | **破坏性新增**：`POST /v1/platform/application-clients` 新增必填 `code`；调用方（仅控制台前端）已同步 |
| 存量库 | 内置客户端编码启动时原地改名；用户自建客户端的既有编码**不动**（若其含数字/连字符，仅表示创建时未受校验，仍可正常使用） |
| 已签发令牌 | `aud` 随内置 client_id 改变 → 旧令牌失效，两个控制台需重新登录 |
| 编码冲突 | 库中已存在同名用户自建客户端时，种子**中断启动**并要求人工确认删除其一 |
| 前端 | 新增表单项；`OAuthClientCreateReq` 类型补 `code` |

---

## 验收

| 编号 | 内容 | 用例 |
|------|------|------|
| V1 | 规则：字母+下划线合法；数字/连字符/大写/下划线开头/空格/空 非法 | `pkg/model`：`TestClientCodePattern` |
| V2 | 两个内置 client_id 常量各自合规且互不相同 | `pkg/model`：`TestSeedBuiltinClientCodesComply` |
| V3 | **后端**：非法 `code` 返回 `ApplicationClientCodeInvalidError` 且不落库 | `svcapplicationclient`：`TestCreateRejectsInvalidClientCode` |
| V4 | **后端**：合法 `code` 原样落库为 `client_id`（不再随机生成） | `svcapplicationclient`：`TestCreateAcceptsClientCodeAsClientID` |
| V5 | **前端**：连字符/数字被表单拦截且不发请求；合法值不报编码错误 | platform-admin-web：`新建客户端的编码校验` |
| V6 | 存量库连字符编码原地改名（保留主键、`source` 收敛），二次执行幂等 | `pkg/seed`：`TestSeedIamMigratesLegacyClientCode` |
| V7 | 网关 aud / back-channel logout 用新值仍通过 | `pkg/middleware`、`pkg/goidc` 既有用例 |

---

## 风险与回滚

| 风险 | 触发 | 应对 |
|------|------|------|
| R1 外部脚本调创建接口未传 `code` | 上线后有 CI/脚本调用该接口 | 返回 400（`binding:"required"`）；按 `AGENTS.md` R5 盘点调用方 |
| R2 旧令牌失效 | 内置 client_id 改变即改 `aud` | 各控制台重新登录；如需平滑，`WithOIDCAudiences` 支持临时同时放行新旧值 |
| R3 新旧编码并存 | 库中已有同名自建客户端 | 启动报错给出两个编码，人工确认后删除其一 |
| 回滚 | 需要恢复服务端生成 | 恢复 DTO/生成函数与常量为连字符口径；已改名的内置行需人工改回（或删库重建） |

---

## 落地记录（2026-09-12）

| 层 | 文件 | 内容 |
|----|------|------|
| 规则 | `pkg/model/application_client.go` | `ClientCodePattern = ^[a-z][a-z_]*$`、`IsValidClientCode`；`Code` 注释改为"创建时由调用方填写" |
| 常量源 | `pkg/model/seed_authority.go` | `SeedBuiltinClientPlatformAdminWeb` / `SeedBuiltinClientTenantAdminWeb` |
| 入参 | `dtoapplicationclient/request.go` | `ApplicationClientCreateReq.Code`（`binding:"required"`） |
| 校验（后端） | `svcapplicationclient/application_client.go` | service 入口校验 → `ApplicationClientCodeInvalidError`；删除 `generateClientCode` / `clientCodePrefix` / uuid·strings 依赖 |
| 错误码 | `pkg/code/permission.go` | 新增 `ApplicationClientCodeInvalidError = 100822` |
| 校验（前端） | `platform-admin-web/src/pages/oauthClient/index.tsx` | `CLIENT_CODE_PATTERN` + 必填/`pattern` 规则 + 「客户端编码」表单项（仅创建态） |
| 类型 | `packages/types/src/platform.ts` | `OAuthClientCreateReq.code` 必填 |
| 种子 | `pkg/seed/seed.go` | 引用常量；`code` 入口校验；legacy 原地改名 + 冲突中断；`findApplicationClientByCode` |
| 网关 | `apps/platformadmin/app.go`、`apps/tenantadmin/app.go` | aud 白名单与 back-channel logout 客户端识别改用常量 |
| 前端默认值 | `apps/*-web/src/main.tsx` | `clientID` 默认值改下划线 |
| 测试 | `pkg/model`、`pkg/seed`、`svcapplicationclient`、platform-admin-web | 4 个新用例（规则/常量自洽/后端拒收与放行/前端拦截）+ 1 个迁移回归 |
| 文档 | `sso-oidc-concepts.md` §3.2、`glossary.md`、`application-integration-guide.md`、`run-and-deploy.md`、`system-design.md`、`api-reference.md`、`AGENTS.md` | 旧"允许连字符 / 服务端生成 UUID"口径全部替换 |
