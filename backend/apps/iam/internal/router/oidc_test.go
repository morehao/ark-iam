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

func TestOIDCRoutesExposeLoginEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config.Conf = &config.Config{
		JWT: config.JWT{SignKey: "test-sign-key"},
		OIDC: config.OIDC{
			Issuer:           "http://localhost:8099/v1/iam/oidc",
			FrontendLoginURL: "http://localhost:3000/oidc/login",
			AllowInsecure:    true,
		},
	}

	engine := gin.New()
	groups := ginserver.NewRouterGroups(engine, "iam", ginserver.Version{Name: gconstant.ApiVersionV1})
	router.InitOIDC(engine, groups)

	routes := engine.Routes()
	paths := make(map[string]map[string]bool, len(routes))
	for _, route := range routes {
		if _, ok := paths[route.Path]; !ok {
			paths[route.Path] = map[string]bool{}
		}
		paths[route.Path][route.Method] = true
	}

	assertRouteRegistered(t, paths, http.MethodPost, "/v1/iam/oidc/login")
	assertRouteMissing(t, paths, http.MethodPost, "/v1/iam/oidc/login/callback")
}

func TestOIDCLoginEndpointBypassesJWTAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config.Conf = &config.Config{
		JWT: config.JWT{SignKey: "test-sign-key"},
		OIDC: config.OIDC{
			Issuer:           "http://localhost:8099/v1/iam/oidc",
			FrontendLoginURL: "http://localhost:3000/oidc/login",
			AllowInsecure:    true,
		},
	}

	engine := gin.New()
	iam.Routers(engine)

	req := httptest.NewRequest(http.MethodPost, "/v1/iam/oidc/login", nil)
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
