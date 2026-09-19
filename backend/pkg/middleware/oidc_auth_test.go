package middleware

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/sdk/rp"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newOIDCAuthTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:oidcauth_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.ApiKeyEntity{}); err != nil {
		t.Fatalf("migrate api_key: %v", err)
	}
	if err := db.AutoMigrate(&model.UserEntity{}); err != nil {
		t.Fatalf("migrate user: %v", err)
	}
	// API Key 鉴权会校验密钥所属租户状态，测试需有 tenant 表。
	if err := db.AutoMigrate(&model.TenantEntity{}); err != nil {
		t.Fatalf("migrate tenant: %v", err)
	}
	dbclient.RegisterDBForTest(dbclient.ServiceNameIam, db)
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

// testKID 是测试 token 的 kid：SDK 校验器要求 kid 必填，且 KeySource 必须能按它取到公钥。
const testKID = "test-kid"

// testKeySource 构造按 kid 取键的进程内 KeySource（等价于生产里 rp.NewKeysFromSet）。
func testKeySource(key *rsa.PublicKey) KeySource {
	return rp.NewKeysFromSet(map[string]*rsa.PublicKey{testKID: key})
}

// testKeySourceWith 构造可容纳多把公钥的 KeySource（多 key 轮换用例）。
func testKeySourceWith(keys map[string]*rsa.PublicKey) KeySource {
	return rp.NewKeysFromSet(keys)
}

// signTestToken 用给定私钥签发带 kid 的 access token。
func signTestToken(t *testing.T, key *rsa.PrivateKey, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = testKID
	s, err := token.SignedString(key)
	require.NoError(t, err)
	return s
}

func makeOIDCToken(t *testing.T, key *rsa.PrivateKey, sub string, tokenUsage string) string {
	t.Helper()
	return makeOIDCTokenWithUser(t, key, sub, tokenUsage, "")
}

func makeOIDCTokenWithUser(t *testing.T, key *rsa.PrivateKey, sub, tokenUsage, userID string) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub":       sub,
		"tenant_id": "1",
		"aud":       "test-client",
		"iss":       "http://localhost:8099/oidc",
		"exp":       time.Now().Add(time.Hour).Unix(),
		"iat":       time.Now().Unix(),
	}
	if tokenUsage != "" {
		claims["token_usage"] = tokenUsage
	}
	if userID != "" {
		claims["user_id"] = userID
	}
	return signTestToken(t, key, claims)
}

func setupRouter(t *testing.T, validate func(ctx *gin.Context, personID string, isMachineToken bool) bool) (*gin.Engine, *rsa.PrivateKey) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	r := gin.New()
	r.Use(OIDCAuth(WithOIDCKeySource(testKeySource(&key.PublicKey)), WithOIDCSSOValidation(validate)))
	r.GET("/v1/test", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"personID": ginFromContext(ctx),
			"userID":   ctx.GetString(gcontext.KeyUserID),
		})
	})
	return r, key
}

