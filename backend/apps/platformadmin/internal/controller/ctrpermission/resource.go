package ctrpermission

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtopermission"
	"github.com/morehao/ark-iam/platformadmin/internal/service/svcpermission"
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

// @Tags 资源管理
// @Summary 创建资源管理
// @accept application/json
// @Produce application/json
// @Param req body dtopermission.ResourceCreateReq true "创建资源管理"
// @Success 200 {object} gincontext.DtoRender{data=dtopermission.ResourceCreateResp}
// @Router /v1/platform/resources [post]
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

// @Tags 资源管理
// @Summary 删除资源管理
// @accept application/json
// @Produce application/json
// @Param resourceID path int true "resourceID"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/platform/resources/{resourceID} [delete]
func (ctr *resourceCtr) Delete(ctx *gin.Context) {
	var req dtopermission.ResourceDeleteReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.resourceSvc.Delete(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}

// @Tags 资源管理
// @Summary 修改资源管理
// @accept application/json
// @Produce application/json
// @Param resourceID path int true "resourceID"
// @Param req body dtopermission.ResourceUpdateReq true "修改资源管理"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/platform/resources/{resourceID} [put]
func (ctr *resourceCtr) Update(ctx *gin.Context) {
	var req dtopermission.ResourceUpdateReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
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

// @Tags 资源管理
// @Summary 资源管理详情
// @accept application/json
// @Produce application/json
// @Param resourceID path int true "resourceID"
// @Success 200 {object} gincontext.DtoRender{data=dtopermission.ResourceDetailResp}
// @Router /v1/platform/resources/{resourceID} [get]
func (ctr *resourceCtr) Detail(ctx *gin.Context) {
	var req dtopermission.ResourceDetailReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
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

// @Tags 资源管理
// @Summary 资源管理列表分页
// @accept application/json
// @Produce application/json
// @Param req query dtopermission.ResourcePageListReq true "资源管理列表分页"
// @Success 200 {object} gincontext.DtoRender{data=dtopermission.ResourcePageListResp}
// @Router /v1/platform/resources [get]
func (ctr *resourceCtr) PageList(ctx *gin.Context) {
	var req dtopermission.ResourcePageListReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
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
