package ctruser

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtouser"
	"github.com/morehao/ark-iam/platformadmin/internal/service/svcuser"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type UserCtr interface {
	Create(ctx *gin.Context)
	Delete(ctx *gin.Context)
	Update(ctx *gin.Context)
	Detail(ctx *gin.Context)
	PageList(ctx *gin.Context)
	UpdatePassword(ctx *gin.Context)
	UpdateStatus(ctx *gin.Context)
	DetailUserLoginLog(ctx *gin.Context)
	PageListUserLoginLog(ctx *gin.Context)
	GetUserLoginLogByUser(ctx *gin.Context)
	CreateUserIdentity(ctx *gin.Context)
	DeleteUserIdentity(ctx *gin.Context)
	UpdateUserIdentity(ctx *gin.Context)
	DetailUserIdentity(ctx *gin.Context)
	PageListUserIdentity(ctx *gin.Context)
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
// @Summary 创建用户管理
// @accept application/json
// @Produce application/json
// @Param req body dtouser.UserCreateReq true "创建用户管理"
// @Success 200 {object} gincontext.DtoRender{data=dtouser.UserCreateResp}
// @Router /v1/platform/users [post]
func (ctr *userCtr) Create(ctx *gin.Context) {
	var req dtouser.UserCreateReq
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

// @Tags 用户管理
// @Summary 删除用户管理
// @accept application/json
// @Produce application/json
// @Param userID path int true "用户ID"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/platform/users/{userID} [delete]
func (ctr *userCtr) Delete(ctx *gin.Context) {
	var req dtouser.UserDeleteReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.userSvc.Delete(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}

// @Tags 用户管理
// @Summary 修改用户管理
// @accept application/json
// @Produce application/json
// @Param userID path int true "用户ID"
// @Param req body dtouser.UserUpdateReq true "修改用户管理"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/platform/users/{userID} [put]
func (ctr *userCtr) Update(ctx *gin.Context) {
	var req dtouser.UserUpdateReq
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

// @Router /v1/platform/login-logs/{loginLogID} [get]
func (ctr *userCtr) DetailUserLoginLog(ctx *gin.Context) {
	var req dtouser.UserLoginLogDetailReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.userSvc.DetailUserLoginLog(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 用户管理
// @Summary 用户登录日志列表分页
// @accept application/json
// @Produce application/json
// @Param req query dtouser.UserLoginLogPageListReq true "用户登录日志列表分页"
// @Success 200 {object} gincontext.DtoRender{data=dtouser.UserLoginLogPageListResp}
// @Router /v1/platform/login-logs [get]
func (ctr *userCtr) PageListUserLoginLog(ctx *gin.Context) {
	var req dtouser.UserLoginLogPageListReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.userSvc.PageListUserLoginLog(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
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
// @Summary 修改用户身份
// @accept application/json
// @Produce application/json
// @Param userID path int true "用户ID"
// @Param identityID path int true "用户身份ID"
// @Param req body dtouser.UserIdentityUpdateReq true "修改用户身份"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/platform/users/{userID}/identities/{identityID} [put]
func (ctr *userCtr) UpdateUserIdentity(ctx *gin.Context) {
	var req dtouser.UserIdentityUpdateReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.userIdentitySvc.Update(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "修改成功")
}

// @Tags 用户管理
// @Summary 用户身份详情
// @accept application/json
// @Produce application/json
// @Param userID path int true "用户ID"
// @Param identityID path int true "用户身份ID"
// @Success 200 {object} gincontext.DtoRender{data=dtouser.UserIdentityDetailResp}
// @Router /v1/platform/users/{userID}/identities/{identityID} [get]
func (ctr *userCtr) DetailUserIdentity(ctx *gin.Context) {
	var req dtouser.UserIdentityDetailReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.userIdentitySvc.Detail(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 用户管理
// @Summary 用户身份列表分页
// @accept application/json
// @Produce application/json
// @Param req query dtouser.UserIdentityPageListReq true "用户身份列表分页"
// @Success 200 {object} gincontext.DtoRender{data=dtouser.UserIdentityPageListResp}
// @Router /v1/platform/user-identities [get]
func (ctr *userCtr) PageListUserIdentity(ctx *gin.Context) {
	var req dtouser.UserIdentityPageListReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.userIdentitySvc.PageList(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
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
