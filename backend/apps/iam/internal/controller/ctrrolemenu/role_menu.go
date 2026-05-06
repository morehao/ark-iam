package ctrrolemenu

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

// Create 角色菜单关系创建
// @Tags 角色菜单关系管理
// @Summary 创建角色菜单关系管理
// @accept application/json
// @Produce application/json
// @Param req body dtopermission.RoleMenuCreateReq true "创建角色菜单关系管理"
// @Success 200 {object} gincontext.DtoRender{data=dtopermission.RoleMenuCreateResp} "{"code": 0, "requestID": "xxx", "data": "ok", "msg": "success"}"
// @Router /v1/iam/permission/create [post]
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

// Delete 角色菜单关系删除
// @Tags 角色菜单关系管理
// @Summary 删除角色菜单关系管理
// @accept application/json
// @Produce application/json
// @Param req body dtopermission.RoleMenuDeleteReq true "删除角色菜单关系管理"
// @Success 200 {object} gincontext.DtoRender{data=string} "{"code": 0, "requestID": "xxx", "data": "ok", "msg": "success"}"
// @Router /v1/iam/permission/delete [post]
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

// PageList 角色菜单关系列表
// @Tags 角色菜单关系管理
// @Summary 角色菜单关系管理列表分页
// @accept application/json
// @Produce application/json
// @Param req body dtopermission.RoleMenuPageListReq true "角色菜单关系管理列表"
// @Success 200 {object} gincontext.DtoRender{data=dtopermission.RoleMenuPageListResp} "{"code": 0, "requestID": "xxx", "data": "ok", "msg": "success"}"
// @Router /v1/iam/permission/pageList [post]
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