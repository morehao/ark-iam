# ark-iam/iam —— 可嵌入 IAM 核心库

让一个新项目「引入即自带」账号认证与管理体系（登录/会话 + 人员/部门/角色/菜单），
支持两种部署形态，共用同一份实现：

- **形态 1：IAM 单独部署**，三方通过 `sdk`（`github.com/morehao/ark-iam/sdk`）验签接入。
- **形态 2：内嵌三方单体**，`iam.New` + `Mount` 把 IAM 装进三方自己的 Go 进程，前端 `embed`，与三方一起部署为一个服务。

> 见 `.dsh/docs/specs/2026-11-06-embeddable-iam-design.md`（设计）与 `.dsh/docs/plans/2026-11-06-embeddable-iam-plan.md`（执行计划）。

## 形态 2：内嵌（最小接入）

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/morehao/ark-iam/iam"
    _ "github.com/morehao/golib/dbaccess/dbgorm/driver/sqlite" // 或 postgres
    _ "github.com/morehao/golib/glog/driver/zap"
)

func main() {
    app, err := iam.New(iam.Config{
        Env:          "dev",
        DB:           iam.DBConfig{DSN: "postgres://user:pass@host:5432/mydb?sslmode=disable"},
        Redis:        &iam.RedisConfig{Addr: "127.0.0.1:6379"}, // 可空
        Domains:      iam.DefaultConsoleDomains,                // 部门/人员/角色/菜单最小集
        SameDatabase: true,                                     // IAM 表与业务表同库，启用撞名检测
    })
    if err != nil { panic(err) }
    defer app.Close()

    if err := app.RunMigrations(context.Background()); err != nil { panic(err) }

    engine := gin.New()
    if err := app.Mount(engine.Group("")); err != nil { panic(err) }        // 挂 API（含 /install、/oidc）
    if err := app.MountFrontend(engine, "/"); err != nil { panic(err) }    // 内嵌登录/初始化页
    engine.Run(":8080")
}
```

启动后：访问 `/install` 初始化管理员 → `/login` 登录。

## 对外 API

| 方法 | 作用 |
|---|---|
| `iam.New(Config) (*App, error)` | 实例化；校验 DSN / 域依赖闭合 / 非 dev 安全配置 |
| `App.RunMigrations(ctx)` | 建表（AutoMigrate 只增不删；`SameDatabase=true` 时含撞名检测） |
| `App.Mount(rg *gin.RouterGroup)` | 按 `Domains` 挂 `/v1/{platform,tenant,auth}` + `/oidc` + `/install`，套鉴权中间件 |
| `App.MountFrontend(engine, basePath)` | 托管内嵌登录页 + `/config.json` 运行时配置 |
| `App.DBGetter()` / `App.KeySource()` / `App.Redis()` | 供宿主复用连接 / OP 公钥 / Redis |
| `App.Close()` | 释放 DB / Redis |

## 域（Domains）与裁剪

`Config.Domains` 为空 = 全量（形态 1 / 全控制台）；形态 2 建议 `DefaultConsoleDomains`
（core-auth + install + person + user + department + role + menu）。

可选企业级域：tenant / application / oauth-client / connector / domain / audit / api-key /
machine-user / invite / tenant-app / self-service。

域依赖：`role` 依赖 `menu`（缺则 `New` 报 `ErrDomainDependency`）。

## 配置要点

- **精简字段**：`DB` / `Redis` / `Env` / `Domains` / `SameDatabase`。
- **完整配置**：`Config.Raw *pkgconfig.Config` —— auth 的 OP（issuer/签名密钥/cookie/控制台回调）需要完整配置。
- **私钥**：dev 自动生成；**非 dev 必须显式配置签名密钥 + encryptionKey + cookieSecure=true**，否则 `New` fail-closed。
- **同库撞名**：`SameDatabase=true` 时，若目标库已存在同名但缺 `id`/`created_at` 的表 → `ErrTableConflict` fail-fast；
  可用 `Config.AllowExistingTables` 白名单放行。
- 三方业务表**不要**用 IAM 的连接查（会被租户插件注入 `tenant_id`），用三方自己的 gorm 连接。

## 形态 1：单独部署 + sdk 接入

部署现网 `gateway`（或分体 auth/platformadmin/tenantadmin），三方 `go get github.com/morehao/ark-iam/sdk`，
按 `sdk/README.md` 用 `rp.NewKeys` + `rp.NewVerifier` 本地验签接入。两形态 token 契约一致（同一 claim 契约）。

## 模块关系

```
iam -> auth/platformadmin/tenantadmin（应用包，复用）-> pkg（model/dao/core/middleware）-> sdk
gateway -> iam.New(全量)（形态 1 的使用者） + rpapi
examples/embedded-iam-demo -> iam（形态 2 的使用者）
```
