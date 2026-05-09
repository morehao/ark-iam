package router

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/config"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func TestConnectorRoutesUseUnifiedEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config.Conf = &config.Config{JWT: config.JWT{SignKey: "test-sign-key"}}

	engine := gin.New()
	groups := ginserver.NewRouterGroups(engine, "iam", ginserver.Version{Name: gconstant.ApiVersionV1})
	RegisterRouter(groups, "iam")

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

	assertRouteMissing(t, paths, http.MethodGet, "/v1/iam/connector/factories")
	assertRouteMissing(t, paths, http.MethodPost, "/v1/iam/connector/:connectorId/authorization-url")
	assertRouteMissing(t, paths, http.MethodPost, "/v1/iam/ssoConnector/create")
	assertRouteMissing(t, paths, http.MethodGet, "/v1/iam/ssoConnector/providers")
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
