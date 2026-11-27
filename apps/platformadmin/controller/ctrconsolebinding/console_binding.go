package ctrconsolebinding

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/apps/platformadmin/dto/dtoconsolebinding"
	"github.com/morehao/ark-iam/apps/platformadmin/service/svcconsolebinding"
	"github.com/morehao/ark-iam/pkg/identity"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/dbaccess/gormdao"
)

type ConsoleBindingCtr interface {
	List(ctx *gin.Context)
	Enable(ctx *gin.Context)
	Disable(ctx *gin.Context)
}

type consoleBindingCtr struct {
	svc svcconsolebinding.ConsoleBindingSvc
}

var _ ConsoleBindingCtr = (*consoleBindingCtr)(nil)

func NewConsoleBindingCtr(db gormdao.DBGetter) ConsoleBindingCtr {
	return &consoleBindingCtr{svc: svcconsolebinding.NewConsoleBindingSvc(db)}
}

// @Tags 控制台绑定
// @Summary 查看 Host binding 列表
// @Router /v1/platform/console-bindings [get]
func (ctr *consoleBindingCtr) List(ctx *gin.Context) {
	var req dtoconsolebinding.ConsoleBindingListReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.svc.List(ctx.Request.Context(), &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 控制台绑定
// @Summary 启用 Host binding
// @Router /v1/platform/console-bindings/{bindingID}/enable [post]
func (ctr *consoleBindingCtr) Enable(ctx *gin.Context) {
	var req dtoconsolebinding.ConsoleBindingEnableReq
	req.Operator = identity.OperatorFromGin(ctx)
	if err := ctx.ShouldBindUri(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.svc.Enable(ctx.Request.Context(), &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "启用成功")
}

// @Tags 控制台绑定
// @Summary 停用 Host binding
// @Router /v1/platform/console-bindings/{bindingID}/disable [post]
func (ctr *consoleBindingCtr) Disable(ctx *gin.Context) {
	var req dtoconsolebinding.ConsoleBindingDisableReq
	req.Operator = identity.OperatorFromGin(ctx)
	if err := ctx.ShouldBindUri(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.svc.Disable(ctx.Request.Context(), &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "停用成功")
}
