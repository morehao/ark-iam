package platformadmin

import (
	"context"

	"github.com/gin-gonic/gin"
	pkgconfig "github.com/morehao/ark-iam/pkg/config"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/middleware"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/oidckit"
	"github.com/morehao/ark-iam/pkg/sso"
	"github.com/morehao/ark-iam/platformadmin/config"
	"github.com/morehao/ark-iam/platformadmin/internal/router"
	"github.com/morehao/golib/biz/gmiddleware/ginmiddleware"
	"github.com/morehao/golib/biz/gserver/gindocs"
	"github.com/morehao/golib/biz/gserver/ginserver"
	"github.com/morehao/golib/glog"
)

const AppName = "platformadmin"

// Init 装配 platformadmin 应用。
//
// injectedKeySource 为 OP 公钥来源：gateway 单体部署由 auth 注入进程内 key set
// （零网络调用）；独立部署传 nil，此时按本应用配置自建（见 oidckit.ResolveKeySource）。
func Init(engine *gin.Engine, Conf *pkgconfig.Config, injectedKeySource oidckit.KeySource) {
	config.Conf = Conf
	keySource, keyErr := oidckit.ResolveKeySource(injectedKeySource, Conf)
	if keyErr != nil {
		// fail-closed：拿不到 OP 公钥就绝不放行任何 token（中间件会统一返回 401）。
		glog.Errorf(context.Background(), "[%s.Init] oidc key source unavailable, err:%v", AppName, keyErr)
	}
	ssoStore := sso.NewSSOSessionStore()

	oidcAuthOpts := []middleware.AuthOption{}
	if Conf != nil {
		// H3：校验 iss 与 aud——只接受本 OP 签发、且 aud 指向本应用 client 的 token，
		// 防止同一 OP 下其它 client 的 token 串用本应用接口。
		if Conf.OIDC.Issuer != "" {
			oidcAuthOpts = append(oidcAuthOpts, middleware.WithOIDCIssuer(Conf.OIDC.Issuer))
		}
		oidcAuthOpts = append(oidcAuthOpts, middleware.WithOIDCAudiences(model.SeedBuiltinClientPlatformAdminWeb))
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
				active, err := ssoStore.HasActiveSession(ctx, personID)
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
			middleware.OIDCAuth(append([]middleware.AuthOption{middleware.WithOIDCKeySource(keySource)}, oidcAuthOpts...)...),
		},
	}})

	if config.Conf.Server.Env == "dev" {
		gindocs.Register(engine.Group("/"+AppName), AppName)
	}

	router.RegisterRouter(routerGroups)
	registerBackChannelLogout(engine, Conf, keySource)
}

// registerBackChannelLogout 挂载本应用的 back-channel logout 接收端。
// 路径使用 app 专属子路径，避免 gateway 聚合部署时与其它应用路由冲突。
func registerBackChannelLogout(engine *gin.Engine, Conf *pkgconfig.Config, keySource oidckit.KeySource) {
	if Conf == nil {
		return
	}
	group := engine.Group("/oidc")
	group.Use(ginmiddleware.CORS())
	basePath := Conf.OIDC.BackChannelLogoutPath
	if basePath == "" {
		basePath = "/bc-logout/platform"
	}
	oidckit.RegisterReceiverRoutes(group, basePath, keySource, Conf.OIDC.Issuer, model.SeedBuiltinClientPlatformAdminWeb, nil)
}
