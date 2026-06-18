# IAM 应用拆分实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将单一的 `backend/apps/iam` 拆分为 `auth`、`platformadmin`、`tenantadmin` 三个独立应用，共享 `pkg/iam/` 包。

**Architecture:** model/dao/object 迁移到 `backend/pkg/iam/`；auth(8081)/platformadmin(8082)/tenantadmin(8083) 各自独立端口；路由前缀 `/v1/{app}/{module}/{operation}`，auth 不加模块前缀直接 `/v1/auth/{operation}`。注意：platformadmin 和 tenantadmin 的 router 包中需要定义 `var OIDCPublicKey *rsa.PublicKey` 以匹配 app.go 的引用，因为 oidcauth 中间件需要该变量。

**Tech Stack:** Go 1.26.1 + Gin + GORM + Go workspace + Redis

---

### Task 1: 创建 pkg/iam 共享包并迁移 model/dao/object

**Files:**
- Create: `backend/pkg/iam/model/` (所有 model 文件)
- Create: `backend/pkg/iam/dao/` (所有 dao 文件)
- Create: `backend/pkg/iam/object/` (所有 object 文件和子目录)
- Modify: `backend/pkg/go.mod`

- [ ] **Step 1: 创建目标目录并复制文件**

```bash
mkdir -p backend/pkg/iam/{model,dao,object}
cp backend/apps/iam/model/*.go backend/pkg/iam/model/
cp backend/apps/iam/dao/*.go backend/pkg/iam/dao/
for d in backend/apps/iam/object/*/; do
  dirname=$(basename "$d")
  mkdir -p "backend/pkg/iam/object/$dirname"
  cp "$d"*.go "backend/pkg/iam/object/$dirname/"
done
```

- [ ] **Step 2: 更新 pkg/iam 中所有 model/object 引用路径**

```bash
find backend/pkg/iam/dao -name "*.go" -exec sed -i '' 's|github.com/morehao/ark-iam/iam/model|github.com/morehao/ark-iam/pkg/iam/model|g' {} +
find backend/pkg/iam/object -name "*.go" -exec sed -i '' 's|github.com/morehao/ark-iam/iam/model|github.com/morehao/ark-iam/pkg/iam/model|g' {} +
find backend/pkg/iam/dao -name "*.go" -exec sed -i '' 's|github.com/morehao/ark-iam/iam/object|github.com/morehao/ark-iam/pkg/iam/object|g' {} +
```

- [ ] **Step 3: 运行 go mod tidy 并验证编译**

```bash
cd backend && go work sync && cd pkg && go mod tidy
cd backend && go build ./pkg/iam/...
```

预期：编译通过。

- [ ] **Step 4: 提交**

```bash
git add backend/pkg/iam/ backend/pkg/go.mod backend/pkg/go.sum
git commit -m "feat: migrate model/dao/object from apps/iam to pkg/iam"
```

---

### Task 2: 创建三个应用骨架（cmd + config + go.mod + app.go）

**Files:**
- Create: `backend/apps/auth/{cmd/main.go,cmd/init.go,config/config.go,go.mod,app.go}`
- Create: `backend/apps/platformadmin/{cmd/main.go,cmd/init.go,config/config.go,go.mod,app.go}`
- Create: `backend/apps/tenantadmin/{cmd/main.go,cmd/init.go,config/config.go,go.mod,app.go}`
- Modify: `backend/go.work`

- [ ] **Step 1: 创建目录结构和 go.mod**

```bash
for app in auth platformadmin tenantadmin; do
  mkdir -p backend/apps/$app/{cmd,config,internal/{controller,service,dto,router,middleware},docs}
  cat > backend/apps/$app/go.mod << GOMOD
module github.com/morehao/ark-iam/$app

go 1.26.1
GOMOD
done
```

- [ ] **Step 2: 创建各应用的 cmd/init.go**

各应用的 `init.go` 基于 `apps/iam/cmd/init.go`，差异仅在 import 路径和 `AppName`：

```go
// backend/apps/auth/cmd/init.go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/morehao/ark-iam/auth"
	"github.com/morehao/ark-iam/auth/config"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gtrace"
	"github.com/morehao/golib/gtrace/otlptracegrpc"
)

var traceProvider *gtrace.Provider

func serverInit() error {
	if err := preInit(); err != nil {
		return err
	}
	if err := initTrace(); err != nil {
		return err
	}
	if err := resourceInit(); err != nil {
		return err
	}
	return nil
}

func preInit() error {
	config.InitConf()
	defaultLogCfg := config.Conf.Log["default"]
	if err := glog.InitLogger(&defaultLogCfg); err != nil {
		return fmt.Errorf("init logger failed: %w", err)
	}
	return nil
}

func resourceInit() error {
	var gormLogConfig *glog.LogConfig
	if cfg, ok := config.Conf.Log["gorm"]; ok {
		gormLogConfig = &cfg
	}
	if err := dbclient.InitMultiDB(config.Conf.DBConfigs, gormLogConfig); err != nil {
		return fmt.Errorf("init db failed: %w", err)
	}

	var redisLogConfig *glog.LogConfig
	if cfg, ok := config.Conf.Log["redis"]; ok {
		redisLogConfig = &cfg
	}
	if err := dbclient.InitRedis(config.Conf.RedisConfig, redisLogConfig); err != nil {
		return fmt.Errorf("init redis failed: %w", err)
	}
	return nil
}

func initTrace() error {
	provider, err := otlptracegrpc.NewGRPCProvider(context.Background(), auth.AppName, config.Conf.Server.Env, config.Conf.Trace)
	if err != nil {
		glog.Errorf(context.Background(), "[%s.initTrace] init trace failed, fallback to disabled mode, err:%v", auth.AppName, err)
		return nil
	}
	traceProvider = provider
	return nil
}

func shutdownTraceProvider() {
	if traceProvider == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := traceProvider.Shutdown(ctx); err != nil {
		glog.Errorf(context.Background(), "[%s.shutdownTraceProvider] shutdown fail, err:%v", auth.AppName, err)
	}
}
```

