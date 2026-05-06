package ctrorganization_role

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/dto/dtoorganization"
	"github.com/morehao/ark-iam/iam/internal/service/svcorganization_role"
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
	organizationRoleSvc svcorganization_role.OrganizationRoleSvc
}

var _ OrganizationRoleCtr = (*organizationRoleCtr)(nil)

func NewOrganizationRoleCtr() OrganizationRoleCtr {
	return &organizationRoleCtr{
		organizationRoleSvc: svcorganization_role.NewOrganizationRoleSvc(),
	}
}

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