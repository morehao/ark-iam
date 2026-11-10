package main

import (
	"context"
	"fmt"
	"time"

	"github.com/morehao/ark-iam/auth"
	"github.com/morehao/ark-iam/auth/config"
	"github.com/morehao/golib/glog"
	_ "github.com/morehao/golib/glog/driver/zap"
	"github.com/morehao/golib/gtrace/otel"
	"github.com/morehao/golib/gtrace/otel/otlptracegrpc"
)

var traceProvider *otel.Provider

// serverInit 只做「与 DB/Redis 无关」的进程前置初始化：日志 + 链路追踪。
// DB/Redis/建表统一交给 sdk/server.New + RunMigrations（分层倒置后由门面编排）。
func serverInit() error {
	config.InitConf()
	if err := preInit(); err != nil {
		return err
	}
	if err := initTrace(); err != nil {
		return err
	}
	return nil
}

func preInit() error {
	defaultLogCfg := config.Conf.Log["default"]
	if err := glog.InitLogger(&defaultLogCfg); err != nil {
		return fmt.Errorf("init logger failed: %w", err)
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
