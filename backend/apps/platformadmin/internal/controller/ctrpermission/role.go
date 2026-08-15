package ctrpermission

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtopermission"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtouser"
	"github.com/morehao/ark-iam/platformadmin/internal/service/svcpermission"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

// RoleCtr 平台排查视角：角色只读查看（列表/详情/成员）。
type RoleCtr interface {
	Detail(ctx *gin.Context)
	PageList(ctx *gin.Context)
	ListUsers(ctx *gin.Context)
}

type roleCtr struct {
	roleSvc svcpermission.RoleSvc
}

var _ RoleCtr = (*roleCtr)(nil)

func NewRoleCtr() RoleCtr {
	return &roleCtr{
		roleSvc: svcpermission.NewRoleSvc(),
	}
}

// @Tags 角色管理
// @Summary 角色管理详情
// @accept application/json
// @Produce application/json
// @Param roleID path int true "角色ID"
// @Success 200 {object} gincontext.DtoRender{data=dtopermission.RoleDetailResp}
// @Router /v1/platform/roles/{roleID} [get]
func (ctr *roleCtr) Detail(ctx *gin.Context) {
	var req dtopermission.RoleDetailReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.roleSvc.Detail(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 角色管理
// @Summary 角色管理列表分页
// @accept application/json
// @Produce application/json
// @Param req query dtopermission.RolePageListReq true "角色管理列表分页"
// @Success 200 {object} gincontext.DtoRender{data=dtopermission.RolePageListResp}
// @Router /v1/platform/roles [get]
func (ctr *roleCtr) PageList(ctx *gin.Context) {
	var req dtopermission.RolePageListReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.roleSvc.PageList(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 角色管理
// @Summary 角色用户列表
// @accept application/json
// @Produce application/json
// @Param roleID path int true "角色ID"
// @Success 200 {object} gincontext.DtoRender{data=dtouser.RoleUserListResp}
// @Router /v1/platform/roles/{roleID}/users [get]
func (ctr *roleCtr) ListUsers(ctx *gin.Context) {
	var req dtouser.RoleUserListReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.roleSvc.ListUsers(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}
