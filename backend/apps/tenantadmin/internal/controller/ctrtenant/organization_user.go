package ctrtenant

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/tenantadmin/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/tenantadmin/internal/service/svctenant"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type OrganizationUserCtr interface {
	Create(ctx *gin.Context)
	Update(ctx *gin.Context)
	Delete(ctx *gin.Context)
	PageList(ctx *gin.Context)
	UpdateUserOrganizations(ctx *gin.Context)
}

type organizationUserCtr struct {
	organizationUserSvc svctenant.OrganizationUserSvc
}

var _ OrganizationUserCtr = (*organizationUserCtr)(nil)

func NewOrganizationUserCtr() OrganizationUserCtr {
	return &organizationUserCtr{
		organizationUserSvc: svctenant.NewOrganizationUserSvc(),
	}
}

// @Tags 组织关系
// @Summary 添加组织关系（primary/secondary/leader）
// @accept application/json
// @Produce application/json
// @Param organizationID path string true "组织ID"
// @Param req body dtotenant.OrganizationUserCreateReq true "添加关系"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.OrganizationUserCreateResp}
// @Router /v1/tenant/organizations/{organizationID}/users [post]
func (ctr *organizationUserCtr) Create(ctx *gin.Context) {
	var req dtotenant.OrganizationUserCreateReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.organizationUserSvc.Create(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 组织关系
// @Summary 更新组织关系（relationType）
// @accept application/json
// @Produce application/json
// @Param organizationID path string true "组织ID"
// @Param userID path string true "用户ID"
// @Param req body dtotenant.OrganizationUserUpdateReq true "更新关系"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/tenant/organizations/{organizationID}/users/{userID} [put]
func (ctr *organizationUserCtr) Update(ctx *gin.Context) {
	var req dtotenant.OrganizationUserUpdateReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.organizationUserSvc.Update(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "修改成功")
}

// @Tags 组织关系
// @Summary 移除组织关系（含 primary/secondary/leader）
// @accept application/json
// @Produce application/json
// @Param organizationID path string true "组织ID"
// @Param userID path string true "用户ID"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/tenant/organizations/{organizationID}/users/{userID} [delete]
func (ctr *organizationUserCtr) Delete(ctx *gin.Context) {
	var req dtotenant.OrganizationUserDeleteReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.organizationUserSvc.Delete(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "移除成功")
}

// @Tags 组织关系
// @Summary 组织关系分页
// @accept application/json
// @Produce application/json
// @Param organizationID path string true "组织ID"
// @Param req query dtotenant.OrganizationUserPageListReq true "关系分页"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.OrganizationUserPageListResp}
// @Router /v1/tenant/organizations/{organizationID}/users [get]
func (ctr *organizationUserCtr) PageList(ctx *gin.Context) {
	var req dtotenant.OrganizationUserPageListReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.organizationUserSvc.PageList(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 组织关系
// @Summary 批量替换用户参与部门（全量替换 secondary 关系）
// @accept application/json
// @Produce application/json
// @Param userID path string true "用户ID"
// @Param req body dtotenant.UserOrganizationsUpdateReq true "批量替换参与部门"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/tenant/users/{userID}/organizations [put]
func (ctr *organizationUserCtr) UpdateUserOrganizations(ctx *gin.Context) {
	var req dtotenant.UserOrganizationsUpdateReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.organizationUserSvc.UpdateUserOrganizations(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "修改成功")
}
