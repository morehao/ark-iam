package router

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/internal/controller/ctroidc"
	"github.com/morehao/ark-iam/auth/internal/middleware"
	pkgmiddleware "github.com/morehao/ark-iam/pkg/middleware"
	"github.com/morehao/ark-iam/pkg/oidckit"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

// RegisterRouter 自装配 auth 全部路由：OIDC provider（签名密钥/logout worker/控制器）、
// 业务路由组与鉴权中间件、各业务模块路由、/oidc 协议路由。
//
// 返回本进程 OP 已发布公钥的进程内 KeySource：gateway 聚合部署时由它注入同进程的
// 其它应用（见 gateway.Init），使内置应用无需网络预取自己的 JWKS、也无需挂载 OP 私钥。
func RegisterRouter(engine *gin.Engine) oidckit.KeySource {
	oidcCtr := ctroidc.NewOIDCCtr()
	registerRouter(engine, oidcCtr)
	return oidcCtr.KeySource()
}

// registerRouter 注册全部路由；oidcCtr 由调用方提供（生产自装配，测试注入轻量实现）。
func registerRouter(engine *gin.Engine, oidcCtr *ctroidc.OIDCCtr) {
	// 业务路由组：auth 作为 OP 自身，验签公钥来自本进程 OP 已发布的 key set
	// （按 kid 取键，OP 多 key 轮换后本进程内立即生效，零网络调用）。
	authOpts := append([]pkgmiddleware.AuthOption{
		pkgmiddleware.WithOIDCKeySource(oidcCtr.KeySource()),
	}, middleware.OIDCBusinessAuthOptions()...)
	routerGroups := ginserver.NewRouterGroups(engine, "auth", []ginserver.VersionGroup{{
		Version: ginserver.ApiVersionV1,
		Middlewares: []gin.HandlerFunc{
			pkgmiddleware.OIDCAuth(authOpts...),
		},
	}})

	authRouter(routerGroups)
	personRouter(routerGroups)
	userSessionRouter(routerGroups)
	connectorRouter(routerGroups)
	InitOIDC(engine, oidcCtr)
}
