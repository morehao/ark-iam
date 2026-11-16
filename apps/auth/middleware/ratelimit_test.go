package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/apps/auth/testutil"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/loginsetting"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/redis/go-redis/v9"
)

func TestLoginRateLimitWithoutRedisFailsOpen(t *testing.T) {
	testutil.SetupSQLite(t)
	gin.SetMode(gin.TestMode)
	dbclient.RedisCli = nil // 无 Redis -> fail-open

	engine := gin.New()
	engine.POST("/oidc/login", LoginRateLimit(), func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"code": 0})
	})
	for i := 0; i < 100; i++ {
		req := httptest.NewRequest(http.MethodPost, "/oidc/login", nil)
		resp := httptest.NewRecorder()
		engine.ServeHTTP(resp, req)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected fail-open pass, got status %d", resp.Code)
		}
	}
}

func TestRateParamsFromSetting(t *testing.T) {
	testutil.SetupSQLite(t)
	ctx := context.Background()
	// 未配置：默认 rate=30 burst=10
	if rate, burst, _ := currentRateParams(); rate != 30 || burst != 10 {
		t.Fatalf("expected default rate=30 burst=10, got rate=%d burst=%d", rate, burst)
	}
	// 配置后生效
	if err := loginsetting.Save(ctx, dbclient.IamDB, loginsetting.Config{
		MaxFailures: 5, WindowSec: 300, LockSec: 900, RatePerMinute: 60, Burst: 20,
	}); err != nil {
		t.Fatalf("save: %v", err)
	}
	if rate, burst, _ := currentRateParams(); rate != 60 || burst != 20 {
		t.Fatalf("expected configured rate=60 burst=20, got rate=%d burst=%d", rate, burst)
	}
}

func TestLoginRateLimitBlocksExcessWithRedis(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis start fail: %v", err)
	}
	defer mr.Close()

	testutil.SetupSQLite(t)
	oldCli := dbclient.RedisCli
	dbclient.RedisCli = redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { dbclient.RedisCli = oldCli }()

	ctx := context.Background()
	if err := loginsetting.Save(ctx, dbclient.IamDB, loginsetting.Config{
		MaxFailures: 5, WindowSec: 300, LockSec: 900, RatePerMinute: 2, Burst: 2,
	}); err != nil {
		t.Fatalf("save: %v", err)
	}

	gin.SetMode(gin.TestMode)
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
