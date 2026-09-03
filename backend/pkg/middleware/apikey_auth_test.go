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
	userDao := dao.NewUserDao(dao.WithDBGetter(func(ctx context.Context) *gorm.DB {
		return db.WithContext(ctx)
	}))

	owner := seedTestOwnerUser(t, userDao, "1", false)
	rawKey, keyHash, keyPrefix := generateTestKey()
	insertTestApiKey(t, apiKeyDao, "1", owner.ID, keyHash, keyPrefix, nil, nil)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	mw := newApiKeyAuthMiddlewareWithDao(apiKeyDao, userDao)
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
	var body struct {
		TenantID string `json:"tenantID"`
		UserID   string `json:"userID"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	// 鉴权身份应等于密钥归属用户（owner），而非创建人
	if body.UserID != owner.ID {
		t.Fatalf("expected ctx userID=owner %s, got %s", owner.ID, body.UserID)
	}
	if body.TenantID != "1" {
		t.Fatalf("expected ctx tenantID=1, got %s", body.TenantID)
	}
}

// TestApiKeyAuthViaXApiKeyHeader 验证通过 x-api-key 请求头鉴权（对齐设计文档 §4.4 机器凭证通道）。
func TestApiKeyAuthViaXApiKeyHeader(t *testing.T) {
	db, cleanup := newTestMiddlewareDB(t)
	defer cleanup()

	apiKeyDao := dao.NewApiKeyDao(dao.WithDBGetter(func(ctx context.Context) *gorm.DB {
		return db.WithContext(ctx)
	}))
	userDao := dao.NewUserDao(dao.WithDBGetter(func(ctx context.Context) *gorm.DB {
		return db.WithContext(ctx)
	}))

	owner := seedTestOwnerUser(t, userDao, "9", true) // 服务账号归属
	rawKey, keyHash, keyPrefix := generateTestKey()
	insertTestApiKey(t, apiKeyDao, "9", owner.ID, keyHash, keyPrefix, nil, nil)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	mw := newApiKeyAuthMiddlewareWithDao(apiKeyDao, userDao)
	r.Use(mw.Middleware())
	r.GET("/test", func(ctx *gin.Context) {
		tenantID, _ := ctx.Get("tenantID")
		userID, _ := ctx.Get("userID")
		userType, _ := ctx.Get(ContextKeyUserType)
		ctx.JSON(http.StatusOK, gin.H{"tenantID": tenantID, "userID": userID, "userType": userType})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("x-api-key", rawKey)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 via x-api-key header, got %d: %s", w.Code, w.Body.String())
	}
	var body struct {
		TenantID string `json:"tenantID"`
		UserID   string `json:"userID"`
		UserType string `json:"userType"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.UserID != owner.ID {
		t.Fatalf("expected ctx userID=machine owner %s, got %s", owner.ID, body.UserID)
	}
	if body.UserType != string(model.UserTypeMachine) {
		t.Fatalf("expected ctx userType=machine, got %s", body.UserType)
	}
}

func TestApiKeyAuthSuspendedOwnerRejected(t *testing.T) {
	db, cleanup := newTestMiddlewareDB(t)
	defer cleanup()

	apiKeyDao := dao.NewApiKeyDao(dao.WithDBGetter(func(ctx context.Context) *gorm.DB {
		return db.WithContext(ctx)
	}))
	userDao := dao.NewUserDao(dao.WithDBGetter(func(ctx context.Context) *gorm.DB {
		return db.WithContext(ctx)
	}))

	owner := seedTestOwnerUser(t, userDao, "1", false)
	owner.IsSuspended = true
	if err := userDao.UpdateMap(context.Background(), owner.ID, map[string]any{"is_suspended": true}); err != nil {
		t.Fatalf("suspend owner: %v", err)
	}

	rawKey, keyHash, keyPrefix := generateTestKey()
	insertTestApiKey(t, apiKeyDao, "1", owner.ID, keyHash, keyPrefix, nil, nil)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	mw := newApiKeyAuthMiddlewareWithDao(apiKeyDao, userDao)
	r.Use(mw.Middleware())
	r.GET("/test", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("x-api-key", rawKey)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for suspended owner, got %d: %s", w.Code, w.Body.String())
	}
}

