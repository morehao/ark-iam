package router

import (
	"github.com/morehao/ark-iam/tenantadmin/controller/ctrtenant"
	"github.com/gin-gonic/gin"
	"github.com/morehao/golib/dbaccess/gormdao"
)

// applicationRouter 租户应用资源路由：
// 应用清单由原 /apps 收敛为 REST 资源名 /applications（前后端同仓发布，旧路由直接删除），
// 并提供应用视角的准入成员列表；成员视角的准入读写挂在 /users/{userID}/applications（router/user.go）。
func applicationRouter(rg *gin.RouterGroup, db gormdao.DBGetter) {
	applicationCtr := ctrtenant.NewApplicationCtr(db)

	rg.GET("/applications", applicationCtr.List)
	rg.GET("/applications/:appID/users", applicationCtr.ListUsers)
}
