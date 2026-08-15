package ctrpermission

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtopermission"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtouser"
	"github.com/morehao/ark-iam/platformadmin/internal/service/svcpermission"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type RoleCtr interface {
	Create(ctx *gin.Context)
	Delete(ctx *gin.Context)
	Update(ctx *gin.Context)
	Detail(ctx *gin.Context)
	PageList(ctx *gin.Context)
	ListUsers(ctx *gin.Context)
	AssignUsers(ctx *gin.Context)
	RemoveUser(ctx *gin.Context)
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
// @Summary 创建角色管理
// @accept application/json
// @Produce application/json
// @Param req body dtopermission.RoleCreateReq true "创建角色管理"
// @Success 200 {object} gincontext.DtoRender{data=dtopermission.RoleCreateResp}
// @Router /v1/platform/role/create [post]
func (ctr *roleCtr) Create(ctx *gin.Context) {
	var req dtopermission.RoleCreateReq
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

// @Tags 角色管理
// @Summary 删除角色管理
// @accept application/json
// @Produce application/json
// @Param req body dtopermission.RoleDeleteReq true "删除角色管理"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/platform/role/delete [post]
func (ctr *roleCtr) Delete(ctx *gin.Context) {
	var req dtopermission.RoleDeleteReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.roleSvc.Delete(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}

// @Tags 角色管理
// @Summary 修改角色管理
// @accept application/json
// @Produce application/json
// @Param req body dtopermission.RoleUpdateReq true "修改角色管理"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/platform/role/update [post]
func (ctr *roleCtr) Update(ctx *gin.Context) {
	var req dtopermission.RoleUpdateReq
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

// @Tags 角色管理
// @Summary 角色管理详情
// @accept application/json
// @Produce application/json
// @Param req query dtopermission.RoleDetailReq true "角色管理详情"
// @Success 200 {object} gincontext.DtoRender{data=dtopermission.RoleDetailResp}
// @Router /v1/platform/role/detail [get]
func (ctr *roleCtr) Detail(ctx *gin.Context) {
	var req dtopermission.RoleDetailReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
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
// @Param req body dtopermission.RolePageListReq true "角色管理列表分页"
// @Success 200 {object} gincontext.DtoRender{data=dtopermission.RolePageListResp}
// @Router /v1/platform/role/pageList [post]
func (ctr *roleCtr) PageList(ctx *gin.Context) {
	var req dtopermission.RolePageListReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
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
// @Param req query dtouser.RoleUserListReq true "角色用户列表"
// @Success 200 {object} gincontext.DtoRender{data=dtouser.RoleUserListResp}
// @Router /v1/platform/role/users [get]
func (ctr *roleCtr) ListUsers(ctx *gin.Context) {
	var req dtouser.RoleUserListReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
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

// @Tags 角色管理
// @Summary 分配用户
// @accept application/json
// @Produce application/json
// @Param req body dtouser.AssignRoleUsersReq true "分配用户"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/platform/role/assignUsers [post]
func (ctr *roleCtr) AssignUsers(ctx *gin.Context) {
	var req dtouser.AssignRoleUsersReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.roleSvc.AssignUsers(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "分配成功")
}

// @Tags 角色管理
// @Summary 移除用户
// @accept application/json
// @Produce application/json
// @Param roleID path int true "角色ID"
// @Param userID path int true "用户ID"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/platform/role/users/{roleID}/{userID} [delete]
func (ctr *roleCtr) RemoveUser(ctx *gin.Context) {
	var req dtouser.RemoveRoleUserReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.roleSvc.RemoveUser(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "移除成功")
}
