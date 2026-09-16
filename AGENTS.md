# AGENTS.md - GoArk 代码库开发指南

本文档为在此代码库中工作的 AI 代理提供开发规范和命令参考。

## 项目概述

GoArk 是一个基于 Gin + GORM 的多应用后端项目，采用前后端分离架构。IAM 后端拆分为四个应用（`auth`/`platformadmin`/`tenantadmin`/`gateway`），共享公共层 `backend/pkg`，并以 `backend/go.work` 作为 Go workspace 管理 5 个模块（apps/auth、apps/gateway、apps/platformadmin、apps/tenantadmin、pkg）。

## 项目结构

```
ark-iam/
├── backend/               # Go 后端项目（go.work 多模块）
│   ├── apps/
│   │   ├── auth/          # 认证网关（登录/注册/token/OIDC），:8081
│   │   ├── platformadmin/ # 平台管理，:8082
│   │   ├── tenantadmin/   # 租户自服务，:8083
│   │   └── gateway/       # 聚合应用（挂载上述三者，单体部署），:8100
│   ├── pkg/               # 公共层，5 模块之一（model/dao/object/core/credential/…）
│   └── Makefile
├── frontend/             # React 前端项目
├── docs/                  # 文档目录
├── Makefile              # 根目录 Makefile
├── AGENTS.md             # 开发规范
└── README.md
```

## 构建与运行命令

所有命令在项目根目录下执行。有效 `APP` 取值为 `auth | platformadmin | tenantadmin | gateway`：

```bash
# 列出所有可用应用
make list-apps

# 构建指定应用
make build APP=auth
make build APP=gateway

# 运行指定应用（开发调试；gateway 单进程聚合三者）
make run APP=auth
make run APP=gateway

# 下载依赖
make deps

# 清理构建产物
make clean
```

应用端口：auth 8081、platformadmin 8082、tenantadmin 8083、gateway 8100。

## 测试命令

```bash
# 运行指定应用的测试（推荐；gateway 只做聚合、无测试用例，请用 auth/platformadmin/tenantadmin）
make test APP=auth

# 注意：go.work 位于 backend/，go 命令须在 backend 目录下执行，且模块按路径逐个 use。
# `cd backend && go test ./...` 会报 "directory prefix . does not contain modules listed in go.work"，
# 必须显式列出各模块目录（或在模块目录内跑 ./...）。
cd backend

# 运行所有模块的测试
go test ./apps/auth/... ./apps/gateway/... ./apps/platformadmin/... ./apps/tenantadmin/... ./pkg/...

# 等价的模块内写法
cd backend/apps/auth && go test ./...

# 运行单个测试函数（共享层领域能力）
cd backend && go test ./pkg/core/user/ -run TestCreate_NewPersonWithDeptRelations -v

# 运行单个测试函数（凭证摘要契约）
cd backend && go test ./pkg/credential/ -run TestHashSecret_Golden -v

# 运行特定包测试
cd backend && go test ./apps/auth/internal/router/... -v

# 生成测试覆盖率报告
cd backend && go test ./apps/platformadmin/internal/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Lint 和代码检查

```bash
# 运行 golangci-lint
make lint

# 仅运行特定 linter
golangci-lint run ./... --disable-all -E golint,errcheck,staticcheck

