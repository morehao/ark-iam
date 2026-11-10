package main

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/gateway/config"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/middleware"
	"github.com/morehao/ark-iam/pkg/seed"
	"github.com/morehao/ark-iam/rpapi"
	"github.com/morehao/ark-iam/sdk/server"
	"github.com/morehao/golib/glog"
)

func main() {
	if err := serverInit(); err != nil {
		panic(fmt.Sprintf("server init failed, error: %v", err))
	}
	if config.Conf.Server.Env == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}
	defer shutdownTraceProvider()
	defer func() {
		if err := glog.Close(); err != nil {
			fmt.Printf("failed to close logger: %v\n", err)
		}
	}()

	// 分层倒置：gateway 是纯 main（不产库、绝不被 import）。
	// 经 sdk/server 门面全量装配（Domains=nil → 全量域），行为与重构前等价：
	// 同一套路由布局、同一套鉴权中间件（OIDCAuth + RequireAppAccess）、同一套 back-channel logout。
	app, err := server.New(server.FromPkgConfig(config.Conf, nil /* 全量 */, "", ""))
	if err != nil {
		panic(fmt.Sprintf("[%s] sdk/server.New fail: %v", appName, err))
	}
	defer app.Close()
	if config.Conf.DB.AutoMigrate {
		if err := app.RunMigrations(context.Background()); err != nil {
			panic(fmt.Sprintf("[%s] RunMigrations fail: %v", appName, err))
		}
	}

	engine := gin.New()
	// 刻意**不开启** engine.ContextWithFallback：service/dao 只收 context.Context，
	// controller 用 ctx.Request.Context() 做唯一一次转换（见
	// docs/design/context-and-tenant-scope.md「边界调整」）。
	engine.Use(gin.Recovery())
	engine.Use(middleware.WithRequestFields())

	// 未初始化守卫：空库启动的业务端点变 409 + 安装引导错误码，放行 /install 与健康/发现。
	guardPrefixes := middleware.BootstrapGuardAllowPrefixes
	if config.Conf.Server.Env == "dev" {
		// dev 的 swagger/redoc 由 gindocs 挂在 /<appName>/...，不属于业务路由。
		guardPrefixes = append(append([]string{}, guardPrefixes...), "/auth", "/platformadmin", "/tenantadmin", "/rpapi")
	}
	engine.Use(middleware.BootstrapGuard(func(ctx context.Context) (bool, error) {
		return seed.IsInitialized(ctx, dbclient.IamDB(ctx))
	}, guardPrefixes...))

	if err := app.Mount(engine.Group("")); err != nil {
		panic(fmt.Sprintf("[%s] Mount fail: %v", appName, err))
	}
	if err := app.MountConfig(engine, ""); err != nil {
		panic(fmt.Sprintf("[%s] MountConfig fail: %v", appName, err))
	}

	// rpapi 不在 sdk/server 门面内（独立只读目录 API，用 sdk/rp 验签），单独挂载。
	// 复用 auth 域已产出的进程内 KeySource（app.KeySource()），rpapi 本地验签、零网络调用，
	// 与重构前 gateway 注入 auth keySource 的行为一致。
	rpapi.Init(engine, config.Conf, app.KeySource())

	if err := engine.Run(fmt.Sprintf(":%s", config.Conf.Server.Port)); err != nil {
		glog.Errorf(context.Background(), "%s run fail, port:%s", appName, config.Conf.Server.Port)
		panic(err)
	}
}
