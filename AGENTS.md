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
│   ├── pkg/               # 公共包（config/middleware/iam/stdb/testsetup 等）
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
# 运行指定应用的测试（推荐）
make test APP=gateway

# 运行所有测试
go test ./...

# 运行单个测试函数
go test ./pkg/iam/service/svcuser -run TestGeneratePassword -v

# 运行特定包测试
go test ./apps/auth/internal/router/... -v

# 生成测试覆盖率报告
go test ./apps/platformadmin/internal/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Lint 和代码检查

```bash
# 运行 golangci-lint
make lint

# 仅运行特定 linter
golangci-lint run ./... --disable-all -E golint,errcheck,staticcheck

# go vet
go vet ./...
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
│   │   ├── dto/dtoxxx/          # DTO 层
│   │   ├── router/              # 路由注册
│   │   └── middleware/          # 中间件
│   ├── model/               # 数据模型
│   └── dao/                 # 数据访问层
├── platformadmin/              # 平台管理（结构同 auth）
├── tenantadmin/                # 租户自服务（结构同 auth）
├── gateway/                    # 聚合应用（挂载 auth/platformadmin/tenantadmin）
pkg/                          # 公共包（跨应用共享：config/middleware/ginserver/iam 等）
```

> 跨应用共享的 model/dao/object 抽取到 `pkg/iam`，通用中间件抽取到 `pkg/middleware`，避免分体间重复代码。

### 命名规范

- **包名**: 纯小写，无下划线连线，简短且按业务领域划分，如 `svcuser`（用户服务）、`svcrole`（角色服务）
- **接口名**: 以 `I` 结尾或使用角色后缀，如 `UserSvc`, `UserCtr`
- **结构体**: 导出使用大驼峰 `UserSvc`，非导出使用小驼峰 `userSvc`
- **文件命名**: 小写下划线，如 `user_service.go`，测试文件 `*_test.go`
- **数据库表**: 下划线命名，如 `user_department`

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
- 路由按领域注册（如 `/v1/platform/user/*` 下包含用户及其相关操作）

### 常量定义规范

**凡字典字符串，禁止硬编码，均需定义成常量。**

| 常量类型 | 存放位置 | 说明 |
|---------|---------|------|
| 数据表名常量 | `model/*.go` | 定义在对应结构体文件 |
| 数据库存储的枚举值 | `model/*.go` | 定义在对应结构体文件 |
| 字典值常量 | `model/*.go` | 所有字典字符串定义成常量 |
| 应用层常量（前端专用） | `internal/constant/` | 状态映射等前端专用常量 |

#### 数据表常量（model 层）

```go
// model/department.go

// 数据表名
const TableNameDepartment = "iam_department"

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
- **ID 字段命名**：Go 字段与 JSON tag 一律 `ID` 全大写（`userID`、`roleID`、`appID`、`connectorID`），禁止 `roleId`/`appId` 等小写 d 写法；路由 path 参数（`:roleID`）与 swagger 注解同步。
- **DTO 命名**：统一 `<业务名词><动词>Req/Resp`（如 `UserCreateReq`、`DomainCreateReq`），禁止 `CreateDomainReq`、裸 `CreateReq` 等变体。
- **DTO ID 类型**：统一 `uint`（与 `gorm.Model.ID` 一致），禁止 `uint64`。
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

- **路径格式**: `/{版本}/{服务标识}/{模块}/{操作}`，如 `/v1/platform/user/create`（`服务标识` 即应用标识段：auth → `/v1/auth`、platformadmin → `/v1/platform`、tenantadmin → `/v1/tenant`，各应用路径互不相同；模块名可跨应用复用，由服务标识段区分归属）
- **版本号**: 放在路径最前，使用 `/v1/`, `/v2/` 等格式
- **服务标识**: 用于区分不同服务/应用，如 `auth`, `platform`, `tenant`（OIDC 标准端点 `/oidc/*` 除外，不随应用分段）
- **模块名**: 对应业务模块，如 `user`, `role`, `menu`
- **操作名**: 使用驼峰命名，如 `create`, `update`, `detail`, `pageList`, `assignDepartment`

**路由层级限制为5层**，避免使用多层嵌套路径。

#### 路由示例

| 模块 | 操作 | 完整路径 |
|------|------|----------|
| user | 创建 | `/v1/platform/user/create` |
| user | 分配部门 | `/v1/platform/user/assignDepartment` |
| role | 列表 | `/v1/platform/role/pageList` |
| 认证操作 | 注册 | `/v1/auth/register`（auth 应用认证操作直接挂服务段，避免 `/v1/auth/auth/*`） |

#### 路由注册

各业务 app 通过 `ginserver.NewRouterGroups(engine, "<服务标识>", ...)` 注册应用前缀（auth: `"auth"`、platformadmin: `"platform"`、tenantadmin: `"tenant"`），然后在 `router/router.go` 中按版本 `MustGetGroup(ginserver.ApiVersionV1)` 注册各模块路由：

```go
routerGroups := ginserver.NewRouterGroups(engine, "platform", ginserver.VersionGroup{
    Version: ginserver.ApiVersionV1,
    // ...
})
```

在各个路由文件中使用 gin 的路由注册方法：

```go
func userRouter(groups *ginserver.RouterGroups) {
    v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
    v1RouterGroup.POST("/user/create", userCtr.Create)
    v1RouterGroup.POST("/user/delete", userCtr.Delete)
    v1RouterGroup.GET("/user/detail", userCtr.Detail)
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
// @Router /v1/platform/user/create [post]
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

## 常用工具

- **依赖管理**: go mod
- **API 文档**: swag (Swag Go)
- **代码生成**: gocli
- **数据库**: GORM with MySQL/PostgreSQL
- **缓存**: Redis
- **链路追踪**: OpenTelemetry
- **日志**: golib/glog