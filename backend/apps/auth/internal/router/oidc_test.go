package router_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/config"
	"github.com/morehao/ark-iam/auth/internal/router"
	pkgconfig "github.com/morehao/ark-iam/pkg/config"
	"github.com/morehao/ark-iam/pkg/testsetup"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func TestOIDCRoutesExposeLoginEndpoint(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	gin.SetMode(gin.TestMode)
	config.Conf = &pkgconfig.Config{
		JWT: pkgconfig.JWT{SignKey: "test-sign-key"},
		OIDC: pkgconfig.OIDC{
			Issuer:           "http://localhost:8099/oidc",
			FrontendLoginURL: "http://localhost:3000/oidc/login",
			AllowInsecure:    true,
		},
	}

	engine := gin.New()
	groups := ginserver.NewRouterGroups(engine, "iam", ginserver.VersionGroup{Version: ginserver.ApiVersionV1})
	router.InitOIDC(engine, groups)

	routes := engine.Routes()
	paths := make(map[string]map[string]bool, len(routes))
	for _, route := range routes {
		if _, ok := paths[route.Path]; !ok {
			paths[route.Path] = map[string]bool{}
		}
		paths[route.Path][route.Method] = true
	}

	assertRouteRegistered(t, paths, http.MethodPost, "/oidc/login")
	assertRouteRegistered(t, paths, http.MethodPost, "/oidc/login/selectTenant")
	assertRouteMissing(t, paths, http.MethodPost, "/oidc/login/callback")
}

func TestOIDCLoginEndpointBypassesJWTAuth(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	gin.SetMode(gin.TestMode)
	config.Conf = &pkgconfig.Config{
		JWT: pkgconfig.JWT{SignKey: "test-sign-key"},
		OIDC: pkgconfig.OIDC{
			Issuer:           "http://localhost:8099/oidc",
			FrontendLoginURL: "http://localhost:3000/oidc/login",
			AllowInsecure:    true,
		},
	}

	engine := gin.New()
	groups := ginserver.NewRouterGroups(engine, "iam", ginserver.VersionGroup{Version: ginserver.ApiVersionV1})
	router.InitOIDC(engine, groups)

	req := httptest.NewRequest(http.MethodPost, "/oidc/login", nil)
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	var body struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if body.Code == http.StatusUnauthorized {
		t.Fatalf("expected OIDC login endpoint to bypass JWT auth, got business code %d", body.Code)
	}
}
