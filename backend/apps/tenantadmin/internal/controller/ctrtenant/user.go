package ctrtenant

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/tenantadmin/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/tenantadmin/internal/service/svctenant"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type UserCtr interface {
	PageList(ctx *gin.Context)
	Create(ctx *gin.Context)
	Detail(ctx *gin.Context)
	Update(ctx *gin.Context)
	ResetPassword(ctx *gin.Context)
	ListRoles(ctx *gin.Context)
	UpdateRoles(ctx *gin.Context)
}

type userCtr struct {
	userSvc svctenant.UserSvc
}

var _ UserCtr = (*userCtr)(nil)

func NewUserCtr() UserCtr {
	return &userCtr{
		userSvc: svctenant.NewUserSvc(),
	}
}

// @Tags 用户
// @Summary 租户内用户列表分页
// @accept application/json
// @Produce application/json
// @Param req query dtotenant.UserPageListReq true "租户内用户列表分页"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.UserPageListResp}
// @Router /v1/tenant/users [get]
func (ctr *userCtr) PageList(ctx *gin.Context) {
	var req dtotenant.UserPageListReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.userSvc.PageList(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 用户
// @Summary 创建租户用户（person 不存在则先创建）
// @accept application/json
// @Produce application/json
// @Param req body dtotenant.UserCreateReq true "创建租户用户"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.UserCreateResp}
// @Router /v1/tenant/users [post]
func (ctr *userCtr) Create(ctx *gin.Context) {
	var req dtotenant.UserCreateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.userSvc.Create(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 用户
// @Summary 用户详情（基础信息 + 组织归属 + 角色）
// @accept application/json
// @Produce application/json
// @Param userID path string true "用户ID"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.UserDetailResp}
// @Router /v1/tenant/users/{userID} [get]
func (ctr *userCtr) Detail(ctx *gin.Context) {
	var req dtotenant.UserDetailReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.userSvc.Detail(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 用户
// @Summary 局部更新用户（姓名/头像/状态）
// @accept application/json
// @Produce application/json
// @Param userID path string true "用户ID"
// @Param req body dtotenant.UserUpdateReq true "更新用户"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/tenant/users/{userID} [patch]
func (ctr *userCtr) Update(ctx *gin.Context) {
	var req dtotenant.UserUpdateReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.userSvc.Update(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "修改成功")
}

// @Tags 用户
// @Summary 重置密码
// @accept application/json
// @Produce application/json
// @Param userID path string true "用户ID"
// @Param req body dtotenant.UserResetPasswordReq true "重置密码"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/tenant/users/{userID}/reset-password [post]
func (ctr *userCtr) ResetPassword(ctx *gin.Context) {
	var req dtotenant.UserResetPasswordReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.userSvc.ResetPassword(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "重置成功")
}

// @Tags 用户
// @Summary 用户已分配角色列表
// @accept application/json
// @Produce application/json
// @Param userID path string true "用户ID"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.UserRolesListResp}
// @Router /v1/tenant/users/{userID}/roles [get]
func (ctr *userCtr) ListRoles(ctx *gin.Context) {
	var req dtotenant.UserRolesListReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.userSvc.ListRoles(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 用户
// @Summary 全量替换用户角色
// @accept application/json
// @Produce application/json
// @Param userID path string true "用户ID"
// @Param req body dtotenant.UserRolesUpdateReq true "全量替换用户角色"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/tenant/users/{userID}/roles [put]
func (ctr *userCtr) UpdateRoles(ctx *gin.Context) {
	var req dtotenant.UserRolesUpdateReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.userSvc.UpdateRoles(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "更新成功")
}
