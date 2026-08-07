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

func makeOIDCToken(t *testing.T, key *rsa.PrivateKey, sub string) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub":       sub,
		"tenant_id": float64(1),
		"aud":       "test-client",
		"iss":       "http://localhost:8099/v1/iam/oidc",
		"exp":       time.Now().Add(time.Hour).Unix(),
		"iat":       time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	s, err := token.SignedString(key)
	require.NoError(t, err)
	return s
}

func setupRouter(t *testing.T, validate func(ctx *gin.Context, personID uint) bool) (*gin.Engine, *rsa.PrivateKey) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	r := gin.New()
	r.Use(OIDCCompatibleAuth("test-secret", func() *rsa.PublicKey { return &key.PublicKey }, WithOIDCSSOValidation(validate)))
	r.GET("/v1/iam/test", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"personID": ginFromContext(ctx),
		})
	})
	return r, key
}

func ginFromContext(ctx *gin.Context) uint {
	v, _ := ctx.Get(gcontext.KeyPersonID)
	id, _ := v.(uint)
	return id
}

func TestOIDCSSOValidationRejectsRevokedSession(t *testing.T) {
	r, key := setupRouter(t, func(ctx *gin.Context, personID uint) bool {
		return false // 会话已撤销
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/iam/test", nil)
	req.Header.Set(AuthHeaderKey, AuthBearer+makeOIDCToken(t, key, "person:88"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.NotContains(t, w.Body.String(), "personID")
}

func TestOIDCSSOValidationAllowsActiveSession(t *testing.T) {
	r, key := setupRouter(t, func(ctx *gin.Context, personID uint) bool {
		return true // 会话有效
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/iam/test", nil)
	req.Header.Set(AuthHeaderKey, AuthBearer+makeOIDCToken(t, key, "person:88"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"personID":88`)
}
