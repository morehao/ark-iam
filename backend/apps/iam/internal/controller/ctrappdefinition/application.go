package ctrappdefinition

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/dto/dtoappdefinition"
	"github.com/morehao/ark-iam/iam/internal/service/svcappdefinition"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type ApplicationCtr interface {
	Create(ctx *gin.Context)
	Update(ctx *gin.Context)
	Delete(ctx *gin.Context)
	Detail(ctx *gin.Context)
	PageList(ctx *gin.Context)
}

type applicationCtr struct {
	svc svcappdefinition.ApplicationSvc
}

var _ ApplicationCtr = (*applicationCtr)(nil)

func NewApplicationCtr() ApplicationCtr {
	return &applicationCtr{svc: svcappdefinition.NewApplicationSvc()}
}

func (ctr *applicationCtr) Create(ctx *gin.Context) {
	var req dtoappdefinition.CreateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.svc.Create(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *applicationCtr) Update(ctx *gin.Context) {
	var req dtoappdefinition.UpdateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.svc.Update(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "修改成功")
}

func (ctr *applicationCtr) Delete(ctx *gin.Context) {
	var req dtoappdefinition.DeleteReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.svc.Delete(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}

func (ctr *applicationCtr) Detail(ctx *gin.Context) {
	var req dtoappdefinition.DetailReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.svc.Detail(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *applicationCtr) PageList(ctx *gin.Context) {
	var req dtoappdefinition.PageListReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.svc.PageList(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}
