package testutil

import (
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"

	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

// NewGinCtx 构造测试用 gin.Context，并按生产中间件的方式写入租户作用域：
//   - tenantID 非空 → 「当前租户」作用域；
//   - tenantID 为空 → 「全部租户」作用域（平台侧跨租户视角：按主键/全局唯一键先定位、
//     建租户等目标租户尚不存在的链路），禁止靠"未声明作用域"获得跨租户可见性。
//
// 测试库与生产库挂载同一份租户隔离插件，未声明作用域会被 fail-closed 拒绝；
// 且 gin 引擎需开启 ContextWithFallback，类型化作用域才能经 Request.Context 透传。
func NewGinCtx(tenantID, userID string) *gin.Context {
	ginCtx, engine := gin.CreateTestContext(nil)
	engine.ContextWithFallback = true
	ginCtx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	ginCtx.Set(gcontext.KeyUserID, userID)
	if tenantID == "" {
		gincontext.SetTenantScope(ginCtx, gcontext.AllScope())
		return ginCtx
	}
	gincontext.SetTenantScope(ginCtx, gcontext.CurrentScope(tenantID))
	return ginCtx
}
