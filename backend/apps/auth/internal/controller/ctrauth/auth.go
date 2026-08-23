package ctrauth

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/internal/dto/dtoauth"
	"github.com/morehao/ark-iam/auth/internal/service/svcauth"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type AuthCtr interface {
	MyTenants(ctx *gin.Context)
	JoinTenant(ctx *gin.Context)
	Logout(ctx *gin.Context)
	LogoutAll(ctx *gin.Context)
	Userinfo(ctx *gin.Context)
}

type authCtr struct {
	authSvc svcauth.AuthSvc
}

var _ AuthCtr = (*authCtr)(nil)

func NewAuthCtr() AuthCtr {
	return &authCtr{
		authSvc: svcauth.NewAuthSvc(),
	}
}

// @Tags 认证
// @Summary 我的租户列表
// @accept application/json
// @Produce application/json
// @Success 200 {object} gincontext.DtoRender{data=dtoauth.MyTenantsResp}
// @Router /v1/auth/me/tenants [get]
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
// @Summary 加入租户
// @accept application/json
// @Produce application/json
// @Param req body dtoauth.JoinTenantReq true "加入租户"
// @Success 200 {object} gincontext.DtoRender{data=dtoauth.JoinTenantResp}
// @Router /v1/auth/joinTenant [post]
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
// @Summary 用户登出（全局：撤销全部 refresh token + SSO 会话）
// @accept application/json
// @Produce application/json
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/auth/logout [post]
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

// LogoutAll 与 Logout 语义一致（均为 person 级全局登出）。
// 会话撤销已由 svcauth.Logout 内部完成，此处不再重复调用 SSO 撤销。
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
// @Router /v1/auth/userinfo [get]
func (ctr *authCtr) Userinfo(ctx *gin.Context) {
	var req dtoauth.UserinfoReq
	res, err := ctr.authSvc.Userinfo(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}