func TestApiKeyAuthInvalidKey(t *testing.T) {
	db, cleanup := newTestMiddlewareDB(t)
	defer cleanup()

	apiKeyDao := dao.NewApiKeyDao(dao.WithDBGetter(func(ctx context.Context) *gorm.DB {
		return db.WithContext(ctx)
	}))
	userDao := dao.NewUserDao(dao.WithDBGetter(func(ctx context.Context) *gorm.DB {
		return db.WithContext(ctx)
	}))

	_, keyHash, keyPrefix := generateTestKey()
	insertTestApiKey(t, apiKeyDao, "1", "", keyHash, keyPrefix, nil, nil)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	mw := newApiKeyAuthMiddlewareWithDao(apiKeyDao, userDao)
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

	userDao := dao.NewUserDao(dao.WithDBGetter(func(ctx context.Context) *gorm.DB {
		return db.WithContext(ctx)
	}))
	rawKey, keyHash, keyPrefix := generateTestKey()
	now := time.Now()
	insertTestApiKey(t, apiKeyDao, "1", "", keyHash, keyPrefix, &now, nil)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	mw := newApiKeyAuthMiddlewareWithDao(apiKeyDao, userDao)
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

	userDao := dao.NewUserDao(dao.WithDBGetter(func(ctx context.Context) *gorm.DB {
		return db.WithContext(ctx)
	}))
	rawKey, keyHash, keyPrefix := generateTestKey()
	past := time.Now().Add(-24 * time.Hour)
	insertTestApiKey(t, apiKeyDao, "1", "", keyHash, keyPrefix, nil, &past)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	mw := newApiKeyAuthMiddlewareWithDao(apiKeyDao, userDao)
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

func insertTestApiKey(t *testing.T, apiKeyDao *dao.ApiKeyDao, tenantID, ownerUserID, keyHash, keyPrefix string, revokedAt *time.Time, expiresAt *time.Time) {
	t.Helper()
	entity := &model.ApiKeyEntity{
		TenantID:    tenantID,
		OwnerUserID: ownerUserID,
		Name:        "Test Middleware Key",
		KeyHash:     keyHash,
		KeyPrefix:   keyPrefix,
		Scope:       json.RawMessage(`{}`),
		ExpiredAt:   expiresAt,
		RevokedAt:   revokedAt,
		CreatedBy:   "42",
	}
	if err := apiKeyDao.Insert(context.Background(), entity); err != nil {
		t.Fatalf("insert test api key: %v", err)
	}
}

// seedTestOwnerUser 播种密钥归属用户（真实用户或服务账号）并返回实体。
func seedTestOwnerUser(t *testing.T, userDao *dao.UserDao, tenantID string, machine bool) *model.UserEntity {
	t.Helper()
	userType := model.UserTypeMember
	name := "Test Owner"
	if machine {
		userType = model.UserTypeMachine
		name = "Test Service Account"
	}
	entity := &model.UserEntity{
		TenantID:   tenantID,
		UserType:   userType,
		Name:       name,
		Profile:    json.RawMessage(`{}`),
		CustomData: json.RawMessage(`{}`),
	}
	if err := userDao.Insert(context.Background(), entity); err != nil {
		t.Fatalf("insert test owner user: %v", err)
	}
	return entity
}

func newTestMiddlewareDB(t *testing.T) (*gorm.DB, func()) {
	t.Helper()
	dsn := fmt.Sprintf("file:%s_%d?mode=memory&cache=shared", sanitizeMiddlewareTestName(t.Name()), time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.ApiKeyEntity{}, &model.UserEntity{}); err != nil {
		t.Fatalf("migrate api_key/user: %v", err)
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
