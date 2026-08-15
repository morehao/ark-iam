package ctrtenant

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/tenantadmin/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/tenantadmin/internal/service/svctenant"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type RoleCtr interface {
	Create(ctx *gin.Context)
	PageList(ctx *gin.Context)
	Detail(ctx *gin.Context)
	Update(ctx *gin.Context)
	Delete(ctx *gin.Context)
	GetMenus(ctx *gin.Context)
	UpdateMenus(ctx *gin.Context)
}

type roleCtr struct {
	roleSvc svctenant.RoleSvc
}

var _ RoleCtr = (*roleCtr)(nil)

func NewRoleCtr() RoleCtr {
	return &roleCtr{
		roleSvc: svctenant.NewRoleSvc(),
	}
}

// @Tags 角色
// @Summary 创建租户角色
// @accept application/json
// @Produce application/json
// @Param req body dtotenant.RoleCreateReq true "创建角色"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.RoleCreateResp}
// @Router /v1/tenant/roles [post]
func (ctr *roleCtr) Create(ctx *gin.Context) {
	var req dtotenant.RoleCreateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.roleSvc.Create(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 角色
// @Summary 租户角色分页列表
// @accept application/json
// @Produce application/json
// @Param req query dtotenant.RolePageListReq true "角色分页列表"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.RolePageListResp}
// @Router /v1/tenant/roles [get]
func (ctr *roleCtr) PageList(ctx *gin.Context) {
	var req dtotenant.RolePageListReq
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

// @Tags 角色
// @Summary 角色详情
// @accept application/json
// @Produce application/json
// @Param roleID path string true "角色ID"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.RoleDetailResp}
// @Router /v1/tenant/roles/{roleID} [get]
func (ctr *roleCtr) Detail(ctx *gin.Context) {
	var req dtotenant.RoleDetailReq
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

// @Tags 角色
// @Summary 更新角色
// @accept application/json
// @Produce application/json
// @Param roleID path string true "角色ID"
// @Param req body dtotenant.RoleUpdateReq true "更新角色"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/tenant/roles/{roleID} [put]
func (ctr *roleCtr) Update(ctx *gin.Context) {
	var req dtotenant.RoleUpdateReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.roleSvc.Update(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "修改成功")
}

// @Tags 角色
// @Summary 删除角色（级联清理成员/菜单关联）
// @accept application/json
// @Produce application/json
// @Param roleID path string true "角色ID"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/tenant/roles/{roleID} [delete]
func (ctr *roleCtr) Delete(ctx *gin.Context) {
	var req dtotenant.RoleDeleteReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.roleSvc.Delete(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}

// @Tags 角色
// @Summary 角色菜单授权回显（菜单树 + 已授权ID）
// @accept application/json
// @Produce application/json
// @Param roleID path string true "角色ID"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.RoleMenuTreeResp}
// @Router /v1/tenant/roles/{roleID}/menus [get]
func (ctr *roleCtr) GetMenus(ctx *gin.Context) {
	var req dtotenant.RoleDetailReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.roleSvc.GetMenus(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 角色
// @Summary 全量替换角色菜单授权
// @accept application/json
// @Produce application/json
// @Param roleID path string true "角色ID"
// @Param req body dtotenant.RoleMenusUpdateReq true "全量替换角色菜单"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/tenant/roles/{roleID}/menus [put]
func (ctr *roleCtr) UpdateMenus(ctx *gin.Context) {
	var req dtotenant.RoleMenusUpdateReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.roleSvc.UpdateMenus(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "更新成功")
}