# go vet（同样按模块逐个执行）
cd backend && for m in apps/auth apps/gateway apps/platformadmin apps/tenantadmin pkg; do (cd $m && go vet ./...); done
```

## 代码规范

### 项目结构

```
apps/
├── auth/                       # 认证网关
│   ├── cmd/                    # 入口函数
│   ├── internal/
│   │   ├── controller/ctrxxx/   # 控制器层 (ctr 前缀)
│   │   ├── service/svcxxx/      # 服务层 (svc 前缀)
│   │   ├── core/oidcop/         # 领域层容器（core 只承载领域层）；oidcop = OIDC Provider 领域层（op.Storage 适配/协议态/持久化/客户端适配）
│   │   ├── dto/dtoxxx/          # DTO 层
│   │   ├── router/              # 路由注册
│   │   └── middleware/          # 中间件
│   ├── model/               # 数据模型
│   └── dao/                 # 数据访问层
├── platformadmin/              # 平台管理（结构同 auth）
├── tenantadmin/                # 租户自服务（结构同 auth）
├── gateway/                    # 聚合应用（挂载 auth/platformadmin/tenantadmin）
pkg/                          # 公共层（跨应用共享，见下方「公共层约定」）
```

> **公共层约定（`pkg/`）**：与应用内层级一一对应，**共享即上提同名目录**——`internal/middleware` ↔ `pkg/middleware`、`internal/core/<域>` ↔ `pkg/core/<域>`、`dto/dto<域>` ↔ `object/obj<域>`；跨应用共享的 model/dao/object 直接平铺在 `pkg/model`、`pkg/dao`、`pkg/object`（**不再套业务域容器**：模块路径 `github.com/morehao/ark-iam/pkg` 已表达域名，套一层会重复且与 `pkg/goidc`、`pkg/seed` 等兄弟包边界矛盾）。
>
> 公共层**禁止**复用应用侧 `svc` 前缀：`svc` 专指应用内服务层，编排不外提；共享层只承载领域不变式（跨表、在调用方事务内、返回实体/哨兵错误）与基础能力，可复用的领域不变式下沉 `pkg/core/<域>`，其余留在各应用。凭证（口令强度 / 临时口令 / API Key·OAuth client secret·refresh token 的生成与摘要）统一在 `pkg/credential`，**全系统只允许一份摘要实现**；OIDC 鉴权中间件在 `pkg/middleware/oidc_auth.go`，RP 侧 back-channel logout 接收端在 `pkg/goidc`。
>
> **领域层容器约定**：应用内领域层统一放 `internal/core/<领域名>`（当前为 `oidcop`）。`core` 只承载绑定框架/协议的领域逻辑，禁止放置工具与辅助代码；`op` 为 OpenID Provider 术语（对应 RP 侧 `pkg/goidc`），非领域层通用后缀，其他领域层按领域名命名（如 `core/session`）。公共层同理：`pkg/core/<域>` 是平铺的领域层容器，**只放领域不变式，不放工具与辅助代码**（工具/基础能力直接放 `pkg/<name>`）。
>
> **OIDC 分层约定**：OP（Provider）侧领域层在 `apps/auth/internal/core/oidcop`（仅 auth 使用，绑定 auth 实体与 zitadel op 框架）；跨应用共享的 OIDC 能力（当前为 RP 侧 `pkg/goidc`）才放 `pkg`。若未来出现第二个 OP 消费者，将 `oidcop` 上提至 `pkg/goidc`。

### 命名规范

- **包名**: 纯小写，无下划线连线，简短且按业务领域划分，如 `svcuser`（用户服务）、`svcrole`（角色服务）
- **接口名**: 以 `I` 结尾或使用角色后缀，如 `UserSvc`, `UserCtr`
- **结构体**: 导出使用大驼峰 `UserSvc`，非导出使用小驼峰 `userSvc`
- **文件命名**: 小写下划线，如 `user_service.go`，测试文件 `*_test.go`
- **数据库表**: 下划线命名，如 `department_user`
- **编码规则（`code` 类字段）**: 两套规则**刻意不同**，不要混用——`application.code`（应用编码，`model.AppCodePattern`，`^[a-z][a-z0-9_]*$`）允许数字；`application_client.code`（= OIDC `client_id`，`model.ClientCodePattern`，`^[a-z][a-z_]*$`）**仅小写字母与下划线、不允许数字**，两者都禁连字符。客户端 `code` 是**创建时的必填入参**（`ApplicationClientCreateReq.Code`，服务端不再生成），且**前后端各校验一份**：前端表单 `pattern`（`platform-admin-web/src/pages/oauthClient` 的 `CLIENT_CODE_PATTERN`）+ 后端 service 入口（`model.IsValidClientCode` → `ApplicationClientCodeInvalidError`）。正则跨语言无法共享，**改一处必须同步另一处**并补两侧回归。内置 `client_id` 常量在 `pkg/model`（`SeedBuiltinClientPlatformAdminWeb` / `SeedBuiltinClientTenantAdminWeb`）：它同时是网关 aud 白名单、back-channel logout 客户端识别与前端 `VITE_OIDC_CLIENT_ID` 默认值的取值来源；存量库的连字符编码由 `pkg/seed` 的 legacy 改名分支原地迁移（保留主键）。
- **`code` 的可改性由「谁按它认行」决定**（见 `docs/design/system-design.md` §4.5「种子数据与字段权威」）：**菜单 `code` 与归属应用、自建应用 `code`、用户自建客户端 `code` 全部可改**——种子按不可见的**种子身份键 `seed_key`**（`menu`/`application` 上的内部列，创建时写入后不变，控制台不可见不可写）认行，租户开通（`ProvisionTenantAdmin`）与退役菜单清理也按它定位；**内置应用 `code` 与内置客户端 `code`（= `client_id`）保持只读**——内置应用的编码仍是各自控制台菜单入口的定位值（`svcpermission.MyTree` 按 `platform_admin` 查应用、tenantadmin `loadConsoleApps` 只保留 `tenant_admin`），内置客户端编码同时是网关 aud 白名单与前端构建期默认值，两者从控制台改名都会当场把对应控制台锁死且界面无法自救（真要换属版本级动作）。新增按编码认行的代码前，先确认该编码是否属于上述"可改"集合。

### 模块划分规范

**按业务领域划分模块，而非按单表划分。**

每个业务领域包含该领域相关的实体、DTO、Service、Controller，放在同一层级目录下。

示例：用户领域（user）包含用户基本信息、用户身份、用户部门关系、用户登录日志等：

```
apps/platformadmin/
├── model/user.go              # 用户领域所有实体
├── dao/user.go               # 用户领域所有数据访问
├── object/user.go            # 用户领域基础对象
└── internal/
    ├── dto/user/             # 用户领域所有 DTO（包含身份、部门关系等）
    │   ├── request.go
    │   └── response.go
    ├── service/svcuser/      # 用户领域服务
    │   └── user.go
    └── controller/ctruser/   # 用户领域控制器
        └── user.go
