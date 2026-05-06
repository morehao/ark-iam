package router

import (
	"github.com/morehao/ark-iam/iam/internal/controller/ctrmenu"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func menuRouter(groups *ginserver.RouterGroups) {
	menuCtr := ctrmenu.NewMenuCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)

	v1RouterGroup.POST("/permission/create", menuCtr.Create)
	v1RouterGroup.POST("/permission/delete", menuCtr.Delete)
	v1RouterGroup.POST("/permission/update", menuCtr.Update)
	v1RouterGroup.GET("/permission/detail", menuCtr.Detail)
	v1RouterGroup.POST("/permission/pageList", menuCtr.PageList)
	v1RouterGroup.GET("/permission/tree", menuCtr.Tree)
}