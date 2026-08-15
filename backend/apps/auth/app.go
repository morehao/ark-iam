package auth

import (
	"crypto/rsa"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/config"
	authRouter "github.com/morehao/ark-iam/auth/internal/router"
	pkgconfig "github.com/morehao/ark-iam/pkg/config"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/iam/sso"
	"github.com/morehao/ark-iam/pkg/middleware"
	"github.com/morehao/golib/biz/gserver/gindocs"
	"github.com/morehao/golib/biz/gserver/ginserver"
	"github.com/morehao/golib/glog"
)

const AppName = "auth"

func Init(engine *gin.Engine, Conf *pkgconfig.Config) {
	config.Conf = Conf

	ssoStore := sso.NewSSOSessionStore()
	oidcAuthOpts := []middleware.AuthOption{}
	if Conf != nil && Conf.OIDC.Issuer != "" {
		// H3：auth 作为 OP 自身，其 /v1/auth/* 接口接受任一合法 client 签发的 token，
		// 因此只校验 iss（必须是本 OP），aud 由各业务应用（RP）自行收紧。
		oidcAuthOpts = append(oidcAuthOpts, middleware.WithOIDCIssuer(Conf.OIDC.Issuer))
	}
	oidcAuthOpts = append(oidcAuthOpts,
		middleware.WithAuthSkipPaths(
			"/v1/auth/register",
			"/v1/auth/connector/callback",
		),
		middleware.WithOIDCSSOValidation(func(ctx *gin.Context, personID string, isMachineToken bool) bool {
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
				glog.Warnf(ctx, "[app.Init] HasActiveSession fail, personID:%s, err:%v", personID, err)
				return true
			}
			return active
		}),
	)
	routerGroups := ginserver.NewRouterGroups(engine, "auth", ginserver.VersionGroup{
		Version: ginserver.ApiVersionV1,
		Middlewares: []gin.HandlerFunc{
			middleware.OIDCCompatibleAuth(func() *rsa.PublicKey { return authRouter.OIDCPublicKey }, oidcAuthOpts...),
		},
	})

	if config.Conf.Server.Env == "dev" {
		gindocs.Register(engine.Group("/"+AppName), AppName)
	}

	authRouter.RegisterRouter(routerGroups, AppName)
	authRouter.InitOIDC(engine, routerGroups)
}
