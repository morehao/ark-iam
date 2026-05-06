package ctruser

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/dto/dtouser"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

// Create 创建用户
// @Tags 用户管理
// @Summary 创建用户
// @accept application/json
// @Produce application/json
// @Param req body dtouser.UserCreateReq true "创建用户"
// @Success 200 {object} gincontext.DtoRender{data=dtouser.UserCreateResp}
// @Router /v1/iam/user/create [post]
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

// Delete 删除用户
// @Tags 用户管理
// @Summary 删除用户
// @accept application/json
// @Produce application/json
// @Param req body dtouser.UserDeleteReq true "删除用户"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/user/delete [post]
func (ctr *userCtr) Delete(ctx *gin.Context) {
	var req dtouser.UserDeleteReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.userSvc.Delete(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}

// Update 修改用户
// @Tags 用户管理
// @Summary 修改用户
// @accept application/json
// @Produce application/json
// @Param req body dtouser.UserUpdateReq true "修改用户"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/user/update [post]
func (ctr *userCtr) Update(ctx *gin.Context) {
	var req dtouser.UserUpdateReq
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

// Detail 用户详情
// @Tags 用户管理
// @Summary 用户详情
// @accept application/json
// @Produce application/json
// @Param req query dtouser.UserDetailReq true "用户详情"
// @Success 200 {object} gincontext.DtoRender{data=dtouser.UserDetailResp}
// @Router /v1/iam/user/detail [get]
func (ctr *userCtr) Detail(ctx *gin.Context) {
	var req dtouser.UserDetailReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
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

// PageList 用户列表
// @Tags 用户管理
// @Summary 用户列表分页
// @accept application/json
// @Produce application/json
// @Param req body dtouser.UserPageListReq true "用户列表"
// @Success 200 {object} gincontext.DtoRender{data=dtouser.UserPageListResp}
// @Router /v1/iam/user/pageList [post]
func (ctr *userCtr) PageList(ctx *gin.Context) {
	var req dtouser.UserPageListReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
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

// UpdatePassword 修改密码
// @Tags 用户管理
// @Summary 修改密码
// @accept application/json
// @Produce application/json
// @Param req body dtouser.UserPasswordUpdateReq true "修改密码"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/user/updatePassword [post]
func (ctr *userCtr) UpdatePassword(ctx *gin.Context) {
	var req dtouser.UserPasswordUpdateReq
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

// UpdateStatus 修改状态
// @Tags 用户管理
// @Summary 修改状态
// @accept application/json
// @Produce application/json
// @Param req body dtouser.UserStatusUpdateReq true "修改状态"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/user/updateStatus [post]
func (ctr *userCtr) UpdateStatus(ctx *gin.Context) {
	var req dtouser.UserStatusUpdateReq
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

// AssignDepartments 分配部门
// @Tags 用户管理
// @Summary 分配部门
// @accept application/json
// @Produce application/json
// @Param req body dtouser.AssignDepartmentsReq true "分配部门"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/user/assignDepartments [post]
func (ctr *userCtr) AssignDepartments(ctx *gin.Context) {
	var req dtouser.AssignDepartmentsReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.userSvc.AssignDepartments(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "分配成功")
}
