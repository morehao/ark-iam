package router

import (
	"github.com/gin-gonic/gin"

	"github.com/morehao/ark-iam/auth/internal/controller/ctrinstall"
	"github.com/morehao/ark-iam/auth/internal/middleware"
	"github.com/morehao/golib/biz/gmiddleware/ginmiddleware"
)

// installRouter 注册初始化引导路由。
//
// 路由规范定位：/install/* 是继 /oidc/*、back-channel logout 之后的**第三类例外（R3）**——
// 它既不是资源 CRUD 也不是业务动作，而是"系统尚未有租户与管理员"时的自举入口：
// 此刻连鉴权主体都不存在，无法挂任何业务前缀，也无法经过业务鉴权中间件。
//
// 为什么不挂在 /v1/auth/*：该前缀下所有路由都经过 pkgmiddleware.OIDCAuth 与未初始化守卫，
// 把唯二需要绕过这两者的端点塞进去，会让"哪些端点需要守卫"退化成逐条白名单（容易漏）。
// 独立前缀让守卫的放行清单只需一条 /install。
//
// 本方案不新增任何业务路由：菜单/权限的后续变动走既有的菜单管理与角色管理接口。
func installRouter(engine *gin.Engine, ctr *ctrinstall.InstallCtr) {
	g := engine.Group("/install")
	g.Use(ginmiddleware.CORS())
	// 状态查询：公开、只读、不含敏感信息。页面靠它决定渲染表单还是"已完成"页，
	// 部署脚本靠它做就绪判定。
	g.GET("/status", ctr.Status)
	// 唯一写入口。令牌校验在 controller 入口（见 ctrinstall.Initialize），
	// 限流在最外层——它必须在令牌校验之前生效，否则暴力猜测不受约束。
	g.POST("/initialize", middleware.BootstrapRateLimit(), ctr.Initialize)
}
