package ctrtenant

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/platformadmin/internal/service/svctenant"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type OrganizationRoleUserCtr interface {
	Create(ctx *gin.Context)
	Delete(ctx *gin.Context)
	PageList(ctx *gin.Context)
}

type organizationRoleUserCtr struct {
	organizationRoleUserSvc svctenant.OrganizationRoleUserSvc
}

var _ OrganizationRoleUserCtr = (*organizationRoleUserCtr)(nil)

func NewOrganizationRoleUserCtr() OrganizationRoleUserCtr {
	return &organizationRoleUserCtr{
		organizationRoleUserSvc: svctenant.NewOrganizationRoleUserSvc(),
	}
}

// @Tags 组织角色用户关联
// @Summary 创建组织角色用户关联
// @accept application/json
// @Produce application/json
// @Param req body dtotenant.OrganizationRoleUserCreateReq true "创建组织角色用户关联"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.OrganizationRoleUserCreateResp}
// @Router /v1/iam/organizationRoleUser/create [post]
func (ctr *organizationRoleUserCtr) Create(ctx *gin.Context) {
	var req dtotenant.OrganizationRoleUserCreateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.organizationRoleUserSvc.Create(ctx, &req)
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
// @Param req body dtotenant.OrganizationRoleUserDeleteReq true "删除组织角色用户关联"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/organizationRoleUser/delete [post]
func (ctr *organizationRoleUserCtr) Delete(ctx *gin.Context) {
	var req dtotenant.OrganizationRoleUserDeleteReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.organizationRoleUserSvc.Delete(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}

// @Tags 组织角色用户关联
// @Summary 组织角色用户关联列表分页
// @accept application/json
// @Produce application/json
// @Param req body dtotenant.OrganizationRoleUserPageListReq true "组织角色用户关联列表分页"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.OrganizationRoleUserPageListResp}
// @Router /v1/iam/organizationRoleUser/pageList [post]
func (ctr *organizationRoleUserCtr) PageList(ctx *gin.Context) {
	var req dtotenant.OrganizationRoleUserPageListReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.organizationRoleUserSvc.PageList(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}