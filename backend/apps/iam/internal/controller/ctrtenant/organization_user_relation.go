package ctrtenant

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