package ctrpermission

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtopermission"
	"github.com/morehao/ark-iam/platformadmin/internal/service/svcpermission"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type MenuCtr interface {
	Create(ctx *gin.Context)
	Delete(ctx *gin.Context)
	Update(ctx *gin.Context)
	Detail(ctx *gin.Context)
	PageList(ctx *gin.Context)
	Tree(ctx *gin.Context)
}

type menuCtr struct {
	menuSvc svcpermission.MenuSvc
}

var _ MenuCtr = (*menuCtr)(nil)

func NewMenuCtr() MenuCtr {
	return &menuCtr{
		menuSvc: svcpermission.NewMenuSvc(),
	}
}

// @Tags 菜单管理
// @Summary 创建菜单管理
// @accept application/json
// @Produce application/json
// @Param req body dtopermission.MenuCreateReq true "创建菜单管理"
// @Success 200 {object} gincontext.DtoRender{data=dtopermission.MenuCreateResp}
// @Router /v1/platform/menus [post]
func (ctr *menuCtr) Create(ctx *gin.Context) {
	var req dtopermission.MenuCreateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.menuSvc.Create(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 菜单管理
// @Summary 删除菜单管理
// @accept application/json
// @Produce application/json
// @Param menuID path int true "menuID"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/platform/menus/{menuID} [delete]
func (ctr *menuCtr) Delete(ctx *gin.Context) {
	var req dtopermission.MenuDeleteReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.menuSvc.Delete(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}

// @Tags 菜单管理
// @Summary 修改菜单管理
// @accept application/json
// @Produce application/json
// @Param menuID path int true "menuID"
// @Param req body dtopermission.MenuUpdateReq true "修改菜单管理"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/platform/menus/{menuID} [put]
func (ctr *menuCtr) Update(ctx *gin.Context) {
	var req dtopermission.MenuUpdateReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.menuSvc.Update(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "修改成功")
}

// @Tags 菜单管理
// @Summary 菜单管理详情
// @accept application/json
// @Produce application/json
// @Param menuID path int true "menuID"
// @Success 200 {object} gincontext.DtoRender{data=dtopermission.MenuDetailResp}
// @Router /v1/platform/menus/{menuID} [get]
func (ctr *menuCtr) Detail(ctx *gin.Context) {
	var req dtopermission.MenuDetailReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.menuSvc.Detail(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 菜单管理
// @Summary 菜单管理列表分页
// @accept application/json
// @Produce application/json
// @Param req query dtopermission.MenuPageListReq true "菜单管理列表分页"
// @Success 200 {object} gincontext.DtoRender{data=dtopermission.MenuPageListResp}
// @Router /v1/platform/menus [get]
func (ctr *menuCtr) PageList(ctx *gin.Context) {
	var req dtopermission.MenuPageListReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.menuSvc.PageList(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 菜单管理
// @Summary 菜单管理树
// @accept application/json
// @Produce application/json
// @Param req query dtopermission.MenuTreeReq true "菜单管理树"
// @Success 200 {object} gincontext.DtoRender{data=dtopermission.MenuTreeResp}
// @Router /v1/platform/menus/tree [get]
func (ctr *menuCtr) Tree(ctx *gin.Context) {
	var req dtopermission.MenuTreeReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.menuSvc.Tree(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}
