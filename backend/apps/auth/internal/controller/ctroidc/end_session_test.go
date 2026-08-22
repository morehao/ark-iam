package ctroidc

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	appconfig "github.com/morehao/ark-iam/auth/config"
	"github.com/morehao/ark-iam/auth/internal/service/svcoidc"
	pkgconfig "github.com/morehao/ark-iam/pkg/config"
)

func TestEndSessionClearsSSOCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	appconfig.Conf = &pkgconfig.Config{}
	engine := gin.New()
	ctr := NewOIDCCtrWithProvider(&svcoidc.OIDCProvider{})
	engine.Any("/oidc/end_session", ctr.EndSession)

	req := httptest.NewRequest(http.MethodGet, "/oidc/end_session", nil)
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	setCookie := resp.Header().Get("Set-Cookie")
	if !strings.Contains(setCookie, "iam_sso_session=") {
		t.Fatalf("expected iam_sso_session clearing cookie, got %q", setCookie)
	}

	if !strings.Contains(setCookie, "Max-Age=0") && !strings.Contains(setCookie, "Expires=") {
		t.Fatalf("expected cookie to be cleared, got %q", setCookie)
	}

	if strings.Contains(setCookie, "Domain=") {
		t.Fatalf("expected cleared cookie to remain host-only by default, got %q", setCookie)
	}
}

func TestProviderHandlerFallsBackWhenProviderMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	ctr := NewOIDCCtrWithProvider(&svcoidc.OIDCProvider{})
	engine.Any("/oidc/keys", ctr.ProviderHandler())

	req := httptest.NewRequest(http.MethodGet, "/oidc/keys", nil)
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200 fallback without provider, got %d", resp.Code)
	}
}

func TestLoggedOutClearsSSOCookieAndRedirects(t *testing.T) {
	gin.SetMode(gin.TestMode)
	appconfig.Conf = &pkgconfig.Config{OIDC: pkgconfig.OIDC{FrontendLoginURL: "http://localhost:4000/oidc/login"}}
	engine := gin.New()
	ctr := NewOIDCCtrWithProvider(&svcoidc.OIDCProvider{})
	engine.GET("/oidc/logged-out", ctr.LoggedOut)

	req := httptest.NewRequest(http.MethodGet, "/oidc/logged-out", nil)
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	if resp.Code != http.StatusFound {
		t.Fatalf("expected 302 redirect, got %d", resp.Code)
	}
	if loc := resp.Header().Get("Location"); loc != "http://localhost:4000/oidc/login" {
		t.Fatalf("unexpected redirect location: %q", loc)
	}
	setCookie := resp.Header().Get("Set-Cookie")
	if !strings.Contains(setCookie, "iam_sso_session=") {
		t.Fatalf("expected iam_sso_session clearing cookie, got %q", setCookie)
	}
}
