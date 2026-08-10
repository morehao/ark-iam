package ctrpermission

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtopermission"
	"github.com/morehao/ark-iam/platformadmin/internal/service/svcpermission"
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

// @Tags 角色菜单
// @Summary 创建角色菜单
// @accept application/json
// @Produce application/json
// @Param req body dtopermission.RoleMenuCreateReq true "创建角色菜单"
// @Success 200 {object} gincontext.DtoRender{data=dtopermission.RoleMenuCreateResp}
// @Router /v1/iam/roleMenu/create [post]
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

// @Tags 角色菜单
// @Summary 删除角色菜单
// @accept application/json
// @Produce application/json
// @Param req body dtopermission.RoleMenuDeleteReq true "删除角色菜单"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/roleMenu/delete [post]
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

// @Tags 角色菜单
// @Summary 角色菜单列表分页
// @accept application/json
// @Produce application/json
// @Param req body dtopermission.RoleMenuPageListReq true "角色菜单列表分页"
// @Success 200 {object} gincontext.DtoRender{data=dtopermission.RoleMenuPageListResp}
// @Router /v1/iam/roleMenu/pageList [post]
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
