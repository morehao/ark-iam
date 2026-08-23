# GoArk 自助注册重设计 · 实现计划

> 状态：**已全部实施 S1-S10**（后端通道A/B、Invite、owner指派、注册即登录、全局开关 + 前端 login-web 注册/加入表单）· 参考主流多租户 IAM（keycloak/zitadel/logto/casdoor）源码模式 + GoArk 现状

---

## 1. 背景与问题

### 1.1 现状病根

| # | 病根 | 位置 |
|---|------|------|
| P1 | `register` 被用成「带 tenantID 加入已有租户」，与 `JoinTenant` 语义重合、无门禁，还丢掉了「开通租户」本义 | `svcauth/auth.go:180`、`dtoauth/request.go` |
| P2 | person 创建两套实现：`pkg/iam/person.FindOrCreate`（tenantadmin 用）vs `svcauth.Register` 自写三连查重 | `pkg/iam/person/`、`svcauth/auth.go:202-239` |
| P3 | **owner 完全悬空**：全项目无任何接口能设 `IsOwner=true`，DB 不会有 owner | 全局 |
| P4 | `AllowJoinByInvite` 死字段零消费；`AllowPersonCreateTenant` 只下发不消费 | `pkg/iam/model/application.go` |
| P5 | `login-web` 无注册页，register 无前端调用 | `frontend/apps/login-web` |
| P6 | 注册后不建 SSO 会话，「注册即登录」不成立 | `svcauth.Register` 只返回 UserID |

### 1.2 已确认决策

1. `register` = 自助开通租户，注册人成为 owner（对标 zitadel `register/org`）
2. 加入已有租户门禁双轨：邀请制为主 + 应用定向为辅，禁止裸 `tenantID` 直入
3. 注册即登录（对标 keycloak/logto/casdoor）
4. 新建完整 Invite 机制
5. 平台建租户不自动建 owner，补显式「指派 owner」接口
6. 允许破坏性调整

---

## 2. 目标模型

### 2.1 最终通道

| 通道 | 端点 | 触发者 | person | tenant | IsOwner | 门禁 |
|---|---|---|---|---|---|---|
| **A 开通租户** | `POST /v1/auth/register`（改造） | 游客 | 新建 | 新建 | **true** | `AllowPersonCreateTenant` + 全局开关 |
| **B 加入已有租户** | `POST /v1/auth/joinTenant`（改造） | 游客/已登录 | FindOrCreate | 已存在 | **false** | 邀请码 **或** 应用定向租户 |

### 2.2 owner 来源收敛
- 仅两渠道：① 通道A 开通（注册人自任 owner）② 管理员显式指派（新增接口）
- 通道B 永不 owner

---

## 3. 变更清单（按依赖顺序）

### S1. 统一 person create（前置）
- 目标：消灭 P2
- 动作：`pkg/iam/person.FindOrCreate` 保持为 person 唯一创建路径
- `svcauth.Register` 与 `svcauth.JoinTenant` 的新逻辑（S2/S3）一律走 `FindOrCreate`，删除自写三连查重
- 复用现有 `person.FindOrCreate(ctx, tx, req)` 签名；返回 `(person, created, error)`

### S2. 通道 A：改造 `POST /v1/auth/register`
**DTO** `dtoauth/request.go`：
```go
type RegisterReq struct {
    TenantName string `json:"tenantName" binding:"required,max=128"` // 租户名
    TenantCode string `json:"tenantCode" binding:"max=64"`           // 可选，空则 tenant-+uuid
    Username     string `json:"username" binding:"max=128"`
    PrimaryEmail string `json:"primaryEmail" binding:"max=128"`
    PrimaryPhone string `json:"primaryPhone" binding:"max=32"`
    Password     string `json:"password" binding:"required,max=128"`
    Name         string `json:"name" binding:"max=128"`
}
type RegisterResp struct {
    UserID   string `json:"userID"`
    TenantID string `json:"tenantID"`
}
```
**service** `svcauth.Register`（重写，事务）：
1. 密码强度 + 至少一 identifier 校验（保留）
2. 校验「允许自助建租户」：读当前应用 `Application.TenantPolicy.AllowPersonCreateTenant` + 全局开关（S5）
3. 事务内：
   a. `person.FindOrCreate` → personID
   b. 插 `TenantEntity{Code, Name, Type=customer}`（Code 空生成，撞唯一索引走 conflict 处理）
   c. 插 `UserEntity{TenantID, PersonID, IsOwner: true, JoinedAt}`
   d. 建根组织节点（复用 `svctenant.tenant.go:64-87`，逻辑上提到 pkg/iam 共用）
4. 注册即登录：`ssoSessionStore.CreateSession(personID, ["pwd"])` + 关联默认租户=新租户（走 S8 封装）
5. 返回 `RegisterResp{UserID, TenantID}`
6. 审计：`audit.ActionTenantCreate` + 新注册动作

### S3. 通道 B：改造 `POST /v1/auth/joinTenant`
**DTO**：
```go
type JoinTenantReq struct {
    InviteCode string `json:"inviteCode,omitempty"` // 邀请码（主门禁）
    // tenantID 不再允许裸传；由邀请或应用定向决定
}
```
**service** `svcauth.JoinTenant`（重写）：
1. 取 person（游客走注册分支建 person；已登录取 token personID）
2. 门禁判定（双轨，皆无则拒绝）：
   - 有 `InviteCode` → 校验 Invite（归属租户、未过期、未用）→ 落邀请绑定租户，可带角色/部门
   - 无 → 应用定向（应用契约绑定默认租户）→ 落该租户