platformadmin/tenantadmin 同理，将 `auth` 替换为对应的 app 名，端口分别用 8082、8083。

- [ ] **Step 3: 创建各应用的 cmd/main.go**

```go
// backend/apps/auth/cmd/main.go
package main

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth"
)

func main() {
	if err := serverInit(); err != nil {
		panic(err)
	}
	defer shutdownTraceProvider()

	engine := gin.Default()
	auth.Routers(engine)
	if err := engine.Run(":8081"); err != nil {
		panic(err)
	}
}
```

```go
// backend/apps/platformadmin/cmd/main.go
package main

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/platformadmin"
)

func main() {
	if err := serverInit(); err != nil {
		panic(err)
	}
	defer shutdownTraceProvider()

	engine := gin.Default()
	platformadmin.Routers(engine)
	if err := engine.Run(":8082"); err != nil {
		panic(err)
	}
}
```

```go
// backend/apps/tenantadmin/cmd/main.go
package main

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/tenantadmin"
)

func main() {
	if err := serverInit(); err != nil {
		panic(err)
	}
	defer shutdownTraceProvider()

	engine := gin.Default()
	tenantadmin.Routers(engine)
	if err := engine.Run(":8083"); err != nil {
		panic(err)
	}
}
```

- [ ] **Step 4: 创建各应用的 config/config.go**

从 `apps/iam/config/config.go` 完整复制，仅改 package 为 `config`。内容与原始完全相同（Config 结构体、InitConf() 函数）。

复制命令：
```bash
for app in auth platformadmin tenantadmin; do
  cp backend/apps/iam/config/config.go backend/apps/$app/config/config.go
done
```

- [ ] **Step 5: 创建各应用的 app.go**

```go
// backend/apps/auth/app.go
package auth

import (
	"crypto/rsa"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/config"
	"github.com/morehao/ark-iam/auth/internal/middleware/oidcauth"
	"github.com/morehao/ark-iam/auth/internal/router"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gmiddleware/ginmiddleware"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

const AppName = "auth"

func Routers(engine *gin.Engine) {
	routerGroups := ginserver.NewRouterGroups(engine, AppName, ginserver.Version{
		Name: gconstant.ApiVersionV1,
		Middlewares: []gin.HandlerFunc{
			oidcauth.OIDCCompatibleAuth(config.Conf.JWT.SignKey, func() *rsa.PublicKey { return router.OIDCPublicKey }, oidcauth.WithAuthSkipPaths(
				"/v1/auth/login",
				"/v1/auth/myTenants",
				"/v1/auth/selectTenant",
				"/v1/auth/register",
				"/v1/auth/refreshToken",
				"/v1/auth/connector/callback",
				"/v1/auth/oidc",
			)),
			ginmiddleware.TokenBlacklistCheck(dbclient.RedisCli, ginmiddleware.WithBlacklistKeyPrefix("auth:token:blacklist:")),
		},
	})

	router.RegisterRouter(routerGroups, AppName)
	router.InitOIDC(engine, routerGroups)
}
```

```go
// backend/apps/platformadmin/app.go
package platformadmin

import (
	"crypto/rsa"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/platformadmin/config"
	"github.com/morehao/ark-iam/platformadmin/internal/middleware/oidcauth"
	"github.com/morehao/ark-iam/platformadmin/internal/router"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gmiddleware/ginmiddleware"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

const AppName = "platformadmin"

func Routers(engine *gin.Engine) {
	routerGroups := ginserver.NewRouterGroups(engine, AppName, ginserver.Version{
		Name: gconstant.ApiVersionV1,
		Middlewares: []gin.HandlerFunc{
			oidcauth.OIDCCompatibleAuth(config.Conf.JWT.SignKey, func() *rsa.PublicKey { return router.OIDCPublicKey }),
			ginmiddleware.TokenBlacklistCheck(dbclient.RedisCli, ginmiddleware.WithBlacklistKeyPrefix("platformadmin:token:blacklist:")),
		},
	})

	router.RegisterRouter(routerGroups, AppName)
}
```

