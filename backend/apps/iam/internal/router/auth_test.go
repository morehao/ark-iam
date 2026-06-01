package router_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	iam "github.com/morehao/ark-iam/iam"
	"github.com/morehao/ark-iam/iam/config"
	"github.com/morehao/ark-iam/iam/internal/router"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func TestAuthAndConnectorRoutesUseUnifiedEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config.Conf = &config.Config{JWT: config.JWT{SignKey: "test-sign-key"}, Server: config.Server{Env: "dev"}}

	engine := gin.New()
	groups := ginserver.NewRouterGroups(engine, "iam", ginserver.Version{Name: gconstant.ApiVersionV1})
	router.RegisterRouter(groups, "iam")

	routes := engine.Routes()
	paths := make(map[string]map[string]bool, len(routes))
	for _, route := range routes {
		if _, ok := paths[route.Path]; !ok {
			paths[route.Path] = map[string]bool{}
		}
		paths[route.Path][route.Method] = true
	}

	assertRouteRegistered(t, paths, http.MethodPost, "/v1/iam/connector/getFactoryList")
	assertRouteRegistered(t, paths, http.MethodPost, "/v1/iam/connector/:connectorId/authorize")
	assertRouteRegistered(t, paths, http.MethodGet, "/v1/iam/connector/callback")
	assertRouteRegistered(t, paths, http.MethodPost, "/v1/iam/login")
	assertRouteRegistered(t, paths, http.MethodPost, "/v1/iam/register")
	assertRouteRegistered(t, paths, http.MethodPost, "/v1/iam/refreshToken")
	assertRouteRegistered(t, paths, http.MethodGet, "/v1/iam/myTenants")
	assertRouteRegistered(t, paths, http.MethodPost, "/v1/iam/selectTenant")
	assertRouteRegistered(t, paths, http.MethodPost, "/v1/iam/switchTenant")
	assertRouteRegistered(t, paths, http.MethodPost, "/v1/iam/logoutAll")
	assertRouteRegistered(t, paths, http.MethodPost, "/v1/iam/user/pageList")
	assertRouteRegistered(t, paths, http.MethodPost, "/v1/iam/user/assignDepartments")
	assertRouteRegistered(t, paths, http.MethodGet, "/v1/iam/user/getUserDepartmentByUser")
	assertRouteRegistered(t, paths, http.MethodPost, "/v1/iam/tenant/create")
	assertRouteRegistered(t, paths, http.MethodPost, "/v1/iam/tenant/delete")
	assertRouteRegistered(t, paths, http.MethodPost, "/v1/iam/tenant/update")
	assertRouteRegistered(t, paths, http.MethodGet, "/v1/iam/tenant/detail")
	assertRouteRegistered(t, paths, http.MethodPost, "/v1/iam/tenant/pageList")
	assertRouteRegistered(t, paths, http.MethodGet, "/v1/iam/userinfo")
	assertRouteRegistered(t, paths, http.MethodPost, "/v1/iam/logout")
	assertRouteRegistered(t, paths, http.MethodGet, "/v1/iam/person/detail")
	assertRouteRegistered(t, paths, http.MethodPost, "/v1/iam/person/updatePassword")

	assertRouteMissing(t, paths, http.MethodGet, "/v1/iam/connector/factories")
	assertRouteMissing(t, paths, http.MethodPost, "/v1/iam/connector/:connectorId/authorization-url")
	assertRouteMissing(t, paths, http.MethodPost, "/v1/iam/ssoConnector/create")
	assertRouteMissing(t, paths, http.MethodGet, "/v1/iam/ssoConnector/providers")
	assertRouteMissing(t, paths, http.MethodPost, "/v1/iam/auth/register")
	assertRouteMissing(t, paths, http.MethodPost, "/v1/iam/auth/loginByPassword")
	assertRouteMissing(t, paths, http.MethodPost, "/v1/iam/auth/selectTenant")
	assertRouteMissing(t, paths, http.MethodPost, "/v1/iam/create")
	assertRouteMissing(t, paths, http.MethodPost, "/v1/iam/delete")
	assertRouteMissing(t, paths, http.MethodPost, "/v1/iam/update")
	assertRouteMissing(t, paths, http.MethodGet, "/v1/iam/detail")
	assertRouteMissing(t, paths, http.MethodPost, "/v1/iam/pageList")
	assertRouteMissing(t, paths, http.MethodPost, "/v1/iam/user/createUserDepartment")
	assertRouteMissing(t, paths, http.MethodPost, "/v1/iam/user/deleteUserDepartment")
	assertRouteMissing(t, paths, http.MethodPost, "/v1/iam/user/updateUserDepartment")
	assertRouteMissing(t, paths, http.MethodGet, "/v1/iam/user/detailUserDepartment")
	assertRouteMissing(t, paths, http.MethodPost, "/v1/iam/user/pageListUserDepartment")
}

func TestIAMRoutersWhitelistPublicAuthEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config.Conf = &config.Config{JWT: config.JWT{SignKey: "test-sign-key"}, Server: config.Server{Env: "dev"}}

	engine := gin.New()
	iam.Routers(engine)

	assertAnonymousRouteAccessible(t, engine, http.MethodPost, "/v1/iam/login")
	assertAnonymousRouteAccessible(t, engine, http.MethodPost, "/v1/iam/register")
	assertAnonymousRouteAccessible(t, engine, http.MethodPost, "/v1/iam/refreshToken")
	assertAnonymousRouteAccessible(t, engine, http.MethodGet, "/v1/iam/connector/callback")
	assertAnonymousRouteAccessible(t, engine, http.MethodGet, "/v1/iam/myTenants")
	assertAnonymousRouteAccessible(t, engine, http.MethodPost, "/v1/iam/selectTenant")

	assertAnonymousRouteBlocked(t, engine, http.MethodPost, "/v1/iam/user/pageList")
	assertAnonymousRouteBlocked(t, engine, http.MethodGet, "/v1/iam/userinfo")
	assertAnonymousRouteBlocked(t, engine, http.MethodPost, "/v1/iam/logout")
	assertAnonymousRouteBlocked(t, engine, http.MethodPost, "/v1/iam/switchTenant")
	assertAnonymousRouteBlocked(t, engine, http.MethodPost, "/v1/iam/logoutAll")
	assertAnonymousRouteBlocked(t, engine, http.MethodGet, "/v1/iam/person/detail")
	assertAnonymousRouteBlocked(t, engine, http.MethodPost, "/v1/iam/person/updatePassword")
}

func assertRouteRegistered(t *testing.T, routes map[string]map[string]bool, method, path string) {
	t.Helper()
	if !routes[path][method] {
		t.Fatalf("expected route %s %s to be registered", method, path)
	}
}

func assertRouteMissing(t *testing.T, routes map[string]map[string]bool, method, path string) {
	t.Helper()
	if routes[path][method] {
		t.Fatalf("expected route %s %s to be removed", method, path)
	}
}

func assertAnonymousRouteAccessible(t *testing.T, engine *gin.Engine, method, path string) {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)
	code := readResponseCode(t, resp)
	if code == http.StatusUnauthorized {
		t.Fatalf("expected route %s %s to bypass auth, got business code %d", method, path, code)
	}
}

func assertAnonymousRouteBlocked(t *testing.T, engine *gin.Engine, method, path string) {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)
	code := readResponseCode(t, resp)
	if code != http.StatusUnauthorized {
		t.Fatalf("expected route %s %s to require auth, got business code %d", method, path, code)
	}
}

func readResponseCode(t *testing.T, resp *httptest.ResponseRecorder) int {
	t.Helper()
	var body struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	return body.Code
}
