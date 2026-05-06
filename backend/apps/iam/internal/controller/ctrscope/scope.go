package ctrscope

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/dto/dtorole"
	"github.com/morehao/ark-iam/iam/internal/service/svcscope"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type ScopeCtr interface {
	Create(ctx *gin.Context)
	Delete(ctx *gin.Context)
	Update(ctx *gin.Context)
	Detail(ctx *gin.Context)
	PageList(ctx *gin.Context)
}

type scopeCtr struct {
	scopeSvc svcscope.ScopeSvc
}

var _ ScopeCtr = (*scopeCtr)(nil)

func NewScopeCtr() ScopeCtr {
	return &scopeCtr{
		scopeSvc: svcscope.NewScopeSvc(),
	}
}

func (ctr *scopeCtr) Create(ctx *gin.Context) {
	var req dtorole.ScopeCreateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.scopeSvc.Create(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *scopeCtr) Delete(ctx *gin.Context) {
	var req dtorole.ScopeDeleteReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.scopeSvc.Delete(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}

func (ctr *scopeCtr) Update(ctx *gin.Context) {
	var req dtorole.ScopeUpdateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.scopeSvc.Update(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "修改成功")
}

func (ctr *scopeCtr) Detail(ctx *gin.Context) {
	var req dtorole.ScopeDetailReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.scopeSvc.Detail(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *scopeCtr) PageList(ctx *gin.Context) {
	var req dtorole.ScopePageListReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.scopeSvc.PageList(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}