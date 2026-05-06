package ctrorganizationrole

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/dto/dtoorganization"
	"github.com/morehao/ark-iam/iam/internal/service/svcorganizationrole"
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
	organizationRoleSvc svcorganizationrole.OrganizationRoleSvc
}

var _ OrganizationRoleCtr = (*organizationRoleCtr)(nil)

func NewOrganizationRoleCtr() OrganizationRoleCtr {
	return &organizationRoleCtr{
		organizationRoleSvc: svcorganizationrole.NewOrganizationRoleSvc(),
	}
}

// Create 组织角色创建
// @Tags 组织角色管理
// @Summary 创建组织角色管理
// @accept application/json
// @Produce application/json
// @Param req body dtoorganization.OrganizationRoleCreateReq true "创建组织角色管理"
// @Success 200 {object} gincontext.DtoRender{data=dtoorganization.OrganizationRoleCreateResp} "{"code": 0, "requestID": "xxx", "data": "ok", "msg": "success"}"
// @Router /v1/iam/organizationrole/create [post]
func (ctr *organizationRoleCtr) Create(ctx *gin.Context) {
	var req dtoorganization.OrganizationRoleCreateReq
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

// Delete 组织角色删除
// @Tags 组织角色管理
// @Summary 删除组织角色管理
// @accept application/json
// @Produce application/json
// @Param req body dtoorganization.OrganizationRoleDeleteReq true "删除组织角色管理"
// @Success 200 {object} gincontext.DtoRender{data=string} "{"code": 0, "requestID": "xxx", "data": "ok", "msg": "success"}"
// @Router /v1/iam/organizationrole/delete [post]
func (ctr *organizationRoleCtr) Delete(ctx *gin.Context) {
	var req dtoorganization.OrganizationRoleDeleteReq
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

// Update 组织角色修改
// @Tags 组织角色管理
// @Summary 修改组织角色管理
// @accept application/json
// @Produce application/json
// @Param req body dtoorganization.OrganizationRoleUpdateReq true "修改组织角色管理"
// @Success 200 {object} gincontext.DtoRender{data=string} "{"code": 0, "requestID": "xxx", "data": "ok", "msg": "修改成功"}"
// @Router /v1/iam/organizationrole/update [post]
func (ctr *organizationRoleCtr) Update(ctx *gin.Context) {
	var req dtoorganization.OrganizationRoleUpdateReq
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

// Detail 组织角色详情
// @Tags 组织角色管理
// @Summary 组织角色管理详情
// @accept application/json
// @Produce application/json
// @Param req query dtoorganization.OrganizationRoleDetailReq true "组织角色管理详情"
// @Success 200 {object} gincontext.DtoRender{data=dtoorganization.OrganizationRoleDetailResp} "{"code": 0, "requestID": "xxx", "data": "ok", "msg": "success"}"
// @Router /v1/iam/organizationrole/detail [get]
func (ctr *organizationRoleCtr) Detail(ctx *gin.Context) {
	var req dtoorganization.OrganizationRoleDetailReq
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

// PageList 组织角色列表
// @Tags 组织角色管理
// @Summary 组织角色管理列表分页
// @accept application/json
// @Produce application/json
// @Param req body dtoorganization.OrganizationRolePageListReq true "组织角色管理列表"
// @Success 200 {object} gincontext.DtoRender{data=dtoorganization.OrganizationRolePageListResp} "{"code": 0, "requestID": "xxx", "data": "ok", "msg": "success"}"
// @Router /v1/iam/organizationrole/pageList [post]
func (ctr *organizationRoleCtr) PageList(ctx *gin.Context) {
	var req dtoorganization.OrganizationRolePageListReq
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