```go
// backend/apps/tenantadmin/app.go
package tenantadmin

import (
	"crypto/rsa"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/tenantadmin/config"
	"github.com/morehao/ark-iam/tenantadmin/internal/middleware/oidcauth"
	"github.com/morehao/ark-iam/tenantadmin/internal/router"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gmiddleware/ginmiddleware"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

const AppName = "tenantadmin"

func Routers(engine *gin.Engine) {
	routerGroups := ginserver.NewRouterGroups(engine, AppName, ginserver.Version{
		Name: gconstant.ApiVersionV1,
		Middlewares: []gin.HandlerFunc{
			oidcauth.OIDCCompatibleAuth(config.Conf.JWT.SignKey, func() *rsa.PublicKey { return router.OIDCPublicKey }),
			ginmiddleware.TokenBlacklistCheck(dbclient.RedisCli, ginmiddleware.WithBlacklistKeyPrefix("tenantadmin:token:blacklist:")),
		},
	})

	router.RegisterRouter(routerGroups, AppName)
}
```

- [ ] **Step 6: 更新 go.work**

在 `backend/go.work` 的 `use (...)` 块中添加三个应用：

```
go 1.26.1

use (
    ./apps/auth
    ./apps/iam
    ./apps/platformadmin
    ./apps/tenantadmin
    ./pkg
)
```

- [ ] **Step 7: 运行 go mod tidy 三个应用**

```bash
for app in auth platformadmin tenantadmin; do
  cd backend/apps/$app && go mod tidy
done
```

- [ ] **Step 8: 编译验证（预期有编译错误，正常）**

```bash
cd backend && go build ./apps/auth/... 2>&1 | head -20
cd backend && go build ./apps/platformadmin/... 2>&1 | head -20
cd backend && go build ./apps/tenantadmin/... 2>&1 | head -20
```

- [ ] **Step 9: 提交**

```bash
git add backend/apps/auth/ backend/apps/platformadmin/ backend/apps/tenantadmin/ backend/go.work backend/go.work.sum
git commit -m "feat: create auth, platformadmin, tenantadmin app skeletons"
```

---

### Task 3: 迁移 middleware 到各应用

**Files:**
- Create: `backend/apps/auth/internal/middleware/oidcauth/oidcauth.go`
- Create: `backend/apps/auth/internal/middleware/silent_oidc.go`
- Create: `backend/apps/auth/internal/middleware/silent_oidc_test.go`
- Create: `backend/apps/auth/internal/middleware/apikey_auth.go`
- Create: `backend/apps/auth/internal/middleware/apikey_auth_test.go`
- Create: `backend/apps/platformadmin/internal/middleware/oidcauth/oidcauth.go`
- Create: `backend/apps/platformadmin/internal/middleware/apikey_auth.go`
- Create: `backend/apps/platformadmin/internal/middleware/apikey_auth_test.go`
- Create: `backend/apps/tenantadmin/internal/middleware/oidcauth/oidcauth.go`

- [ ] **Step 1: 复制 middleware 到各应用**

```bash
# auth
cp -r backend/apps/iam/internal/middleware/oidcauth backend/apps/auth/internal/middleware/
cp backend/apps/iam/internal/middleware/silent_oidc.go backend/apps/auth/internal/middleware/
cp backend/apps/iam/internal/middleware/silent_oidc_test.go backend/apps/auth/internal/middleware/
cp backend/apps/iam/internal/middleware/apikey_auth.go backend/apps/auth/internal/middleware/
cp backend/apps/iam/internal/middleware/apikey_auth_test.go backend/apps/auth/internal/middleware/

# platformadmin
cp -r backend/apps/iam/internal/middleware/oidcauth backend/apps/platformadmin/internal/middleware/
cp backend/apps/iam/internal/middleware/apikey_auth.go backend/apps/platformadmin/internal/middleware/
cp backend/apps/iam/internal/middleware/apikey_auth_test.go backend/apps/platformadmin/internal/middleware/

# tenantadmin
cp -r backend/apps/iam/internal/middleware/oidcauth backend/apps/tenantadmin/internal/middleware/
```

- [ ] **Step 2: 全局替换 import 路径**

```bash
# auth 应用：iam/ -> auth/，然后 model 修正为 pkg/iam/model
find backend/apps/auth/internal/middleware -name "*.go" -exec sed -i '' 's|github.com/morehao/ark-iam/iam/|github.com/morehao/ark-iam/auth/|g' {} +
find backend/apps/auth/internal/middleware -name "*.go" -exec sed -i '' 's|github.com/morehao/ark-iam/auth/model|github.com/morehao/ark-iam/pkg/iam/model|g' {} +

# platformadmin 应用
find backend/apps/platformadmin/internal/middleware -name "*.go" -exec sed -i '' 's|github.com/morehao/ark-iam/iam/|github.com/morehao/ark-iam/platformadmin/|g' {} +
find backend/apps/platformadmin/internal/middleware -name "*.go" -exec sed -i '' 's|github.com/morehao/ark-iam/platformadmin/model|github.com/morehao/ark-iam/pkg/iam/model|g' {} +

# tenantadmin 应用
find backend/apps/tenantadmin/internal/middleware -name "*.go" -exec sed -i '' 's|github.com/morehao/ark-iam/iam/|github.com/morehao/ark-iam/tenantadmin/|g' {} +
find backend/apps/tenantadmin/internal/middleware -name "*.go" -exec sed -i '' 's|github.com/morehao/ark-iam/tenantadmin/model|github.com/morehao/ark-iam/pkg/iam/model|g' {} +
```

