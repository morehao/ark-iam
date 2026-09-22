package main

import (
	"context"
	"fmt"
	"time"

	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/rpapi"
	"github.com/morehao/ark-iam/rpapi/config"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/glog"
	_ "github.com/morehao/golib/glog/driver/zap"
	"github.com/morehao/golib/gtrace/otel"
	"github.com/morehao/golib/gtrace/otel/otlptracegrpc"
)

var traceProvider *otel.Provider

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
	// 启动期不隶属任何租户：显式声明「全部租户」作用域。
	// fail-closed 下不允许用缺失作用域来获得跨租户可见性（会直接报错），
	// 因此 AutoMigrate 必须在此显式声明。
	//
	// 启动期**只建表、不写数据**：内置数据由初始化页面触发的一次性引导写入
	// （pkg/seed.Bootstrap，见 docs/design/system-design.md §4.5）。
	bootstrapCtx := gcontext.WithTenantScope(context.Background(), gcontext.AllScope())
	if config.Conf.DB.AutoMigrate {
		if err := model.AutoMigrateAll(dbclient.IamDB(bootstrapCtx)); err != nil {
			return fmt.Errorf("auto migrate failed: %w", err)
		}
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
	provider, err := otlptracegrpc.NewGRPCProvider(context.Background(), rpapi.AppName, config.Conf.Server.Env, config.Conf.Trace)
	if err != nil {
		glog.Errorf(context.Background(), "[%s.initTrace] init trace failed, fallback to disabled mode, err:%v", rpapi.AppName, err)
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
		glog.Errorf(context.Background(), "[%s.shutdownTraceProvider] shutdown fail, err:%v", rpapi.AppName, err)
	}
}
