package svcpermission

import (
	"context"

	"github.com/gin-gonic/gin"

	"github.com/morehao/ark-iam/pkg/object/objidentity"
	"github.com/morehao/golib/biz/gcontext"
)

// requestCtx 把测试用的 gin 夹具转成 service 现在接收的纯 context.Context，
// 与生产 controller 的实现完全一致（ctx.Request.Context()）。
func requestCtx(ginCtx *gin.Context) context.Context {
	if ginCtx == nil || ginCtx.Request == nil {
		return context.Background()
	}
	return ginCtx.Request.Context()
}

// operatorOf 从 gin 夹具取操作人，填进 DTO 的 Operator 字段（等价于 controller 的
// identity.OperatorFromGin）。
func operatorOf(ginCtx *gin.Context) objidentity.Operator {
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
