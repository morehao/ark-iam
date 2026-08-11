package ctrtenant

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/tenantadmin/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/tenantadmin/internal/service/svctenant"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type OrganizationUserCtr interface {
	Create(ctx *gin.Context)
	Delete(ctx *gin.Context)
	PageList(ctx *gin.Context)
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

// @Tags 组织用户关联
// @Summary 创建组织用户关联
// @accept application/json
// @Produce application/json
// @Param req body dtotenant.OrganizationUserCreateReq true "创建组织用户关联"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.OrganizationUserCreateResp}
// @Router /v1/iam/organizationUser/create [post]
func (ctr *organizationUserCtr) Create(ctx *gin.Context) {
	var req dtotenant.OrganizationUserCreateReq
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

// @Tags 组织用户关联
// @Summary 删除组织用户关联
// @accept application/json
// @Produce application/json
// @Param req body dtotenant.OrganizationUserDeleteReq true "删除组织用户关联"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/organizationUser/delete [post]
func (ctr *organizationUserCtr) Delete(ctx *gin.Context) {
	var req dtotenant.OrganizationUserDeleteReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.organizationUserSvc.Delete(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}

// @Tags 组织用户关联
// @Summary 组织用户关联列表分页
// @accept application/json
// @Produce application/json
// @Param req body dtotenant.OrganizationUserPageListReq true "组织用户关联列表分页"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.OrganizationUserPageListResp}
// @Router /v1/iam/organizationUser/pageList [post]
func (ctr *organizationUserCtr) PageList(ctx *gin.Context) {
	var req dtotenant.OrganizationUserPageListReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
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