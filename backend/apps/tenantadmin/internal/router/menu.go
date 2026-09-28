package router

import (
	"github.com/morehao/ark-iam/tenantadmin/internal/controller/ctrtenant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

// tenantMenuRouter 租户侧菜单路由。
// service 段 /v1/tenant 已与 /v1/platform/menus/tree 区分归属。
func tenantMenuRouter(groups *ginserver.RouterGroups) {
	tenantMenuCtr := ctrtenant.NewTenantMenuCtr()

	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	v1RouterGroup.GET("/menus/tree", tenantMenuCtr.Tree)
}
