package platformadmin

import (
	"crypto/rsa"

	"github.com/gin-gonic/gin"
	pkgconfig "github.com/morehao/ark-iam/pkg/config"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/goidc"
	"github.com/morehao/ark-iam/pkg/iam/sso"
	"github.com/morehao/ark-iam/pkg/middleware"
	"github.com/morehao/ark-iam/platformadmin/config"
	"github.com/morehao/ark-iam/platformadmin/internal/router"
	"github.com/morehao/golib/biz/gmiddleware/ginmiddleware"
	"github.com/morehao/golib/biz/gserver/gindocs"
	"github.com/morehao/golib/biz/gserver/ginserver"
	"github.com/morehao/golib/glog"
)

const AppName = "platformadmin"

func Init(engine *gin.Engine, Conf *pkgconfig.Config) {
	config.Conf = Conf
	getOIDCPublicKey := middleware.LoadSigningPublicKey(Conf)
	ssoStore := sso.NewSSOSessionStore()

	oidcAuthOpts := []middleware.AuthOption{}
	if Conf != nil {
		// H3：校验 iss 与 aud——只接受本 OP 签发、且 aud 指向本应用 client 的 token，
		// 防止同一 OP 下其它 client 的 token 串用本应用接口。
		if Conf.OIDC.Issuer != "" {
			oidcAuthOpts = append(oidcAuthOpts, middleware.WithOIDCIssuer(Conf.OIDC.Issuer))
		}
		oidcAuthOpts = append(oidcAuthOpts, middleware.WithOIDCAudiences("platform-admin-web"))
	}
	if Conf != nil && Conf.OIDC.EnableSSOSessionValidation {
		// 请求粒度 SSO 会话活性校验：任一应用登出（撤销该 person 全部 SSO 会话）后，
		// 本应用的下一次请求即判 401，实现"一处登出、处处登出"的即时性。
		// 前提：本应用与 auth 共享同一认证 Redis。
		oidcAuthOpts = append(oidcAuthOpts,
			middleware.WithOIDCSSOValidation(func(ctx *gin.Context, personID string, isMachineToken bool) bool {
				if isMachineToken {
					return true
				}
				if dbclient.RedisCli == nil {
					return true // fail-open：无 Redis 时无法校验
				}
				active, err := ssoStore.HasActiveSession(ctx.Request.Context(), personID)
				if err != nil {
					glog.Warnf(ctx, "[platformadmin.Init] HasActiveSession fail, personID:%s, err:%v", personID, err)
					return true
				}
				return active
			}))
	}

	routerGroups := ginserver.NewRouterGroups(engine, "platform", []ginserver.VersionGroup{{
		Version: ginserver.ApiVersionV1,
		Middlewares: []gin.HandlerFunc{
			middleware.OIDCCompatibleAuth(getOIDCPublicKey, oidcAuthOpts...),
		},
	}})

	if config.Conf.Server.Env == "dev" {
		gindocs.Register(engine.Group("/"+AppName), AppName)
	}

	router.RegisterRouter(routerGroups)
	registerBackChannelLogout(engine, Conf, getOIDCPublicKey)
}

// registerBackChannelLogout 挂载本应用的 back-channel logout 接收端。
// 路径使用 app 专属子路径，避免 gateway 聚合部署时与其它应用路由冲突。
func registerBackChannelLogout(engine *gin.Engine, Conf *pkgconfig.Config, getOIDCPublicKey func() *rsa.PublicKey) {
	if Conf == nil {
		return
	}
	group := engine.Group("/oidc")
	group.Use(ginmiddleware.CORS())
	basePath := Conf.OIDC.BackChannelLogoutPath
	if basePath == "" {
		basePath = "/bc-logout/platform"
	}
	goidc.RegisterReceiverRoutes(group, basePath, getOIDCPublicKey, Conf.OIDC.Issuer, "platform-admin-web", nil)
}
