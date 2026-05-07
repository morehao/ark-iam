package ctrpermission

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/dto/dtopermission"
	"github.com/morehao/ark-iam/iam/internal/service/svcpermission"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type RoleMenuCtr interface {
	Create(ctx *gin.Context)
	Delete(ctx *gin.Context)
	PageList(ctx *gin.Context)
}

type roleMenuCtr struct {
	roleMenuSvc svcpermission.RoleMenuSvc
}

var _ RoleMenuCtr = (*roleMenuCtr)(nil)

func NewRoleMenuCtr() RoleMenuCtr {
	return &roleMenuCtr{
		roleMenuSvc: svcpermission.NewRoleMenuSvc(),
	}
}

func (ctr *roleMenuCtr) Create(ctx *gin.Context) {
	var req dtopermission.RoleMenuCreateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.roleMenuSvc.Create(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *roleMenuCtr) Delete(ctx *gin.Context) {
	var req dtopermission.RoleMenuDeleteReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.roleMenuSvc.Delete(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}

func (ctr *roleMenuCtr) PageList(ctx *gin.Context) {
	var req dtopermission.RoleMenuPageListReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.roleMenuSvc.PageList(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}