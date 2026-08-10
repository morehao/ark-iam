package ctrauth

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/internal/dto/dtoauth"
	"github.com/morehao/ark-iam/auth/internal/service/svcauth"
	"github.com/morehao/ark-iam/auth/internal/service/svcsso"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/glog"
)

type AuthCtr interface {
	MyTenants(ctx *gin.Context)
	Register(ctx *gin.Context)
	JoinTenant(ctx *gin.Context)
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
// @Router /v1/auth/register [post]
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
// @Summary 用户登出
// @accept application/json
// @Produce application/json
// @Param req body dtoauth.LogoutReq true "用户登出"
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
	// 撤销该 person 的全部 SSO session，实现"一处登出、处处登出"的全局登出语义
	personID := gincontext.GetPersonID(ctx)
	if personID != 0 {
		if err := svcsso.RevokeSSOSessionsByPersonID(ctx.Request.Context(), personID); err != nil {
			glog.Errorf(ctx, "[ctrauth.LogoutAll] RevokeSSOSessionsByPersonID fail, personID:%d, err:%v", personID, err)
		}
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