```

错误码和路由也按领域划分：
- 一个业务领域共享一套错误码段（如 user 领域用 1005XX）
- 路由按领域注册（如 `/v1/platform/users/*` 下包含用户及其相关操作）

### 常量定义规范

**凡字典字符串，禁止硬编码，均需定义成常量。**

| 常量类型 | 存放位置 | 说明 |
|---------|---------|------|
| 数据表名常量 | `model/*.go` | 定义在对应结构体文件 |
| 数据库存储的枚举值 | `model/*.go` | 定义在对应结构体文件 |
| 字典值常量 | `model/*.go` | 所有字典字符串定义成常量 |
| 应用层常量（前端专用） | `internal/constant/` | 状态映射等前端专用常量 |

#### 强类型字符串枚举（字典常量全链路复用）

**凡定义成常量的一组字典字符串，必须声明具名类型，并让实体/DAO/DTO/Service/测试全链路复用该类型与常量。**

```go
// model/department_user.go
// 具名类型 + 常量（禁止硬编码、禁止在其他层裸写字符串）
type DeptUserRelationType string

const (
    DeptUserRelationPrimary   DeptUserRelationType = "primary"   // 行政主部门，每用户至多 1 行
    DeptUserRelationSecondary DeptUserRelationType = "secondary" // 跨部门参与，可多条
    DeptUserRelationLeader    DeptUserRelationType = "leader"    // 负责人，可多条
)
```

**硬规则（新增字典枚举必守）：**

1. **字段类型用具名类型，不用 `string`**：实体、DAO Cond、DTO 请求/响应的枚举字段一律声明为该具名类型（如 `RelationType DeptUserRelationType`），而非 `string`——编译期即可杜绝拼错枚举值。
2. **全链路用常量**：赋值、传参、比较一律引用常量，如 `model.DeptUserRelationPrimary`，**禁止** `string(model.DeptUserRelationX)` 强转、**禁止**显式类型转换换别的枚举类型、**禁止**裸字面量 `"primary"`/`"admin"` 出现在非定义处。
3. **非法值校验归 service**：请求来自前端（JSON/form 绑定原始类型），service 入口用 `switch` + 常量白名单判合法，非法返回对应功能级错误码；合法值命中常量直接使用。
4. **JSON/DB 向下兼容**：具名类型的底层是 `string`，JSON 序列化仍是普通字符串、gorm 存 varchar，前端和数据库均无感知；DTO 包允许 import `pkg/model`（单向下游，无环）。

#### 数据表常量（model 层）

```go
// model/department.go

// 数据表名
const TableNameDepartment = "department"

// 业务枚举类型
type DeptStatus string

// 字典值常量（禁止硬编码）
const (
    DeptStatusActive   DeptStatus = "active"
    DeptStatusInactive DeptStatus = "inactive"
)

func (DepartmentEntity) TableName() string {
    return TableNameDepartment
}
```

#### 应用层常量（internal/constant/）

前端专用的状态映射等常量，放在应用的 `internal/constant/` 目录下：

```go
// apps/platformadmin/internal/constant/status.go
package constant

const (
    StatusEnabled  = "enable"
    StatusDisabled = "disable"
)

var StatusTextMap = map[string]string{
    StatusEnabled:  "启用",
    StatusDisabled: "停用",
}
```

### Import 排序

按以下顺序分组，无空行分隔：

1. 标准库 (`fmt`, `strings`, `time`...)
2. 第三方库 (`github.com/gin-gonic/gin`, `github.com/stretchr/testify`...)
3. 项目内部包 (`github.com/morehao/ark-iam/apps/platformadmin/...`, `github.com/morehao/ark-iam/pkg/...`)
4. 关联库 (`github.com/morehao/golib/...`)

```go
import (
    "strings"

    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify"

    "github.com/morehao/ark-iam/platformadmin/internal/dto/dtouser"
    "github.com/morehao/ark-iam/pkg/code"
    "github.com/morehao/golib/glog"
)
```

### 接口定义与依赖注入

使用接口定义服务层，通过构造函数注入：

```go
type UserSvc interface {
    Create(ctx *gin.Context, req *dtouser.UserCreateReq) (*dtouser.UserCreateResp, error)
}

type userSvc struct {
}

var _ UserSvc = (*userSvc)(nil)  // 编译时接口检查

func NewUserSvc() UserSvc {
    return &userSvc{}
}
```

### 服务层数据访问与测试约定

- **服务层直接调用 `dao.NewXxxDao()`**，不定义 repository 接口/adapter/函数变量等中间层；跨表操作在调用点直接使用对应 dao。
- **ID 字段命名**：Go 字段与 JSON tag 一律 `ID` 全大写（`userID`、`roleID`、`appID`、`connectorID`），禁止 `roleId`/`appId` 等小写 d 写法；路由 path 参数（`:roleID`）与 swagger 注解同步。**path 是 ID 的唯一来源**：凡带 `uri:"xxxID"` 的字段，DTO tag 用 `json:"-" uri:"xxxID"`（禁止挂 `json:"xxxID"`/`form:"xxxID"`，防 body/query 覆盖 path 造成参数污染）。
- **DTO 命名**：统一 `<业务名词><动词>Req/Resp`（如 `UserCreateReq`、`DomainCreateReq`），禁止 `CreateDomainReq`、裸 `CreateReq` 等变体。
- **DTO ID 类型**：统一 `string`（字符串主键，UUID v7，由 `gormdao.BaseEntity` 自动生成；见 `docs/design/system-design.md` §4.1），禁止 `uint`/`uint64`。
- **前后端时间交互**：前后端交互的所有时间字段统一使用**秒级 int64 时间戳**（Unix 秒，即 `.Unix()`）。包括新建、编辑、筛选、展示等一切 DTO 请求与响应字段，禁止用 `string` 承载格式化时间（如 `"2006-01-02 15:04:05"`）。出参可空时间用指针 `*int64`（无值返回 `null`），入参可空时间用 `int64`（无值传 `0`）；`gobject.OperatorBaseInfo` 内嵌的 `CreatedAt`/`UpdatedAt` 已是 int64，禁止覆盖为 string。service 层出参用 `x.Unix()`，入参解析用 `time.Unix(req.ExpiredAt, 0)`。
- **单元测试**：统一使用各 app `testutil.SetupSQLite(t, entities...)`（内存 SQLite 注册为全局 iam 库），服务内部 `dao.NewXxxDao()` 自动落测试库，直接断言真实 dao 行为；不写 stub/注入 seam。注意 sqlite 对 `not null` JSON 列（`profile`/`config` 等）与 `joined_at` 需要显式播种值。

### 错误处理

- 使用统一的错误码包 `github.com/morehao/ark-iam/pkg/code`
- 业务错误通过 `code.GetError(code.XXXError)` 返回
- 错误日志使用 `glog.Errorf(ctx, "[module.Method] msg, err:%v", err)`

```go
if err != nil {
    glog.Errorf(ctx, "[svcuser.Create] daoUser GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
    return nil, code.GetError(code.UserCreateError)
}
```

#### err 判断与业务边界判断必须分离

`err != nil` 属于**不可预期的系统错误**（数据库/网络/IO 等），无论来自 DAO 还是任何会返回 error 的调用，都必须**先单独判断**，记录日志后返回该操作的功能级错误码；**不得**与可预期的业务边界判断（如结果为 nil、主键为空、不属于当前租户、列表长度不匹配）混在同一个 `if` 里。混写会掩盖真实根因（把系统错误伪装成"XX 不存在/不匹配"）且丢失错误日志。

- 不可预期错误（`err != nil`）→ 单独 `if`，`glog.Errorf` 后返回功能级错误码（`CreateError`/`UpdateError`/`DeleteError`/`GetDetailError` 等）
- 可预期业务边界（`entity == nil`/`ID == ""`/`TenantID != tenantID`/`len != 0` 等）→ 单独 `if`，返回 `NotExistError` 等业务错误码

错误示范（err 与边界混写）：

```go
// 错误：系统错误被掩盖为 DepartmentNotExistError，且未记日志
parent, err := dao.NewDepartmentDao().GetByID(ctx, req.DepartmentID)
if err != nil || !departmentVisibleToTenant(parent, tenantID) {
    return nil, code.GetError(code.DepartmentNotExistError)
}
```

正确示范（分离）：

```go
parent, err := dao.NewDepartmentDao().GetByID(ctx, req.DepartmentID)
if err != nil {
    glog.Errorf(ctx, "[svcdepartment.Children] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
    return nil, code.GetError(code.DepartmentGetPageListError)
}
if !departmentVisibleToTenant(parent, tenantID) {
    return nil, code.GetError(code.DepartmentNotExistError)
}
```

### 事务处理

使用 `dbclient` 封装的事务：

```go
txErr := dbclient.IamDB(ctx).Transaction(func(tx *gorm.DB) error {
    result, err = user.CreatePersonWithUser(ctx, tx, params)
    if err != nil {
        return err
    }
    return nil
})
if txErr != nil {
    glog.Errorf(ctx, "[svcuser.Create] Transaction fail, err:%v", txErr)
    return nil, code.GetError(code.UserCreateError)
}
```

### 上下文传递与租户作用域（must）

含 `tenant_id` 列的表，其 SELECT/UPDATE/DELETE 由租户隔离插件（`pkg/dbclient/tenant_scope.go`）自动注入 `tenant_id = ?`；**ctx 未声明作用域时 fail-closed 报错**（`dbclient.ErrTenantScopeMissing`），绝不静默放行成跨租户读。

1. **ctx 一路直传 `*gin.Context`**：controller → service → dao → 第三方（OIDC provider、glog、限流器）都传同一个 gin ctx（引擎已开 `ContextWithFallback`，gin 上下文本身即合法 `context.Context`）。**禁止 `ctx.Request.Context()`**（全仓白名单为空）；异步/后台续跑用 `gincontext.AsyncContext(ctx)`（保留取值与作用域、去掉取消，且不持有会被复用的 gin 上下文）。
2. **作用域只能显式声明**，禁止用"ctx 碰巧没有作用域"表达跨租户可见：
   - 当前租户：中间件调用一次 `gincontext.SetTenantScope(ctx, gcontext.CurrentScope(tenantID))`（写入请求上下文，并投影 gin Keys 供 `gincontext.Get*String` 读取）；
   - 指定它租户：`dbclient.ExplicitTenantContext(ctx, tenantID)`（写在具名 dao 方法内，调用方零判断）；
   - 跨全部租户：`dbclient.CrossTenantContext(ctx)`（按全局唯一键反查、自然人级操作、启动期迁移/种子）；
   - 非请求入口（seed、worker、后台任务）：显式 `gcontext.WithTenantScope(ctx, gcontext.AllScope()/CurrentScope(tenantID))`。
3. **新增不含 `tenant_id` 的全局表**必须同步登记到 `pkg/dbclient/gorm.go` 的 `tenantScopeSkipTables`，否则带租户上下文查询会报 `no such column`。
4. **测试同样受约束**：`testutil.SetupSQLite` 挂载同一份插件，测试里对租户表的断言查询也必须带作用域（用被测服务同一个 gin ctx，或 `dbclient.CrossTenantContext(...)` 显式跨租户）。
5. 灰度回退仅有 `dbclient.SetMissingTenantScopeMode(dbclient.MissingTenantScopeWarn)` 一个开关（只告警不拦截），**不得**作为长期方案。

### Controller 返回模式

统一使用 `gincontext` 封装响应：

```go
func (ctr *userCtr) Create(ctx *gin.Context) {
    var req dtouser.UserCreateReq
    if err := ctx.ShouldBindJSON(&req); err != nil {
        gincontext.Fail(ctx, err)
        return
    }
    res, err := ctr.userSvc.Create(ctx, &req)
    if err != nil {
        gincontext.Fail(ctx, err)
        return
    }
    gincontext.Success(ctx, res)
}
```

### API 路由规范

采用**规则化混合**风格（REST 资源式为主 + 显式动作式补充）。完整规范即下方三条硬规则与命名/方法/关联/层级约定；接口清单与端点总览见 `docs/design/api-reference.md`。

**三条硬规则（新增路由必须按序判定）**：

- **R1 资源 CRUD → REST**：资源的增删改查/列表/详情/树，用「集合 + 方法 + ID」表达。路径格式 `/{版本}/{服务标识}/{资源}[/{id}[/{子资源}]]`（如 `/v1/platform/users/{userID}/identities`；`服务标识` 即应用标识段：auth → `/v1/auth`、platformadmin → `/v1/platform`、tenantadmin → `/v1/tenant`，各应用路径互不相同；资源名可跨应用复用，由服务标识段区分归属）
- **R2 业务动作 → 动作子路径**：状态流转/触发副作用类操作用 `POST /资源/{id}/动作`（如 `POST /v1/tenant/api-keys/{apiKeyID}/revoke`）；认证/会话类动作挂 `/v1/auth` 动作式专用段（`joinTenant`/`logout`/`logoutAll`/`userinfo`；`register` 已下线，自助注册走 `/oidc/registerPerson` + `/oidc/createTenant`）
- **R3 标准协议 → 专用前缀**：`/oidc/*`、back-channel logout、docs 不走业务路由规范，保持不动

**资源命名**：复数 + kebab-case（`users`、`application-clients`、`api-keys`、`tenant-applications`），禁止驼峰（`applicationClient`）。ID 路径参数一律 `{xxxID}` 全大写（`{userID}`、`{roleID}`、`{appID}`），与 DTO JSON tag 及 swagger 注解同步。

**HTTP 方法语义**：`GET` 查询（含分页列表）／`POST` 创建与动作子路径（`/id/action`）／`PUT` 全量更新与批量授权（全量替换集合）／`PATCH` 局部更新（如状态）／`DELETE` 删除。**禁止** `POST /xxx/delete`、`POST /xxx/update`、`POST /xxx/pageList` 等动作式写法。

**关联建模**：从属资源用子资源（`/users/{userID}/identities`）；多对多关联用父资源子集合两端视角（`/users/{userID}/roles` 与 `/roles/{roleID}/menus`，服务账号对应 `/machine-users/{machineUserID}/roles`），批量授权 = `PUT` 集合全量替换；跨父资源检索保留顶层只读资源（`/v1/platform/user-identities`、`/v1/platform/login-logs`）。

**层级限制**：集合层级 ≤ 3（路径段 ≤ 6：`v1 / service / collection / {id} / subcollection / {id}`），禁止更深嵌套。

**当前用户资源**：`/v1/auth/me`（原 person）、`/v1/auth/me/tenants`（原 myTenants）、`/v1/auth/me/sessions`（原 user/sessions）。

#### 路由示例

| 资源 | 操作 | 完整路径 |
|------|------|----------|
| user | 创建 | `POST /v1/platform/users` |
| department | 创建部门节点 | `POST /v1/tenant/departments` |
| department | 部门树 | `GET /v1/tenant/departments/tree` |
| apiKey | 吊销（动作） | `POST /v1/tenant/api-keys/{apiKeyID}/revoke` |
| 认证操作 | 加入租户（动作） | `POST /v1/auth/joinTenant`（auth 应用认证操作直接挂服务段，避免 `/v1/auth/auth/*`）；自助**注册**不在业务路由，收口在协议端点 `POST /oidc/registerPerson` + `POST /oidc/createTenant` |

#### 路由注册

各业务 app 通过 `ginserver.NewRouterGroups(engine, "<服务标识>", ...)` 注册应用前缀（auth: `"auth"`、platformadmin: `"platform"`、tenantadmin: `"tenant"`），然后在 `router/router.go` 中按版本 `MustGetGroup(ginserver.ApiVersionV1)` 注册各模块路由：

```go
routerGroups := ginserver.NewRouterGroups(engine, "platform", []ginserver.VersionGroup{{
    Version: ginserver.ApiVersionV1,
    // ...
}})
```

在各个路由文件中使用 gin 的路由注册方法（统一按「create → list → tree/detail → update → delete → 动作/子资源」顺序排列）：

```go
func userRouter(groups *ginserver.RouterGroups) {
    v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
    v1RouterGroup.POST("/users", userCtr.Create)
    v1RouterGroup.GET("/users", userCtr.PageList)
    v1RouterGroup.GET("/users/:userID", userCtr.Detail)
    v1RouterGroup.PUT("/users/:userID", userCtr.Update)
    v1RouterGroup.PATCH("/users/:userID", userCtr.UpdateStatus)
    v1RouterGroup.DELETE("/users/:userID", userCtr.Delete)
    v1RouterGroup.GET("/users/:userID/identities", userCtr.GetUserIdentityByUser)
}
```

### Swagger 文档

使用 Swag Go 注解，需包含以下注释：

```go
// @Tags 用户管理
// @Summary 创建用户管理
// @accept application/json
// @Produce application/json
// @Param req body dtouser.UserCreateReq true "创建用户管理"
// @Success 200 {object} gincontext.DtoRender{data=dtouser.UserCreateResp}
// @Router /v1/platform/users [post]
```

生成文档：

```bash
make swag APP=auth
```

### 测试规范

- 测试文件放在同包或 `testutil` 包中
- 使用 `testutil.Initialize()` 初始化测试环境
- 使用标准库 `testing` 包和 `testify/assert`

```go
package svcuser

import (
    "testing"

    "github.com/morehao/golib/gcrypto"
)

func TestGeneratePassword(t *testing.T) {
    hash, err := gcrypto.GeneratePasswordHash("password")
    if err != nil {
        t.Fatalf("GeneratePasswordHash failed: %v", err)
    }
    if err := gcrypto.ComparePasswordHash(hash, "password"); err != nil {
        t.Errorf("ComparePasswordHash failed: %v", err)
    }
}
```

### 数据库 Schema 变更约定（按新项目处理）

本项目按**全新项目**维护 schema，**不写数据迁移脚本**：`AutoMigrate` 只新增缺失的表/列/索引，**不删不改**既有结构（`db.auto_migrate`），`pkg/seed` 亦只做幂等 upsert。因此**列/表下线（删字段、改名、类型或语义替换）一律按新项目处理**：

- **下线即彻底删代码**：model 字段、DAO Cond、DTO/object、service、controller/router、前端类型与 `docs/design` 同步删除，并全仓 `grep` 确认零残留（参考 `tenant.is_suspended` → `tenant.status`、`system` 模块下线）。
- **菜单下线额外两步**：从 `seedMenus` 删除定义后，必须在 `pkg/seed/retired_menu.go` 的 `retiredMenus` 登记（父目录与子菜单一并登记），否则存量库会残留指向已删除页面的死链菜单；判断依据是「base 版 `seedMenus` 与当前定义的差集」——`seedMenus` 只 upsert 不下线，而全新库测试永远测不出这类残留。退役清理是**物理删除**（不留墓碑）：版本级下线允许未来重新上线同名 `seed_key`。
- **菜单的"行"归运维**（见 `docs/design/system-design.md` §4.5）：控制台可新增根菜单/子菜单、可删除任意菜单（含内置菜单）。删除是软删除，软删行仍带 `seed_key`，即"该菜单已被人为下线"的**墓碑**——`seedMenus` 命中墓碑即跳过创建（检查必须早于 `(app_id, code)` 兜底，否则会误认领运维自建的同 code 菜单）。因此不要假设"内置菜单行一定存在"，也不要从控制台删除后又指望种子把它建回来。
- **禁止在代码里写兼容旧库的分支**：不加回填、不加 `DROP COLUMN`、不引入 `information_schema`/`Migrator()` 判定——这会污染 AutoMigrate「只增不删」的契约，且对新项目零收益。
- **旧库残留列/表属预期**（不再是事实源、不被读写），处置方式是**开发/测试库删库重建**：重建后 AutoMigrate + Seed 产出的结构即目标结构（见 `docs/design/run-and-deploy.md` §2.3）。
- **种子写入遵循字段权威矩阵**（单一写者，见 `pkg/model/seed_authority.go` 与 `docs/design/system-design.md` §4.5）：每字段显式声明 `reconcile`（种子收敛，控制台必须拒写）/ `create_only`（只播种，归运维，种子不回写）/ `migrate_once`（值匹配一次性改名，登记在 `pkg/seed` 的 `seedMigrations`）；禁止绕过矩阵手写回填 `if`、禁止用 `reconcile` 表达改名（会覆盖运维改动），新增内置字段必须先声明语义再实现。
- **`reconcile` 的准入判据**（2026-09-12 两轮收窄后的最终形态见 `docs/design/system-design.md` §4.5）：只保留**安全不变式**——`source`（内置标记）、`admin_type`、平台租户 `status`（被挂起整栈失联）。**种子认行不再经由任何控制台可写字段**：菜单/应用按不可见的 `seed_key` 认行，`application_client` 按 `code` 认行。因此展示、结构、编码类字段一律 `create_only` 归运维——种子不回写；其中**内置应用与内置客户端的 `code` 另由 service 拒改**（见上方「`code` 的可改性」）。种子在执行侧只按矩阵过滤（`reconcileFields`），**禁止**在 `pkg/seed` 里硬编码字段归属。
- **确需保全旧数据时**，把一次性 SQL 写进部署文档交执行方在升级前运行，而不是塞进启动流程。

### 代码生成

项目使用 `gocli` 工具进行代码生成：

```bash
# 生成 API 路由和控制器
make codegen APP=auth COMMAND=api

# 生成模块代码
make codegen APP=auth COMMAND=module

# 生成模型代码
make codegen APP=auth COMMAND=model
```

### Docker 支持

```bash
# 构建 Docker 镜像
make docker-build APP=auth

# 运行 Docker 容器
make docker-run APP=auth
```

## 前端规范

前端工作区为 pnpm monorepo（`frontend/`）：`packages/{types,api,auth,ui}` 为共享包，`apps/{login-web,platform-admin-web,tenant-admin-web}` 为业务应用。统一视觉语言与设计令牌见 `frontend/DESIGN.md`（代码事实源为 `frontend/packages/ui/src/theme.ts` 的 `tokens`），业务页面一律复用 `@ark-iam/ui` 共享组件，禁止硬编码色值与自造样式。

### 列表页操作列规范（操作列收敛 + 名称即详情入口）

所有列表页（Table）的操作列遵守以下硬规则。规范细节同步维护在 `frontend/DESIGN.md` §7.3，实现统一收敛到 `@ark-iam/ui` 的 `actionColumn`（内部 `RowActions`）与 `nameColumn`（内部 `NameLink`）。

**R1 操作列一律用 `actionColumn` 声明：最多横排 3 个操作，超过即纵向展开为「更多」下拉，且始终 `fixed: 'right'`。**

- 操作数 ≤ `max`（默认 3）：全部以 `Button type="link" size="small"` 横排；危险操作加 `danger`。
- 操作数 > `max`：保留前 `max-1` 个高频操作横排，其余收进「更多」下拉（菜单项纵向排列），避免操作列被撑宽、行内按钮挤成一团。这是后台表格的主流做法，与 Ant Design Table 官方「操作」示例的 `Delete + More actions` 形态一致。
- **操作列永远 `fixed: 'right'`**：由 `actionColumn` 统一保证，横向滚动时钉在右侧，杜绝「操作列要横滑很久才看得到」。
- **列宽由 `actionColumnWidth(max)` 自动给出**，页面**禁止**手写 `width: 120/200/240`；横排按钮文案保持简短（≤4 个汉字），文案更长时降低 `max` 让它落入「更多」下拉，或显式传 `width`。
- 统一用 `actionColumn` 渲染，**禁止**在页面内手写 `Space + Button/Popconfirm` 拼装操作列，也禁止给操作列另写 `title`/`key`/`width`/`fixed`：

```tsx
import { actionColumn } from '@ark-iam/ui'

actionColumn<TenantItem>({
  max: 2,
  actions: (r) => [
    { key: 'edit', label: '编辑', onClick: () => handleEdit(r) },
    { key: 'roles', label: '授权角色', onClick: () => handleAuth(r) },
    { key: 'reset', label: '重置密码', onClick: () => handleReset(r) },
    { key: 'delete', label: '删除', danger: true, confirm: '确认删除？', onClick: () => void handleDelete(r) },
  ],
})
```

- 二次确认统一通过 `RowAction.confirm` 声明：横排操作用 `Popconfirm` 就地气泡确认，下拉菜单项用 `Modal.confirm`（下拉会先关闭，气泡无法稳定锚定）。页面不得再自行包裹 `Popconfirm`。
- 运行时隐藏某操作（如已吊销密钥不再展示「吊销」）用 `RowAction.hidden`，不要写条件 JSX 破坏操作数量统计。
- 操作列表顺序按重要程度从高到低排列，被收起的是次要/危险操作。

**R2 详情通过点击名称进入，名称必须有可点击的 UI 展示。**

- 列表不再提供独立的「详情」操作按钮，详情入口收敛到名称列：名称渲染为主色链接（hover 下划线 + 手型光标），让用户一眼看出可点击。
- 统一用 `nameColumn`（内部 `NameLink`）：`nameColumn<T>({ title: '名称', dataIndex: 'name', onClick: (r) => void openDetail(r) })`；名称过长自动省略号截断并悬浮展示全称，编码/日志键类名称传 `monospace`。
- 名称成为详情入口后，操作列删除「详情」项；若某表删除「详情」后已无任何操作，则整体移除操作列（`scroll.x` 由 `tableScrollX` 自动收窄，无需手改）。

### 列表页列宽与横向滚动规范（列宽收敛 + 操作列常驻）

这是「列建得太宽 → 一进列表就有横向滚动条 → 操作列要横滑很久才看到」的根因治理，规范细节见 `frontend/DESIGN.md` §7.3。

- **列宽优先取共享常量或列工厂**：`ID_COL_WIDTH`(130) / `NAME_COL_WIDTH`(180) / `CODE_COL_WIDTH`(150) / `TAG_COL_WIDTH`(110) / `STATUS_COL_WIDTH`(100) / `COUNT_COL_WIDTH`(90) / `TEXT_COL_WIDTH`(200) / `LONG_TEXT_COL_WIDTH`(320)，或直接用 `idColumn` / `nameColumn` / `textColumn` / `timeColumn` / `actionColumn` 工厂。常量不合适时可在调用处传显式 `width`（`scroll.x` 由 `tableScrollX` 求和，不会漂移），但**同类列在各页必须同宽**，禁止同一字段一处 150、一处 180，也禁止与内容无关的整百凑数宽度。
- **所有列都必须有显式列宽**：`tableLayout="fixed"` 下没有宽度的列会被压扁，因此即使是自由文本列也要给 `TEXT_COL_WIDTH` / `LONG_TEXT_COL_WIDTH`。
- **Table 必须写 `tableLayout="fixed"`**：auto 布局下列宽只是建议值，长文本会把列撑开，`EllipsisCell` / `TimeCell` 的省略号与 `nowrap` 都会失效。
- **`scroll.x` 必须写 `scroll={tableScrollX(columns)}`**（由列宽求和得出），**禁止手写 `scroll={{ x: 1490 }}`**：手写值会随列增删漂移（历史 bug：API Key 页实际合计 1580 却仍写 1520），且写大了会让本可放下的表格强制出现横滚条。总宽小于容器时表格按 `min-width: 100%` 铺满，不留白也无滚动条。
- **文本列一律走 `textColumn`**（内部 `EllipsisCell`：超长省略号截断 + 悬浮全文），等宽语义（编码 / key / 域名 / IP）传 `monospace: true`；不要用裸 `<span>` + 手写样式，也不要用裸字符串 render（长文本会换行撑高行、或把列撑宽）。

### 列表页时间列规范（创建时间 + 更新时间）

所有列表页（Table）的时间列遵守以下硬规则，规范细节同步维护在 `frontend/DESIGN.md` §7.3。

**R3 一律展示「创建时间」，可编辑业务主体同时展示「更新时间」。**

- 所有列表页必须有**创建时间**列；记录本身可被编辑/状态流转的**业务主体**（租户、应用、OAuth 客户端、域名、租户应用、菜单、角色、成员、服务账号、部门、API Key 等）还必须同时有**更新时间**列。
- 后端列表 DTO 必须同步回传 `createdAt`/`updatedAt`（秒级 int64 时间戳）：DTO 加字段、service 出参用 `x.Unix()` 赋值、并补测试断言两个字段均 `> 0`（参考 `TestTenantPageListReturnsTimeFields`）。**禁止前端用其他字段派生更新时间**。
- **纯追加型 / 不可变记录不设更新时间列**：审计日志、登录日志、客户端密钥（OAuth Secret）、第三方身份绑定等 `updated_at` 恒等于 `created_at`，展示无意义。这类记录若已有事件时间列（如登录日志的「登录时间」）即视为已承担创建时间语义，不再重复加「创建时间」列。
- 时间列一律用 `timeColumn<T>({ title, dataIndex })`（列宽由 `TIME_COL_WIDTH` / `TIME_COL_WIDTH_RELATIVE` 给，禁止手写 150/160/170 造成折行）；两列位置统一在状态列之后、操作列之前（`scroll.x` 由 `tableScrollX(columns)` 自动跟随，无需手改）。
- 可空时间（`过期时间`/`最后使用`/`验证时间`）用 `placeholder` 表达空值语义，次要时间用 `relative: true`；`创建时间`/`更新时间` 一律绝对时间。

## 常用工具

- **依赖管理**: go mod
- **API 文档**: swag (Swag Go)
- **代码生成**: gocli
- **数据库**: GORM with PostgreSQL（主库，启动时 AutoMigrate 自动建表 + 幂等种子数据），测试用 SQLite
- **缓存**: Redis
- **链路追踪**: OpenTelemetry
- **日志**: golib/glog