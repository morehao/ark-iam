package ctrauth

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtoauth"
	"github.com/morehao/ark-iam/platformadmin/internal/service/svcauth"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type AuthCtr interface {
	Login(ctx *gin.Context)
	SelectTenant(ctx *gin.Context)
	SwitchTenant(ctx *gin.Context)
	MyTenants(ctx *gin.Context)
	Register(ctx *gin.Context)
	JoinTenant(ctx *gin.Context)
	RefreshToken(ctx *gin.Context)
	Logout(ctx *gin.Context)
	LogoutAll(ctx *gin.Context)
	Userinfo(ctx *gin.Context)
}

type authCtr struct {
	authSvc svcauth.AuthSvc
}

var _ AuthCtr = (*authCtr)(nil)

func NewAuthCtr(authSvc svcauth.AuthSvc) AuthCtr {
	return &authCtr{
		authSvc: authSvc,
	}
}

// @Tags 认证
// @Summary 用户登录
// @accept application/json
// @Produce application/json
// @Param req body dtoauth.LoginReq true "用户登录"
// @Success 200 {object} gincontext.DtoRender{data=dtoauth.LoginResp}
// @Router /v1/iam/auth/login [post]
func (ctr *authCtr) Login(ctx *gin.Context) {
	var req dtoauth.LoginReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.authSvc.Login(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *authCtr) SelectTenant(ctx *gin.Context) {
	var req dtoauth.SelectTenantReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.authSvc.SelectTenant(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *authCtr) SwitchTenant(ctx *gin.Context) {
	var req dtoauth.SwitchTenantReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.authSvc.SwitchTenant(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *authCtr) MyTenants(ctx *gin.Context) {
	var req dtoauth.MyTenantsReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.authSvc.MyTenants(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 认证
// @Summary 用户注册
// @accept application/json
// @Produce application/json
// @Param req body dtoauth.RegisterReq true "用户注册"
// @Success 200 {object} gincontext.DtoRender{data=dtoauth.RegisterResp}
// @Router /v1/iam/auth/register [post]
func (ctr *authCtr) Register(ctx *gin.Context) {
	var req dtoauth.RegisterReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.authSvc.Register(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 认证
// @Summary 加入租户
// @accept application/json
// @Produce application/json
// @Param req body dtoauth.JoinTenantReq true "加入租户"
// @Success 200 {object} gincontext.DtoRender{data=dtoauth.JoinTenantResp}
// @Router /v1/iam/auth/joinTenant [post]
func (ctr *authCtr) JoinTenant(ctx *gin.Context) {
	var req dtoauth.JoinTenantReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.authSvc.JoinTenant(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 认证
// @Summary 刷新令牌
// @accept application/json
// @Produce application/json
// @Param req body dtoauth.RefreshTokenReq true "刷新令牌"
// @Success 200 {object} gincontext.DtoRender{data=dtoauth.RefreshTokenResp}
// @Router /v1/iam/auth/refreshToken [post]
func (ctr *authCtr) RefreshToken(ctx *gin.Context) {
	var req dtoauth.RefreshTokenReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.authSvc.RefreshToken(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 认证
// @Summary 用户登出
// @accept application/json
// @Produce application/json
// @Param req body dtoauth.LogoutReq true "用户登出"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/auth/logout [post]
func (ctr *authCtr) Logout(ctx *gin.Context) {
	var req dtoauth.LogoutReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.authSvc.Logout(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "登出成功")
}

func (ctr *authCtr) LogoutAll(ctx *gin.Context) {
	var req dtoauth.LogoutAllReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.authSvc.LogoutAll(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "登出成功")
}

// @Tags 认证
// @Summary 获取用户信息
// @accept application/json
// @Produce application/json
// @Param req query dtoauth.UserinfoReq true "获取用户信息"
// @Success 200 {object} gincontext.DtoRender{data=dtoauth.UserinfoResp}
// @Router /v1/iam/auth/userinfo [get]
func (ctr *authCtr) Userinfo(ctx *gin.Context) {
	var req dtoauth.UserinfoReq
	res, err := ctr.authSvc.Userinfo(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}
