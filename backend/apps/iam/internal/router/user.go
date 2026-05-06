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

	v1RouterGroup.POST("/user/identity/create", userCtr.CreateUserIdentity)
	v1RouterGroup.POST("/user/identity/delete", userCtr.DeleteUserIdentity)
	v1RouterGroup.POST("/user/identity/update", userCtr.UpdateUserIdentity)
	v1RouterGroup.GET("/user/identity/detail", userCtr.DetailUserIdentity)
	v1RouterGroup.POST("/user/identity/pageList", userCtr.PageListUserIdentity)
	v1RouterGroup.GET("/user/identity/getByUser", userCtr.GetUserIdentityByUser)

	v1RouterGroup.GET("/user/login-log/detail", userCtr.DetailUserLoginLog)
	v1RouterGroup.POST("/user/login-log/pageList", userCtr.PageListUserLoginLog)
	v1RouterGroup.GET("/user/login-log/getByUser", userCtr.GetUserLoginLogByUser)

	v1RouterGroup.POST("/user/department/create", userCtr.CreateUserDepartmentRelation)
	v1RouterGroup.POST("/user/department/delete", userCtr.DeleteUserDepartmentRelation)
	v1RouterGroup.POST("/user/department/update", userCtr.UpdateUserDepartmentRelation)
	v1RouterGroup.GET("/user/department/detail", userCtr.DetailUserDepartmentRelation)
	v1RouterGroup.POST("/user/department/pageList", userCtr.PageListUserDepartmentRelation)
	v1RouterGroup.GET("/user/department/getByUser", userCtr.GetUserDepartmentRelationByUser)
	v1RouterGroup.POST("/user/department/assign", userCtr.AssignDepartments)
}
