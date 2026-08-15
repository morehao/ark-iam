package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/internal/core/oidcop"
	"github.com/stretchr/testify/assert"
)

func TestTenantHint_stashesHintInGinContext(t *testing.T) {
	r := gin.New()
	r.Any("/authorize", TenantHint(), func(ctx *gin.Context) {
		assert.Equal(t, "5", ctx.GetString(ginKeyTenantHint))
		ctx.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/authorize?tenant=5", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTenantHint_noTenant(t *testing.T) {
	r := gin.New()
	r.Any("/authorize", TenantHint(), func(ctx *gin.Context) {
		assert.Equal(t, "", ctx.GetString(ginKeyTenantHint))
		ctx.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/authorize", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTenantHint_passesAnyNonEmptyHint(t *testing.T) {
	// string-id 改造后 tenant hint 是 UUID 字符串：非空即透传，合法性由下游成员租户校验兜底。
	r := gin.New()
	r.Any("/authorize", TenantHint(), func(ctx *gin.Context) {
		assert.Equal(t, "abc", ctx.GetString(ginKeyTenantHint))
		ctx.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/authorize?tenant=abc", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestResourceHint_stashesHintInGinContext(t *testing.T) {
	r := gin.New()
	r.Any("/oauth/token", ResourceHint(), func(ctx *gin.Context) {
		assert.Equal(t, "https://api.example.com", ctx.GetString(ginKeyResourceHint))
		ctx.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", nil)
	req.PostForm = map[string][]string{"resource": {"https://api.example.com"}}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCarryOIDCHints_movesHintsIntoRequestContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Any("/authorize", func(ctx *gin.Context) {
		ctx.Set(ginKeyTenantHint, "5")
		ctx.Set(ginKeyResourceHint, "https://api.example.com")
		CarryOIDCHints(ctx)
		assert.Equal(t, "5", ctx.Request.Context().Value(oidcop.TenantHintKey))
		assert.Equal(t, "https://api.example.com", ctx.Request.Context().Value(oidcop.ResourceHintKey))
		ctx.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/authorize", nil)
	engine.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCarryOIDCHints_noHints_noop(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Any("/authorize", func(ctx *gin.Context) {
		CarryOIDCHints(ctx)
		assert.Nil(t, ctx.Request.Context().Value(oidcop.TenantHintKey))
		ctx.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/authorize", nil)
	engine.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
