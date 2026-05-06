package ctruser

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/dto/dtouser"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

// CreateUserIdentity 创建用户身份
// @Tags 用户管理
// @Summary 创建用户身份
// @accept application/json
// @Produce application/json
// @Param req body dtouser.UserIdentityCreateReq true "创建用户身份"
// @Success 200 {object} gincontext.DtoRender{data=dtouser.UserIdentityCreateResp}
// @Router /v1/iam/user/createUserIdentity [post]
func (ctr *userCtr) CreateUserIdentity(ctx *gin.Context) {
	var req dtouser.UserIdentityCreateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.userSvc.CreateUserIdentity(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// DeleteUserIdentity 删除用户身份
// @Tags 用户管理
// @Summary 删除用户身份
// @accept application/json
// @Produce application/json
// @Param req body dtouser.UserIdentityDeleteReq true "删除用户身份"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/user/deleteUserIdentity [post]
func (ctr *userCtr) DeleteUserIdentity(ctx *gin.Context) {
	var req dtouser.UserIdentityDeleteReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.userSvc.DeleteUserIdentity(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}

// UpdateUserIdentity 修改用户身份
// @Tags 用户管理
// @Summary 修改用户身份
// @accept application/json
// @Produce application/json
// @Param req body dtouser.UserIdentityUpdateReq true "修改用户身份"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/user/updateUserIdentity [post]
func (ctr *userCtr) UpdateUserIdentity(ctx *gin.Context) {
	var req dtouser.UserIdentityUpdateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.userSvc.UpdateUserIdentity(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "修改成功")
}

// DetailUserIdentity 用户身份详情
// @Tags 用户管理
// @Summary 用户身份详情
// @accept application/json
// @Produce application/json
// @Param req query dtouser.UserIdentityDetailReq true "用户身份详情"
// @Success 200 {object} gincontext.DtoRender{data=dtouser.UserIdentityDetailResp}
// @Router /v1/iam/user/detailUserIdentity [get]
func (ctr *userCtr) DetailUserIdentity(ctx *gin.Context) {
	var req dtouser.UserIdentityDetailReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.userSvc.DetailUserIdentity(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// PageListUserIdentity 用户身份列表
// @Tags 用户管理
// @Summary 用户身份列表分页
// @accept application/json
// @Produce application/json
// @Param req body dtouser.UserIdentityPageListReq true "用户身份列表"
// @Success 200 {object} gincontext.DtoRender{data=dtouser.UserIdentityPageListResp}
// @Router /v1/iam/user/pageListUserIdentity [post]
func (ctr *userCtr) PageListUserIdentity(ctx *gin.Context) {
	var req dtouser.UserIdentityPageListReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.userSvc.PageListUserIdentity(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// GetUserIdentityByUser 获取用户身份
// @Tags 用户管理
// @Summary 获取用户身份
// @accept application/json
// @Produce application/json
// @Param req query dtouser.UserIdentityByUserReq true "获取用户身份"
// @Success 200 {object} gincontext.DtoRender{data=dtouser.UserIdentityPageListResp}
// @Router /v1/iam/user/getUserIdentityByUser [get]
func (ctr *userCtr) GetUserIdentityByUser(ctx *gin.Context) {
	var req dtouser.UserIdentityByUserReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.userSvc.GetUserIdentityByUser(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}
