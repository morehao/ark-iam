package ctrpermission

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/apps/platformadmin/dto/dtopermission"
	"github.com/morehao/ark-iam/apps/platformadmin/service/svcpermission"
	"github.com/morehao/ark-iam/pkg/identity"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/dbaccess/gormdao"
)

type MenuCtr interface {
	Detail(ctx *gin.Context)
	PageList(ctx *gin.Context)
	Tree(ctx *gin.Context)
	MyTree(ctx *gin.Context)
}

type menuCtr struct {
	menuSvc svcpermission.MenuSvc
}

var _ MenuCtr = (*menuCtr)(nil)

func NewMenuCtr(db gormdao.DBGetter) MenuCtr {
	return &menuCtr{
		menuSvc: svcpermission.NewMenuSvc(db),
	}
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
	res, err := ctr.menuSvc.Detail(ctx.Request.Context(), &req)
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
	res, err := ctr.menuSvc.PageList(ctx.Request.Context(), &req)
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
	res, err := ctr.menuSvc.Tree(ctx.Request.Context(), &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 菜单管理
// @Summary 当前用户可见菜单树（侧边栏动态菜单）
// @accept application/json
// @Produce application/json
// @Success 200 {object} gincontext.DtoRender{data=dtopermission.MenuMyTreeResp}
// @Router /v1/platform/menus/my [get]
func (ctr *menuCtr) MyTree(ctx *gin.Context) {
	req := dtopermission.MenuMyTreeReq{Operator: identity.OperatorFromGin(ctx)}
	res, err := ctr.menuSvc.MyTree(ctx.Request.Context(), &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}
