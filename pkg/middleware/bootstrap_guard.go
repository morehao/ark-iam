package middleware

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/morehao/ark-iam/pkg/code"

	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/glog"
)

// BootstrapGuardAllowPrefixes 守卫放行清单：这些路径在**未初始化**时也必须可用。
//
// 为什么是前缀（而不是逐条枚举）：放行的是"自举与健康"两类——
//   - /install：唯一的写入口与状态查询，未初始化时前端只能靠它工作；
//   - /oidc 下的健康检查/发现/登出回调：部署脚本与探针靠它们判断进程是否就绪，
//     若被守卫拦成 409，编排层会把"等待初始化"误判成"服务起不来"而反复重启。
//
// 业务路由一律不放行：未初始化时它们本就无法工作（没有租户、没有账号），
// 放行只会让失败模式从"明确的未初始化"退化为"各种找不到数据"。
var BootstrapGuardAllowPrefixes = []string{
	"/install",
	"/oidc/healthz",
	"/oidc/ready",
	"/oidc/.well-known",
	"/oidc/keys",
	"/oidc/logged-out",
}

// BootstrapProbe 报告系统是否已完成首次初始化。返回 error 表示**无法判定**。
type BootstrapProbe func(ctx context.Context) (bool, error)

// BootstrapGuard 在系统尚未完成首次初始化时拦截业务端点（返回 409 + InstallationPendingError）。
//
// 存在意义：切断"启动期播种"之后，空库启动的进程不再有任何可用账号，若不放行任何提示，
// 访问者只会看到一片 401/500（且无法区分"没初始化"与"服务坏了"）。守卫把这个状态变成
// 一个**明确、可执行**的响应：请先访问 /install。
//
// 两个关键实现选择：
//
//  1. **一次性闩锁**：一旦探针报告已初始化，结果永久缓存到进程结束，之后每个请求零开销。
//     "已初始化"是单向状态（没有反向操作），因此缓存不会失效；这也让守卫在生产上
//     完全不产生额外 DB 查询——只有部署首启、尚未初始化时才会反复探测。
//  2. **探测失败按未初始化处理**（fail-closed 但保守）：探针报错时返回 503 而非放行，
//     并附上系统错误语义；绝不能因为"查不出来"就放行业务请求。
type bootstrapGuard struct {
	probe BootstrapProbe
	// allowPrefixes 放行前缀（默认 BootstrapGuardAllowPrefixes）。
	allowPrefixes []string
	// settled 为 true 表示已确认初始化，此后的请求直接放行（读锁保护，写只发生一次）。
	settled bool
	mu      sync.RWMutex
}

// BootstrapGuard 构造未初始化守卫（见 bootstrapGuard 注释）。allowPrefixes 为空时用内置清单。
//
// 用法（应用装配时挂到 engine 上，早于业务路由与鉴权中间件）：
//
//	engine.Use(middleware.BootstrapGuard(func(ctx context.Context) (bool, error) {
//		return seed.IsInitialized(ctx, dbclient.IamDB(ctx))
//	}))
func BootstrapGuard(probe BootstrapProbe, allowPrefixes ...string) gin.HandlerFunc {
	if len(allowPrefixes) == 0 {
		allowPrefixes = BootstrapGuardAllowPrefixes
	}
	g := &bootstrapGuard{probe: probe, allowPrefixes: allowPrefixes}
	return g.handle
}

func (g *bootstrapGuard) handle(ctx *gin.Context) {
	if isSkippedPath(ctx.Request.URL.Path, g.allowPrefixes) {
		ctx.Next()
		return
	}

	initialized, err := g.initialized(ctx)
	if err != nil {
		// 无法判定 ≠ 可以放行：给 503 明确表达"服务暂时不能处理业务请求"。
		glog.Errorf(ctx, "[bootstrapguard] probe fail, err:%v", err)
		gincontext.FailWithStatus(ctx, 503, code.GetError(code.InstallationInitializeError))
		ctx.Abort()
		return
	}
	if !initialized {
		// 409：请求本身没问题，是系统状态不允许（与"已初始化后重复初始化"同一语义）。
		gincontext.FailWithStatus(ctx, 409, code.GetError(code.InstallationPendingError))
		ctx.Abort()
		return
	}
	ctx.Next()
}

// initialized 返回系统是否已初始化；已确认后直接命中闩锁（零 DB 开销）。
func (g *bootstrapGuard) initialized(ctx *gin.Context) (bool, error) {
	g.mu.RLock()
	settled := g.settled
	g.mu.RUnlock()
	if settled {
		return true, nil
	}

	ok, err := g.probe(ctx)
	if err != nil {
		// 探测失败不落闩锁：下一次请求继续探测（可能只是 DB 抖动）。
		return false, err
	}
	if ok {
		g.mu.Lock()
		g.settled = true
		g.mu.Unlock()
	}
	return ok, nil
}

// ErrBootstrapProbeTimeout 探针超时的语义错误（供调用方包装自身错误时比对）。
var ErrBootstrapProbeTimeout = errors.New("bootstrap probe timeout")

// BootstrapProbeWithTimeout 给探针加超时：守卫在请求路径上，探针是 DB 查询，
// 不能让"数据库慢"直接演变成"所有请求都挂住"。
func BootstrapProbeWithTimeout(probe BootstrapProbe, timeout time.Duration) BootstrapProbe {
	return func(ctx context.Context) (bool, error) {
		probeCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		done := make(chan struct{})
		var (
			ok  bool
			err error
		)
		go func() {
			defer close(done)
			ok, err = probe(probeCtx)
		}()
		select {
		case <-done:
			return ok, err
		case <-probeCtx.Done():
			return false, ErrBootstrapProbeTimeout
		}
	}
}
