package router

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/internal/controller/ctroidc"
)

// TestRegisterAllRoutes 全量注册认证网关路由（业务 v1 路由 + /oidc 协议路由），
// 验证无重复/冲突路由。gin 在注册冲突路由时会 panic，测试失败即代表路由表冲突。
// 使用轻量 OIDC 控制器（nil provider），避免测试依赖真实 provider 装配（Redis/密钥）。
func TestRegisterAllRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	registerRouter(engine, ctroidc.NewOIDCCtrWithProvider(nil))
}