- [ ] **Step 3: go mod tidy 和编译验证**

```bash
cd backend/apps/auth && go mod tidy
cd backend/apps/platformadmin && go mod tidy
cd backend/apps/tenantadmin && go mod tidy
cd backend && go build ./apps/auth/... ./apps/platformadmin/... ./apps/tenantadmin/...
```

- [ ] **Step 4: 提交**

```bash
git add backend/apps/auth/internal/middleware/ backend/apps/platformadmin/internal/middleware/ backend/apps/tenantadmin/internal/middleware/
git add backend/apps/auth/go.mod backend/apps/auth/go.sum
git add backend/apps/platformadmin/go.mod backend/apps/platformadmin/go.sum
git add backend/apps/tenantadmin/go.mod backend/apps/tenantadmin/go.sum
git commit -m "feat: copy middleware to auth, platformadmin, tenantadmin apps"
```

### Task 4: 迁移 auth 应用业务代码

**Files:**
- Create: `backend/apps/auth/internal/service/svcauth/` (全部)
- Create: `backend/apps/auth/internal/service/svcoidc/` (全部)
- Create: `backend/apps/auth/internal/service/svcperson/` (全部)
- Create: `backend/apps/auth/internal/service/svcsession/` (全部)
- Create: `backend/apps/auth/internal/controller/ctrauth/` (全部)
- Create: `backend/apps/auth/internal/controller/ctroidc/` (全部)
- Create: `backend/apps/auth/internal/controller/ctrperson/` (全部)
- Create: `backend/apps/auth/internal/controller/ctrsession/` (全部)
- Create: `backend/apps/auth/internal/dto/dtoauth/` (全部)
- Create: `backend/apps/auth/internal/dto/dtooidc/` (全部)
- Create: `backend/apps/auth/internal/dto/dtoperson/` (全部)
- Create: `backend/apps/auth/internal/dto/dtouser/` (仅 session.go)
- Create: `backend/apps/auth/internal/router/router.go`
- Create: `backend/apps/auth/internal/router/auth.go`
- Create: `backend/apps/auth/internal/router/oidc.go`
- Create: `backend/apps/auth/internal/router/person.go`

- [ ] **Step 1: 复制 service/controller/dto**

```bash
cp -r backend/apps/iam/internal/service/svcauth backend/apps/auth/internal/service/
cp -r backend/apps/iam/internal/service/svcoidc backend/apps/auth/internal/service/
cp -r backend/apps/iam/internal/service/svcperson backend/apps/auth/internal/service/
cp -r backend/apps/iam/internal/service/svcsession backend/apps/auth/internal/service/
cp -r backend/apps/iam/internal/controller/ctrauth backend/apps/auth/internal/controller/
cp -r backend/apps/iam/internal/controller/ctroidc backend/apps/auth/internal/controller/
cp -r backend/apps/iam/internal/controller/ctrperson backend/apps/auth/internal/controller/
cp -r backend/apps/iam/internal/controller/ctrsession backend/apps/auth/internal/controller/
cp -r backend/apps/iam/internal/dto/dtoauth backend/apps/auth/internal/dto/
cp -r backend/apps/iam/internal/dto/dtooidc backend/apps/auth/internal/dto/
cp -r backend/apps/iam/internal/dto/dtoperson backend/apps/auth/internal/dto/
cp -r backend/apps/iam/internal/dto/dtouser/session.go backend/apps/auth/internal/dto/dtouser/
```

- [ ] **Step 2: 全局替换 import 路径**

```bash
find backend/apps/auth/internal -name "*.go" -exec sed -i '' 's|github.com/morehao/ark-iam/iam/|github.com/morehao/ark-iam/auth/|g' {} +
find backend/apps/auth/internal -name "*.go" -exec sed -i '' 's|github.com/morehao/ark-iam/auth/model|github.com/morehao/ark-iam/pkg/iam/model|g' {} +
find backend/apps/auth/internal -name "*.go" -exec sed -i '' 's|github.com/morehao/ark-iam/auth/dao|github.com/morehao/ark-iam/pkg/iam/dao|g' {} +
find backend/apps/auth/internal -name "*.go" -exec sed -i '' 's|github.com/morehao/ark-iam/auth/object|github.com/morehao/ark-iam/pkg/iam/object|g' {} +
```

（注意：此步骤中 model/dao/object 的 sed 必须放在 `iam/` -> `auth/` 的 sed 之后执行，因为 `iam/model` 会被先替换成 `auth/model`，然后再纠正为 `pkg/iam/model`）

- [ ] **Step 3: 创建 router/router.go**

```go
package router

import "github.com/morehao/golib/biz/gserver/ginserver"

func RegisterRouter(groups *ginserver.RouterGroups, appName string) {
	authRouter(groups)
	personRouter(groups)
}
```

- [ ] **Step 4: 创建 router/auth.go**

