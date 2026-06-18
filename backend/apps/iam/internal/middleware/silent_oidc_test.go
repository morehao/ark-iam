package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSilentSSORequired_NoCookie_ReturnsLoginRequired(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	var nextCalled bool
	r.Use(SilentSSORequired("iam_sso_session"))
	r.GET("/authorize", func(ctx *gin.Context) {
		nextCalled = true
		ctx.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet,
		"/authorize?prompt=none&redirect_uri=http%3A%2F%2Flocalhost%3A3000%2Fauth%2Fcallback&state=abc",
		nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if nextCalled {
		t.Fatal("next handler should not be called when SSO cookie is missing")
	}
	if w.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", w.Code)
	}
	loc := w.Header().Get("Location")
	if loc != "http://localhost:3000/auth/callback?error=login_required&state=abc" {
		t.Fatalf("unexpected Location: %s", loc)
	}
}

func TestSilentSSORequired_HasCookie_CallsNext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	var nextCalled bool
	r.Use(SilentSSORequired("iam_sso_session"))
	r.GET("/authorize", func(ctx *gin.Context) {
		nextCalled = true
		ctx.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet,
		"/authorize?prompt=none&redirect_uri=http%3A%2F%2Flocalhost%3A3000%2Fauth%2Fcallback",
		nil)
	req.AddCookie(&http.Cookie{Name: "iam_sso_session", Value: "valid-session-id"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !nextCalled {
		t.Fatal("next handler should be called when SSO cookie is present")
	}
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestSilentSSORequired_NonSilentPrompt_CallsNext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	var nextCalled bool
	r.Use(SilentSSORequired("iam_sso_session"))
	r.GET("/authorize", func(ctx *gin.Context) {
		nextCalled = true
		ctx.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet,
		"/authorize?prompt=login&redirect_uri=http%3A%2F%2Flocalhost%3A3000%2Fauth%2Fcallback",
		nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !nextCalled {
		t.Fatal("next handler should be called for non-silent prompt")
	}
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestSilentSSORequired_NoRedirectURI_Returns400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	var nextCalled bool
	r.Use(SilentSSORequired("iam_sso_session"))
	r.GET("/authorize", func(ctx *gin.Context) {
		nextCalled = true
		ctx.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/authorize?prompt=none", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if nextCalled {
		t.Fatal("next handler should not be called when redirect_uri is missing")
	}
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestSilentSSORequired_NoState_OmitsStateParam(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SilentSSORequired("iam_sso_session"))
	r.GET("/authorize", func(ctx *gin.Context) {
		ctx.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet,
		"/authorize?prompt=none&redirect_uri=http%3A%2F%2Flocalhost%3A3000%2Fauth%2Fcallback",
		nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", w.Code)
	}
	loc := w.Header().Get("Location")
	if loc != "http://localhost:3000/auth/callback?error=login_required" {
		t.Fatalf("unexpected Location: %s", loc)
	}
}
