package router

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

// TestRegisterAllRoutes 全量注册平台管理路由，验证无重复/冲突路由。
// gin 在注册冲突路由时会 panic，测试失败即代表路由表冲突。
func TestRegisterAllRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	groups := ginserver.NewRouterGroups(engine, "platform", ginserver.VersionGroup{
		Version: ginserver.ApiVersionV1,
	})
	RegisterRouter(groups)
}
