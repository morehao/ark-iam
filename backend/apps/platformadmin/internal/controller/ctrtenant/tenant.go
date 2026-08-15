package ctrtenant

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/gctx"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/platformadmin/internal/service/svctenant"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type TenantCtr interface {
	Create(ctx *gin.Context)
	CreateAsOwner(ctx *gin.Context)
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

// @Tags 租户管理
// @Summary 创建租户管理
// @accept application/json
// @Produce application/json
// @Param req body dtotenant.TenantCreateReq true "创建租户管理"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.TenantCreateResp}
// @Router /v1/platform/tenants [post]
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

// @Tags 租户管理
// @Summary 0租户自然人自助创建租户并成为租户 owner
// @accept application/json
// @Produce application/json
// @Param req body dtotenant.TenantCreateAsOwnerReq true "创建租户并成为owner"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.TenantCreateAsOwnerResp}
// @Router /v1/platform/tenants/createAsOwner [post]
func (ctr *tenantCtr) CreateAsOwner(ctx *gin.Context) {
	var req dtotenant.TenantCreateAsOwnerReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	req.PersonID = gctx.GetPersonID(ctx)
	res, err := ctr.tenantSvc.CreateTenantAsOwner(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 租户管理
// @Summary 删除租户管理
// @accept application/json
// @Produce application/json
// @Param tenantID path int true "tenantID"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/platform/tenants/{tenantID} [delete]
func (ctr *tenantCtr) Delete(ctx *gin.Context) {
	var req dtotenant.TenantDeleteReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.tenantSvc.Delete(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}

// @Tags 租户管理
// @Summary 修改租户管理
// @accept application/json
// @Produce application/json
// @Param req body dtotenant.TenantUpdateReq true "修改租户管理"
// @Param tenantID path int true "tenantID"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/platform/tenants/{tenantID} [put]
func (ctr *tenantCtr) Update(ctx *gin.Context) {
	var req dtotenant.TenantUpdateReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
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

// @Tags 租户管理
// @Summary 租户管理详情
// @accept application/json
// @Produce application/json
// @Param tenantID path int true "tenantID"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.TenantDetailResp}
// @Router /v1/platform/tenants/{tenantID} [get]
func (ctr *tenantCtr) Detail(ctx *gin.Context) {
	var req dtotenant.TenantDetailReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
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

// @Tags 租户管理
// @Summary 租户管理列表分页
// @accept application/json
// @Produce application/json
// @Param req query dtotenant.TenantPageListReq true "租户管理列表分页"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.TenantPageListResp}
// @Router /v1/platform/tenants [get]
func (ctr *tenantCtr) PageList(ctx *gin.Context) {
	var req dtotenant.TenantPageListReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
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
