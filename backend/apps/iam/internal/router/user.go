package router

import (
	"github.com/morehao/ark-iam/iam/internal/controller/ctrsession"
	"github.com/morehao/ark-iam/iam/internal/controller/ctruser"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func userRouter(groups *ginserver.RouterGroups) {
	userCtr := ctruser.NewUserCtr()
	sessionCtr := ctrsession.NewSessionCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)

	v1RouterGroup.POST("/user/create", userCtr.Create)
	v1RouterGroup.POST("/user/delete", userCtr.Delete)
	v1RouterGroup.POST("/user/update", userCtr.Update)
	v1RouterGroup.GET("/user/detail", userCtr.Detail)
	v1RouterGroup.POST("/user/pageList", userCtr.PageList)
	v1RouterGroup.POST("/user/updatePassword", userCtr.UpdatePassword)
	v1RouterGroup.POST("/user/updateStatus", userCtr.UpdateStatus)

	v1RouterGroup.POST("/user/createUserIdentity", userCtr.CreateUserIdentity)
	v1RouterGroup.POST("/user/deleteUserIdentity", userCtr.DeleteUserIdentity)
	v1RouterGroup.POST("/user/updateUserIdentity", userCtr.UpdateUserIdentity)
	v1RouterGroup.GET("/user/detailUserIdentity", userCtr.DetailUserIdentity)
	v1RouterGroup.POST("/user/pageListUserIdentity", userCtr.PageListUserIdentity)
	v1RouterGroup.GET("/user/getUserIdentityByUser", userCtr.GetUserIdentityByUser)

	v1RouterGroup.GET("/user/detailUserLoginLog", userCtr.DetailUserLoginLog)
	v1RouterGroup.POST("/user/pageListUserLoginLog", userCtr.PageListUserLoginLog)
	v1RouterGroup.GET("/user/getUserLoginLogByUser", userCtr.GetUserLoginLogByUser)

	v1RouterGroup.GET("/user/getUserDepartmentByUser", userCtr.GetUserDepartmentByUser)
	v1RouterGroup.POST("/user/assignDepartments", userCtr.AssignDepartments)

	v1RouterGroup.GET("/user/sessions", sessionCtr.List)
	v1RouterGroup.DELETE("/user/sessions", sessionCtr.RevokeAll)
	v1RouterGroup.DELETE("/user/sessions/:sessionId", sessionCtr.Revoke)
}
