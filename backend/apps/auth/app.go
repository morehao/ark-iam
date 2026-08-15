package auth

import (
	"crypto/rsa"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/config"
	authRouter "github.com/morehao/ark-iam/auth/internal/router"
	pkgconfig "github.com/morehao/ark-iam/pkg/config"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/iam/sso"
	"github.com/morehao/ark-iam/pkg/middleware/oidcauth"
	"github.com/morehao/ark-iam/pkg/token"
	"github.com/morehao/golib/biz/gmiddleware/ginmiddleware"
	"github.com/morehao/golib/biz/gserver/gindocs"
	"github.com/morehao/golib/biz/gserver/ginserver"
	"github.com/morehao/golib/glog"
)

const AppName = "auth"

func Init(engine *gin.Engine, Conf *pkgconfig.Config) {
	config.Conf = Conf

	ssoStore := sso.NewSSOSessionStore()
	routerGroups := ginserver.NewRouterGroups(engine, "auth", ginserver.VersionGroup{
		Version: ginserver.ApiVersionV1,
		Middlewares: []gin.HandlerFunc{
			oidcauth.OIDCCompatibleAuth(func() *rsa.PublicKey { return authRouter.OIDCPublicKey },
				oidcauth.WithAuthSkipPaths(
					"/v1/auth/register",
					"/v1/auth/connector/callback",
				),
				oidcauth.WithOIDCSSOValidation(func(ctx *gin.Context, personID uint, isMachineToken bool) bool {
					// 机器凭证（client_credentials/API Key）不依赖浏览器 SSO 会话活性，直接放行
					if isMachineToken {
						return true
					}
					// 无 Redis 时无法校验会话，采取放行（fail-open），避免破坏无 Redis 的环境
					if dbclient.RedisCli == nil {
						return true
					}
					active, err := ssoStore.HasActiveSession(ctx.Request.Context(), personID)
					if err != nil {
						glog.Warnf(ctx, "[app.Init] HasActiveSession fail, personID:%d, err:%v", personID, err)
						return true
					}
					return active
				})),
			ginmiddleware.TokenBlacklistCheck(dbclient.RedisCli,
				ginmiddleware.WithBlacklistKeyPrefix(token.TokenBlacklistKeyPrefix)),
		},
	})

	if config.Conf.Server.Env == "dev" {
		gindocs.Register(engine.Group("/"+AppName), AppName)
	}

	authRouter.RegisterRouter(routerGroups, AppName)
	authRouter.InitOIDC(engine, routerGroups)
}
