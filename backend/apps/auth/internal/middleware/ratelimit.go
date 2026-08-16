package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/config"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/ratelimit"
)

type rateLimitParams struct {
	rate   int
	burst  int
	period time.Duration
}

// defaultRateLimitParams 返回登录限流的默认参数。
func defaultRateLimitParams() rateLimitParams {
	return rateLimitParams{rate: 30, burst: 10, period: time.Minute}
}

// loginRateLimitParams 从配置读取限流参数，未配置或非法时回退默认值。
func loginRateLimitParams() rateLimitParams {
	p := defaultRateLimitParams()
	if config.Conf == nil {
		return p
	}
	if v := config.Conf.Security.Login.RatePerMinute; v > 0 {
		p.rate = v
	}
	if v := config.Conf.Security.Login.Burst; v > 0 {
		p.burst = v
	}
	return p
}

// LoginRateLimit 登录接口频率限流中间件（防暴力破解/CC）。
// 按客户端 IP 维度限流，key 形如 "oidc:login:ip:<ip>"；
// Redis 不可用时 fail-open 放行（与 svcloginguard 的降级策略一致）。
// 注意与 svcloginguard（失败累计锁定）互补：限流挡高频，锁定挡低频持续尝试。
func LoginRateLimit() gin.HandlerFunc {
	if dbclient.RedisCli == nil {
		return func(ctx *gin.Context) { ctx.Next() }
	}
	p := loginRateLimitParams()
	limiter, err := ratelimit.NewLimiter(
		ratelimit.WithRedisClient(dbclient.RedisCli),
		ratelimit.WithRate(p.rate),
		ratelimit.WithBurst(p.burst),
		ratelimit.WithPeriod(p.period),
	)
	if err != nil {
		// 限流器构造失败时 fail-open，避免破坏登录流程
		return func(ctx *gin.Context) { ctx.Next() }
	}
	return func(ctx *gin.Context) {
		key := "oidc:login:ip:" + gincontext.GetClientIP(ctx)
		if !limiter.Allow(ctx.Request.Context(), key) {
			gincontext.Fail(ctx, code.GetError(code.LoginRateLimitedError))
			ctx.Abort()
			return
		}
		ctx.Next()
	}
}
