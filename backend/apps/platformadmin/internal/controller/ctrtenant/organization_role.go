package ctrtenant

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/platformadmin/internal/service/svctenant"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type OrganizationRoleCtr interface {
	Create(ctx *gin.Context)
	Delete(ctx *gin.Context)
	Update(ctx *gin.Context)
	Detail(ctx *gin.Context)
	PageList(ctx *gin.Context)
}

type organizationRoleCtr struct {
	organizationRoleSvc svctenant.OrganizationRoleSvc
}

var _ OrganizationRoleCtr = (*organizationRoleCtr)(nil)

func NewOrganizationRoleCtr() OrganizationRoleCtr {
	return &organizationRoleCtr{
		organizationRoleSvc: svctenant.NewOrganizationRoleSvc(),
	}
}

// @Tags 组织角色
// @Summary 创建组织角色
// @accept application/json
// @Produce application/json
// @Param req body dtotenant.OrganizationRoleCreateReq true "创建组织角色"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.OrganizationRoleCreateResp}
// @Router /v1/iam/organizationRole/create [post]
func (ctr *organizationRoleCtr) Create(ctx *gin.Context) {
	var req dtotenant.OrganizationRoleCreateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.organizationRoleSvc.Create(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 组织角色
// @Summary 删除组织角色
// @accept application/json
// @Produce application/json
// @Param req body dtotenant.OrganizationRoleDeleteReq true "删除组织角色"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/organizationRole/delete [post]
func (ctr *organizationRoleCtr) Delete(ctx *gin.Context) {
	var req dtotenant.OrganizationRoleDeleteReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.organizationRoleSvc.Delete(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}

// @Tags 组织角色
// @Summary 修改组织角色
// @accept application/json
// @Produce application/json
// @Param req body dtotenant.OrganizationRoleUpdateReq true "修改组织角色"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/organizationRole/update [post]
func (ctr *organizationRoleCtr) Update(ctx *gin.Context) {
	var req dtotenant.OrganizationRoleUpdateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.organizationRoleSvc.Update(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "修改成功")
}

// @Tags 组织角色
// @Summary 组织角色详情
// @accept application/json
// @Produce application/json
// @Param req query dtotenant.OrganizationRoleDetailReq true "组织角色详情"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.OrganizationRoleDetailResp}
// @Router /v1/iam/organizationRole/detail [get]
func (ctr *organizationRoleCtr) Detail(ctx *gin.Context) {
	var req dtotenant.OrganizationRoleDetailReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.organizationRoleSvc.Detail(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 组织角色
// @Summary 组织角色列表分页
// @accept application/json
// @Produce application/json
// @Param req body dtotenant.OrganizationRolePageListReq true "组织角色列表分页"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.OrganizationRolePageListResp}
// @Router /v1/iam/organizationRole/pageList [post]
func (ctr *organizationRoleCtr) PageList(ctx *gin.Context) {
	var req dtotenant.OrganizationRolePageListReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.organizationRoleSvc.PageList(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}