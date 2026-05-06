package ctruser

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/service/svcuser"
)

type UserCtr interface {
	Create(ctx *gin.Context)
	Delete(ctx *gin.Context)
	Update(ctx *gin.Context)
	Detail(ctx *gin.Context)
	PageList(ctx *gin.Context)
	UpdatePassword(ctx *gin.Context)
	UpdateStatus(ctx *gin.Context)
	CreateUserIdentity(ctx *gin.Context)
	DeleteUserIdentity(ctx *gin.Context)
	UpdateUserIdentity(ctx *gin.Context)
	DetailUserIdentity(ctx *gin.Context)
	PageListUserIdentity(ctx *gin.Context)
	GetUserIdentityByUser(ctx *gin.Context)
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
}

type userCtr struct {
	userSvc svcuser.UserSvc
}

var _ UserCtr = (*userCtr)(nil)

func NewUserCtr() UserCtr {
	return &userCtr{
		userSvc: svcuser.NewUserSvc(),
	}
}