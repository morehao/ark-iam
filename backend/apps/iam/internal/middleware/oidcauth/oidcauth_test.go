package oidcauth

import (
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeOIDCToken(t *testing.T, key *rsa.PrivateKey, sub string, tokenUsage string) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub":       sub,
		"tenant_id": float64(1),
		"aud":       "test-client",
		"iss":       "http://localhost:8099/oidc",
		"exp":       time.Now().Add(time.Hour).Unix(),
		"iat":       time.Now().Unix(),
	}
	if tokenUsage != "" {
		claims["token_usage"] = tokenUsage
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	s, err := token.SignedString(key)
	require.NoError(t, err)
	return s
}

func setupRouter(t *testing.T, validate func(ctx *gin.Context, personID uint, isMachineToken bool) bool) (*gin.Engine, *rsa.PrivateKey) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	r := gin.New()
	r.Use(OIDCCompatibleAuth(func() *rsa.PublicKey { return &key.PublicKey }, WithOIDCSSOValidation(validate)))
	r.GET("/v1/iam/test", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"personID": ginFromContext(ctx),
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
			"orgId":     float64(1),
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
	r, _ := setupRouter(t, func(ctx *gin.Context, personID uint, isMachineToken bool) bool {
		return true // 会话有效，但 RS256 验签失败应直接 401
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/iam/test", nil)
	req.Header.Set(AuthHeaderKey, AuthBearer+makeInternalHS256Token(t, "person:88"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func ginFromContext(ctx *gin.Context) uint {
	v, _ := ctx.Get(gcontext.KeyPersonID)
	id, _ := v.(uint)
	return id
}

func TestOIDCSSOValidationRejectsRevokedSession(t *testing.T) {
	r, key := setupRouter(t, func(ctx *gin.Context, personID uint, isMachineToken bool) bool {
		return false // 会话已撤销
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/iam/test", nil)
	req.Header.Set(AuthHeaderKey, AuthBearer+makeOIDCToken(t, key, "person:88", ""))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.NotContains(t, w.Body.String(), "personID")
}

func TestOIDCSSOValidationAllowsActiveSession(t *testing.T) {
	r, key := setupRouter(t, func(ctx *gin.Context, personID uint, isMachineToken bool) bool {
		return true // 会话有效
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/iam/test", nil)
	req.Header.Set(AuthHeaderKey, AuthBearer+makeOIDCToken(t, key, "person:88", ""))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"personID":88`)
}

func TestOIDCSSOValidationMachineTokenBypassesRevokedSession(t *testing.T) {
	// 机器令牌（token_usage=machine）通过 SSO 校验器时即使自然人的浏览器会话已撤销也放行。
	// 校验器按生产契约（app.go）对机器令牌直接 short-circuit 返回 true。
	r, key := setupRouter(t, func(ctx *gin.Context, personID uint, isMachineToken bool) bool {
		return isMachineToken // 非机器令牌返回 false（会话撤销），机器令牌放行
	})

	// 机器令牌：即便自然人会话被撤销也应 200
	req := httptest.NewRequest(http.MethodGet, "/v1/iam/test", nil)
	req.Header.Set(AuthHeaderKey, AuthBearer+makeOIDCToken(t, key, "person:88", "machine"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"personID":88`)

	// 非机器令牌：会话撤销应 401
	req2 := httptest.NewRequest(http.MethodGet, "/v1/iam/test", nil)
	req2.Header.Set(AuthHeaderKey, AuthBearer+makeOIDCToken(t, key, "person:88", ""))
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusUnauthorized, w2.Code)
}
