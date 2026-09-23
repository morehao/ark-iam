package main

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth"
	"github.com/morehao/ark-iam/auth/config"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/middleware"
	"github.com/morehao/ark-iam/pkg/seed"
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
		guardPrefixes = append(append([]string{}, guardPrefixes...), "/auth")
	}
	engine.Use(middleware.BootstrapGuard(func(ctx context.Context) (bool, error) {
		return seed.IsInitialized(ctx, dbclient.IamDB(ctx))
	}, guardPrefixes...))
	// H1：默认不信任任何代理（直接用 RemoteAddr），避免客户端伪造
	// X-Forwarded-For 绕过按 IP 维度的登录限流/锁定；部署在反向代理后时
	// 在配置 server.trustedProxies 中显式声明可信代理 CIDR。
	if len(config.Conf.Server.TrustedProxies) > 0 {
		if err := engine.SetTrustedProxies(config.Conf.Server.TrustedProxies); err != nil {
			glog.Errorf(context.Background(), "%s set trusted proxies fail, err:%v", auth.AppName, err)
			panic(err)
		}
	} else {
		// nil = 不信任任何代理，客户端 IP 直取 RemoteAddr（Gin 对空列表不返回错误，此处仍与上一分支统一处理）
		if err := engine.SetTrustedProxies(nil); err != nil {
			glog.Errorf(context.Background(), "%s set trusted proxies fail, err:%v", auth.AppName, err)
			panic(err)
		}
	}
	auth.Init(engine, config.Conf)

	if err := engine.Run(fmt.Sprintf(":%s", config.Conf.Server.Port)); err != nil {
		glog.Errorf(context.Background(), "%s run fail, port:%s", auth.AppName, config.Conf.Server.Port)
		panic(err)
	}
}