```go
package router

import (
	"github.com/morehao/ark-iam/auth/config"
	"github.com/morehao/ark-iam/auth/internal/controller/ctrauth"
	"github.com/morehao/ark-iam/auth/internal/controller/ctrsession"
	"github.com/morehao/ark-iam/auth/internal/service/svcauth"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func authRouter(groups *ginserver.RouterGroups) {
	authSvc := svcauth.NewAuthSvc(config.Conf.JWT.SignKey)
	authCtr := ctrauth.NewAuthCtr(authSvc)
	sessionCtr := ctrsession.NewSessionCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/auth/login", authCtr.Login)
	v1RouterGroup.GET("/auth/myTenants", authCtr.MyTenants)
	v1RouterGroup.POST("/auth/selectTenant", authCtr.SelectTenant)
	v1RouterGroup.POST("/auth/switchTenant", authCtr.SwitchTenant)
	v1RouterGroup.POST("/auth/register", authCtr.Register)
	v1RouterGroup.POST("/auth/joinTenant", authCtr.JoinTenant)
	v1RouterGroup.POST("/auth/refreshToken", authCtr.RefreshToken)
	v1RouterGroup.POST("/auth/logout", authCtr.Logout)
	v1RouterGroup.POST("/auth/logoutAll", authCtr.LogoutAll)
	v1RouterGroup.GET("/auth/userinfo", authCtr.Userinfo)

	v1RouterGroup.GET("/auth/user/sessions", sessionCtr.List)
	v1RouterGroup.DELETE("/auth/user/sessions", sessionCtr.RevokeAll)
	v1RouterGroup.DELETE("/auth/user/sessions/:sessionId", sessionCtr.Revoke)
}

func connectorRouter(groups *ginserver.RouterGroups) {
	connectorCtr := ctrauth.NewConnectorCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.GET("/auth/connector/callback", connectorCtr.Callback)
}
```

- [ ] **Step 5: 创建 router/person.go**

```go
package router

import (
	"github.com/morehao/ark-iam/auth/config"
	"github.com/morehao/ark-iam/auth/internal/controller/ctrperson"
	"github.com/morehao/ark-iam/auth/internal/service/svcauth"
	"github.com/morehao/ark-iam/auth/internal/service/svcperson"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func personRouter(groups *ginserver.RouterGroups) {
	authSvc := svcauth.NewAuthSvc(config.Conf.JWT.SignKey)
	personCtr := ctrperson.NewPersonCtr(
		svcperson.NewPersonProfileSvc(),
		authSvc,
	)

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.GET("/auth/person/detail", personCtr.Detail)
	v1RouterGroup.POST("/auth/person/updatePassword", personCtr.UpdatePassword)
}
```

- [ ] **Step 6: 创建 router/oidc.go**

参照原 `apps/iam/internal/router/oidc.go`，OIDC 路由前缀从 `/v1/iam/oidc` 改为 `/v1/auth/oidc`：

```go
package router

import (
	"context"
	"crypto/rsa"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/config"
	"github.com/morehao/ark-iam/auth/internal/controller/ctroidc"
	"github.com/morehao/ark-iam/auth/internal/service/svcoidc"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gmiddleware/ginmiddleware"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

var OIDCPublicKey *rsa.PublicKey

func InitOIDC(engine *gin.Engine, groups *ginserver.RouterGroups) {
	issuer := config.Conf.OIDC.Issuer
	if issuer == "" {
		port := config.Conf.Server.Port
		if port == "" {
			port = "8081"
		}
		issuer = fmt.Sprintf("http://localhost:%s/v1/auth/oidc", port)
	}
	provider, err := svcoidc.SetupOIDCProvider(issuer)
	if err != nil {
		panic(err)
	}

	signingKey, err := provider.Storage.SigningKey(context.Background())
	if err != nil {
		panic(err)
	}
	OIDCPublicKey = &signingKey.Key().(*rsa.PrivateKey).PublicKey

	ctr := ctroidc.NewOIDCCtr(provider)
	ssoCookieDomain := config.Conf.OIDC.SSOCookieDomain()

	v1Group := groups.MustGetGroup(gconstant.ApiVersionV1)
	oidcGroup := v1Group.Group("/auth/oidc")
	oidcGroup.Use(ginmiddleware.CORS())
	oidcGroup.POST("/login", ctr.Login)
	oidcGroup.GET("/sso-login", ctr.SSOLogin)
	oidcGroup.GET("/logged-out", func(ctx *gin.Context) {
		ctx.SetCookie("iam_sso_session", "", -1, "/", ssoCookieDomain, false, true)
		ctx.Redirect(302, config.Conf.OIDC.FrontendLoginURL)
	})
	svcoidc.RegisterProviderRoutes(oidcGroup, provider, "iam_sso_session")
}
```

- [ ] **Step 7: 注册 connectorRouter 到 router.go**

在 router.go 中添加 `connectorRouter(groups)`。

- [ ] **Step 8: go mod tidy 和编译验证**

```bash
cd backend/apps/auth && go mod tidy
cd backend && go build ./apps/auth/...
```

修复所有编译错误。

- [ ] **Step 9: 提交**

```bash
git add backend/apps/auth/internal/ backend/apps/auth/go.mod backend/apps/auth/go.sum
git commit -m "feat: migrate auth app business code"
```

---

### Task 5: 迁移 platformadmin 应用业务代码

