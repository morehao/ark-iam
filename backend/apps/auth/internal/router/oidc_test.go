package router_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/config"
	"github.com/morehao/ark-iam/auth/internal/controller/ctroidc"
	"github.com/morehao/ark-iam/auth/internal/router"
	pkgconfig "github.com/morehao/ark-iam/pkg/config"
	"github.com/morehao/ark-iam/pkg/testsetup"
)

func TestOIDCRoutesExposeLoginEndpoint(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)

	gin.SetMode(gin.TestMode)
	config.Conf = &pkgconfig.Config{
		JWT: pkgconfig.JWT{SignKey: "test-sign-key"},
		OIDC: pkgconfig.OIDC{
			Issuer:           "http://localhost:8099/oidc",
			FrontendLoginURL: "http://localhost:4000/oidc/login",
			AllowInsecure:    true,
		},
	}

	engine := gin.New()
	// InitOIDC 只做路由注册；测试用 nil provider 的控制器即可验证路由表
	router.InitOIDC(engine, ctroidc.NewOIDCCtrWithProvider(nil))

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
	assertRouteRegistered(t, paths, http.MethodPost, "/oidc/registerPerson")
	assertRouteRegistered(t, paths, http.MethodPost, "/oidc/createTenant")
	assertRouteMissing(t, paths, http.MethodPost, "/oidc/login/callback")
}

func TestOIDCLoginEndpointBypassesJWTAuth(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)

	gin.SetMode(gin.TestMode)
	config.Conf = &pkgconfig.Config{
		JWT: pkgconfig.JWT{SignKey: "test-sign-key"},
		OIDC: pkgconfig.OIDC{
			Issuer:           "http://localhost:8099/oidc",
			FrontendLoginURL: "http://localhost:4000/oidc/login",
			AllowInsecure:    true,
		},
	}

	engine := gin.New()
	// InitOIDC 只做路由注册；测试用 nil provider 的控制器即可验证路由表
	router.InitOIDC(engine, ctroidc.NewOIDCCtrWithProvider(nil))

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
