package ctrtenant

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/platformadmin/internal/service/svctenant"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type OrganizationCtr interface {
	Tree(ctx *gin.Context)
	GetUserOrganizations(ctx *gin.Context)
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
// @Summary 组织树（平台只读）
// @accept application/json
// @Produce application/json
// @Param req query dtotenant.OrganizationTreeReq true "组织树查询"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.OrganizationTreeResp}
// @Router /v1/platform/organizations/tree [get]
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
// @Summary 用户组织归属（平台只读）
// @accept application/json
// @Produce application/json
// @Param userID path string true "用户ID"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.UserOrganizationListResp}
// @Router /v1/platform/users/{userID}/organizations [get]
func (ctr *organizationCtr) GetUserOrganizations(ctx *gin.Context) {
	var req dtotenant.UserOrganizationListReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.organizationSvc.GetUserOrganizations(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}
