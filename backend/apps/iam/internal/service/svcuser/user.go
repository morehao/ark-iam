package svcuser

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/dto/dtouser"
)

type UserSvc interface {
	Create(ctx *gin.Context, req *dtouser.UserCreateReq) (*dtouser.UserCreateResp, error)
	Delete(ctx *gin.Context, req *dtouser.UserDeleteReq) error
	Update(ctx *gin.Context, req *dtouser.UserUpdateReq) error
	Detail(ctx *gin.Context, req *dtouser.UserDetailReq) (*dtouser.UserDetailResp, error)
	PageList(ctx *gin.Context, req *dtouser.UserPageListReq) (*dtouser.UserPageListResp, error)
	UpdatePassword(ctx *gin.Context, req *dtouser.UserPasswordUpdateReq) error
	UpdateStatus(ctx *gin.Context, req *dtouser.UserStatusUpdateReq) error
	CreateUserIdentity(ctx *gin.Context, req *dtouser.UserIdentityCreateReq) (*dtouser.UserIdentityCreateResp, error)
	DeleteUserIdentity(ctx *gin.Context, req *dtouser.UserIdentityDeleteReq) error
	UpdateUserIdentity(ctx *gin.Context, req *dtouser.UserIdentityUpdateReq) error
	DetailUserIdentity(ctx *gin.Context, req *dtouser.UserIdentityDetailReq) (*dtouser.UserIdentityDetailResp, error)
	PageListUserIdentity(ctx *gin.Context, req *dtouser.UserIdentityPageListReq) (*dtouser.UserIdentityPageListResp, error)
	GetUserIdentityByUser(ctx *gin.Context, req *dtouser.UserIdentityByUserReq) (*dtouser.UserIdentityPageListResp, error)
	DetailUserLoginLog(ctx *gin.Context, req *dtouser.UserLoginLogDetailReq) (*dtouser.UserLoginLogDetailResp, error)
	PageListUserLoginLog(ctx *gin.Context, req *dtouser.UserLoginLogPageListReq) (*dtouser.UserLoginLogPageListResp, error)
	GetUserLoginLogByUser(ctx *gin.Context, req *dtouser.UserLoginLogByUserReq) (*dtouser.UserLoginLogPageListResp, error)
	CreateUserDepartmentRelation(ctx *gin.Context, req *dtouser.UserDepartmentRelationCreateReq) (*dtouser.UserDepartmentRelationCreateResp, error)
	DeleteUserDepartmentRelation(ctx *gin.Context, req *dtouser.UserDepartmentRelationDeleteReq) error
	UpdateUserDepartmentRelation(ctx *gin.Context, req *dtouser.UserDepartmentRelationUpdateReq) error
	DetailUserDepartmentRelation(ctx *gin.Context, req *dtouser.UserDepartmentRelationDetailReq) (*dtouser.UserDepartmentRelationDetailResp, error)
	PageListUserDepartmentRelation(ctx *gin.Context, req *dtouser.UserDepartmentRelationPageListReq) (*dtouser.UserDepartmentRelationPageListResp, error)
	GetUserDepartmentRelationByUser(ctx *gin.Context, req *dtouser.UserDepartmentRelationByUserReq) (*dtouser.UserDepartmentRelationPageListResp, error)
	AssignDepartments(ctx *gin.Context, req *dtouser.AssignDepartmentsReq) error
}

type userSvc struct {
}

var _ UserSvc = (*userSvc)(nil)

func NewUserSvc() UserSvc {
	return &userSvc{}
}