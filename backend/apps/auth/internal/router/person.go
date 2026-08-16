package router

import (
	"github.com/morehao/ark-iam/auth/internal/controller/ctrperson"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func personRouter(groups *ginserver.RouterGroups) {
	personCtr := ctrperson.NewPersonCtr()
	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	v1RouterGroup.GET("/me", personCtr.Detail)
	v1RouterGroup.POST("/me/changePassword", personCtr.UpdatePassword)
}
