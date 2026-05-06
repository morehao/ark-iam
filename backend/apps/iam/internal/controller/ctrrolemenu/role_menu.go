package ctrrolemenu

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/dto/dtorole"
	"github.com/morehao/ark-iam/iam/internal/service/svcrolemenu"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type RoleMenuCtr interface {
	Create(ctx *gin.Context)
	Delete(ctx *gin.Context)
	PageList(ctx *gin.Context)
}

type roleMenuCtr struct {
	roleMenuSvc svcrolemenu.RoleMenuSvc
}

var _ RoleMenuCtr = (*roleMenuCtr)(nil)

func NewRoleMenuCtr() RoleMenuCtr {
	return &roleMenuCtr{
		roleMenuSvc: svcrolemenu.NewRoleMenuSvc(),
	}
}

// Create 角色菜单关系创建
// @Tags 角色菜单关系管理
// @Summary 创建角色菜单关系管理
// @accept application/json
// @Produce application/json
// @Param req body dtorole.RoleMenuCreateReq true "创建角色菜单关系管理"
// @Success 200 {object} gincontext.DtoRender{data=dtorole.RoleMenuCreateResp} "{"code": 0, "requestID": "xxx", "data": "ok", "msg": "success"}"
// @Router /v1/iam/rolemenu/create [post]
func (ctr *roleMenuCtr) Create(ctx *gin.Context) {
	var req dtorole.RoleMenuCreateReq
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
// @Param req body dtorole.RoleMenuDeleteReq true "删除角色菜单关系管理"
// @Success 200 {object} gincontext.DtoRender{data=string} "{"code": 0, "requestID": "xxx", "data": "ok", "msg": "success"}"
// @Router /v1/iam/rolemenu/delete [post]
func (ctr *roleMenuCtr) Delete(ctx *gin.Context) {
	var req dtorole.RoleMenuDeleteReq
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
// @Param req body dtorole.RoleMenuPageListReq true "角色菜单关系管理列表"
// @Success 200 {object} gincontext.DtoRender{data=dtorole.RoleMenuPageListResp} "{"code": 0, "requestID": "xxx", "data": "ok", "msg": "success"}"
// @Router /v1/iam/rolemenu/pageList [post]
func (ctr *roleMenuCtr) PageList(ctx *gin.Context) {
	var req dtorole.RoleMenuPageListReq
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