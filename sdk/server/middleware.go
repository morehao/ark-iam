package server

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/middleware"
	"github.com/morehao/ark-iam/pkg/oidckit"
	pkgconfig "github.com/morehao/ark-iam/pkg/config"
	"github.com/morehao/ark-iam/pkg/sso"
	"github.com/morehao/golib/biz/gmiddleware/ginmiddleware"
)

// authMiddlewareForGroup 构造某个内置控制台应用（platform/tenant）版本组的鉴权中间件：
// OIDCAuth（验签 + iss/aud 收紧 + 可选 SSO 会话活性校验）+ RequireAppAccess（应用准入门禁）。
//
// 与现网三 app 的中间件装配逐段等价，配置来源为完整 pkgconfig.Config。
func (a *App) authMiddlewareForGroup(conf *pkgconfig.Config, audience string) []gin.HandlerFunc {
	opts := []middleware.AuthOption{middleware.WithOIDCKeySource(a.keySource)}
	if conf != nil && conf.OIDC.Issuer != "" {
		opts = append(opts, middleware.WithOIDCIssuer(conf.OIDC.Issuer))
	}
	opts = append(opts, middleware.WithOIDCAudiences(audience))
	if conf != nil && conf.OIDC.EnableSSOSessionValidation {
		ssoStore := sso.NewSSOSessionStore()
		opts = append(opts, middleware.WithOIDCSSOValidation(func(ctx context.Context, personID string, isMachineToken bool) bool {
			if isMachineToken {
				return true
			}
			if dbclient.RedisCli == nil {
				return true
			}
			active, err := ssoStore.HasActiveSession(ctx, personID)
			if err != nil {
				return true
			}
			return active
		}))
	}
	return []gin.HandlerFunc{
		middleware.OIDCAuth(opts...),
		middleware.RequireAppAccess(),
	}
}

// registerBackChannelLogout 挂载 back-channel logout 接收端（/oidc/bc-logout/{platform|tenant}）。
func (a *App) registerBackChannelLogout(rg *gin.RouterGroup, conf *pkgconfig.Config, path, clientID string) {
	if conf == nil || a.keySource == nil {
		return
	}
	g := rg.Group("/oidc")
	g.Use(ginmiddleware.CORS())
	oidckit.RegisterReceiverRoutes(g, path, a.keySource, conf.OIDC.Issuer, clientID, nil)
}
