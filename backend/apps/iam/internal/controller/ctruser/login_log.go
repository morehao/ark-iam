package ctruser

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/dto/dtouser"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

// DetailUserLoginLog 登录日志详情
// @Tags 用户管理
// @Summary 登录日志详情
// @accept application/json
// @Produce application/json
// @Param req query dtouser.UserLoginLogDetailReq true "登录日志详情"
// @Success 200 {object} gincontext.DtoRender{data=dtouser.UserLoginLogDetailResp}
// @Router /v1/iam/user/detailUserLoginLog [get]
func (ctr *userCtr) DetailUserLoginLog(ctx *gin.Context) {
	var req dtouser.UserLoginLogDetailReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
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

// PageListUserLoginLog 登录日志列表
// @Tags 用户管理
// @Summary 登录日志列表分页
// @accept application/json
// @Produce application/json
// @Param req body dtouser.UserLoginLogPageListReq true "登录日志列表"
// @Success 200 {object} gincontext.DtoRender{data=dtouser.UserLoginLogPageListResp}
// @Router /v1/iam/user/pageListUserLoginLog [post]
func (ctr *userCtr) PageListUserLoginLog(ctx *gin.Context) {
	var req dtouser.UserLoginLogPageListReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
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

// GetUserLoginLogByUser 获取用户登录日志
// @Tags 用户管理
// @Summary 获取用户登录日志
// @accept application/json
// @Produce application/json
// @Param req query dtouser.UserLoginLogByUserReq true "获取用户登录日志"
// @Success 200 {object} gincontext.DtoRender{data=dtouser.UserLoginLogPageListResp}
// @Router /v1/iam/user/getUserLoginLogByUser [get]
func (ctr *userCtr) GetUserLoginLogByUser(ctx *gin.Context) {
	var req dtouser.UserLoginLogByUserReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
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
