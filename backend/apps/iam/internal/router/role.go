package router

import (
	"github.com/morehao/ark-iam/iam/internal/controller/ctrrole"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func roleRouter(groups *ginserver.RouterGroups) {
	roleCtr := ctrrole.NewRoleCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)

	v1RouterGroup.POST("/role/create", roleCtr.Create)
	v1RouterGroup.POST("/role/delete", roleCtr.Delete)
	v1RouterGroup.POST("/role/update", roleCtr.Update)
	v1RouterGroup.GET("/role/detail", roleCtr.Detail)
	v1RouterGroup.POST("/role/pageList", roleCtr.PageList)
}