package ctruser

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/dto/dtouser"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

// CreateUserDepartmentRelation 创建用户部门关系
// @Tags 用户管理
// @Summary 创建用户部门关系
// @accept application/json
// @Produce application/json
// @Param req body dtouser.UserDepartmentRelationCreateReq true "创建用户部门关系"
// @Success 200 {object} gincontext.DtoRender{data=dtouser.UserDepartmentRelationCreateResp}
// @Router /v1/iam/user/createUserDepartmentRelation [post]
func (ctr *userCtr) CreateUserDepartmentRelation(ctx *gin.Context) {
	var req dtouser.UserDepartmentRelationCreateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.userSvc.CreateUserDepartmentRelation(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// DeleteUserDepartmentRelation 删除用户部门关系
// @Tags 用户管理
// @Summary 删除用户部门关系
// @accept application/json
// @Produce application/json
// @Param req body dtouser.UserDepartmentRelationDeleteReq true "删除用户部门关系"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/user/deleteUserDepartmentRelation [post]
func (ctr *userCtr) DeleteUserDepartmentRelation(ctx *gin.Context) {
	var req dtouser.UserDepartmentRelationDeleteReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.userSvc.DeleteUserDepartmentRelation(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}

// UpdateUserDepartmentRelation 修改用户部门关系
// @Tags 用户管理
// @Summary 修改用户部门关系
// @accept application/json
// @Produce application/json
// @Param req body dtouser.UserDepartmentRelationUpdateReq true "修改用户部门关系"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/user/updateUserDepartmentRelation [post]
func (ctr *userCtr) UpdateUserDepartmentRelation(ctx *gin.Context) {
	var req dtouser.UserDepartmentRelationUpdateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.userSvc.UpdateUserDepartmentRelation(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "修改成功")
}

// DetailUserDepartmentRelation 用户部门关系详情
// @Tags 用户管理
// @Summary 用户部门关系详情
// @accept application/json
// @Produce application/json
// @Param req query dtouser.UserDepartmentRelationDetailReq true "用户部门关系详情"
// @Success 200 {object} gincontext.DtoRender{data=dtouser.UserDepartmentRelationDetailResp}
// @Router /v1/iam/user/detailUserDepartmentRelation [get]
func (ctr *userCtr) DetailUserDepartmentRelation(ctx *gin.Context) {
	var req dtouser.UserDepartmentRelationDetailReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.userSvc.DetailUserDepartmentRelation(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// PageListUserDepartmentRelation 用户部门关系列表
// @Tags 用户管理
// @Summary 用户部门关系列表分页
// @accept application/json
// @Produce application/json
// @Param req body dtouser.UserDepartmentRelationPageListReq true "用户部门关系列表"
// @Success 200 {object} gincontext.DtoRender{data=dtouser.UserDepartmentRelationPageListResp}
// @Router /v1/iam/user/pageListUserDepartmentRelation [post]
func (ctr *userCtr) PageListUserDepartmentRelation(ctx *gin.Context) {
	var req dtouser.UserDepartmentRelationPageListReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.userSvc.PageListUserDepartmentRelation(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// GetUserDepartmentRelationByUser 获取用户部门关系
// @Tags 用户管理
// @Summary 获取用户部门关系
// @accept application/json
// @Produce application/json
// @Param req query dtouser.UserDepartmentRelationByUserReq true "获取用户部门关系"
// @Success 200 {object} gincontext.DtoRender{data=dtouser.UserDepartmentRelationPageListResp}
// @Router /v1/iam/user/getUserDepartmentRelationByUser [get]
func (ctr *userCtr) GetUserDepartmentRelationByUser(ctx *gin.Context) {
	var req dtouser.UserDepartmentRelationByUserReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.userSvc.GetUserDepartmentRelationByUser(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}
