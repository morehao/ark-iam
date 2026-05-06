package ctrtenantuserrelation

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/iam/internal/service/svctenant"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type OrganizationUserRelationCtr interface {
	Create(ctx *gin.Context)
	Delete(ctx *gin.Context)
	PageList(ctx *gin.Context)
}

type organizationUserRelationCtr struct {
	organizationUserRelationSvc svctenant.OrganizationUserRelationSvc
}

var _ OrganizationUserRelationCtr = (*organizationUserRelationCtr)(nil)

func NewOrganizationUserRelationCtr() OrganizationUserRelationCtr {
	return &organizationUserRelationCtr{
		organizationUserRelationSvc: svctenant.NewOrganizationUserRelationSvc(),
	}
}

// Create 组织用户关系创建
// @Tags 组织用户关系管理
// @Summary 创建组织用户关系管理
// @accept application/json
// @Produce application/json
// @Param req body dtotenant.OrganizationUserRelationCreateReq true "创建组织用户关系管理"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.OrganizationUserRelationCreateResp} "{"code": 0, "requestID": "xxx", "data": "ok", "msg": "success"}"
// @Router /v1/iam/organization/create [post]
func (ctr *organizationUserRelationCtr) Create(ctx *gin.Context) {
	var req dtotenant.OrganizationUserRelationCreateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.organizationUserRelationSvc.Create(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// Delete 组织用户关系删除
// @Tags 组织用户关系管理
// @Summary 删除组织用户关系管理
// @accept application/json
// @Produce application/json
// @Param req body dtotenant.OrganizationUserRelationDeleteReq true "删除组织用户关系管理"
// @Success 200 {object} gincontext.DtoRender{data=string} "{"code": 0, "requestID": "xxx", "data": "ok", "msg": "success"}"
// @Router /v1/iam/organization/delete [post]
func (ctr *organizationUserRelationCtr) Delete(ctx *gin.Context) {
	var req dtotenant.OrganizationUserRelationDeleteReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.organizationUserRelationSvc.Delete(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}

// PageList 组织用户关系列表
// @Tags 组织用户关系管理
// @Summary 组织用户关系管理列表分页
// @accept application/json
// @Produce application/json
// @Param req body dtotenant.OrganizationUserRelationPageListReq true "组织用户关系管理列表"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.OrganizationUserRelationPageListResp} "{"code": 0, "requestID": "xxx", "data": "ok", "msg": "success"}"
// @Router /v1/iam/organization/pageList [post]
func (ctr *organizationUserRelationCtr) PageList(ctx *gin.Context) {
	var req dtotenant.OrganizationUserRelationPageListReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.organizationUserRelationSvc.PageList(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}