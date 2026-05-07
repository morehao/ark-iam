package ctrpermission

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/dto/dtopermission"
	"github.com/morehao/ark-iam/iam/internal/service/svcpermission"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type UserRoleCtr interface {
	Create(ctx *gin.Context)
	Delete(ctx *gin.Context)
	PageList(ctx *gin.Context)
}

type userRoleCtr struct {
	userRoleSvc svcpermission.UserRoleSvc
}

var _ UserRoleCtr = (*userRoleCtr)(nil)

func NewUserRoleCtr() UserRoleCtr {
	return &userRoleCtr{
		userRoleSvc: svcpermission.NewUserRoleSvc(),
	}
}

func (ctr *userRoleCtr) Create(ctx *gin.Context) {
	var req dtopermission.UserRoleCreateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.userRoleSvc.Create(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *userRoleCtr) Delete(ctx *gin.Context) {
	var req dtopermission.UserRoleDeleteReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.userRoleSvc.Delete(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}

func (ctr *userRoleCtr) PageList(ctx *gin.Context) {
	var req dtopermission.UserRolePageListReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.userRoleSvc.PageList(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}