// Package auth 是 auth 域的实现库（可被 sdk/server import）。
//
// 它由原 iam/auth 的 DI 重构版迁入：域内 controller/service/core/dto/router/middleware
// 全部扁平平铺在本包之下（不再有 internal/），对外通过 Register 装配入口暴露。
//
// 依赖方向（分层倒置）：sdk/server（门面）→ apps/auth（本库）→ pkg。
// 本库**不得** import sdk（除无）。
package auth

import (
	"github.com/gin-gonic/gin"
	authrouter "github.com/morehao/ark-iam/backend/apps/auth/router"
	"github.com/morehao/ark-iam/pkg/oidckit"
	"github.com/morehao/golib/dbaccess/gormdao"
)

// AppName 是 auth 应用的名称（日志/健康检查/文档前缀）。
const AppName = "auth"

// Options 控制 auth 域挂载哪些部分（转承 auth/router.Options）。
type Options = authrouter.Options

// Register 装配 auth 域全部路由，返回本进程 OP 的进程内 KeySource（供 platform/tenant
// 域鉴权中间件验签注入）。
//
//	authRG          —— 业务路由组，通常为 /v1/auth
//	protocolEngine  —— /oidc、/install 等协议端点直接挂在其根
//	db              —— 数据访问器
//
// 注意：调用前必须已通过 auth/config.Set 注入运行时配置。
func Register(authRG *gin.RouterGroup, protocolEngine *gin.RouterGroup, db gormdao.DBGetter, opts Options) oidckit.KeySource {
	return authrouter.Register(authRG, protocolEngine, db, opts)
}
