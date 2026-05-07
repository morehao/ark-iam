package ctrtenant

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/iam/internal/service/svctenant"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type OrganizationRoleUserRelationCtr interface {
	Create(ctx *gin.Context)
	Delete(ctx *gin.Context)
	PageList(ctx *gin.Context)
}

type organizationRoleUserRelationCtr struct {
	organizationRoleUserRelationSvc svctenant.OrganizationRoleUserRelationSvc
}

var _ OrganizationRoleUserRelationCtr = (*organizationRoleUserRelationCtr)(nil)

func NewOrganizationRoleUserRelationCtr() OrganizationRoleUserRelationCtr {
	return &organizationRoleUserRelationCtr{
		organizationRoleUserRelationSvc: svctenant.NewOrganizationRoleUserRelationSvc(),
	}
}

// @Tags 组织角色用户关联
// @Summary 创建组织角色用户关联
// @accept application/json
// @Produce application/json
// @Param req body dtotenant.OrganizationRoleUserRelationCreateReq true "创建组织角色用户关联"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.OrganizationRoleUserRelationCreateResp}
// @Router /v1/iam/organizationRoleUser/create [post]
func (ctr *organizationRoleUserRelationCtr) Create(ctx *gin.Context) {
	var req dtotenant.OrganizationRoleUserRelationCreateReq
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

// @Tags 组织角色用户关联
// @Summary 删除组织角色用户关联
// @accept application/json
// @Produce application/json
// @Param req body dtotenant.OrganizationRoleUserRelationDeleteReq true "删除组织角色用户关联"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/organizationRoleUser/delete [post]
func (ctr *organizationRoleUserRelationCtr) Delete(ctx *gin.Context) {
	var req dtotenant.OrganizationRoleUserRelationDeleteReq
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

// @Tags 组织角色用户关联
// @Summary 组织角色用户关联列表分页
// @accept application/json
// @Produce application/json
// @Param req body dtotenant.OrganizationRoleUserRelationPageListReq true "组织角色用户关联列表分页"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.OrganizationRoleUserRelationPageListResp}
// @Router /v1/iam/organizationRoleUser/pageList [post]
func (ctr *organizationRoleUserRelationCtr) PageList(ctx *gin.Context) {
	var req dtotenant.OrganizationRoleUserRelationPageListReq
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