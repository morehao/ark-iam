package svcoidc

import (
	"net/http"

	"github.com/gin-gonic/gin"
	appconfig "github.com/morehao/ark-iam/iam/config"
	"github.com/morehao/ark-iam/iam/internal/middleware"
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
	ssoStore := NewSSOSessionStore()
	silentAuth := middleware.SilentSSORequired("iam_sso_session", middleware.WithSessionValidator(func(ctx *gin.Context, sessionID string) error {
		_, err := ssoStore.ValidateSession(ctx.Request.Context(), sessionID)
		return err
	}))
	routerGroup.Any("/authorize", silentAuth, handler)
	routerGroup.Any("/authorize/callback", handler)
	routerGroup.Any("/oauth/token", handler)
	routerGroup.Any("/oauth/introspect", handler)
	routerGroup.Any("/userinfo", handler)
	routerGroup.Any("/revoke", handler)
	routerGroup.Any("/end_session", func(ctx *gin.Context) {
		ctx.SetCookie(ssoSessionCookieName, "", -1, "/", appconfig.Conf.OIDC.SSOCookieDomain(), false, true)
		handler(ctx)
	})
	routerGroup.Any("/keys", handler)
	routerGroup.Any("/healthz", handler)
	routerGroup.Any("/ready", handler)
}
