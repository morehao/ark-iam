package router

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/apps/platformadmin/controller/ctrapplicationclient"
	"github.com/morehao/ark-iam/apps/platformadmin/controller/ctrpermission"
	"github.com/morehao/golib/dbaccess/gormdao"
)

// 角色（role）不再有平台端入口：角色按租户归属，读写与成员管理全部收敛到 tenantadmin，
// 平台侧不再提供跨租户角色只读视图，避免绕过租户授权边界。
//
// 菜单：平台侧只保留只读（PageList/Tree/Detail/MyTree）。菜单写能力已下放归属租户
// （tenantadmin /v1/tenant/applications/{appID}/menus），平台不再维护任何应用菜单写。
func menuRouter(rg *gin.RouterGroup, db gormdao.DBGetter) {
	menuCtr := ctrpermission.NewMenuCtr(db)
	rg.GET("/menus", menuCtr.PageList)
	rg.GET("/menus/tree", menuCtr.Tree)
	rg.GET("/menus/my", menuCtr.MyTree)
	rg.GET("/menus/:menuID", menuCtr.Detail)
}

func applicationClientRouter(rg *gin.RouterGroup, db gormdao.DBGetter) {
	appCtr := ctrapplicationclient.NewApplicationClientCtr(db)
	rg.POST("/application-clients", appCtr.Create)
	rg.GET("/application-clients", appCtr.PageList)
	rg.GET("/application-clients/:applicationClientID", appCtr.Detail)
	rg.PUT("/application-clients/:applicationClientID", appCtr.Update)
	rg.DELETE("/application-clients/:applicationClientID", appCtr.Delete)
	// 客户端密钥（子资源）
	rg.GET("/application-clients/:applicationClientID/secrets", appCtr.ListSecrets)
	rg.POST("/application-clients/:applicationClientID/secrets", appCtr.CreateSecret)
	rg.DELETE("/application-clients/:applicationClientID/secrets/:secretID", appCtr.DeleteSecret)
}
