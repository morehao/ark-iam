package router

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// TestRegisterAllRoutes 全量注册平台管理路由（经 Registry），验证无重复/冲突路由。
// gin 在注册冲突路由时会 panic，测试失败即代表路由表冲突。
func TestRegisterAllRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	rg := engine.Group("/v1/platform")
	db := func(context.Context) *gorm.DB { return nil }
	for _, fn := range Registry {
		fn(rg, db)
	}
}
