package router

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/controller/ctroidc"
	"gorm.io/gorm"
)

// TestRegisterAllRoutes 全量注册认证网关路由（业务 v1 路由 + /oidc 协议路由），
// 验证无重复/冲突路由。使用轻量 OIDC 控制器（nil provider），避免真实 provider 装配。
func TestRegisterAllRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	db := func(context.Context) *gorm.DB { return nil }
	authRG := engine.Group("/v1/auth")
	registerRouter(authRG, engine.Group("/"), db, ctroidc.NewOIDCCtrWithProvider(db, nil),
		Options{CoreAuth: true, Install: true, Connector: true})
}
