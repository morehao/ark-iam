# AGENTS.md - GoArk 代码库开发指南

本文档为在此代码库中工作的 AI 代理提供开发规范和命令参考。

## 项目概述

GoArk 是一个基于 Gin + GORM 的多应用 Go 后端项目，支持 IAM（身份认证）和 Demo 两个应用模块。

## 构建与运行命令

```bash
# 列出所有可用应用
make list-apps

# 构建指定应用
make build APP=demo
make build APP=iam

# 运行指定应用
make run APP=demo
make run APP=iam

# 下载依赖
make deps

# 清理构建产物
make clean
```

## 测试命令

```bash
# 运行指定应用的测试（推荐）
make test APP=demo
make test APP=iam

# 运行所有测试
go test ./...

# 运行单个测试函数
go test ./apps/demo/internal/service/svcuser -run TestGeneratePassword -v

# 运行特定包测试
go test ./apps/iam/internal/service/svcuser/... -v

# 生成测试覆盖率报告
go test ./apps/demo/internal/... -coverprofile=coverage.out
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
├── demo/                      # Demo 应用
│   ├── cmd/                   # 入口函数
│   ├── internal/
│   │   ├── controller/ctrxxx/  # 控制器层 (ctr 前缀)
│   │   ├── service/svcxxx/     # 服务层 (svc 前缀)
│   │   ├── dto/dtoxxx/         # DTO 层
│   │   ├── router/             # 路由注册
│   │   └── middleware/         # 中间件
│   ├── model/              # 数据模型
│   └── dao/                # 数据访问层
├── iam/                       # IAM 应用 (同上结构)
│   └── internal/
│       ├── constant/           # 应用层常量（前端专用）
pkg/                          # 公共包
```

### 命名规范

- **包名**: 小写，简短，如 `svcuser`, `dtouser`
- **接口名**: 以 `I` 结尾或使用角色后缀，如 `UserSvc`, `UserCtr`
- **结构体**: 导出使用大驼峰 `UserSvc`，非导出使用小驼峰 `userSvc`
- **文件命名**: 小写下划线，如 `user_service.go`，测试文件 `*_test.go`
- **数据库表**: 下划线命名，如 `user_department`

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
// apps/iam/internal/constant/status.go
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
3. 项目内部包 (`github.com/morehao/goark/apps/iam/...`, `github.com/morehao/goark/pkg/...`)
4. 关联库 (`github.com/morehao/golib/...`)

```go
import (
    "strings"

    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify"

    "github.com/morehao/goark/apps/iam/internal/dto/dtouser"
    "github.com/morehao/goark/pkg/code"
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

### 错误处理

- 使用统一的错误码包 `github.com/morehao/goark/pkg/code`
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

- **路径格式**: `/{版本}/{app}/{模块}/{操作}`，如 `/v1/iam/user/create`
- **版本号**: 放在路径最前，使用 `/v1/`, `/v2/` 等格式
- **App 标识**: 用于区分不同应用，如 `iam`, `demo`
- **模块名**: 对应业务模块，如 `user`, `role`, `menu`
- **操作名**: 使用驼峰命名，如 `create`, `update`, `detail`, `pageList`, `assignDepartment`

**路由层级限制为4层**，避免使用多层嵌套路径。

#### 路由示例

| 模块 | 操作 | 完整路径 |
|------|------|----------|
| user | 创建 | `/v1/iam/user/create` |
| user | 分配部门 | `/v1/iam/user/assignDepartment` |
| role | 列表 | `/v1/iam/role/pageList` |

#### 路由注册

在 `router/router.go` 中注册路由，先按版本分组，再按应用分组：

```go
v1AuthGroup := groups.AuthGroup.Group("/v1")
iamGroup := v1AuthGroup.Group("/iam")

userRouter(iamGroup)
roleRouter(iamGroup)
```

在各个路由文件中使用 gin 的路由注册方法：

```go
func userRouter(routerGroup *gin.RouterGroup) {
    routerGroup.POST("/user/create", userCtr.Create)
    routerGroup.POST("/user/delete", userCtr.Delete)
    routerGroup.GET("/user/detail", userCtr.Detail)
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
// @Router /v1/demo/user/create [post]
```

生成文档：

```bash
make swag APP=demo
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
make codegen APP=demo COMMAND=api

# 生成模块代码
make codegen APP=demo COMMAND=module

# 生成模型代码
make codegen APP=demo COMMAND=model
```

### Docker 支持

```bash
# 构建 Docker 镜像
make docker-build APP=demo

# 运行 Docker 容器
make docker-run APP=demo
```

## 常用工具

- **依赖管理**: go mod
- **API 文档**: swag (Swag Go)
- **代码生成**: gocli
- **数据库**: GORM with MySQL/PostgreSQL
- **缓存**: Redis
- **链路追踪**: OpenTelemetry
- **日志**: golib/glog