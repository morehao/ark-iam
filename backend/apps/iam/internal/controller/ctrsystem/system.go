package ctrsystem

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/dto/dtosystem"
	"github.com/morehao/ark-iam/iam/internal/service/svcsystem"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type SystemCtr interface {
	Create(ctx *gin.Context)
	Delete(ctx *gin.Context)
	Update(ctx *gin.Context)
	Detail(ctx *gin.Context)
	PageList(ctx *gin.Context)
}

type systemCtr struct {
	systemSvc svcsystem.SystemSvc
}

var _ SystemCtr = (*systemCtr)(nil)

func NewSystemCtr() SystemCtr {
	return &systemCtr{
		systemSvc: svcsystem.NewSystemSvc(),
	}
}

func (ctr *systemCtr) Create(ctx *gin.Context) {
	var req dtosystem.SystemCreateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.systemSvc.Create(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *systemCtr) Delete(ctx *gin.Context) {
	var req dtosystem.SystemDeleteReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.systemSvc.Delete(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}

func (ctr *systemCtr) Update(ctx *gin.Context) {
	var req dtosystem.SystemUpdateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.systemSvc.Update(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "修改成功")
}

func (ctr *systemCtr) Detail(ctx *gin.Context) {
	var req dtosystem.SystemDetailReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.systemSvc.Detail(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *systemCtr) PageList(ctx *gin.Context) {
	var req dtosystem.SystemPageListReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.systemSvc.PageList(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}