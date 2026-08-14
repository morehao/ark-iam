package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestApiKeyAuthValidKey(t *testing.T) {
	db, cleanup := newTestMiddlewareDB(t)
	defer cleanup()

	apiKeyDao := dao.NewApiKeyDao(dao.WithDBGetter(func(ctx context.Context) *gorm.DB {
		return db.WithContext(ctx)
	}))

	rawKey, keyHash, keyPrefix := generateTestKey()
	insertTestApiKey(t, apiKeyDao, 1, keyHash, keyPrefix, nil, nil)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	mw := newApiKeyAuthMiddlewareWithDao(apiKeyDao)
	r.Use(mw.Middleware())
	r.GET("/test", func(ctx *gin.Context) {
		tenantID, _ := ctx.Get("tenantID")
		userID, _ := ctx.Get("userID")
		ctx.JSON(http.StatusOK, gin.H{"tenantID": tenantID, "userID": userID})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+rawKey)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// TestApiKeyAuthViaXApiKeyHeader 验证通过 x-api-key 请求头鉴权（对齐设计文档 §4.4 机器凭证通道）。
func TestApiKeyAuthViaXApiKeyHeader(t *testing.T) {
	db, cleanup := newTestMiddlewareDB(t)
	defer cleanup()

	apiKeyDao := dao.NewApiKeyDao(dao.WithDBGetter(func(ctx context.Context) *gorm.DB {
		return db.WithContext(ctx)
	}))

	rawKey, keyHash, keyPrefix := generateTestKey()
	insertTestApiKey(t, apiKeyDao, 9, keyHash, keyPrefix, nil, nil)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	mw := newApiKeyAuthMiddlewareWithDao(apiKeyDao)
	r.Use(mw.Middleware())
	r.GET("/test", func(ctx *gin.Context) {
		tenantID, _ := ctx.Get("tenantID")
		userID, _ := ctx.Get("userID")
		ctx.JSON(http.StatusOK, gin.H{"tenantID": tenantID, "userID": userID})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("x-api-key", rawKey)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 via x-api-key header, got %d: %s", w.Code, w.Body.String())
	}
}

func TestApiKeyAuthInvalidKey(t *testing.T) {
	db, cleanup := newTestMiddlewareDB(t)
	defer cleanup()

	apiKeyDao := dao.NewApiKeyDao(dao.WithDBGetter(func(ctx context.Context) *gorm.DB {
		return db.WithContext(ctx)
	}))

	_, keyHash, keyPrefix := generateTestKey()
	insertTestApiKey(t, apiKeyDao, 1, keyHash, keyPrefix, nil, nil)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	mw := newApiKeyAuthMiddlewareWithDao(apiKeyDao)
	r.Use(mw.Middleware())
	r.GET("/test", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer some-other-key")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestApiKeyAuthMissingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ApiKeyAuth())
	r.GET("/test", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestApiKeyAuthEmptyBearer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ApiKeyAuth())
	r.GET("/test", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer ")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestApiKeyAuthRevokedKey(t *testing.T) {
	db, cleanup := newTestMiddlewareDB(t)
	defer cleanup()

	apiKeyDao := dao.NewApiKeyDao(dao.WithDBGetter(func(ctx context.Context) *gorm.DB {
		return db.WithContext(ctx)
	}))

	rawKey, keyHash, keyPrefix := generateTestKey()
	now := time.Now()
	insertTestApiKey(t, apiKeyDao, 1, keyHash, keyPrefix, &now, nil)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	mw := newApiKeyAuthMiddlewareWithDao(apiKeyDao)
	r.Use(mw.Middleware())
	r.GET("/test", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+rawKey)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for revoked key, got %d: %s", w.Code, w.Body.String())
	}
}

func TestApiKeyAuthExpiredKey(t *testing.T) {
	db, cleanup := newTestMiddlewareDB(t)
	defer cleanup()

	apiKeyDao := dao.NewApiKeyDao(dao.WithDBGetter(func(ctx context.Context) *gorm.DB {
		return db.WithContext(ctx)
	}))

	rawKey, keyHash, keyPrefix := generateTestKey()
	past := time.Now().Add(-24 * time.Hour)
	insertTestApiKey(t, apiKeyDao, 1, keyHash, keyPrefix, nil, &past)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	mw := newApiKeyAuthMiddlewareWithDao(apiKeyDao)
	r.Use(mw.Middleware())
	r.GET("/test", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+rawKey)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for expired key, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHashApiKey(t *testing.T) {
	key := "test-key-64-chars-1234567890123456789012345678901234567890123456"
	hash := hashApiKey(key)
	if len(hash) != 64 {
		t.Fatalf("expected SHA256 hex length 64, got %d", len(hash))
	}
	expected := sha256.Sum256([]byte(key))
	if hash != hex.EncodeToString(expected[:]) {
		t.Fatal("hash mismatch")
	}
}

func generateTestKey() (rawKey, keyHash, keyPrefix string) {
	b := make([]byte, 32)
	for i := range b {
		b[i] = byte(i % 256)
	}
	rawKey = hex.EncodeToString(b)
	sum := sha256.Sum256([]byte(rawKey))
	keyHash = hex.EncodeToString(sum[:])
	keyPrefix = rawKey[:7]
	return
}

func insertTestApiKey(t *testing.T, apiKeyDao *dao.ApiKeyDao, tenantID uint, keyHash, keyPrefix string, revokedAt *time.Time, expiresAt *time.Time) {
	t.Helper()
	entity := &model.ApiKeyEntity{
		TenantID:  tenantID,
		Name:      "Test Middleware Key",
		KeyHash:   keyHash,
		KeyPrefix: keyPrefix,
		Scope:     json.RawMessage(`{}`),
		ExpiredAt: expiresAt,
		RevokedAt: revokedAt,
		CreatedBy: 42,
	}
	if err := apiKeyDao.Insert(context.Background(), entity); err != nil {
		t.Fatalf("insert test api key: %v", err)
	}
}

func newTestMiddlewareDB(t *testing.T) (*gorm.DB, func()) {
	t.Helper()
	dsn := fmt.Sprintf("file:%s_%d?mode=memory&cache=shared", sanitizeMiddlewareTestName(t.Name()), time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.ApiKeyEntity{}); err != nil {
		t.Fatalf("migrate api_key: %v", err)
	}
	cleanup := func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	}
	return db, cleanup
}

func sanitizeMiddlewareTestName(name string) string {
	replacer := strings.NewReplacer("/", "_", " ", "_", ":", "_")
	return replacer.Replace(name)
}
