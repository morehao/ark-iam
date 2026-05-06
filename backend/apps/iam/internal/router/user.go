package router

import (
	"github.com/morehao/ark-iam/iam/internal/controller/ctruser"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func userRouter(groups *ginserver.RouterGroups) {
	userCtr := ctruser.NewUserCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)

	v1RouterGroup.POST("/user/create", userCtr.Create)
	v1RouterGroup.POST("/user/delete", userCtr.Delete)
	v1RouterGroup.POST("/user/update", userCtr.Update)
	v1RouterGroup.GET("/user/detail", userCtr.Detail)
	v1RouterGroup.POST("/user/pageList", userCtr.PageList)
	v1RouterGroup.POST("/user/updatePassword", userCtr.UpdatePassword)
	v1RouterGroup.POST("/user/updateStatus", userCtr.UpdateStatus)
	v1RouterGroup.POST("/userIdentity/create", userCtr.CreateUserIdentity)
	v1RouterGroup.POST("/userIdentity/delete", userCtr.DeleteUserIdentity)
	v1RouterGroup.POST("/userIdentity/update", userCtr.UpdateUserIdentity)
	v1RouterGroup.GET("/userIdentity/detail", userCtr.DetailUserIdentity)
	v1RouterGroup.POST("/userIdentity/pageList", userCtr.PageListUserIdentity)
	v1RouterGroup.GET("/userIdentity/getByUser", userCtr.GetUserIdentityByUser)
	v1RouterGroup.GET("/userLoginLog/detail", userCtr.DetailUserLoginLog)
	v1RouterGroup.POST("/userLoginLog/pageList", userCtr.PageListUserLoginLog)
	v1RouterGroup.GET("/userLoginLog/getByUser", userCtr.GetUserLoginLogByUser)
	v1RouterGroup.POST("/userDepartmentRelation/create", userCtr.CreateUserDepartmentRelation)
	v1RouterGroup.POST("/userDepartmentRelation/delete", userCtr.DeleteUserDepartmentRelation)
	v1RouterGroup.POST("/userDepartmentRelation/update", userCtr.UpdateUserDepartmentRelation)
	v1RouterGroup.GET("/userDepartmentRelation/detail", userCtr.DetailUserDepartmentRelation)
	v1RouterGroup.POST("/userDepartmentRelation/pageList", userCtr.PageListUserDepartmentRelation)
	v1RouterGroup.GET("/userDepartmentRelation/getByUser", userCtr.GetUserDepartmentRelationByUser)
	v1RouterGroup.POST("/userDepartmentRelation/assignDepartments", userCtr.AssignDepartments)
}