**Files:**
- Create: `backend/apps/platformadmin/internal/service/svcuser/`、`svctenant/`、`svcpermission/`、`svcapplication/`、`svcapikey/`、`svcdomain/`、`svcoauthclient/`、`svctenantapplication/`
- Create: `backend/apps/platformadmin/internal/controller/ctruser/`、`ctrtenant/`、`ctrpermission/`、`ctrapplication/`、`ctrapikey/`、`ctrdomain/`、`ctroauthclient/`、`ctrtenantapplication/`
- Create: `backend/apps/platformadmin/internal/dto/dtouser/`、`dtotenant/`、`dtopermission/`、`dtoapplication/`、`dtoapikey/`、`dtodomain/`、`dtooauthclient/`、`dtotenantapplication/`、`dtoconnector/`
- Create: `backend/apps/platformadmin/internal/router/` (所有路由文件 + router.go)

- [ ] **Step 1: 复制 service**

```bash
for svc in svcuser svctenant svcpermission svcapplication svcapikey svcdomain svcoauthclient svctenantapplication; do
  cp -r backend/apps/iam/internal/service/$svc backend/apps/platformadmin/internal/service/
done
```

- [ ] **Step 2: 复制 controller**

```bash
for ctr in ctruser ctrtenant ctrpermission ctrapplication ctrapikey ctrdomain ctroauthclient ctrtenantapplication; do
  cp -r backend/apps/iam/internal/controller/$ctr backend/apps/platformadmin/internal/controller/
done
```

- [ ] **Step 3: 复制 dto**

```bash
for dto in dtouser dtotenant dtopermission dtoapplication dtoapikey dtodomain dtooauthclient dtotenantapplication dtoconnector; do
  cp -r backend/apps/iam/internal/dto/$dto backend/apps/platformadmin/internal/dto/
done
```

- [ ] **Step 4: 全局替换 import 路径**

```bash
find backend/apps/platformadmin/internal -name "*.go" -exec sed -i '' 's|github.com/morehao/ark-iam/iam/|github.com/morehao/ark-iam/platformadmin/|g' {} +
find backend/apps/platformadmin/internal -name "*.go" -exec sed -i '' 's|github.com/morehao/ark-iam/platformadmin/model|github.com/morehao/ark-iam/pkg/iam/model|g' {} +
find backend/apps/platformadmin/internal -name "*.go" -exec sed -i '' 's|github.com/morehao/ark-iam/platformadmin/dao|github.com/morehao/ark-iam/pkg/iam/dao|g' {} +
find backend/apps/platformadmin/internal -name "*.go" -exec sed -i '' 's|github.com/morehao/ark-iam/platformadmin/object|github.com/morehao/ark-iam/pkg/iam/object|g' {} +
```

- [ ] **Step 5: 创建 router/router.go**

```go
package router

import (
	"crypto/rsa"

	"github.com/morehao/golib/biz/gserver/ginserver"
)

var OIDCPublicKey *rsa.PublicKey // platformadmin 不生成 OIDC 密钥，但 app.go 需要引用

func RegisterRouter(groups *ginserver.RouterGroups, appName string) {
	tenantRouter(groups)
	apiKeyRouter(groups)
	userRouter(groups)
	roleRouter(groups)
	menuRouter(groups)
	scopeRouter(groups)
	resourceRouter(groups)
	roleMenuRouter(groups)
	roleScopeRouter(groups)
	userRoleRouter(groups)
	oauthClientRouter(groups)
	applicationRouter(groups)
	connectorRouter(groups)
	departmentRouter(groups)
	organizationRouter(groups)
	systemRouter(groups)
	organizationRoleRouter(groups)
	organizationUserRouter(groups)
	organizationRoleUserRouter(groups)
	logRouter(groups)
	domainRouter(groups)
	tenantApplicationRouter(groups)
}
```

- [ ] **Step 6: 创建各路由文件，路径前缀改为 `/v1/platformadmin/{module}/{operation}`**

从 `apps/iam/internal/router/` 复制每个路由文件，修改路由路径前缀。

示例 - `user.go`：
```go
package router

import (
	"github.com/morehao/ark-iam/platformadmin/internal/controller/ctruser"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func userRouter(groups *ginserver.RouterGroups) {
	userCtr := ctruser.NewUserCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/platformadmin/user/create", userCtr.Create)
	v1RouterGroup.POST("/platformadmin/user/delete", userCtr.Delete)
	v1RouterGroup.POST("/platformadmin/user/update", userCtr.Update)
	v1RouterGroup.GET("/platformadmin/user/detail", userCtr.Detail)
	v1RouterGroup.POST("/platformadmin/user/pageList", userCtr.PageList)
	v1RouterGroup.POST("/platformadmin/user/updatePassword", userCtr.UpdatePassword)
	v1RouterGroup.POST("/platformadmin/user/updateStatus", userCtr.UpdateStatus)

	v1RouterGroup.POST("/platformadmin/user/createUserIdentity", userCtr.CreateUserIdentity)
	v1RouterGroup.POST("/platformadmin/user/deleteUserIdentity", userCtr.DeleteUserIdentity)
	v1RouterGroup.POST("/platformadmin/user/updateUserIdentity", userCtr.UpdateUserIdentity)
	v1RouterGroup.GET("/platformadmin/user/detailUserIdentity", userCtr.DetailUserIdentity)
	v1RouterGroup.POST("/platformadmin/user/pageListUserIdentity", userCtr.PageListUserIdentity)
	v1RouterGroup.GET("/platformadmin/user/getUserIdentityByUser", userCtr.GetUserIdentityByUser)

	v1RouterGroup.GET("/platformadmin/user/detailUserLoginLog", userCtr.DetailUserLoginLog)
	v1RouterGroup.POST("/platformadmin/user/pageListUserLoginLog", userCtr.PageListUserLoginLog)
	v1RouterGroup.GET("/platformadmin/user/getUserLoginLogByUser", userCtr.GetUserLoginLogByUser)

	v1RouterGroup.GET("/platformadmin/user/getUserDepartmentByUser", userCtr.GetUserDepartmentByUser)
	v1RouterGroup.POST("/platformadmin/user/assignDepartments", userCtr.AssignDepartments)
}
```

