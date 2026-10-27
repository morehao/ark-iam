package testutil

import (
	"context"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"

	"github.com/morehao/ark-iam/pkg/object/objidentity"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

// NewGinCtx 构造测试用 gin.Context，并按生产中间件的方式写入租户作用域：
//   - tenantID 非空 → 「当前租户」作用域；
//   - tenantID 为空 → 「全部租户」作用域（平台侧跨租户视角：按主键/全局唯一键先定位、
//     建租户等目标租户尚不存在的链路），禁止靠"未声明作用域"获得跨租户可见性。
//
// 测试库与生产库挂载同一份租户隔离插件，未声明作用域会被 fail-closed 拒绝。
//
// 注意：生产引擎**不再开启** engine.ContextWithFallback，service 只收 context.Context。
// 需要用 gin.Context 的地方仅限于 controller 级测试；调 service 时请用
// RequestCtx(ginCtx) 取纯 ctx，并用 Operator(ginCtx) 填 DTO 的 Operator 字段。
func NewGinCtx(tenantID, userID string) *gin.Context {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	ginCtx.Set(gcontext.KeyUserID, userID)
	if tenantID == "" {
		gincontext.SetTenantScope(ginCtx, gcontext.AllScope())
		return ginCtx
	}
	gincontext.SetTenantScope(ginCtx, gcontext.CurrentScope(tenantID))
	return ginCtx
}

// RequestCtx 返回 gin.Context 对应的纯请求上下文——与服务在 controller 里收到的
// 分毫不差（controller 的实现就是 ctx.Request.Context()）。
//
// 租户作用域随 Request.Context 透传（gincontext.SetTenantScope 写入的是请求上下文，
// 不依赖 ContextWithFallback），因此 service/dao 拿到的纯 ctx 能正常隔离。
func RequestCtx(ginCtx *gin.Context) context.Context {
	if ginCtx == nil || ginCtx.Request == nil {
		return context.Background()
	}
	return ginCtx.Request.Context()
}

// Operator 按生产 controller 的方式取操作人，供测试填入 DTO 的 Operator 字段。
func Operator(ginCtx *gin.Context) objidentity.Operator {
	if ginCtx == nil {
		return objidentity.Operator{}
	}
	op := objidentity.Operator{
		UserID:   ginCtx.GetString(gcontext.KeyUserID),
		PersonID: ginCtx.GetString(gcontext.KeyPersonID),
	}
	if ginCtx.Request != nil {
		op.UserAgent = ginCtx.Request.UserAgent()
	}
	return op
}

// NewRequestCtx 直接构造"纯 ctx + 操作人"，不经过 gin——这是身份走 DTO 给 service
// 单测带来的主要收益：不需要造 gin 引擎。
//
// tenantID 为空 → AllScope（跨租户视角，与 NewGinCtx 语义一致）。
func NewRequestCtx(tenantID, userID string) (context.Context, objidentity.Operator) {
	ctx := context.Background()
	if tenantID == "" {
		ctx = gcontext.WithTenantScope(ctx, gcontext.AllScope())
	} else {
		ctx = gcontext.WithTenantScope(ctx, gcontext.CurrentScope(tenantID))
	}
	return ctx, objidentity.Operator{UserID: userID}
}
