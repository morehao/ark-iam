package main

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/apps/platformadmin"
	"github.com/morehao/ark-iam/apps/platformadmin/config"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/middleware"
	"github.com/morehao/ark-iam/pkg/seed"
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

	// 分层倒置：进程入口经 sdk/server 门面装配 platformadmin 域库。
	// 独立部署无同进程 OP：platform 域鉴权中间件的 KeySource 由 server 按
	// Config.Issuer（= auth 的 issuer，见 config.yaml oidc.issuer）远程拉 JWKS 解析。
	app, err := server.New(server.FromPkgConfig(config.Conf, platformDomains(), "", config.Conf.OIDC.Issuer))
	if err != nil {
		panic(fmt.Sprintf("[%s] sdk/server.New fail: %v", platformadmin.AppName, err))
	}
	defer app.Close()
	if config.Conf.DB.AutoMigrate {
		if err := app.RunMigrations(context.Background()); err != nil {
			panic(fmt.Sprintf("[%s] RunMigrations fail: %v", platformadmin.AppName, err))
		}
	}

	engine := gin.New()
	// 刻意**不开启** engine.ContextWithFallback：service/dao 只收 context.Context，
	// controller 用 ctx.Request.Context() 做唯一一次转换。
	engine.Use(gin.Recovery())
	engine.Use(middleware.WithRequestFields())

	guardPrefixes := middleware.BootstrapGuardAllowPrefixes
	if config.Conf.Server.Env == "dev" {
		guardPrefixes = append(append([]string{}, guardPrefixes...), "/platformadmin")
	}
	engine.Use(middleware.BootstrapGuard(func(ctx context.Context) (bool, error) {
		return seed.IsInitialized(ctx, dbclient.IamDB(ctx))
	}, guardPrefixes...))

	if err := app.Mount(engine.Group("")); err != nil {
		panic(fmt.Sprintf("[%s] Mount fail: %v", platformadmin.AppName, err))
	}
	if err := app.MountConfig(engine, ""); err != nil {
		panic(fmt.Sprintf("[%s] MountConfig fail: %v", platformadmin.AppName, err))
	}

	if err := engine.Run(fmt.Sprintf(":%s", config.Conf.Server.Port)); err != nil {
		glog.Errorf(context.Background(), "%s run fail, port:%s", platformadmin.AppName, config.Conf.Server.Port)
		panic(err)
	}
}

// platformDomains 返回 platformadmin 独立进程的域集（menu/tenant/application/oauth-client/
// domain/tenant-app/audit；menu 为 role 依赖在 tenant 进程侧，这里按平台控制台页挂全）。
func platformDomains() []server.Domain {
	return []server.Domain{
		server.DomainMenu,
		server.DomainTenant,
		server.DomainApplication,
		server.DomainOAuthClient,
		server.DomainDomain,
		server.DomainTenantApp,
		server.DomainAudit,
	}
}
