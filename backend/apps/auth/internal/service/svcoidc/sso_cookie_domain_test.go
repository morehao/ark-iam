package svcoidc_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/config"
	"github.com/morehao/ark-iam/auth/internal/router"
	"github.com/morehao/ark-iam/pkg/testsetup"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func TestLoggedOutUsesHostOnlyCookieByDefault(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	gin.SetMode(gin.TestMode)
	config.Conf = &config.Config{
		JWT: config.JWT{SignKey: "test-key"},
		OIDC: config.OIDC{
			Issuer:           "http://localhost:8099/v1/iam/oidc",
			FrontendLoginURL: "http://localhost:3003/login",
			AllowInsecure:    true,
		},
	}

	engine := gin.New()
	groups := ginserver.NewRouterGroups(engine, "iam", ginserver.Version{Name: gconstant.ApiVersionV1})
	router.InitOIDC(engine, groups)

	req := httptest.NewRequest(http.MethodGet, "/v1/iam/oidc/logged-out", nil)
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
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	gin.SetMode(gin.TestMode)
	config.Conf = &config.Config{
		JWT: config.JWT{SignKey: "test-key"},
		OIDC: config.OIDC{
			Issuer:           "http://localhost:8099/v1/iam/oidc",
			FrontendLoginURL: "http://localhost:3003/login",
			CookieDomain:     "example.com",
			AllowInsecure:    true,
		},
	}

	engine := gin.New()
	groups := ginserver.NewRouterGroups(engine, "iam", ginserver.Version{Name: gconstant.ApiVersionV1})
	router.InitOIDC(engine, groups)

	req := httptest.NewRequest(http.MethodGet, "/v1/iam/oidc/logged-out", nil)
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	setCookie := resp.Header().Get("Set-Cookie")
	if !strings.Contains(setCookie, "Domain=example.com") {
		t.Fatalf("expected configured cookie domain, got %q", setCookie)
	}
}
