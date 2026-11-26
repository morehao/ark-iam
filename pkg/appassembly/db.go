package appassembly

import (
	"context"
	"fmt"

	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/runtimesetting"
	"github.com/morehao/golib/dbaccess/dbgorm"
	"github.com/morehao/golib/dbaccess/dbredis"
	"gorm.io/gorm"
)

// defaultDBService 是未指定 DB.Service 时的缺省服务名。
const defaultDBService = "iam"

// initDB 建立门面的数据库连接（含租户隔离插件），并包装为 DBGetter。
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
		return fmt.Errorf("pkg/appassembly: init db failed: %w", err)
	}
	a.storeRaw = db
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
	service := a.cfg.Redis.Service
	if service == "" {
		service = defaultDBService // "iam"
	}
	cfg := dbredis.RedisConfig{
		Service:  service,
		Addr:     a.cfg.Redis.Addr,
		Password: a.cfg.Redis.Password,
		DB:       a.cfg.Redis.DB,
	}
	client, err := dbredis.New(&cfg)
	if err != nil {
		return fmt.Errorf("pkg/appassembly: init redis failed: %w", err)
	}
	a.redis = client
	return nil
}

// Close 释放实例持有的连接资源（DB 底层连接、Redis）。
func (a *App) Close() error {
	a.closeOnce.Do(func() {
		if !a.ownsProcess {
			return
		}
		if a.stopLogoutWorker != nil {
			a.stopLogoutWorker()
		}
		if a.closeKeys != nil {
			a.closeKeys()
		}
		if a.setting != nil {
			runtimesetting.SetProcessStore(nil)
			dbclient.SetIamDB(nil)
			dbclient.SetRedis(nil)
		}
		a.closeErr = a.closeResources()
		appActive.Store(false)
	})
	return a.closeErr
}

func (a *App) closeResources() error {
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
	if a.settingCache != nil {
		if cerr := a.settingCache.Close(); cerr != nil && firstErr == nil {
			firstErr = cerr
		}
	}
	return firstErr
}
