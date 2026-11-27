package appassembly

import (
	"fmt"
	"time"

	"github.com/morehao/ark-iam/pkg/runtimesetting"
	"github.com/morehao/golib/gcache"
	_ "github.com/morehao/golib/gcache/driver/memory" // 兜底 driver（无 Redis 时）
	_ "github.com/morehao/golib/gcache/driver/redis"  // 生产 driver
	"github.com/morehao/golib/gsetting"
)

// settingCacheTTL 是运行时配置读缓存的 TTL。
//
// 配置读多写少、写路径会主动失效，故 TTL 只需覆盖「写后到过期前」读到旧值的最坏窗口；
// 1 分钟兼顾命中率与新鲜度（见 gsetting README 缓存语义）。
const settingCacheTTL = time.Minute

// initSetting 初始化运行时配置中心实例（golib gsetting），挂在 App 上供各消费者共享。
//
// 一进程一实例语义：页面写（platformadmin）与业务读（notify/security.login）在同一进程内
// 共用同一 *gsetting.Setting，避免 gsetting 包级 AdminAPI 被多次 New 覆盖（见设计 §10 R2）。
//
// 读路径缓存（gsetting.WithCache）：有 Redis 用 redis driver（跨实例共享、写后各实例按 TTL
// 自然过期），无 Redis（dev）退化 memory driver。缓存的是 DB 原值（加密项为密文），不落明文。
// 写路径（Create/Update/Delete）由 gsetting 自动按 (group,key) 失效，无需业务处理。
//
// core_setting 由 RunMigrations 统一建表。
func (a *App) initSetting() error {
	if a.storeRaw == nil {
		return fmt.Errorf("pkg/appassembly: init setting failed: db not initialized")
	}

	cache, err := a.buildSettingCache()
	if err != nil {
		return fmt.Errorf("pkg/appassembly: init setting cache failed: %w", err)
	}
	a.settingCache = cache

	var opts []gsetting.Option
	if cache != nil {
		opts = append(opts, gsetting.WithCache(cache, settingCacheTTL))
	}
	store, err := runtimesetting.New(a.storeRaw, opts...)
	if err != nil {
		return fmt.Errorf("pkg/appassembly: init setting failed: %w", err)
	}
	a.setting = store
	return nil
}

// buildSettingCache 依部署环境构造读缓存：有 Redis 用 redis driver，否则 memory driver。
func (a *App) buildSettingCache() (gcache.Cache, error) {
	if a.cfg.Redis != nil && a.cfg.Redis.Addr != "" {
		service := a.cfg.Redis.Service
		if service == "" {
			service = "iam"
		}
		return gcache.New(gcache.Config{
			Driver:     gcache.DriverRedis,
			DefaultTTL: settingCacheTTL,
			Redis: &gcache.RedisConfig{
				Service:  service,
				Addr:     a.cfg.Redis.Addr,
				Password: a.cfg.Redis.Password,
				DB:       a.cfg.Redis.DB,
			},
		})
	}
	// 无 Redis（dev / 单机）：进程内 memory 缓存。
	return gcache.New(gcache.Config{
		Driver:     gcache.DriverMemory,
		DefaultTTL: settingCacheTTL,
	})
}

// Setting 返回实例持有的运行时配置中心句柄；未初始化时为 nil。
func (a *App) Setting() *gsetting.Setting {
	if a.setting == nil {
		return nil
	}
	return a.setting.Setting
}
