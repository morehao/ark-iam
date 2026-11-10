// Package auth 是 auth 域的实现库（可被 sdk/server import）。
//
// 它由原 iam/auth 的 DI 重构版迁入：域内 controller/service/core/dto/router/middleware
// 全部扁平平铺在本包之下（不再有 internal/），对外通过 Mount（稳定装配面）暴露。
//
// 依赖方向（分层倒置）：sdk/server/internal（门面）→ apps/auth（本库）→ pkg → contract。
// 本库**不得** import sdk；对 pkg 只依赖稳定面（本包内部再收敛 router/config/svcdb，
// 使 sdk/server/internal 只依赖 apps/auth 顶层包）。
package auth

import (
	"github.com/gin-gonic/gin"
	authconfig "github.com/morehao/ark-iam/backend/apps/auth/config"
	authrouter "github.com/morehao/ark-iam/backend/apps/auth/router"
	svcdb "github.com/morehao/ark-iam/backend/apps/auth/service/svcdb"
	pkgconfig "github.com/morehao/ark-iam/pkg/config"
	"github.com/morehao/ark-iam/pkg/oidckit"
	"github.com/morehao/golib/dbaccess/gormdao"
)

// AppName 是 auth 应用的名称（日志/健康检查/文档前缀）。
const AppName = "auth"

// Options 控制 auth 域挂载哪些部分（转承 auth/router.Options）。
type Options = authrouter.Options

// Mount 装配 auth 域全部路由（稳定装配面）：内部完成运行时配置注入（config.Set）、
// 数据访问器注入（svcdb.Set）与路由注册，返回本进程 OP 的进程内 KeySource
// （contract.KeySource，供 platform/tenant 域鉴权中间件验签注入）。
//
// sdk/server/internal 只需依赖本包顶层，不再直接 import apps/auth/{config,router,service/svcdb}。
//
//	authRG    —— 业务路由组，通常为 /v1/auth
//	engine    —— /oidc、/install 等协议端点直接挂在其根
//	db        —— 数据访问器
//	conf      —— 完整运行时配置（可为 nil，仅测试）
//	opts      —— 选择挂载哪些 auth 子域
func Mount(authRG *gin.RouterGroup, engine *gin.RouterGroup, db gormdao.DBGetter, conf *pkgconfig.Config, opts Options) oidckit.KeySource {
	authconfig.Set(conf)
	svcdb.Set(db)
	return authrouter.Register(authRG, engine, db, opts)
}

// Register 保留历史入口（等价 Mount，但不注入配置/svcdb；调用方需自行先 Set）。
//
// Deprecated: 请改用 Mount（稳定装配面，内部收敛配置/数据库注入）。
func Register(authRG *gin.RouterGroup, protocolEngine *gin.RouterGroup, db gormdao.DBGetter, opts Options) oidckit.KeySource {
	return authrouter.Register(authRG, protocolEngine, db, opts)
}

// Issuer 返回当前注入配置的 OP issuer（供上层构造验签器收紧 iss；未配置返回空）。
func Issuer() string {
	if authconfig.Conf != nil {
		return authconfig.Conf.OIDC.Issuer
	}
	return ""
}
