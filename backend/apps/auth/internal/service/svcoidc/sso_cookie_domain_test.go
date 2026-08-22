package svcoidc_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/config"
	"github.com/morehao/ark-iam/auth/internal/controller/ctroidc"
	"github.com/morehao/ark-iam/auth/internal/router"
	pkgconfig "github.com/morehao/ark-iam/pkg/config"
	"github.com/morehao/ark-iam/pkg/testsetup"
)

func TestLoggedOutUsesHostOnlyCookieByDefault(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)

	gin.SetMode(gin.TestMode)
	config.Conf = &pkgconfig.Config{
		JWT: pkgconfig.JWT{SignKey: "test-key"},
		OIDC: pkgconfig.OIDC{
			Issuer:           "http://localhost:8099/oidc",
			FrontendLoginURL: "http://localhost:4000/login",
			AllowInsecure:    true,
		},
	}

	engine := gin.New()
	// InitOIDC 只做路由注册；LoggedOut 不依赖 provider，nil 即可
	router.InitOIDC(engine, ctroidc.NewOIDCCtrWithProvider(nil))

	req := httptest.NewRequest(http.MethodGet, "/oidc/logged-out", nil)
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	setCookie := resp.Header().Get("Set-Cookie")
	if !strings.Contains(setCookie, "iam_sso_session=") {
		t.Fatalf("expected iam_sso_session clearing cookie, got %q", setCookie)
	}
	if strings.Contains(setCookie, "Domain=") {
		t.Fatalf("expected logged-out to clear host-only cookie by default, got %q", setCookie)
	}

	t.Logf("Set-Cookie header: %s", setCookie)
}

func TestLoggedOutUsesConfiguredCookieDomain(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)

	gin.SetMode(gin.TestMode)
	config.Conf = &pkgconfig.Config{
		JWT: pkgconfig.JWT{SignKey: "test-key"},
		OIDC: pkgconfig.OIDC{
			Issuer:           "http://localhost:8099/oidc",
			FrontendLoginURL: "http://localhost:4000/login",
			CookieDomain:     "example.com",
			AllowInsecure:    true,
		},
	}

	engine := gin.New()
	// InitOIDC 只做路由注册；LoggedOut 不依赖 provider，nil 即可
	router.InitOIDC(engine, ctroidc.NewOIDCCtrWithProvider(nil))

	req := httptest.NewRequest(http.MethodGet, "/oidc/logged-out", nil)
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	setCookie := resp.Header().Get("Set-Cookie")
	if !strings.Contains(setCookie, "Domain=example.com") {
		t.Fatalf("expected configured cookie domain, got %q", setCookie)
	}
}