对所有路由文件做同样操作：在原路径前加 `/platformadmin`。

- [ ] **Step 7: go mod tidy 和编译验证**

```bash
cd backend/apps/platformadmin && go mod tidy
cd backend && go build ./apps/platformadmin/...
```

修复所有编译错误。

- [ ] **Step 8: 提交**

```bash
git add backend/apps/platformadmin/internal/ backend/apps/platformadmin/go.mod backend/apps/platformadmin/go.sum
git commit -m "feat: migrate platformadmin app business code"
```

---

### Task 6: 迁移 tenantadmin 应用业务代码

**Files:**
- Create: `backend/apps/tenantadmin/internal/service/svctenant/organization_role.go`
- Create: `backend/apps/tenantadmin/internal/service/svctenant/organization_user.go`
- Create: `backend/apps/tenantadmin/internal/service/svctenant/organization_role_user.go`
- Create: `backend/apps/tenantadmin/internal/controller/ctrtenant/` (organization* 相关控制器)
- Create: `backend/apps/tenantadmin/internal/dto/dtotenant/` (organization* 相关 DTO)
- Create: `backend/apps/tenantadmin/internal/router/router.go`
- Create: `backend/apps/tenantadmin/internal/router/organization.go`

- [ ] **Step 1: 探索 ctrtenant 中 organization 相关控制器方法**

先查看 current ctrtenant controller 的具体结构：
```bash
ls backend/apps/iam/internal/controller/ctrtenant/
grep "organization\|Organization" backend/apps/iam/internal/controller/ctrtenant/*.go -l
```

根据 AGENTS.md 的项目规范，ctrtenant 按领域划分，organization 相关 controller 可能在独立文件中。

- [ ] **Step 2: 复制 service 文件**

```bash
mkdir -p backend/apps/tenantadmin/internal/service/svctenant
cp backend/apps/iam/internal/service/svctenant/organization_role.go backend/apps/tenantadmin/internal/service/svctenant/
cp backend/apps/iam/internal/service/svctenant/organization_user.go backend/apps/tenantadmin/internal/service/svctenant/
cp backend/apps/iam/internal/service/svctenant/organization_role_user.go backend/apps/tenantadmin/internal/service/svctenant/
```

- [ ] **Step 3: 复制 controller 文件**

```bash
mkdir -p backend/apps/tenantadmin/internal/controller/ctrtenant
# 复制 organization 相关的 controller 文件
cp backend/apps/iam/internal/controller/ctrtenant/organization_role*.go backend/apps/tenantadmin/internal/controller/ctrtenant/ 2>/dev/null
cp backend/apps/iam/internal/controller/ctrtenant/organization_user*.go backend/apps/tenantadmin/internal/controller/ctrtenant/ 2>/dev/null
```

（具体文件名需要检查后确定）

- [ ] **Step 4: 复制 dto**

```bash
mkdir -p backend/apps/tenantadmin/internal/dto/dtotenant
cp backend/apps/iam/internal/dto/dtotenant/organization_*.go backend/apps/tenantadmin/internal/dto/dtotenant/
```

- [ ] **Step 5: 全局替换 import 路径**

```bash
find backend/apps/tenantadmin/internal -name "*.go" -exec sed -i '' 's|github.com/morehao/ark-iam/iam/|github.com/morehao/ark-iam/tenantadmin/|g' {} +
find backend/apps/tenantadmin/internal -name "*.go" -exec sed -i '' 's|github.com/morehao/ark-iam/tenantadmin/model|github.com/morehao/ark-iam/pkg/iam/model|g' {} +
find backend/apps/tenantadmin/internal -name "*.go" -exec sed -i '' 's|github.com/morehao/ark-iam/tenantadmin/dao|github.com/morehao/ark-iam/pkg/iam/dao|g' {} +
find backend/apps/tenantadmin/internal -name "*.go" -exec sed -i '' 's|github.com/morehao/ark-iam/tenantadmin/object|github.com/morehao/ark-iam/pkg/iam/object|g' {} +
```

- [ ] **Step 6: 创建 router 文件**

```go
// router.go
package router

import (
	"crypto/rsa"

	"github.com/morehao/golib/biz/gserver/ginserver"
)

var OIDCPublicKey *rsa.PublicKey

func RegisterRouter(groups *ginserver.RouterGroups, appName string) {
	organizationRoleRouter(groups)
	organizationUserRouter(groups)
	organizationRoleUserRouter(groups)
}
```

