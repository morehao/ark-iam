package svcoidc

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	appconfig "github.com/morehao/ark-iam/auth/config"
)

func TestEndSessionClearsSSOCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	appconfig.Conf = &appconfig.Config{}
	engine := gin.New()
	group := engine.Group("/v1/iam/oidc")
	RegisterProviderRoutes(group, &OIDCProvider{}, "iam_sso_session")

	req := httptest.NewRequest(http.MethodGet, "/v1/iam/oidc/end_session", nil)
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
