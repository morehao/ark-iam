// Package loginsetting 是「登录安全」运行时配置（gsetting 分组 security.login）的读写封装：
// 登录失败锁定阈值 + 登录接口频率限流参数。
//
// 消费者：apps/auth 的 svcloginguard（锁定，现读）与 middleware/ratelimit（限流，懒重建）；
// 播种：sdk/server 首启从 yaml security 段写入一次（分组为空时）。
package loginsetting

import (
	"context"
	"fmt"
	"sync"

	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/gsetting"
)

// SettingGroup 是登录安全在 gsetting 中的配置分组名。
const SettingGroup = "security.login"

// 字段键（禁止散落裸字符串）。
const (
	KeyMaxFailures   = "maxFailures"
	KeyWindowSec     = "windowSec"
	KeyLockSec       = "lockSec"
	KeyRatePerMinute = "ratePerMinute"
	KeyBurst         = "burst"
)

// 默认值（未配置/读取失败时的安全回退，与既有默认一致）。
const (
	DefaultMaxFailures   = 5
	DefaultWindowSec     = 300
	DefaultLockSec       = 900
	DefaultRatePerMinute = 30
	DefaultBurst         = 10
)

// Config 是登录安全的运行时参数。
type Config struct {
	MaxFailures   int // 失败次数阈值
	WindowSec     int // 失败计数窗口（秒）
	LockSec       int // 锁定时长（秒）
	RatePerMinute int // 限流：每分钟请求数
	Burst         int // 限流：突发容量
}

// WithDefaults 用默认值填充零值字段（< =0 视为未设置）。
func (c Config) WithDefaults() Config {
	if c.MaxFailures <= 0 {
		c.MaxFailures = DefaultMaxFailures
	}
	if c.WindowSec <= 0 {
		c.WindowSec = DefaultWindowSec
	}
	if c.LockSec <= 0 {
		c.LockSec = DefaultLockSec
	}
	if c.RatePerMinute <= 0 {
		c.RatePerMinute = DefaultRatePerMinute
	}
	if c.Burst <= 0 {
		c.Burst = DefaultBurst
	}
	return c
}

// instanceCache 按 *gorm.DB 缓存 gsetting 实例，避免重复 Migrate。
var (
	instanceMu    sync.Mutex
	instanceCache = map[any]*gsetting.Setting{}
)

func settingFor(db gormdao.DBGetter) (*gsetting.Setting, error) {
	raw := db(context.Background())
	instanceMu.Lock()
	defer instanceMu.Unlock()
	if s, ok := instanceCache[raw]; ok {
		return s, nil
	}
	s, err := gsetting.New(raw)
	if err != nil {
		return nil, err
	}
	instanceCache[raw] = s
	return s, nil
}

// Load 读取 security.login 分组；字段缺失/读取错误按默认值填充（不返回错误，安全回退）。
func Load(ctx context.Context, db gormdao.DBGetter) Config {
	s, err := settingFor(db)
	if err != nil {
		return Config{}.WithDefaults()
	}
	return loadFrom(ctx, s).WithDefaults()
}

func loadFrom(ctx context.Context, s *gsetting.Setting) Config {
	read := func(key string) int {
		v, err := s.GetInt64(ctx, SettingGroup, key)
		if err != nil {
			return 0 // 行不存在/解析失败 -> 0（WithDefaults 会补默认）
		}
		return int(v)
	}
	return Config{
		MaxFailures:   read(KeyMaxFailures),
		WindowSec:     read(KeyWindowSec),
		LockSec:       read(KeyLockSec),
		RatePerMinute: read(KeyRatePerMinute),
		Burst:         read(KeyBurst),
	}
}

// Save 将 config 逐字段写回（upsert）。
func Save(ctx context.Context, db gormdao.DBGetter, c Config) error {
	s, err := settingFor(db)
	if err != nil {
		return fmt.Errorf("loginsetting: init gsetting: %w", err)
	}
	return saveTo(ctx, s, c)
}

func saveTo(ctx context.Context, s *gsetting.Setting, c Config) error {
	admin := gsetting.GetAdmin()
	entries := []struct {
		key  string
		val  int
		desc string
	}{
		{KeyMaxFailures, c.MaxFailures, "失败次数阈值：窗口内连续失败达到该次数即锁定"},
		{KeyWindowSec, c.WindowSec, "失败计数窗口（秒）"},
		{KeyLockSec, c.LockSec, "锁定时长（秒）"},
		{KeyRatePerMinute, c.RatePerMinute, "登录限流：每分钟允许的最大请求数（按 IP）"},
		{KeyBurst, c.Burst, "登录限流：突发容量（令牌桶最大容量）"},
	}
	for _, e := range entries {
		// gsetting 不接受空值；用 WithDefaults 后各项均 >0，正常不会为 0。
		if e.val <= 0 {
			continue
		}
		if err := admin.Create(ctx, &gsetting.CreateReq{
			Group:       SettingGroup,
			Key:         e.key,
			ValueType:   string(gsetting.ValueTypeInt),
			Value:       fmt.Sprintf("%d", e.val),
			Description: e.desc,
		}); err != nil {
			return fmt.Errorf("loginsetting: save %s: %w", e.key, err)
		}
	}
	return nil
}

// IsEmpty 判断分组是否无任何行（用于首启播种判据）。
func IsEmpty(ctx context.Context, db gormdao.DBGetter) (bool, error) {
	s, err := settingFor(db)
	if err != nil {
		return false, err
	}
	v, err := s.GetInt64(ctx, SettingGroup, KeyMaxFailures)
	if err != nil {
		// 取不到（行不存在）视为空分组。
		return true, nil
	}
	_ = v
	return false, nil
}
