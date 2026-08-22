package ctrtenant

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/tenantadmin/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/tenantadmin/internal/service/svctenant"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type OrganizationCtr interface {
	Create(ctx *gin.Context)
	Tree(ctx *gin.Context)
	Children(ctx *gin.Context)
	Update(ctx *gin.Context)
	UpdateStatus(ctx *gin.Context)
	Delete(ctx *gin.Context)
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
// @Summary 创建组织节点
// @accept application/json
// @Produce application/json
// @Param req body dtotenant.OrganizationCreateReq true "创建组织"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.OrganizationCreateResp}
// @Router /v1/tenant/organizations [post]
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
// @Summary 组织树
// @accept application/json
// @Produce application/json
// @Param req query dtotenant.OrganizationTreeReq true "组织树查询"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.OrganizationTreeResp}
// @Router /v1/tenant/organizations/tree [get]
func (ctr *organizationCtr) Tree(ctx *gin.Context) {
	var req dtotenant.OrganizationTreeReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.organizationSvc.Tree(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 组织
// @Summary 某部门直属子部门分页列表
// @accept application/json
// @Produce application/json
// @Param organizationID path string true "部门ID(父节点)"
// @Param req query dtotenant.OrganizationChildrenReq true "子部门查询"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.OrganizationChildrenResp}
// @Router /v1/tenant/organizations/{organizationID}/children [get]
func (ctr *organizationCtr) Children(ctx *gin.Context) {
	var req dtotenant.OrganizationChildrenReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.organizationSvc.Children(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 组织
// @Summary 修改组织（改 parentID 即移动节点）
// @accept application/json
// @Produce application/json
// @Param organizationID path string true "组织ID"
// @Param req body dtotenant.OrganizationUpdateReq true "修改组织"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/tenant/organizations/{organizationID} [put]
func (ctr *organizationCtr) Update(ctx *gin.Context) {
	var req dtotenant.OrganizationUpdateReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
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
// @Summary 更新组织状态（启停用）
// @accept application/json
// @Produce application/json
// @Param organizationID path string true "组织ID"
// @Param req body dtotenant.OrganizationStatusReq true "更新状态"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/tenant/organizations/{organizationID} [patch]
func (ctr *organizationCtr) UpdateStatus(ctx *gin.Context) {
	var req dtotenant.OrganizationStatusReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.organizationSvc.UpdateStatus(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "修改成功")
}

// @Tags 组织
// @Summary 删除组织（有子节点/成员需 ?cascade=1）
// @accept application/json
// @Produce application/json
// @Param organizationID path string true "组织ID"
// @Param cascade query bool false "级联删除子树与成员"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/tenant/organizations/{organizationID} [delete]
func (ctr *organizationCtr) Delete(ctx *gin.Context) {
	var req dtotenant.OrganizationDeleteReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.organizationSvc.Delete(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}
