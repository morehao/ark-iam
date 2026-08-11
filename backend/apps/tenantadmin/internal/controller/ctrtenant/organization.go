package ctrtenant

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/tenantadmin/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/tenantadmin/internal/service/svctenant"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type OrganizationCtr interface {
	Create(ctx *gin.Context)
	Delete(ctx *gin.Context)
	Update(ctx *gin.Context)
	Detail(ctx *gin.Context)
	PageList(ctx *gin.Context)
}

type organizationCtr struct {
	organizationSvc svctenant.OrganizationSvc
}

var _ OrganizationCtr = (*organizationCtr)(nil)

func NewOrganizationCtr() OrganizationCtr {
	return &organizationCtr{
		organizationSvc: svctenant.NewOrganizationSvc(),
	}
}

// @Tags 组织
// @Summary 创建组织
// @accept application/json
// @Produce application/json
// @Param req body dtotenant.OrganizationCreateReq true "创建组织"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.OrganizationCreateResp}
// @Router /v1/iam/organization/create [post]
func (ctr *organizationCtr) Create(ctx *gin.Context) {
	var req dtotenant.OrganizationCreateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.organizationSvc.Create(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 组织
// @Summary 删除组织
// @accept application/json
// @Produce application/json
// @Param req body dtotenant.OrganizationDeleteReq true "删除组织"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/organization/delete [post]
func (ctr *organizationCtr) Delete(ctx *gin.Context) {
	var req dtotenant.OrganizationDeleteReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.organizationSvc.Delete(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}

// @Tags 组织
// @Summary 修改组织
// @accept application/json
// @Produce application/json
// @Param req body dtotenant.OrganizationUpdateReq true "修改组织"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/organization/update [post]
func (ctr *organizationCtr) Update(ctx *gin.Context) {
	var req dtotenant.OrganizationUpdateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.organizationSvc.Update(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "修改成功")
}

// @Tags 组织
// @Summary 组织详情
// @accept application/json
// @Produce application/json
// @Param req query dtotenant.OrganizationDetailReq true "组织详情"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.OrganizationDetailResp}
// @Router /v1/iam/organization/detail [get]
func (ctr *organizationCtr) Detail(ctx *gin.Context) {
	var req dtotenant.OrganizationDetailReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.organizationSvc.Detail(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 组织
// @Summary 组织列表分页
// @accept application/json
// @Produce application/json
// @Param req body dtotenant.OrganizationPageListReq true "组织列表分页"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.OrganizationPageListResp}
// @Router /v1/iam/organization/pageList [post]
func (ctr *organizationCtr) PageList(ctx *gin.Context) {
	var req dtotenant.OrganizationPageListReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.organizationSvc.PageList(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}