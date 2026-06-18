package router

import (
	"github.com/morehao/ark-iam/platformadmin/internal/controller/ctrsession"
	"github.com/morehao/ark-iam/platformadmin/internal/controller/ctruser"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func userRouter(groups *ginserver.RouterGroups) {
	userCtr := ctruser.NewUserCtr()
	sessionCtr := ctrsession.NewSessionCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)

	v1RouterGroup.POST("/platformadmin/user/create", userCtr.Create)
	v1RouterGroup.POST("/platformadmin/user/delete", userCtr.Delete)
	v1RouterGroup.POST("/platformadmin/user/update", userCtr.Update)
	v1RouterGroup.GET("/platformadmin/user/detail", userCtr.Detail)
	v1RouterGroup.POST("/platformadmin/user/pageList", userCtr.PageList)
	v1RouterGroup.POST("/platformadmin/user/updatePassword", userCtr.UpdatePassword)
	v1RouterGroup.POST("/platformadmin/user/updateStatus", userCtr.UpdateStatus)

	v1RouterGroup.POST("/platformadmin/user/createUserIdentity", userCtr.CreateUserIdentity)
	v1RouterGroup.POST("/platformadmin/user/deleteUserIdentity", userCtr.DeleteUserIdentity)
	v1RouterGroup.POST("/platformadmin/user/updateUserIdentity", userCtr.UpdateUserIdentity)
	v1RouterGroup.GET("/platformadmin/user/detailUserIdentity", userCtr.DetailUserIdentity)
	v1RouterGroup.POST("/platformadmin/user/pageListUserIdentity", userCtr.PageListUserIdentity)
	v1RouterGroup.GET("/platformadmin/user/getUserIdentityByUser", userCtr.GetUserIdentityByUser)

	v1RouterGroup.GET("/platformadmin/user/detailUserLoginLog", userCtr.DetailUserLoginLog)
	v1RouterGroup.POST("/platformadmin/user/pageListUserLoginLog", userCtr.PageListUserLoginLog)
	v1RouterGroup.GET("/platformadmin/user/getUserLoginLogByUser", userCtr.GetUserLoginLogByUser)

	v1RouterGroup.GET("/platformadmin/user/getUserDepartmentByUser", userCtr.GetUserDepartmentByUser)
	v1RouterGroup.POST("/platformadmin/user/assignDepartments", userCtr.AssignDepartments)

	v1RouterGroup.GET("/platformadmin/user/sessions", sessionCtr.List)
	v1RouterGroup.DELETE("/platformadmin/user/sessions", sessionCtr.RevokeAll)
	v1RouterGroup.DELETE("/platformadmin/user/sessions/:sessionId", sessionCtr.Revoke)
}
