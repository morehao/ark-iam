package main

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/middleware"
	"github.com/morehao/ark-iam/pkg/seed"
	"github.com/morehao/ark-iam/rpapi"
	"github.com/morehao/ark-iam/rpapi/config"
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

	engine := gin.New()
	// 让 gin 上下文可当作 context.Context 传给任何下游（OIDC provider、glog、异步任务）：
	// 仅在开启该开关时 gin 才把 Done/Err/Deadline/Value 转发到请求的 context。
	engine.ContextWithFallback = true
	engine.Use(gin.Recovery())

	// 未初始化守卫（见 pkg/middleware 的 bootstrap_guard.go）：空库启动的进程没有任何账号，
	// 业务端点若照常放行只会返回一片 401/500，且无法区分"没初始化"与"服务坏了"。
	// 守卫把这个状态变成明确的 409 + 安装引导错误码，并把 /install 与健康/发现端点放行，
	// 让访问者被引导到初始化页面。已初始化后闩锁生效，此后零额外查询。
	guardPrefixes := middleware.BootstrapGuardAllowPrefixes
	if config.Conf.Server.Env == "dev" {
		// dev 的 swagger/redoc 由 gindocs 挂在 /<appName>/...，不属于业务路由；
		// 放行以免"想查接口文档却被告知先初始化"。
		guardPrefixes = append(append([]string{}, guardPrefixes...), "/rpapi")
	}
	engine.Use(middleware.BootstrapGuard(func(ctx context.Context) (bool, error) {
		return seed.IsInitialized(ctx, dbclient.IamDB(ctx))
	}, guardPrefixes...))
	// 独立部署没有同进程 OP：传 nil，由应用按自身配置解析 JWKS 端点。
	rpapi.Init(engine, config.Conf, nil)

	if err := engine.Run(fmt.Sprintf(":%s", config.Conf.Server.Port)); err != nil {
		glog.Errorf(context.Background(), "%s run fail, port:%s", rpapi.AppName, config.Conf.Server.Port)
		panic(err)
	}
}
