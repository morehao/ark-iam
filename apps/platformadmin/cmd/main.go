package main

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/apps/platformadmin"
	app "github.com/morehao/ark-iam/apps/platformadmin/app"
	"github.com/morehao/ark-iam/apps/platformadmin/config"
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
	// H1：默认不信任任何代理（直取 RemoteAddr），避免伪造 X-Forwarded-For 绕过按 IP 的限流/锁定。
	if err := engine.SetTrustedProxies(config.Conf.Server.TrustedProxies); err != nil {
		glog.Errorf(context.Background(), "%s set trusted proxies fail, err:%v", platformadmin.AppName, err)
		panic(err)
	}

	// 分层倒置：进程入口经 apps/platformadmin/app 装配（内部 pkg/appassembly + 域库），cmd 极薄。
	closer, err := app.Routers(engine)
	if err != nil {
		panic(fmt.Sprintf("[%s] Routers fail: %v", platformadmin.AppName, err))
	}
	defer func() {
		if cerr := closer.Close(); cerr != nil {
			glog.Errorf(context.Background(), "[%s] closer.Close fail, err:%v", platformadmin.AppName, cerr)
		}
	}()

	if err := engine.Run(fmt.Sprintf(":%s", config.Conf.Server.Port)); err != nil {
		glog.Errorf(context.Background(), "%s run fail, port:%s", platformadmin.AppName, config.Conf.Server.Port)
		panic(err)
	}
}
