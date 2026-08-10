package iam

import (
	"crypto/rsa"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/config"
	_ "github.com/morehao/ark-iam/iam/docs"
	"github.com/morehao/ark-iam/iam/internal/router"
	"github.com/morehao/ark-iam/iam/internal/service/svcsso"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/middleware/oidcauth"
	"github.com/morehao/golib/biz/gmiddleware/ginmiddleware"
	"github.com/morehao/golib/biz/gserver/gindocs"
	"github.com/morehao/golib/biz/gserver/ginserver"
	"github.com/morehao/golib/glog"
)

const AppName = "iam"

func Routers(engine *gin.Engine) {
	ssoStore := svcsso.NewSSOSessionStore()
	routerGroups := ginserver.NewRouterGroups(engine, AppName, ginserver.VersionGroup{
		Version: ginserver.ApiVersionV1,
		Middlewares: []gin.HandlerFunc{
			oidcauth.OIDCCompatibleAuth(func() *rsa.PublicKey { return router.OIDCPublicKey }, oidcauth.WithAuthSkipPaths(
				"/v1/iam/org/getConfigsByDomain",
				"/v1/iam/auth/register",
				"/v1/iam/connector/callback",
			), oidcauth.WithOIDCSSOValidation(func(ctx *gin.Context, personID uint, isMachineToken bool) bool {
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
					glog.Warnf(ctx, "[app.Routers] HasActiveSession fail, personID:%d, err:%v", personID, err)
					return true
				}
				return active
			})),
			ginmiddleware.TokenBlacklistCheck(dbclient.RedisCli, ginmiddleware.WithBlacklistKeyPrefix("iam:token:blacklist:")),
		},
	})

	if config.Conf.Server.Env == "dev" {
		gindocs.Register(engine.Group("/"+AppName), AppName)
	}

	router.RegisterRouter(routerGroups, AppName)
	router.InitOIDC(engine, routerGroups)
}
