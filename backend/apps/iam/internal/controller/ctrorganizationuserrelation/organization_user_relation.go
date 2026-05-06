package ctrorganizationuserrelation

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/dto/dtoorganization"
	"github.com/morehao/ark-iam/iam/internal/service/svcorganizationuserrelation"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type OrganizationUserRelationCtr interface {
	Create(ctx *gin.Context)
	Delete(ctx *gin.Context)
	PageList(ctx *gin.Context)
}

type organizationUserRelationCtr struct {
	organizationUserRelationSvc svcorganizationuserrelation.OrganizationUserRelationSvc
}

var _ OrganizationUserRelationCtr = (*organizationUserRelationCtr)(nil)

func NewOrganizationUserRelationCtr() OrganizationUserRelationCtr {
	return &organizationUserRelationCtr{
		organizationUserRelationSvc: svcorganizationuserrelation.NewOrganizationUserRelationSvc(),
	}
}

// Create 组织用户关系创建
// @Tags 组织用户关系管理
// @Summary 创建组织用户关系管理
// @accept application/json
// @Produce application/json
// @Param req body dtoorganization.OrganizationUserRelationCreateReq true "创建组织用户关系管理"
// @Success 200 {object} gincontext.DtoRender{data=dtoorganization.OrganizationUserRelationCreateResp} "{"code": 0, "requestID": "xxx", "data": "ok", "msg": "success"}"
// @Router /v1/iam/organizationuserrelation/create [post]
func (ctr *organizationUserRelationCtr) Create(ctx *gin.Context) {
	var req dtoorganization.OrganizationUserRelationCreateReq
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
// @Param req body dtoorganization.OrganizationUserRelationDeleteReq true "删除组织用户关系管理"
// @Success 200 {object} gincontext.DtoRender{data=string} "{"code": 0, "requestID": "xxx", "data": "ok", "msg": "success"}"
// @Router /v1/iam/organizationuserrelation/delete [post]
func (ctr *organizationUserRelationCtr) Delete(ctx *gin.Context) {
	var req dtoorganization.OrganizationUserRelationDeleteReq
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
// @Param req body dtoorganization.OrganizationUserRelationPageListReq true "组织用户关系管理列表"
// @Success 200 {object} gincontext.DtoRender{data=dtoorganization.OrganizationUserRelationPageListResp} "{"code": 0, "requestID": "xxx", "data": "ok", "msg": "success"}"
// @Router /v1/iam/organizationuserrelation/pageList [post]
func (ctr *organizationUserRelationCtr) PageList(ctx *gin.Context) {
	var req dtoorganization.OrganizationUserRelationPageListReq
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