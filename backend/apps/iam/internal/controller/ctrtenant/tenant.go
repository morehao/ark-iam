package ctrtenant

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/iam/internal/service/svctenant"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type TenantCtr interface {
	Create(ctx *gin.Context)
	Delete(ctx *gin.Context)
	Update(ctx *gin.Context)
	Detail(ctx *gin.Context)
	PageList(ctx *gin.Context)
}

type tenantCtr struct {
	tenantSvc svctenant.TenantSvc
}

var _ TenantCtr = (*tenantCtr)(nil)

func NewTenantCtr() TenantCtr {
	return &tenantCtr{
		tenantSvc: svctenant.NewTenantSvc(),
	}
}

// Create 创建租户管理
// @Tags 租户管理
// @Summary 创建租户管理
// @accept application/json
// @Produce application/json
// @Param req body dtotenant.TenantCreateReq true "创建租户管理"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.TenantCreateResp} "{"code": 0, "requestID": "xxx", "data": "ok", "msg": "success"}"
// @Router /v1/iam/tenant/create [post]
func (ctr *tenantCtr) Create(ctx *gin.Context) {
	var req dtotenant.TenantCreateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.tenantSvc.Create(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// Delete 删除租户管理
// @Tags 租户管理
// @Summary 删除租户管理
// @accept application/json
// @Produce application/json
// @Param req body dtotenant.TenantDeleteReq true "删除租户管理"
// @Success 200 {object} gincontext.DtoRender{data=string} "{"code": 0, "requestID": "xxx", "data": "ok", "msg": "success"}"
// @Router /v1/iam/tenant/delete [post]
func (ctr *tenantCtr) Delete(ctx *gin.Context) {
	var req dtotenant.TenantDeleteReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}

	if err := ctr.tenantSvc.Delete(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}

// Update 修改租户管理
// @Tags 租户管理
// @Summary 修改租户管理
// @accept application/json
// @Produce application/json
// @Param req body dtotenant.TenantUpdateReq true "修改租户管理"
// @Success 200 {object} gincontext.DtoRender{data=string} "{"code": 0, "requestID": "xxx", "data": "ok", "msg": "修改成功"}"
// @Router /v1/iam/tenant/update [post]
func (ctr *tenantCtr) Update(ctx *gin.Context) {
	var req dtotenant.TenantUpdateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.tenantSvc.Update(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "修改成功")
}

// Detail 租户管理详情
// @Tags 租户管理
// @Summary 租户管理详情
// @accept application/json
// @Produce application/json
// @Param req query dtotenant.TenantDetailReq true "租户管理详情"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.TenantDetailResp} "{"code": 0, "requestID": "xxx", "data": "ok", "msg": "success"}"
// @Router /v1/iam/tenant/detail [get]
func (ctr *tenantCtr) Detail(ctx *gin.Context) {
	var req dtotenant.TenantDetailReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.tenantSvc.Detail(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// PageList 租户管理列表
// @Tags 租户管理
// @Summary 租户管理列表分页
// @accept application/json
// @Produce application/json
// @Param req body dtotenant.TenantPageListReq true "租户管理列表"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.TenantPageListResp} "{"code": 0, "requestID": "xxx", "data": "ok", "msg": "success"}"
// @Router /v1/iam/tenant/pageList [post]
func (ctr *tenantCtr) PageList(ctx *gin.Context) {
	var req dtotenant.TenantPageListReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.tenantSvc.PageList(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}
