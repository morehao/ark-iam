package ctrpermission

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/dto/dtopermission"
	"github.com/morehao/ark-iam/iam/internal/service/svcpermission"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type ResourceCtr interface {
	Create(ctx *gin.Context)
	Delete(ctx *gin.Context)
	Update(ctx *gin.Context)
	Detail(ctx *gin.Context)
	PageList(ctx *gin.Context)
}

type resourceCtr struct {
	resourceSvc svcpermission.ResourceSvc
}

var _ ResourceCtr = (*resourceCtr)(nil)

func NewResourceCtr() ResourceCtr {
	return &resourceCtr{
		resourceSvc: svcpermission.NewResourceSvc(),
	}
}

func (ctr *resourceCtr) Create(ctx *gin.Context) {
	var req dtopermission.ResourceCreateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.resourceSvc.Create(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *resourceCtr) Delete(ctx *gin.Context) {
	var req dtopermission.ResourceDeleteReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.resourceSvc.Delete(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}

func (ctr *resourceCtr) Update(ctx *gin.Context) {
	var req dtopermission.ResourceUpdateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.resourceSvc.Update(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "修改成功")
}

func (ctr *resourceCtr) Detail(ctx *gin.Context) {
	var req dtopermission.ResourceDetailReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.resourceSvc.Detail(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *resourceCtr) PageList(ctx *gin.Context) {
	var req dtopermission.ResourcePageListReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.resourceSvc.PageList(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}