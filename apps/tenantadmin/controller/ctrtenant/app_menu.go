package ctrtenant

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/apps/tenantadmin/dto/dtotenant"
	"github.com/morehao/ark-iam/apps/tenantadmin/service/svctenant"
	"github.com/morehao/ark-iam/pkg/identity"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/dbaccess/gormdao"
)

// AppMenuCtr 租户侧「应用菜单」维护控制器（只能维护归属当前租户的应用菜单）。
type AppMenuCtr interface {
	Create(ctx *gin.Context)
	Update(ctx *gin.Context)
	Delete(ctx *gin.Context)
	Detail(ctx *gin.Context)
	Tree(ctx *gin.Context)
}

type appMenuCtr struct {
	appMenuSvc svctenant.AppMenuSvc
}

var _ AppMenuCtr = (*appMenuCtr)(nil)

func NewAppMenuCtr(db gormdao.DBGetter) AppMenuCtr {
	return &appMenuCtr{appMenuSvc: svctenant.NewAppMenuSvc(db)}
}

// @Tags 租户菜单
// @Summary 创建应用菜单
// @accept application/json
// @Produce application/json
// @Param appID path string true "应用ID"
// @Param req body dtotenant.AppMenuCreateReq true "创建应用菜单"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.AppMenuCreateResp}
// @Router /v1/tenant/applications/{appID}/menus [post]
func (ctr *appMenuCtr) Create(ctx *gin.Context) {
	var req dtotenant.AppMenuCreateReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	// path 是 appID 的唯一来源，覆盖 body 里可能伪造的 AppID
	if err := ctx.ShouldBindUri(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	req.Operator = identity.OperatorFromGin(ctx)
	res, err := ctr.appMenuSvc.Create(ctx.Request.Context(), &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 租户菜单
// @Summary 更新应用菜单
// @accept application/json
// @Produce application/json
// @Param appID path string true "应用ID"
// @Param menuID path string true "菜单ID"
// @Param req body dtotenant.AppMenuUpdateReq true "更新应用菜单"
// @Success 200 {object} gincontext.DtoRender
// @Router /v1/tenant/applications/{appID}/menus/{menuID} [put]
func (ctr *appMenuCtr) Update(ctx *gin.Context) {
	var req dtotenant.AppMenuUpdateReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctx.ShouldBindUri(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	req.Operator = identity.OperatorFromGin(ctx)
	if err := ctr.appMenuSvc.Update(ctx.Request.Context(), &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, nil)
}

// @Tags 租户菜单
// @Summary 删除应用菜单
// @accept application/json
// @Produce application/json
// @Param appID path string true "应用ID"
// @Param menuID path string true "菜单ID"
// @Success 200 {object} gincontext.DtoRender
// @Router /v1/tenant/applications/{appID}/menus/{menuID} [delete]
func (ctr *appMenuCtr) Delete(ctx *gin.Context) {
	var req dtotenant.AppMenuDeleteReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	req.Operator = identity.OperatorFromGin(ctx)
	if err := ctr.appMenuSvc.Delete(ctx.Request.Context(), &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, nil)
}

// @Tags 租户菜单
// @Summary 应用菜单详情
// @accept application/json
// @Produce application/json
// @Param appID path string true "应用ID"
// @Param menuID path string true "菜单ID"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.MenuTreeItem}
// @Router /v1/tenant/applications/{appID}/menus/{menuID} [get]
func (ctr *appMenuCtr) Detail(ctx *gin.Context) {
	var req dtotenant.AppMenuDetailReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	req.Operator = identity.OperatorFromGin(ctx)
	res, err := ctr.appMenuSvc.Detail(ctx.Request.Context(), &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 租户菜单
// @Summary 应用菜单树（管理视图）
// @accept application/json
// @Produce application/json
// @Param appID path string true "应用ID"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.MenuTreeResp}
// @Router /v1/tenant/applications/{appID}/menus/tree [get]
func (ctr *appMenuCtr) Tree(ctx *gin.Context) {
	var req dtotenant.AppMenuTreeReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	req.Operator = identity.OperatorFromGin(ctx)
	res, err := ctr.appMenuSvc.Tree(ctx.Request.Context(), &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}
