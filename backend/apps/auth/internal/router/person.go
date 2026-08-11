package router

import (
	"github.com/morehao/ark-iam/auth/internal/controller/ctrperson"
	"github.com/morehao/ark-iam/auth/internal/service/svcperson"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func personRouter(groups *ginserver.RouterGroups) {
	personCtr := ctrperson.NewPersonCtr(svcperson.NewPersonProfileSvc())

	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	v1RouterGroup.GET("/person/detail", personCtr.Detail)
	v1RouterGroup.POST("/person/updatePassword", personCtr.UpdatePassword)
}
