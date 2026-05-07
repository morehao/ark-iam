package ctrpermission

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/dto/dtopermission"
	"github.com/morehao/ark-iam/iam/internal/service/svcpermission"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type RoleScopeCtr interface {
	Create(ctx *gin.Context)
	Delete(ctx *gin.Context)
	PageList(ctx *gin.Context)
}

type roleScopeCtr struct {
	roleScopeSvc svcpermission.RoleScopeSvc
}

var _ RoleScopeCtr = (*roleScopeCtr)(nil)

func NewRoleScopeCtr() RoleScopeCtr {
	return &roleScopeCtr{
		roleScopeSvc: svcpermission.NewRoleScopeSvc(),
	}
}

func (ctr *roleScopeCtr) Create(ctx *gin.Context) {
	var req dtopermission.RoleScopeCreateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.roleScopeSvc.Create(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *roleScopeCtr) Delete(ctx *gin.Context) {
	var req dtopermission.RoleScopeDeleteReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.roleScopeSvc.Delete(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}

func (ctr *roleScopeCtr) PageList(ctx *gin.Context) {
	var req dtopermission.RoleScopePageListReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.roleScopeSvc.PageList(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}