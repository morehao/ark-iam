package ctrorganizationroleuserrelation

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/dto/dtoorganization"
	"github.com/morehao/ark-iam/iam/internal/service/svcorganizationroleuserrelation"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type OrganizationRoleUserRelationCtr interface {
	Create(ctx *gin.Context)
	Delete(ctx *gin.Context)
	PageList(ctx *gin.Context)
}

type organizationRoleUserRelationCtr struct {
	organizationRoleUserRelationSvc svcorganizationroleuserrelation.OrganizationRoleUserRelationSvc
}

var _ OrganizationRoleUserRelationCtr = (*organizationRoleUserRelationCtr)(nil)

func NewOrganizationRoleUserRelationCtr() OrganizationRoleUserRelationCtr {
	return &organizationRoleUserRelationCtr{
		organizationRoleUserRelationSvc: svcorganizationroleuserrelation.NewOrganizationRoleUserRelationSvc(),
	}
}

// Create 组织角色用户关系创建
// @Tags 组织角色用户关系管理
// @Summary 创建组织角色用户关系管理
// @accept application/json
// @Produce application/json
// @Param req body dtoorganization.OrganizationRoleUserRelationCreateReq true "创建组织角色用户关系管理"
// @Success 200 {object} gincontext.DtoRender{data=dtoorganization.OrganizationRoleUserRelationCreateResp} "{"code": 0, "requestID": "xxx", "data": "ok", "msg": "success"}"
// @Router /v1/iam/organizationroleuserrelation/create [post]
func (ctr *organizationRoleUserRelationCtr) Create(ctx *gin.Context) {
	var req dtoorganization.OrganizationRoleUserRelationCreateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.organizationRoleUserRelationSvc.Create(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// Delete 组织角色用户关系删除
// @Tags 组织角色用户关系管理
// @Summary 删除组织角色用户关系管理
// @accept application/json
// @Produce application/json
// @Param req body dtoorganization.OrganizationRoleUserRelationDeleteReq true "删除组织角色用户关系管理"
// @Success 200 {object} gincontext.DtoRender{data=string} "{"code": 0, "requestID": "xxx", "data": "ok", "msg": "success"}"
// @Router /v1/iam/organizationroleuserrelation/delete [post]
func (ctr *organizationRoleUserRelationCtr) Delete(ctx *gin.Context) {
	var req dtoorganization.OrganizationRoleUserRelationDeleteReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.organizationRoleUserRelationSvc.Delete(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}

// PageList 组织角色用户关系列表
// @Tags 组织角色用户关系管理
// @Summary 组织角色用户关系管理列表分页
// @accept application/json
// @Produce application/json
// @Param req body dtoorganization.OrganizationRoleUserRelationPageListReq true "组织角色用户关系管理列表"
// @Success 200 {object} gincontext.DtoRender{data=dtoorganization.OrganizationRoleUserRelationPageListResp} "{"code": 0, "requestID": "xxx", "data": "ok", "msg": "success"}"
// @Router /v1/iam/organizationroleuserrelation/pageList [post]
func (ctr *organizationRoleUserRelationCtr) PageList(ctx *gin.Context) {
	var req dtoorganization.OrganizationRoleUserRelationPageListReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.organizationRoleUserRelationSvc.PageList(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}