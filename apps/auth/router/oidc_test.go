package router_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/apps/auth/config"
	"github.com/morehao/ark-iam/apps/auth/controller/ctroidc"
	"github.com/morehao/ark-iam/apps/auth/router"
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
	router.InitOIDC(engine.Group("/"), ctroidc.NewOIDCCtrWithProvider(nil, nil))

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
	router.InitOIDC(engine.Group("/"), ctroidc.NewOIDCCtrWithProvider(nil, nil))

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

// TestOIDCCORSSendsCredentialsForCookie 验证 /oidc/* 在 golib v1.36.2 下仍能携带
// SSO 会话 cookie：CORS 必须下发 Access-Control-Allow-Credentials: true，且回显具体
// Origin（不能用通配 *）。未配置白名单时动态回显请求 Origin。
func TestOIDCCORSSendsCredentialsForCookie(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)

	gin.SetMode(gin.TestMode)
	config.Conf = &pkgconfig.Config{
		JWT:  pkgconfig.JWT{SignKey: "test-sign-key"},
		OIDC: pkgconfig.OIDC{Issuer: "http://localhost:8099/oidc", AllowInsecure: true},
	}

	engine := gin.New()
	router.InitOIDC(engine.Group("/"), ctroidc.NewOIDCCtrWithProvider(nil, nil))

	// 用真实的 POST 请求（登录端点）验证：跨源请求会带上 Origin，CORS 中间件应回显
	// Origin 并下发 Allow-Credentials。注意 gin 仅在路由命中时才执行分组中间件，
	// 故这里用已注册的 POST 方法（预检 OPTIONS 在本项目部署上由反向代理/同源处理）。
	req := httptest.NewRequest(http.MethodPost, "/oidc/login", nil)
	req.Header.Set("Origin", "https://login.example.com")
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	if got := resp.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("expected Allow-Credentials=true for /oidc/*, got %q", got)
	}
	if got := resp.Header().Get("Access-Control-Allow-Origin"); got != "https://login.example.com" {
		t.Fatalf("expected echoed origin, got %q", got)
	}
}

// TestOIDCCORSAllowlistRejectsUnknownOrigin 验证配置了 CORSAllowOrigins 白名单时，
// 非白名单 Origin 不会被放行（不回显 Origin、不下发凭证）。
func TestOIDCCORSAllowlistRejectsUnknownOrigin(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)

	gin.SetMode(gin.TestMode)
	config.Conf = &pkgconfig.Config{
		JWT: pkgconfig.JWT{SignKey: "test-sign-key"},
		OIDC: pkgconfig.OIDC{
			Issuer:           "http://localhost:8099/oidc",
			AllowInsecure:    true,
			CORSAllowOrigins: []string{"https://console.example.com"},
		},
	}

	engine := gin.New()
	router.InitOIDC(engine.Group("/"), ctroidc.NewOIDCCtrWithProvider(nil, nil))

	req := httptest.NewRequest(http.MethodPost, "/oidc/login", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	if got := resp.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no Allow-Origin for non-allowlisted origin, got %q", got)
	}
}
