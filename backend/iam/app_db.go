package iam

import (
	"context"
	"fmt"

	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/dbaccess/dbgorm"
	"github.com/morehao/golib/dbaccess/dbredis"
	"gorm.io/gorm"
)

// defaultDBService 是未指定 DB.Service 时的缺省服务名。
const defaultDBService = "iam"

// initDB 建立 IAM 的数据库连接（含租户隔离插件），并包装为 DBGetter。
func (a *App) initDB() error {
	service := a.cfg.DB.Service
	if service == "" {
		service = defaultDBService
	}
	db, err := dbclient.NewStore(dbgorm.Config{
		URL:     a.cfg.DB.DSN,
		Service: service,
	})
	if err != nil {
		return fmt.Errorf("iam: init db failed: %w", err)
	}
	a.storeRaw = db
	// 将实例连接同时注册为全局 iam 库：一进程一 IAM 实例下，仍通过 dbclient.IamDB 取连接的
	// 共享代码（pkg/sso 会话审计、pkg/core 默认路径等）与本实例共用同一份连接。
	dbclient.SetIamDB(db)
	a.dbGetter = func(ctx context.Context) *gorm.DB {
		return db.WithContext(ctx)
	}
	return nil
}

// initRedis 建立 Redis 连接；未配置则跳过。
func (a *App) initRedis() error {
	if a.cfg.Redis == nil || a.cfg.Redis.Addr == "" {
		return nil
	}
	cfg := dbredis.RedisConfig{
		Service:  a.cfg.Redis.Service,
		Addr:     a.cfg.Redis.Addr,
		Password: a.cfg.Redis.Password,
		DB:       a.cfg.Redis.DB,
	}
	client, err := dbredis.New(&cfg)
	if err != nil {
		return fmt.Errorf("iam: init redis failed: %w", err)
	}
	a.redis = client
	return nil
}

// Close 释放实例持有的连接资源（DB 底层连接、Redis）。
func (a *App) Close() error {
	var firstErr error
	if a.storeRaw != nil {
		if sqlDB, err := a.storeRaw.DB(); err == nil {
			if cerr := sqlDB.Close(); cerr != nil {
				firstErr = cerr
			}
		}
	}
	if a.redis != nil {
		if cerr := a.redis.Close(); cerr != nil && firstErr == nil {
			firstErr = cerr
		}
	}
	return firstErr
}
