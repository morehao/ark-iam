package router

import (
	"github.com/morehao/ark-iam/platformadmin/internal/controller/ctrapplicationclient"
	"github.com/morehao/ark-iam/platformadmin/internal/controller/ctrpermission"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

// 角色（role）不再有平台端入口：角色按租户归属，读写与成员管理全部收敛到 tenantadmin，
// 平台侧不再提供跨租户角色只读视图，避免绕过租户授权边界。
func menuRouter(groups *ginserver.RouterGroups) {
	menuCtr := ctrpermission.NewMenuCtr()
	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	v1RouterGroup.POST("/menus", menuCtr.Create)
	v1RouterGroup.GET("/menus", menuCtr.PageList)
	v1RouterGroup.GET("/menus/tree", menuCtr.Tree)
	v1RouterGroup.GET("/menus/my", menuCtr.MyTree)
	v1RouterGroup.GET("/menus/:menuID", menuCtr.Detail)
	v1RouterGroup.PUT("/menus/:menuID", menuCtr.Update)
	v1RouterGroup.DELETE("/menus/:menuID", menuCtr.Delete)
}

func applicationClientRouter(groups *ginserver.RouterGroups) {
	appCtr := ctrapplicationclient.NewApplicationClientCtr()
	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	v1RouterGroup.POST("/application-clients", appCtr.Create)
	v1RouterGroup.GET("/application-clients", appCtr.PageList)
	v1RouterGroup.GET("/application-clients/:applicationClientID", appCtr.Detail)
	v1RouterGroup.PUT("/application-clients/:applicationClientID", appCtr.Update)
	v1RouterGroup.DELETE("/application-clients/:applicationClientID", appCtr.Delete)
	// 客户端密钥（子资源）
	v1RouterGroup.GET("/application-clients/:applicationClientID/secrets", appCtr.ListSecrets)
	v1RouterGroup.POST("/application-clients/:applicationClientID/secrets", appCtr.CreateSecret)
	v1RouterGroup.DELETE("/application-clients/:applicationClientID/secrets/:secretID", appCtr.DeleteSecret)
}