3. 查重 `user(personID+tenantID)`，已存在 → `AlreadyJoinedError`
4. 插 `UserEntity{IsOwner:false}`
5. 注册即登录（游客场景）
- 激活 `AllowJoinByInvite` 死字段

### S4. 新增 Invite 机制
**Model** `pkg/iam/model/invite.go`（新表 `tenant_invite`）：
```go
type InviteStatus string // pending / accepted / revoked / expired
type InviteEntity struct {
    gormdao.BaseEntity
    TenantID   string          // 归属租户
    Code       string          `uniqueIndex` // 邀请码
    Role       json.RawMessage // 可带角色集合 []
    OrgNodeID  string          // 可带部门节点
    ExpiresAt  *time.Time
    Status     InviteStatus
    MaxUses    int
    CreatedBy  string
}
```
- `pkg/iam/dao/invite.go`：`InviteDao`
- tenantadmin：
  - `POST /v1/tenant/invites`（生成，绑租户+角色+部门+有效期）
  - `DELETE /v1/tenant/invites/{inviteID}`（撤销）
  - `GET /v1/tenant/invites`（列表）
  - UI 邀请管理
- auth：`svcauth` 邀请校验解析（S3 用）
- 鉴权：只有该租户 owner/管理员可生成/撤销（见 S6+S7）

### S5. 激活 TenantPolicy
- `AllowPersonCreateTenant`：通道A 门禁判定读取（缺失/未设视为 false）
- `AllowJoinByInvite`：通道B 门禁判定读取
- 新增全局开关：「允许公开自助建租户」（对标 zitadel `DisallowPublicOrgRegistration`），放配置/实例级；关闭时通道A 直接 404 或返回门禁错误

### S6. 新增 owner 显式指派（补齐 P3）
- 接口（tenantadmin / platformadmin）：
  - `POST /v1/tenant/members/{userID}/grant-owner`
  - `POST /v1/tenant/members/{userID}/revoke-owner`
- service：校验操作者是该租户 owner/平台管理员；`UserEntity.IsOwner` 翻转
- 鉴权：租户级 `WithTenantOwnerAbilities` 中间件（新增）
- UI：成员管理里的 owner 操作
- `svctenant.TenantCreate` 不变（无 owner）

### S7. 鉴权中间件（通道AB + owner 指派共用）
- 新增租户内「是否 owner/管理员」判定工具，供通道A 门禁、通道B、S6 owner 指派复用
- 复用现有 `pkg/middleware` OIDC 鉴权；新增租户 owner 校验 handler

### S8. 注册即登录统一封装
- 把 `svcoidc` 的 `CreateSession`/SSO 会话逻辑封装为可被 `svcauth` 复用的入口（幂等）
- 通道A/B 的「注册即登录」调它

### S9. 前端 login-web
- `/login` 加「注册」入口（对标 `prompt=create`）
- `/register/org`（通道A：租户名+编码+姓名+邮箱+密码）
- `/join`（通道B：邀请码 + 邮箱/密码）
- `api.ts` 补 `registerOrg` / `joinWithInvite` 封装
- vite proxy 已指向 gateway:8100，无需改

### S10. 破坏性面同步
- `RegisterReq`/`RegisterResp`/`JoinTenantReq` 变更 → swagger、测试、前端同步
- 错误码：接入 `AuthRegisterFailedError(110012)`；新增 `AuthJoinNotAllowedError`、`Invite*Error`、`OwnerGrantError`、`TenantRegisterNotAllowedError`
- 更新 `svcauth/auth_test.go`、`auth_integration_test.go`
- 文档 `docs/design/system-design.md §5.1` 重写为通道AB双轨

---

## 4. 关键风险与注意

1. **owner 鉴权环**：授予 owner 方自身须为 owner/管理员，否则人人自封 owner——S6/S7 必须优先落地
2. **通道A 风控**：影响面大（对标 zitadel 全局可关）。需 `LoginRateLimit` + 全局开关 + 审计，默认生产关闭
3. **注册即登录**：`svcauth.Register` 当前独立表单、非 OIDC authorize 流程，建 `CreateSession` 需复用 `svcoidc` 会话存储，注意幂等
4. **根组织复用**：通道A 与平台建租户共用根组织逻辑，上提到 `pkg/iam` 避免复制
5. **Code 唯一冲突**：通道A 建租户撞 `tenant_code` 唯一索引的处理
6. **Invite 一次性/过期**：状态机 pending→accepted/revoked/expired

---

## 5. 实施顺序（退出 plan mode 后）

1. S1 统一 FindOrCreate
2. S7 租户 owner 鉴权工具 + S6 owner 指派接口（先补 P3 悬空 + 鉴权环）
3. S2 通道A + S5 AllowPersonCreateTenant + 全局开关
4. S3 通道B + S4 Invite + AllowJoinByInvite
5. S8 注册即登录统一封装
6. S10 破坏性面同步 + 测试 + 文档
7. S9 前端 login-web（独立可后置）

每步 `make test APP=...` + `go test ./...` 验证，完成后按 verification-before-completion 跑 lint/test。
