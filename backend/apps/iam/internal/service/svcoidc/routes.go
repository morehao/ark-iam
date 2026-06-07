package svcoidc

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zitadel/oidc/v3/pkg/oidc"
)

func RegisterProviderRoutes(routerGroup *gin.RouterGroup, provider *OIDCProvider, ssoSessionCookieName string) {
	var handler gin.HandlerFunc
	if provider != nil && provider.Provider != nil {
		handler = gin.WrapH(http.StripPrefix("/v1/iam/oidc", provider.Provider))
	} else {
		handler = func(ctx *gin.Context) {
			ctx.Status(http.StatusOK)
		}
	}
	routerGroup.GET(oidc.DiscoveryEndpoint, handler)
	routerGroup.Any("/authorize", handler)
	routerGroup.Any("/authorize/callback", handler)
	routerGroup.Any("/oauth/token", handler)
	routerGroup.Any("/oauth/introspect", handler)
	routerGroup.Any("/userinfo", handler)
	routerGroup.Any("/revoke", handler)
	routerGroup.Any("/end_session", func(ctx *gin.Context) {
		ctx.SetCookie(ssoSessionCookieName, "", -1, "/", "", false, true)
		handler(ctx)
	})
	routerGroup.Any("/keys", handler)
	routerGroup.Any("/healthz", handler)
	routerGroup.Any("/ready", handler)
}
