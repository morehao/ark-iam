package middleware

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/loginsetting"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/ratelimit"
)

// limiterEntry 是「一组参数 + 对应 limiter」的缓存条目，用配置指纹判断是否需重建。
type limiterEntry struct {
	rate   int
	burst  int
	period time.Duration
	lim    ratelimit.Limiter
}

// limiterSlot 是按 key 前缀区分的限流器缓存槽（B1 懒重建 + 原子换）。
//
// 理由（见设计 §5.6b）：ratelimit.Limiter 有后台 goroutine，必须 Close() 释放，
// 不能每请求重建；参数变更极低频，故每次请求比对配置指纹，不一致才 Close 旧的 + 重建。
type limiterSlot struct {
	ptr atomic.Pointer[limiterEntry]
	mu  sync.Mutex // 串行化重建，避免并发重复 Close/重建
}

var (
	slotMu sync.Mutex
	slots  = map[string]*limiterSlot{}
)

func slotFor(keyPrefix string) *limiterSlot {
	slotMu.Lock()
	defer slotMu.Unlock()
	if s, ok := slots[keyPrefix]; ok {
		return s
	}
	s := &limiterSlot{}
	slots[keyPrefix] = s
	return s
}

// currentRateParams 现读登录安全配置中的限流参数（rate/burst/period）。
// 读失败按默认值（loginsetting.Load 内已 WithDefaults）。
func currentRateParams() (rate, burst int, period time.Duration) {
	c := loginsetting.Load(context.Background(), dbclient.IamDB)
	return c.RatePerMinute, c.Burst, time.Minute
}

// limiterFor 返回 keyPrefix 对应的可用 limiter：指纹一致复用，不一致重建（B1）。
func limiterFor(keyPrefix string) ratelimit.Limiter {
	rate, burst, period := currentRateParams()

	slot := slotFor(keyPrefix)
	cur := slot.ptr.Load()
	if cur != nil && cur.rate == rate && cur.burst == burst && cur.period == period {
		return cur.lim
	}

	slot.mu.Lock()
	defer slot.mu.Unlock()
	// double-check：拿锁后可能已被其它并发请求重建。
	cur = slot.ptr.Load()
	if cur != nil && cur.rate == rate && cur.burst == burst && cur.period == period {
		return cur.lim
	}
	newLim, err := ratelimit.NewLimiter(
		ratelimit.WithRedisClient(dbclient.RedisCli),
		ratelimit.WithRate(rate),
		ratelimit.WithBurst(burst),
		ratelimit.WithPeriod(period),
	)
	if err != nil {
		// 构造失败 fail-open：直接放行（返回 nil，调用方判定）。
		return nil
	}
	slot.ptr.Store(&limiterEntry{rate: rate, burst: burst, period: period, lim: newLim})
	if cur != nil && cur.lim != nil {
		cur.lim.Close() // 释放旧实例的后台 goroutine
	}
	return newLim
}

// LoginRateLimit 登录接口频率限流中间件（防暴力破解/CC）。
// 按客户端 IP 维度限流，key 形如 "oidc:login:ip:<ip>"；
// Redis 不可用时 fail-open 放行（与 svcloginguard 的降级策略一致）。
func LoginRateLimit() gin.HandlerFunc {
	return NewLoginRateLimit("oidc:login:ip:")
}

// PasswordChangeRateLimit 修改密码接口频率限流（按 IP），防止对旧密码的暴力试探。
func PasswordChangeRateLimit() gin.HandlerFunc {
	return NewLoginRateLimit("password:change:ip:")
}

// BootstrapRateLimit 初始化接口频率限流（按 IP）。
//
// 需要它的理由：/install/initialize 是**未认证写面**，且令牌是静态共享密钥——
// 无限流时它可以被离线暴力猜测，也会成为一条无需鉴权的写入压力入口。
func BootstrapRateLimit() gin.HandlerFunc {
	return NewLoginRateLimit("install:initialize:ip:")
}

// NewLoginRateLimit 按指定 key 前缀构造登录/敏感操作的 IP 频率限流中间件。
//
// 限流参数在请求时现读（B1 懒重建），保存配置后无需重启即可生效（下一请求自愈）。
func NewLoginRateLimit(keyPrefix string) gin.HandlerFunc {
	if dbclient.RedisCli == nil {
		return func(ctx *gin.Context) { ctx.Next() }
	}
	return func(ctx *gin.Context) {
		limiter := limiterFor(keyPrefix)
		if limiter == nil {
			// 构造失败：fail-open。
			ctx.Next()
			return
		}
		key := keyPrefix + gincontext.GetClientIP(ctx)
		if !limiter.Allow(ctx, key) {
			gincontext.Fail(ctx, code.GetError(code.LoginRateLimitedError))
			ctx.Abort()
			return
		}
		ctx.Next()
	}
}
