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