func makeInternalHS256Token(t *testing.T, sub string) string {
	t.Helper()
	claims := jwt.MapClaims{
		"customData": map[string]interface{}{
			"userId":    float64(1),
			"personId":  float64(88),
			"tenantId":  float64(1),
			"deptId":    float64(1),
			"userType":  "user",
			"tokenType": "auth",
		},
		"sub": sub,
		"aud": "test-client",
		"iss": "test-issuer",
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := token.SignedString([]byte("test-secret"))
	require.NoError(t, err)
	return s
}

func TestRejectsInternalHS256Token(t *testing.T) {
	r, _ := setupRouter(t, func(ctx *gin.Context, personID string, isMachineToken bool) bool {
		return true // 会话有效，但 RS256 验签失败应直接 401
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/test", nil)
	req.Header.Set(AuthHeaderKey, AuthBearer+makeInternalHS256Token(t, "person:88"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func ginFromContext(ctx *gin.Context) string {
	v, _ := ctx.Get(gcontext.KeyPersonID)
	id, _ := v.(string)
	return id
}

func TestOIDCSSOValidationRejectsRevokedSession(t *testing.T) {
	r, key := setupRouter(t, func(ctx *gin.Context, personID string, isMachineToken bool) bool {
		return false // 会话已撤销
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/test", nil)
	req.Header.Set(AuthHeaderKey, AuthBearer+makeOIDCToken(t, key, "person:88", ""))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.NotContains(t, w.Body.String(), "personID")
}

func TestOIDCSSOValidationAllowsActiveSession(t *testing.T) {
	// person token 反查租户用户需要测试库：建立 SQLite 并 seed (tenant=1, person=88) 的用户
	_ = newOIDCAuthTestDB(t)
	seedDefaultOIDCUser(t)
	t.Cleanup(func() { dbclient.ClearDBForTest(dbclient.ServiceNameIam) })

	r, key := setupRouter(t, func(ctx *gin.Context, personID string, isMachineToken bool) bool {
		return true // 会话有效
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/test", nil)
	req.Header.Set(AuthHeaderKey, AuthBearer+makeOIDCToken(t, key, "person:88", ""))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"personID":"88"`)
	// person token 反查租户用户成功，userID 应被注入上下文
	assert.NotEqual(t, "", w.Body)
	assert.Contains(t, w.Body.String(), `"userID"`)
}

func TestOIDCSSOValidationMachineTokenBypassesRevokedSession(t *testing.T) {
	// 机器令牌（token_usage=machine）通过 SSO 校验器时即使自然人的浏览器会话已撤销也放行。
	// 校验器按生产契约（app.go）对机器令牌直接 short-circuit 返回 true。
	// 非机器 person 分支会反查租户用户：注册空 user 表，令 (tenant=1,person=88) 反查为空 → 401。
	_ = newOIDCAuthTestDB(t)
	t.Cleanup(func() { dbclient.ClearDBForTest(dbclient.ServiceNameIam) })

	r, key := setupRouter(t, func(ctx *gin.Context, personID string, isMachineToken bool) bool {
		return isMachineToken // 非机器令牌返回 false（会话撤销），机器令牌放行
	})

	// 机器令牌：即便自然人会话被撤销也应 200
	req := httptest.NewRequest(http.MethodGet, "/v1/test", nil)
	req.Header.Set(AuthHeaderKey, AuthBearer+makeOIDCToken(t, key, "person:88", "machine"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"personID":"88"`)

	// 非机器令牌：会话撤销应 401
	req2 := httptest.NewRequest(http.MethodGet, "/v1/test", nil)
	req2.Header.Set(AuthHeaderKey, AuthBearer+makeOIDCToken(t, key, "person:88", ""))
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusUnauthorized, w2.Code)
}

// TestAPIKeyParallelAuth 验证 x-api-key 通道与管理 OIDC 并行鉴权（任一通过即放行）。
// 无 OIDC token，仅携带合法 API Key 也应 200；非法 API Key 应 401。
func TestAPIKeyParallelAuth(t *testing.T) {
	// 与其余单元测试一致：内存 SQLite 注册为全局 iam 库，不依赖真实数据库。
	_ = newOIDCAuthTestDB(t)
	t.Cleanup(func() { dbclient.ClearDBForTest(dbclient.ServiceNameIam) })

	rawKey, keyHash := apiKeyHashForTest(t)

	// API Key 鉴权要求所属租户存在且为 active。
	tenant := &model.TenantEntity{
		Code:   "parallel-auth-tenant",
		Name:   "Parallel Auth Tenant",
		Type:   model.TenantTypeCustomer,
		Status: model.TenantStatusActive,
	}
	tenant.ID = "1"
	if err := dao.NewTenantDao().Insert(context.Background(), tenant); err != nil {
		t.Fatalf("seed api key tenant: %v", err)
	}

	owner := &model.UserEntity{
		TenantID: "1",
		UserType: model.UserTypeMachine,
		Name:     "parallel-auth-service-account",
	}
	if err := dao.NewUserDao().Insert(context.Background(), owner); err != nil {
		t.Fatalf("seed api key owner: %v", err)
	}
	t.Cleanup(func() {
		_ = dbclient.IamDB(context.Background()).Where("id = ?", owner.ID).Delete(&model.UserEntity{}).Error
	})

	seed := &model.ApiKeyEntity{
		TenantID:    "1",
		OwnerUserID: owner.ID,
		Name:        "parallel-auth-test",
		KeyHash:     keyHash,
		KeyPrefix:   rawKey[:7],
		CreatedBy:   "1",
	}
	if err := dao.NewApiKeyDao().Insert(context.Background(), seed); err != nil {
		t.Fatalf("seed api key: %v", err)
	}
	t.Cleanup(func() {
		_ = dbclient.IamDB(context.Background()).Where("id = ?", seed.ID).Delete(&model.ApiKeyEntity{}).Error
	})

	r, _ := setupRouter(t, func(ctx *gin.Context, personID string, isMachineToken bool) bool {
		return true
	})

	// 仅 x-api-key（无 OIDC token）应放行
	req := httptest.NewRequest(http.MethodGet, "/v1/test", nil)
	req.Header.Set("x-api-key", rawKey)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "x-api-key 应能独立鉴权通过")

	// 非法 API Key 应 401
	reqBad := httptest.NewRequest(http.MethodGet, "/v1/test", nil)
	reqBad.Header.Set("x-api-key", "invalid-key-value")
	wBad := httptest.NewRecorder()
	r.ServeHTTP(wBad, reqBad)
	assert.Equal(t, http.StatusUnauthorized, wBad.Code)
}

func apiKeyHashForTest(t *testing.T) (raw, hash string) {
	t.Helper()
	raw = "test-raw-key-1234567890abcdef"
	sum := sha256.Sum256([]byte(raw))
	return raw, hex.EncodeToString(sum[:])
}

// seedDefaultOIDCUser 在全局测试 iam 库写入 person=88 对应租户 1 下的用户，供 person token 反查使用。
// 需显式播种 not null JSON（profile/custom_data）与 joined_at，兼容 SQLite。
func seedDefaultOIDCUser(t *testing.T) {
	t.Helper()
	now := time.Now()
	user := &model.UserEntity{
		TenantID: "1",
		PersonID: "88",
		Name:     "oidc-user",
		JoinedAt: &now,
		Status:   model.UserStatusActive,
	}
	if err := dao.NewUserDao().Insert(context.Background(), user); err != nil {
		t.Fatalf("seed oidc user: %v", err)
	}
	// 内存 SQLite 随连接关闭清空，无需显式删除；亦避免 cleanup 顺序访问已清除的全局 DB。
}

func TestRejectsTokenWithWrongIssuer(t *testing.T) {
	// H3：配置 issuer 后，iss 不匹配的 token 一律拒绝
	gin.SetMode(gin.TestMode)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	r := gin.New()
	r.Use(OIDCAuth(WithOIDCKeySource(testKeySource(&key.PublicKey)),
		WithOIDCIssuer("http://localhost:8099/oidc")))
	r.GET("/v1/test", func(ctx *gin.Context) { ctx.Status(http.StatusOK) })

	s := signTestToken(t, key, jwt.MapClaims{
		"sub":       "person:88",
		"tenant_id": "1",
		"aud":       "test-client",
		"iss":       "http://evil.example.com/oidc",
		"exp":       time.Now().Add(time.Hour).Unix(),
		"iat":       time.Now().Unix(),
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/test", nil)
	req.Header.Set(AuthHeaderKey, AuthBearer+s)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRejectsTokenWithWrongAudience(t *testing.T) {
	// H3：配置 audiences 后，aud 不含本应用 client_id 的 token 一律拒绝（跨 client 串用防护）
	gin.SetMode(gin.TestMode)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	r := gin.New()
	r.Use(OIDCAuth(WithOIDCKeySource(testKeySource(&key.PublicKey)),
		WithOIDCIssuer("http://localhost:8099/oidc"),
		WithOIDCAudiences("platform_admin_web")))
	r.GET("/v1/test", func(ctx *gin.Context) { ctx.Status(http.StatusOK) })

	s := signTestToken(t, key, jwt.MapClaims{
		"sub":       "person:88",
		"tenant_id": "1",
		"aud":       "tenant_admin_web", // 另一个 client 的 token
		"iss":       "http://localhost:8099/oidc",
		"exp":       time.Now().Add(time.Hour).Unix(),
		"iat":       time.Now().Unix(),
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/test", nil)
	req.Header.Set(AuthHeaderKey, AuthBearer+s)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAcceptsTokenWithMatchingIssuerAndAudience(t *testing.T) {
	// H3：iss/aud 均匹配时放行（person token 需要测试库反查租户用户）
	_ = newOIDCAuthTestDB(t)
	seedDefaultOIDCUser(t)
	t.Cleanup(func() { dbclient.ClearDBForTest(dbclient.ServiceNameIam) })

	gin.SetMode(gin.TestMode)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	r := gin.New()
	r.Use(OIDCAuth(WithOIDCKeySource(testKeySource(&key.PublicKey)),
		WithOIDCIssuer("http://localhost:8099/oidc"),
		WithOIDCAudiences("platform_admin_web"),
		WithOIDCSSOValidation(func(ctx *gin.Context, personID string, isMachineToken bool) bool { return true })))
	r.GET("/v1/test", func(ctx *gin.Context) { ctx.Status(http.StatusOK) })

	s := signTestToken(t, key, jwt.MapClaims{
		"sub":       "person:88",
		"tenant_id": "1",
		"aud":       "platform_admin_web",
		"iss":       "http://localhost:8099/oidc",
		"exp":       time.Now().Add(time.Hour).Unix(),
		"iat":       time.Now().Unix(),
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/test", nil)
	req.Header.Set(AuthHeaderKey, AuthBearer+s)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestOIDCPersonWithoutUserRejected person token 在该租户无对应用户时拒绝（非租户成员）。
func TestOIDCPersonWithoutUserRejected(t *testing.T) {
	// 建立空 user 表 DB（不 seed），令唯一 person token 的反查为空 → 401
	_ = newOIDCAuthTestDB(t)
	t.Cleanup(func() { dbclient.ClearDBForTest(dbclient.ServiceNameIam) })

	r, key := setupRouter(t, func(ctx *gin.Context, personID string, isMachineToken bool) bool {
		return true
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/test", nil)
	req.Header.Set(AuthHeaderKey, AuthBearer+makeOIDCToken(t, key, "person:88", ""))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestOIDCUserIDFromTokenClaim 锁定「令牌携带 user_id 时不再查库」的行为：
// claims.user_id 即下游审计操作者，且与 tenant_user 行主键一致。
func TestOIDCUserIDFromTokenClaim(t *testing.T) {
	// 建库但不 seed 任何 user：若中间件仍走反查会因"非租户成员"而 401。
	_ = newOIDCAuthTestDB(t)
	t.Cleanup(func() { dbclient.ClearDBForTest(dbclient.ServiceNameIam) })

	r, key := setupRouter(t, func(ctx *gin.Context, personID string, isMachineToken bool) bool {
		return true
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/test", nil)
	req.Header.Set(AuthHeaderKey, AuthBearer+makeOIDCTokenWithUser(t, key, "person:88", "", "tu-777"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"userID":"tu-777"`, "user_id 必须直接取令牌声明")
	assert.Contains(t, w.Body.String(), `"personID":"88"`)
}

// TestOIDCMachineTokenCarriesUserID 机器凭证同样下发 user_id（机器主体），且不查库。
func TestOIDCMachineTokenCarriesUserID(t *testing.T) {
	_ = newOIDCAuthTestDB(t)
	t.Cleanup(func() { dbclient.ClearDBForTest(dbclient.ServiceNameIam) })

	r, key := setupRouter(t, func(ctx *gin.Context, personID string, isMachineToken bool) bool {
		return true
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/test", nil)
	req.Header.Set(AuthHeaderKey, AuthBearer+makeOIDCTokenWithUser(t, key, "ak_1234567", "machine", "owner-9"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"userID":"owner-9"`)
}

// TestOIDCMultiKeyRotation 覆盖最小多 key 轮换：
// active 换 key 后，旧 key 仍在 key set 里时存量 token 继续可用；
// 摘除旧 key 后立即失效（紧急轮换语义）。
func TestOIDCMultiKeyRotation(t *testing.T) {
	_ = newOIDCAuthTestDB(t)
	seedDefaultOIDCUser(t)
	t.Cleanup(func() { dbclient.ClearDBForTest(dbclient.ServiceNameIam) })

	oldKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	newKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	const oldKID = "kid-old"
	const newKID = "kid-new"

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(OIDCAuth(
		WithOIDCKeySource(testKeySourceWith(map[string]*rsa.PublicKey{
			oldKID: &oldKey.PublicKey,
			newKID: &newKey.PublicKey,
		})),
		WithOIDCIssuer("http://localhost:8099/oidc"),
		WithOIDCAudiences("test-client"),
	))
	engine.GET("/v1/test", func(ctx *gin.Context) { ctx.Status(http.StatusOK) })

	oldToken := signTokenWithKID(t, oldKey, oldKID)
	newToken := signTokenWithKID(t, newKey, newKID)

	for name, token := range map[string]string{"旧 key 存量 token": oldToken, "新 key token": newToken} {
		req := httptest.NewRequest(http.MethodGet, "/v1/test", nil)
		req.Header.Set(AuthHeaderKey, AuthBearer+token)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "%s 应在过渡期内继续可用", name)
	}

	// 摘除旧 key（模拟紧急轮换）：旧 token 立即 401，新 token 不受影响。
	engine2 := gin.New()
	engine2.Use(OIDCAuth(
		WithOIDCKeySource(testKeySourceWith(map[string]*rsa.PublicKey{newKID: &newKey.PublicKey})),
		WithOIDCIssuer("http://localhost:8099/oidc"),
		WithOIDCAudiences("test-client"),
	))
	engine2.GET("/v1/test", func(ctx *gin.Context) { ctx.Status(http.StatusOK) })

	reqOld := httptest.NewRequest(http.MethodGet, "/v1/test", nil)
	reqOld.Header.Set(AuthHeaderKey, AuthBearer+oldToken)
	wOld := httptest.NewRecorder()
	engine2.ServeHTTP(wOld, reqOld)
	assert.Equal(t, http.StatusUnauthorized, wOld.Code, "摘除旧 key 后旧 token 必须立即失效")

	reqNew := httptest.NewRequest(http.MethodGet, "/v1/test", nil)
	reqNew.Header.Set(AuthHeaderKey, AuthBearer+newToken)
	wNew := httptest.NewRecorder()
	engine2.ServeHTTP(wNew, reqNew)
	assert.Equal(t, http.StatusOK, wNew.Code, "新 active key 签发的 token 不受影响")
}

func signTokenWithKID(t *testing.T, key *rsa.PrivateKey, kid string) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub":       "person:88",
		"tenant_id": "1",
		"aud":       "test-client",
		"iss":       "http://localhost:8099/oidc",
		"exp":       time.Now().Add(time.Hour).Unix(),
		"iat":       time.Now().Unix(),
		"user_id":   "tu-88",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	s, err := token.SignedString(key)
	require.NoError(t, err)
	return s
}
