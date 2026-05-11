package router

import (
	"github.com/morehao/ark-iam/iam/config"
	"github.com/morehao/ark-iam/iam/internal/controller/ctrperson"
	"github.com/morehao/ark-iam/iam/internal/service/svcauth"
	"github.com/morehao/ark-iam/iam/internal/service/svcperson"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func personRouter(groups *ginserver.RouterGroups) {
	authSvc := svcauth.NewAuthSvc(config.Conf.JWT.SignKey)
	personCtr := ctrperson.NewPersonCtr(
		svcperson.NewPersonProfileSvc(),
		authSvc,
	)

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.GET("/person/detail", personCtr.Detail)
	v1RouterGroup.POST("/person/updatePassword", personCtr.UpdatePassword)
}