```go
// organization.go
package router

import (
	"github.com/morehao/ark-iam/tenantadmin/internal/controller/ctrtenant"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func organizationRoleRouter(groups *ginserver.RouterGroups) {
	organizationRoleCtr := ctrtenant.NewOrganizationRoleCtr()
	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/tenantadmin/organizationRole/create", organizationRoleCtr.Create)
	v1RouterGroup.POST("/tenantadmin/organizationRole/delete", organizationRoleCtr.Delete)
	v1RouterGroup.POST("/tenantadmin/organizationRole/update", organizationRoleCtr.Update)
	v1RouterGroup.GET("/tenantadmin/organizationRole/detail", organizationRoleCtr.Detail)
	v1RouterGroup.POST("/tenantadmin/organizationRole/pageList", organizationRoleCtr.PageList)
}

func organizationUserRouter(groups *ginserver.RouterGroups) {
	organizationUserCtr := ctrtenant.NewOrganizationUserCtr()
	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/tenantadmin/organizationUser/create", organizationUserCtr.Create)
	v1RouterGroup.POST("/tenantadmin/organizationUser/delete", organizationUserCtr.Delete)
	v1RouterGroup.POST("/tenantadmin/organizationUser/pageList", organizationUserCtr.PageList)
}

func organizationRoleUserRouter(groups *ginserver.RouterGroups) {
	organizationRoleUserCtr := ctrtenant.NewOrganizationRoleUserCtr()
	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/tenantadmin/organizationRoleUser/create", organizationRoleUserCtr.Create)
	v1RouterGroup.POST("/tenantadmin/organizationRoleUser/delete", organizationRoleUserCtr.Delete)
	v1RouterGroup.POST("/tenantadmin/organizationRoleUser/pageList", organizationRoleUserCtr.PageList)
}
```

- [ ] **Step 7: go mod tidy 和编译验证**

```bash
cd backend/apps/tenantadmin && go mod tidy
cd backend && go build ./apps/tenantadmin/...
```

- [ ] **Step 8: 提交**

```bash
git add backend/apps/tenantadmin/internal/ backend/apps/tenantadmin/go.mod backend/apps/tenantadmin/go.sum
git commit -m "feat: migrate tenantadmin app business code"
```

---

### Task 7: 从原 iam 应用删除已迁移代码，更新配置和测试

**Files:**
- Delete: `backend/apps/iam/model/`
- Delete: `backend/apps/iam/dao/`
- Delete: `backend/apps/iam/object/`
- Delete: `backend/apps/iam/internal/` (整个目录)
- Modify: `backend/apps/iam/go.mod`
- Modify: `backend/apps/iam/app.go`
- Modify: `backend/apps/iam/cmd/init.go`

- [ ] **Step 1: 更新原 iam 应用的 internal 代码内 model/dao 引用路径**

```bash
find backend/apps/iam/internal -name "*.go" -exec sed -i '' 's|github.com/morehao/ark-iam/iam/model|github.com/morehao/ark-iam/pkg/iam/model|g' {} +
find backend/apps/iam/internal -name "*.go" -exec sed -i '' 's|github.com/morehao/ark-iam/iam/dao|github.com/morehao/ark-iam/pkg/iam/dao|g' {} +
find backend/apps/iam/internal -name "*.go" -exec sed -i '' 's|github.com/morehao/ark-iam/iam/object|github.com/morehao/ark-iam/pkg/iam/object|g' {} +
```

- [ ] **Step 2: 删除已迁移的 model/dao/object 目录**

```bash
rm -rf backend/apps/iam/model backend/apps/iam/dao backend/apps/iam/object
```

- [ ] **Step 3: go mod tidy 原 iam 应用**

```bash
cd backend/apps/iam && go mod tidy
```

- [ ] **Step 4: 编译验证整个项目**

```bash
cd backend && go build ./...
```

- [ ] **Step 5: 运行测试**

```bash
cd backend && go test ./apps/iam/internal/... -v
cd backend && go test ./apps/auth/internal/... -v
cd backend && go test ./apps/platformadmin/internal/... -v
cd backend && go test ./apps/tenantadmin/internal/... -v
```

- [ ] **Step 6: 提交**

```bash
git add -A
git commit -m "refactor: cleanup original iam app, all tests pass"
```

---

### Task 8: 验证三个应用独立启动

- [ ] **Step 1: 分别启动三个应用验证端口不冲突**

```bash
make run APP=auth &
sleep 3 && curl -s http://localhost:8081/v1/auth/userinfo | head -5
kill %1

make run APP=platformadmin &
sleep 3 && curl -s http://localhost:8082/v1/platformadmin/tenant/pageList | head -5
kill %1

make run APP=tenantadmin &
sleep 3 && curl -s http://localhost:8083/v1/tenantadmin/organizationRole/pageList | head -5
kill %1
```

- [ ] **Step 2: 最终编译验证全项目**

```bash
cd backend && go build ./...
```

- [ ] **Step 3: 提交总结**

```bash
git commit -m "feat: complete IAM split into auth, platformadmin, tenantadmin apps" --allow-empty
```
