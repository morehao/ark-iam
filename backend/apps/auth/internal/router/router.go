package router

import (
	"crypto/rsa"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/internal/controller/ctroidc"
	"github.com/morehao/ark-iam/auth/internal/middleware"
	pkgmiddleware "github.com/morehao/ark-iam/pkg/middleware"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

// RegisterRouter 自装配 auth 全部路由：OIDC provider（签名密钥/logout worker/控制器）、
// 业务路由组与鉴权中间件、各业务模块路由、/oidc 协议路由。
func RegisterRouter(engine *gin.Engine) {
	oidcCtr := ctroidc.NewOIDCCtr()
	registerRouter(engine, oidcCtr)
}

// registerRouter 注册全部路由；oidcCtr 由调用方提供（生产自装配，测试注入轻量实现）。
func registerRouter(engine *gin.Engine, oidcCtr *ctroidc.OIDCCtr) {
	// 业务路由组：auth 作为 OP 自身，鉴权中间件公钥来自本进程 OP 的运行时签名密钥
	routerGroups := ginserver.NewRouterGroups(engine, "auth", []ginserver.VersionGroup{{
		Version: ginserver.ApiVersionV1,
		Middlewares: []gin.HandlerFunc{
			pkgmiddleware.OIDCCompatibleAuth(func() *rsa.PublicKey { return oidcCtr.PublicKey() }, middleware.OIDCBusinessAuthOptions()...),
		},
	}})

	authRouter(routerGroups)
	personRouter(routerGroups)
	userSessionRouter(routerGroups)
	connectorRouter(routerGroups)
	InitOIDC(engine, oidcCtr)
}
