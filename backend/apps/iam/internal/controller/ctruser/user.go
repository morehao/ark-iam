package ctruser

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/dto/dtouser"
	"github.com/morehao/ark-iam/iam/internal/service/svcuser"
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
	CreateUserDepartmentRelation(ctx *gin.Context)
	DeleteUserDepartmentRelation(ctx *gin.Context)
	UpdateUserDepartmentRelation(ctx *gin.Context)
	DetailUserDepartmentRelation(ctx *gin.Context)
	PageListUserDepartmentRelation(ctx *gin.Context)
	GetUserDepartmentRelationByUser(ctx *gin.Context)
	AssignDepartments(ctx *gin.Context)
	CreateUserIdentity(ctx *gin.Context)
	DeleteUserIdentity(ctx *gin.Context)
	UpdateUserIdentity(ctx *gin.Context)
	DetailUserIdentity(ctx *gin.Context)
	PageListUserIdentity(ctx *gin.Context)
	GetUserIdentityByUser(ctx *gin.Context)
}

type userCtr struct {
	userSvc          svcuser.UserSvc
	userIdentitySvc svcuser.UserIdentitySvc
}

var _ UserCtr = (*userCtr)(nil)

func NewUserCtr() UserCtr {
	return &userCtr{
		userSvc:          svcuser.NewUserSvc(),
		userIdentitySvc: svcuser.NewUserIdentitySvc(),
	}
}

// @Tags 用户管理
// @Summary 创建用户管理
// @accept application/json
// @Produce application/json
// @Param req body dtouser.UserCreateReq true "创建用户管理"
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

// @Tags 用户管理
// @Summary 删除用户管理
// @accept application/json
// @Produce application/json
// @Param req body dtouser.UserDeleteReq true "删除用户管理"
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

// @Tags 用户管理
// @Summary 修改用户管理
// @accept application/json
// @Produce application/json
// @Param req body dtouser.UserUpdateReq true "修改用户管理"
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

// @Tags 用户管理
// @Summary 用户管理详情
// @accept application/json
// @Produce application/json
// @Param req query dtouser.UserDetailReq true "用户管理详情"
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

// @Tags 用户管理
// @Summary 用户管理列表分页
// @accept application/json
// @Produce application/json
// @Param req body dtouser.UserPageListReq true "用户管理列表分页"
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

// @Tags 用户管理
// @Summary 修改用户密码
// @accept application/json
// @Produce application/json
// @Param req body dtouser.UserPasswordUpdateReq true "修改用户密码"
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

// @Tags 用户管理
// @Summary 修改用户状态
// @accept application/json
// @Produce application/json
// @Param req body dtouser.UserStatusUpdateReq true "修改用户状态"
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

// @Tags 用户管理
// @Summary 分配用户部门
// @accept application/json
// @Produce application/json
// @Param req body dtouser.AssignDepartmentsReq true "分配用户部门"
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

// @Tags 用户管理
// @Summary 用户登录日志详情
// @accept application/json
// @Produce application/json
// @Param req query dtouser.UserLoginLogDetailReq true "用户登录日志详情"
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

// @Tags 用户管理
// @Summary 用户登录日志列表分页
// @accept application/json
// @Produce application/json
// @Param req body dtouser.UserLoginLogPageListReq true "用户登录日志列表分页"
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

// @Tags 用户管理
// @Summary 创建用户部门关联
// @accept application/json
// @Produce application/json
// @Param req body dtouser.UserDepartmentRelationCreateReq true "创建用户部门关联"
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

// @Tags 用户管理
// @Summary 删除用户部门关联
// @accept application/json
// @Produce application/json
// @Param req body dtouser.UserDepartmentRelationDeleteReq true "删除用户部门关联"
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

// @Tags 用户管理
// @Summary 修改用户部门关联
// @accept application/json
// @Produce application/json
// @Param req body dtouser.UserDepartmentRelationUpdateReq true "修改用户部门关联"
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

// @Tags 用户管理
// @Summary 用户部门关联详情
// @accept application/json
// @Produce application/json
// @Param req query dtouser.UserDepartmentRelationDetailReq true "用户部门关联详情"
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

// @Tags 用户管理
// @Summary 用户部门关联列表分页
// @accept application/json
// @Produce application/json
// @Param req body dtouser.UserDepartmentRelationPageListReq true "用户部门关联列表分页"
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

// @Tags 用户管理
// @Summary 获取用户部门关联
// @accept application/json
// @Produce application/json
// @Param req query dtouser.UserDepartmentRelationByUserReq true "获取用户部门关联"
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
// @Param req body dtouser.UserIdentityDeleteReq true "删除用户身份"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/user/deleteUserIdentity [post]
func (ctr *userCtr) DeleteUserIdentity(ctx *gin.Context) {
	var req dtouser.UserIdentityDeleteReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
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
// @Param req body dtouser.UserIdentityUpdateReq true "修改用户身份"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/user/updateUserIdentity [post]
func (ctr *userCtr) UpdateUserIdentity(ctx *gin.Context) {
	var req dtouser.UserIdentityUpdateReq
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
// @Param req query dtouser.UserIdentityDetailReq true "用户身份详情"
// @Success 200 {object} gincontext.DtoRender{data=dtouser.UserIdentityDetailResp}
// @Router /v1/iam/user/detailUserIdentity [get]
func (ctr *userCtr) DetailUserIdentity(ctx *gin.Context) {
	var req dtouser.UserIdentityDetailReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
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
// @Param req body dtouser.UserIdentityPageListReq true "用户身份列表分页"
// @Success 200 {object} gincontext.DtoRender{data=dtouser.UserIdentityPageListResp}
// @Router /v1/iam/user/pageListUserIdentity [post]
func (ctr *userCtr) PageListUserIdentity(ctx *gin.Context) {
	var req dtouser.UserIdentityPageListReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
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
// @Param req query dtouser.UserIdentityByUserReq true "获取用户身份"
// @Success 200 {object} gincontext.DtoRender{data=dtouser.UserIdentityPageListResp}
// @Router /v1/iam/user/getUserIdentityByUser [get]
func (ctr *userCtr) GetUserIdentityByUser(ctx *gin.Context) {
	var req dtouser.UserIdentityByUserReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
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
