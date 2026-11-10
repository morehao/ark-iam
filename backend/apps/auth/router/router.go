package router

import (
	"github.com/gin-gonic/gin"
	authconfig "github.com/morehao/ark-iam/backend/apps/auth/config"
	authmiddleware "github.com/morehao/ark-iam/backend/apps/auth/middleware"
	"github.com/morehao/ark-iam/backend/apps/auth/controller/ctroidc"
	"github.com/morehao/ark-iam/backend/apps/auth/controller/ctrinstall"
	pkgmiddleware "github.com/morehao/ark-iam/pkg/middleware"
	"github.com/morehao/ark-iam/pkg/oidckit"
	"github.com/morehao/golib/dbaccess/gormdao"
)

// Options 控制 auth 域挂载哪些部分。
type Options struct {
	// CoreAuth 挂业务路由（me/logout/sessions/userinfo 等）+ /oidc 协议。默认 true。
	CoreAuth bool
	// Install 挂 /install 初始化引导。默认 true（空库自举必需）。
	Install bool
	// Connector 挂 connector 身份联合路由（企业级，可关）。默认 false（瘦版）。
	Connector bool
	// SelfService 挂多租户自助（joinTenant/me-tenants/registerPerson/createTenant，可关）。
	SelfService bool
}

// Register 装配 auth 域全部路由，返回本进程 OP 的进程内 KeySource（供 platform/tenant
// 域中间件验签注入）。
//
//	authRG          —— 业务路由组，通常为 /v1/auth（本函数在其下注册 /me、/logout 等）
//	protocolEngine  —— /oidc、/install 等协议端点直接挂在 engine 根
//	db              —— 数据访问器
//
// 注意：调用前必须已通过 iam/auth/config.Set 注入配置；OP provider 由 NewOIDCCtr 自装配。
func Register(authRG *gin.RouterGroup, protocolEngine *gin.RouterGroup, db gormdao.DBGetter, opts Options) oidckit.KeySource {
	oidcCtr := ctroidc.NewOIDCCtr(db)
	// 业务路由组 /v1/auth/* 必须经过 OIDC 鉴权中间件（验签 + 身份注入 + iss/SSO 会话校验），
	// 否则 /me、/me/sessions 等取不到 Operator，会误报「用户不存在」。
	// 这里在注册业务路由前对该组挂中间件，使其在 gateway 聚合部署与 auth 独立部署下行为一致
	// （原 iam 版把该职责留在调用方易漏挂；统一收敛到域库 Register 内）。
	authOpts := append([]pkgmiddleware.AuthOption{
		pkgmiddleware.WithOIDCKeySource(oidcCtr.KeySource()),
		pkgmiddleware.WithVerifier(oidcCtr.Verifier(authIssuer())),
	}, authmiddleware.OIDCBusinessAuthOptions()...)
	authRG.Use(pkgmiddleware.OIDCAuth(authOpts...))
	registerRouter(authRG, protocolEngine, db, oidcCtr, opts)
	return oidcCtr.KeySource()
}

func registerRouter(authRG *gin.RouterGroup, protocolEngine *gin.RouterGroup, db gormdao.DBGetter, oidcCtr *ctroidc.OIDCCtr, opts Options) {
	if opts.CoreAuth {
		authRouter(authRG, db)
		personRouter(authRG, db)
		userSessionRouter(authRG, db)
		if opts.Connector {
			connectorRouter(authRG, db)
		}
		InitOIDC(protocolEngine, oidcCtr)
	}
	if opts.Install {
		installRouter(protocolEngine, ctrinstall.NewInstallCtr(db))
	}
}

// authIssuer 返回本 OP 的 issuer（供构造 access token 校验器收紧 iss；未配置时为空）。
func authIssuer() string {
	if authconfig.Conf != nil {
		return authconfig.Conf.OIDC.Issuer
	}
	return ""
}
