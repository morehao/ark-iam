package ctruser

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtouser"
	"github.com/morehao/ark-iam/platformadmin/internal/service/svcuser"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

// UserCtr 平台排查视角：跨租户用户目录只读 + 挂起/恢复 + 重置密码 + 身份/登录日志子资源。
type UserCtr interface {
	Detail(ctx *gin.Context)
	PageList(ctx *gin.Context)
	UpdatePassword(ctx *gin.Context)
	UpdateStatus(ctx *gin.Context)
	UpdateOwner(ctx *gin.Context)
	GetUserLoginLogByUser(ctx *gin.Context)
	CreateUserIdentity(ctx *gin.Context)
	DeleteUserIdentity(ctx *gin.Context)
	GetUserIdentityByUser(ctx *gin.Context)
}

type userCtr struct {
	userSvc         svcuser.UserSvc
	userIdentitySvc svcuser.UserIdentitySvc
}

var _ UserCtr = (*userCtr)(nil)

func NewUserCtr() UserCtr {
	return &userCtr{
		userSvc:         svcuser.NewUserSvc(),
		userIdentitySvc: svcuser.NewUserIdentitySvc(),
	}
}

// @Tags 用户管理
// @Summary 用户管理详情
// @accept application/json
// @Produce application/json
// @Param userID path int true "用户ID"
// @Success 200 {object} gincontext.DtoRender{data=dtouser.UserDetailResp}
// @Router /v1/platform/users/{userID} [get]
func (ctr *userCtr) Detail(ctx *gin.Context) {
	var req dtouser.UserDetailReq
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

// @Tags 用户管理
// @Summary 用户管理列表分页
// @accept application/json
// @Produce application/json
// @Param req query dtouser.UserPageListReq true "用户管理列表分页"
// @Success 200 {object} gincontext.DtoRender{data=dtouser.UserPageListResp}
// @Router /v1/platform/users [get]
func (ctr *userCtr) PageList(ctx *gin.Context) {
	var req dtouser.UserPageListReq
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

// @Tags 用户管理
// @Summary 修改用户密码
// @accept application/json
// @Produce application/json
// @Param userID path int true "用户ID"
// @Param req body dtouser.UserPasswordUpdateReq true "修改用户密码"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/platform/users/{userID}/changePassword [post]
func (ctr *userCtr) UpdatePassword(ctx *gin.Context) {
	var req dtouser.UserPasswordUpdateReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.userSvc.UpdatePassword(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "修改成功")
}

// @Tags 用户管理
// @Summary 修改用户状态
// @accept application/json
// @Produce application/json
// @Param userID path int true "用户ID"
// @Param req body dtouser.UserStatusUpdateReq true "修改用户状态"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/platform/users/{userID} [patch]
func (ctr *userCtr) UpdateStatus(ctx *gin.Context) {
	var req dtouser.UserStatusUpdateReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.userSvc.UpdateStatus(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "修改成功")
}

// @Tags 用户管理
// @Summary 设置租户拥有者（平台管理员显式指派/取消）
// @accept application/json
// @Produce application/json
// @Param userID path int true "用户ID"
// @Param req body dtouser.UserOwnerUpdateReq true "设置租户拥有者"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/platform/users/{userID}/owner [put]
func (ctr *userCtr) UpdateOwner(ctx *gin.Context) {
	var req dtouser.UserOwnerUpdateReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.userSvc.UpdateOwner(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "设置成功")
}

// @Tags 用户管理
// @Summary 获取用户登录日志
// @accept application/json
// @Produce application/json
// @Param userID path int true "用户ID"
// @Success 200 {object} gincontext.DtoRender{data=dtouser.UserLoginLogPageListResp}
// @Router /v1/platform/users/{userID}/login-logs [get]
func (ctr *userCtr) GetUserLoginLogByUser(ctx *gin.Context) {
	var req dtouser.UserLoginLogByUserReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.userSvc.GetUserLoginLogByUser(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 用户管理
// @Summary 创建用户身份
// @accept application/json
// @Produce application/json
// @Param userID path int true "用户ID"
// @Param req body dtouser.UserIdentityCreateReq true "创建用户身份"
// @Success 200 {object} gincontext.DtoRender{data=dtouser.UserIdentityCreateResp}
// @Router /v1/platform/users/{userID}/identities [post]
func (ctr *userCtr) CreateUserIdentity(ctx *gin.Context) {
	var req dtouser.UserIdentityCreateReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.userIdentitySvc.Create(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 用户管理
// @Summary 删除用户身份
// @accept application/json
// @Produce application/json
// @Param userID path int true "用户ID"
// @Param identityID path int true "用户身份ID"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/platform/users/{userID}/identities/{identityID} [delete]
func (ctr *userCtr) DeleteUserIdentity(ctx *gin.Context) {
	var req dtouser.UserIdentityDeleteReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.userIdentitySvc.Delete(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}

// @Tags 用户管理
// @Summary 获取用户身份
// @accept application/json
// @Produce application/json
// @Param userID path int true "用户ID"
// @Success 200 {object} gincontext.DtoRender{data=dtouser.UserIdentityPageListResp}
// @Router /v1/platform/users/{userID}/identities [get]
func (ctr *userCtr) GetUserIdentityByUser(ctx *gin.Context) {
	var req dtouser.UserIdentityByUserReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.userIdentitySvc.GetByUser(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}
