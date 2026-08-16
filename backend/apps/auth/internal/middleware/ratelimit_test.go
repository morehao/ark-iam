package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/config"
	"github.com/morehao/ark-iam/pkg/code"
	pkgconfig "github.com/morehao/ark-iam/pkg/config"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/testsetup"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/redis/go-redis/v9"
)

func TestLoginRateLimitWithoutRedisFailsOpen(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)

	gin.SetMode(gin.TestMode)
	config.Conf = &pkgconfig.Config{}
	// 显式模拟无 Redis（dbclient.RedisCli == nil），确保走 fail-open 分支
	dbclient.RedisCli = nil

	engine := gin.New()
	engine.POST("/oidc/login", LoginRateLimit(), func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"code": 0})
	})

	// 无 Redis 时中间件应放行，不 panic
	for i := 0; i < 100; i++ {
		req := httptest.NewRequest(http.MethodPost, "/oidc/login", nil)
		resp := httptest.NewRecorder()
		engine.ServeHTTP(resp, req)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected fail-open pass, got status %d", resp.Code)
		}
	}
}

func TestLoginRateLimitParamsDefaultsAndConfig(t *testing.T) {
	// 未配置限流参数时使用默认值
	config.Conf = &pkgconfig.Config{}
	if got := loginRateLimitParams(); got.rate != 30 || got.burst != 10 {
		t.Fatalf("expected default rate=30 burst=10, got %+v", got)
	}
	// 配置后生效
	config.Conf = &pkgconfig.Config{Security: pkgconfig.SecurityConfig{
		Login: pkgconfig.LoginGuardConfig{RatePerMinute: 60, Burst: 20},
	}}
	if got := loginRateLimitParams(); got.rate != 60 || got.burst != 20 {
		t.Fatalf("expected configured rate=60 burst=20, got %+v", got)
	}
	// config 为 nil 时仍回退默认值
	config.Conf = nil
	if got := loginRateLimitParams(); got.rate != 30 || got.burst != 10 {
		t.Fatalf("expected nil-config fallback rate=30 burst=10, got %+v", got)
	}
}

func TestLoginRateLimitBlocksExcessWithRedis(t *testing.T) {
	// miniredis 提供内存 Redis，验证限流真实生效（rate=2/min burst=2，第 3 次请求被拦）
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis start fail: %v", err)
	}
	defer mr.Close()

	oldCli := dbclient.RedisCli
	dbclient.RedisCli = redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { dbclient.RedisCli = oldCli }()

	gin.SetMode(gin.TestMode)
	config.Conf = &pkgconfig.Config{Security: pkgconfig.SecurityConfig{
		Login: pkgconfig.LoginGuardConfig{RatePerMinute: 2, Burst: 2},
	}}

	engine := gin.New()
	engine.POST("/oidc/login", LoginRateLimit(), func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"code": 0})
	})

	type respBody struct {
		Code int `json:"code"`
	}
	codes := make([]int, 0, 3)
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodPost, "/oidc/login", nil)
		resp := httptest.NewRecorder()
		engine.ServeHTTP(resp, req)
		var body respBody
		if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		codes = append(codes, body.Code)
	}

	if codes[0] != 0 || codes[1] != 0 {
		t.Fatalf("expected first two requests to pass, got %v", codes)
	}
	if codes[2] != code.LoginRateLimitedError {
		t.Fatalf("expected third request rate-limited (code %d), got %v", code.LoginRateLimitedError, codes)
	}
}

func TestLoginRateLimitedErrorRenderable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.GET("/x", func(ctx *gin.Context) {
		gincontext.Fail(ctx, code.GetError(code.LoginRateLimitedError))
	})
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	var body struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body.Code != code.LoginRateLimitedError {
		t.Fatalf("expected code %d, got %d", code.LoginRateLimitedError, body.Code)
	}
}
