package main

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/platformadmin"
	"github.com/morehao/ark-iam/platformadmin/config"
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
	platformadmin.Init(engine, config.Conf)

	if err := engine.Run(fmt.Sprintf(":%s", config.Conf.Server.Port)); err != nil {
		glog.Errorf(context.Background(), "%s run fail, port:%s", platformadmin.AppName, config.Conf.Server.Port)
		panic(err)
	}
}
