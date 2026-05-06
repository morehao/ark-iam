package router

import (
	"github.com/morehao/ark-iam/iam/internal/controller/ctrapplication"
	"github.com/morehao/ark-iam/iam/internal/controller/ctrmenu"
	"github.com/morehao/ark-iam/iam/internal/controller/ctrresource"
	"github.com/morehao/ark-iam/iam/internal/controller/ctrrole"
	"github.com/morehao/ark-iam/iam/internal/controller/ctrrolemenu"
	"github.com/morehao/ark-iam/iam/internal/controller/ctrrolescope"
	"github.com/morehao/ark-iam/iam/internal/controller/ctrscope"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func permissionRouter(groups *ginserver.RouterGroups) {
	roleCtr := ctrrole.NewRoleCtr()
	menuCtr := ctrmenu.NewMenuCtr()
	scopeCtr := ctrscope.NewScopeCtr()
	appCtr := ctrapplication.NewApplicationCtr()
	resourceCtr := ctrresource.NewResourceCtr()
	roleMenuCtr := ctrrolemenu.NewRoleMenuCtr()
	roleScopeCtr := ctrrolescope.NewRoleScopeCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)

	v1RouterGroup.POST("/permission/role/create", roleCtr.Create)
	v1RouterGroup.POST("/permission/role/delete", roleCtr.Delete)
	v1RouterGroup.POST("/permission/role/update", roleCtr.Update)
	v1RouterGroup.GET("/permission/role/detail", roleCtr.Detail)
	v1RouterGroup.POST("/permission/role/pageList", roleCtr.PageList)

	v1RouterGroup.POST("/permission/menu/create", menuCtr.Create)
	v1RouterGroup.POST("/permission/menu/delete", menuCtr.Delete)
	v1RouterGroup.POST("/permission/menu/update", menuCtr.Update)
	v1RouterGroup.GET("/permission/menu/detail", menuCtr.Detail)
	v1RouterGroup.POST("/permission/menu/pageList", menuCtr.PageList)
	v1RouterGroup.GET("/permission/menu/tree", menuCtr.Tree)

	v1RouterGroup.POST("/permission/scope/create", scopeCtr.Create)
	v1RouterGroup.POST("/permission/scope/delete", scopeCtr.Delete)
	v1RouterGroup.POST("/permission/scope/update", scopeCtr.Update)
	v1RouterGroup.GET("/permission/scope/detail", scopeCtr.Detail)
	v1RouterGroup.POST("/permission/scope/pageList", scopeCtr.PageList)

	v1RouterGroup.POST("/permission/application/create", appCtr.Create)
	v1RouterGroup.POST("/permission/application/delete", appCtr.Delete)
	v1RouterGroup.POST("/permission/application/update", appCtr.Update)
	v1RouterGroup.GET("/permission/application/detail", appCtr.Detail)
	v1RouterGroup.POST("/permission/application/pageList", appCtr.PageList)

	v1RouterGroup.POST("/permission/resource/create", resourceCtr.Create)
	v1RouterGroup.POST("/permission/resource/delete", resourceCtr.Delete)
	v1RouterGroup.POST("/permission/resource/update", resourceCtr.Update)
	v1RouterGroup.GET("/permission/resource/detail", resourceCtr.Detail)
	v1RouterGroup.POST("/permission/resource/pageList", resourceCtr.PageList)

	v1RouterGroup.POST("/permission/roleMenu/create", roleMenuCtr.Create)
	v1RouterGroup.POST("/permission/roleMenu/delete", roleMenuCtr.Delete)
	v1RouterGroup.POST("/permission/roleMenu/pageList", roleMenuCtr.PageList)

	v1RouterGroup.POST("/permission/roleScope/create", roleScopeCtr.Create)
	v1RouterGroup.POST("/permission/roleScope/delete", roleScopeCtr.Delete)
	v1RouterGroup.POST("/permission/roleScope/pageList", roleScopeCtr.PageList)
}