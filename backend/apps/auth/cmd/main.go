package main

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth"
	"github.com/morehao/ark-iam/auth/config"
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
	engine.Use(gin.Recovery())
	// H1：默认不信任任何代理（直接用 RemoteAddr），避免客户端伪造
	// X-Forwarded-For 绕过按 IP 维度的登录限流/锁定；部署在反向代理后时
	// 在配置 server.trustedProxies 中显式声明可信代理 CIDR。
	if len(config.Conf.Server.TrustedProxies) > 0 {
		if err := engine.SetTrustedProxies(config.Conf.Server.TrustedProxies); err != nil {
			glog.Errorf(context.Background(), "%s set trusted proxies fail, err:%v", auth.AppName, err)
			panic(err)
		}
	} else {
		engine.SetTrustedProxies(nil)
	}
	auth.Init(engine, config.Conf)

	if err := engine.Run(fmt.Sprintf(":%s", config.Conf.Server.Port)); err != nil {
		glog.Errorf(context.Background(), "%s run fail, port:%s", auth.AppName, config.Conf.Server.Port)
		panic(err)
	}
}
