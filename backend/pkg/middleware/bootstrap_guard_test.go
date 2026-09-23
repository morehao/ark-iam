package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/morehao/ark-iam/pkg/code"
)

// newGuardEngine 构造挂了守卫的最小引擎：/v1/business 为业务端点，其余为放行端点。
func newGuardEngine(probe BootstrapProbe, allowPrefixes ...string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(BootstrapGuard(probe, allowPrefixes...))
	engine.GET("/v1/business/ping", func(ctx *gin.Context) { ctx.String(http.StatusOK, "pong") })
	engine.GET("/install/status", func(ctx *gin.Context) { ctx.String(http.StatusOK, "install") })
	engine.GET("/oidc/healthz", func(ctx *gin.Context) { ctx.String(http.StatusOK, "healthy") })
	engine.GET("/oidc/.well-known/openid-configuration", func(ctx *gin.Context) { ctx.String(http.StatusOK, "discovery") })
	engine.GET("/oidc/authorize", func(ctx *gin.Context) { ctx.String(http.StatusOK, "authorize") })
	return engine
}

func doGet(engine *gin.Engine, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

// TestBootstrapGuard_BlocksBusinessWhenUninitialized 未初始化时业务端点必须被拦成 409 +
// InstallationPendingError，且响应体带上错误码（前端据此跳转 /install）。
func TestBootstrapGuard_BlocksBusinessWhenUninitialized(t *testing.T) {
	engine := newGuardEngine(func(context.Context) (bool, error) { return false, nil })
	rec := doGet(engine, "/v1/business/ping")
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body %q: %v", rec.Body.String(), err)
	}
	if got := intOf(body["code"]); got != code.InstallationPendingError {
		t.Errorf("响应 code = %v, want %d（body=%s）", body["code"], code.InstallationPendingError, rec.Body.String())
	}
}

// TestBootstrapGuard_AllowsInstallAndHealth 未初始化时 /install 与健康/发现端点必须放行。
//
// 反面：若健康检查也被拦成 409，编排层会把"等待初始化"误判成"服务起不来"而反复重启，
// 把一次正常部署变成重启风暴。
func TestBootstrapGuard_AllowsInstallAndHealth(t *testing.T) {
	engine := newGuardEngine(func(context.Context) (bool, error) { return false, nil })
	for _, path := range []string{
		"/install/status",
		"/oidc/healthz",
		"/oidc/.well-known/openid-configuration",
	} {
		if rec := doGet(engine, path); rec.Code != http.StatusOK {
			t.Errorf("%s status = %d, want 200（未初始化时也必须放行）", path, rec.Code)
		}
	}
	// 放行清单是"自举与健康"两类，业务性的 /oidc/authorize 不在其中：
	// 未初始化时没有租户与账号，授权请求无法工作，明确拦下比让它 401 更好定位。
	if rec := doGet(engine, "/oidc/authorize"); rec.Code != http.StatusConflict {
		t.Errorf("/oidc/authorize status = %d, want 409（不在放行清单内）", rec.Code)
	}
}

// TestBootstrapGuard_AllowsAfterInitialized 已初始化后业务端点放行。
func TestBootstrapGuard_AllowsAfterInitialized(t *testing.T) {
	engine := newGuardEngine(func(context.Context) (bool, error) { return true, nil })
	if rec := doGet(engine, "/v1/business/ping"); rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

// TestBootstrapGuard_ProbeFailClosed 探针报错时**不得放行**（503），
// 且不落闩锁——数据库抖动恢复后必须能继续正常工作。
func TestBootstrapGuard_ProbeFailClosed(t *testing.T) {
	var calls int32
	var failing atomic.Bool
	failing.Store(true)
	engine := newGuardEngine(func(context.Context) (bool, error) {
		atomic.AddInt32(&calls, 1)
		if failing.Load() {
			return false, errors.New("db down")
		}
		return true, nil
	})

	if rec := doGet(engine, "/v1/business/ping"); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("探针失败 status = %d, want 503（绝不放行）", rec.Code)
	}
	failing.Store(false)
	if rec := doGet(engine, "/v1/business/ping"); rec.Code != http.StatusOK {
		t.Fatalf("恢复后 status = %d, want 200（探测失败不得落闩锁）", rec.Code)
	}
	if n := atomic.LoadInt32(&calls); n != 2 {
		t.Errorf("探针调用次数 = %d, want 2（失败不缓存）", n)
	}
}

// TestBootstrapGuard_LatchesAfterInitialized 确认初始化后闩锁生效：
// 后续请求不再探测（生产上"已初始化"是绝大多数请求，守卫必须零 DB 开销）。
func TestBootstrapGuard_LatchesAfterInitialized(t *testing.T) {
	var calls int32
	engine := newGuardEngine(func(context.Context) (bool, error) {
		atomic.AddInt32(&calls, 1)
		return true, nil
	})
	for i := 0; i < 5; i++ {
		if rec := doGet(engine, "/v1/business/ping"); rec.Code != http.StatusOK {
			t.Fatalf("第 %d 次 status = %d, want 200", i+1, rec.Code)
		}
	}
	if n := atomic.LoadInt32(&calls); n != 1 {
		t.Errorf("探针调用次数 = %d, want 1（一旦确认初始化即永久放行，零额外查询）", n)
	}
	// 放行端点即使已初始化也不消耗探测（先判前缀）
	before := atomic.LoadInt32(&calls)
	if rec := doGet(engine, "/install/status"); rec.Code != http.StatusOK {
		t.Fatalf("install status = %d", rec.Code)
	}
	if n := atomic.LoadInt32(&calls); n != before {
		t.Errorf("放行端点不应触发探测，调用次数 %d -> %d", before, n)
	}
}

// TestBootstrapGuard_CustomAllowPrefixes 自定义放行清单生效（应用可按部署需要追加）。
func TestBootstrapGuard_CustomAllowPrefixes(t *testing.T) {
	engine := newGuardEngine(func(context.Context) (bool, error) { return false, nil }, "/v1/business")
	if rec := doGet(engine, "/v1/business/ping"); rec.Code != http.StatusOK {
		t.Errorf("自定义放行前缀未生效: status = %d, want 200", rec.Code)
	}
	if rec := doGet(engine, "/install/status"); rec.Code != http.StatusConflict {
		t.Errorf("自定义清单应替换默认清单: status = %d, want 409", rec.Code)
	}
}

// TestBootstrapProbeWithTimeout 探针超时必须返回超时错误（不无限挂住请求路径）。
func TestBootstrapProbeWithTimeout(t *testing.T) {
	slow := BootstrapProbeWithTimeout(func(ctx context.Context) (bool, error) {
		<-ctx.Done()
		return false, ctx.Err()
	}, 20*time.Millisecond)
	start := time.Now()
	if _, err := slow(context.Background()); !errors.Is(err, ErrBootstrapProbeTimeout) {
		t.Fatalf("err = %v, want ErrBootstrapProbeTimeout", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("超时未生效，耗时 %v", elapsed)
	}

	fast := BootstrapProbeWithTimeout(func(context.Context) (bool, error) { return true, nil }, time.Second)
	ok, err := fast(context.Background())
	if err != nil || !ok {
		t.Errorf("正常探针返回 (%v, %v), want (true, nil)", ok, err)
	}
}

// intOf 从 JSON 解出的数值取 int（json 解成 float64）。
func intOf(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	}
	return -1
}
