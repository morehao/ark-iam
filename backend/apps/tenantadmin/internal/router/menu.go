package router

import (
	"github.com/morehao/ark-iam/tenantadmin/internal/controller/ctrtenant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

// tenantMenuRouter 租户侧菜单路由。
// 路径使用 /myMenu/tree，避免与 platformadmin 的 /menu/tree 在 gateway 聚合时冲突。
func tenantMenuRouter(groups *ginserver.RouterGroups) {
	tenantMenuCtr := ctrtenant.NewTenantMenuCtr()

	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	v1RouterGroup.GET("/myMenu/tree", tenantMenuCtr.Tree)
}
