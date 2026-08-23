package router

import (
	"github.com/morehao/ark-iam/tenantadmin/internal/controller/ctrtenant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func inviteRouter(groups *ginserver.RouterGroups) {
	inviteCtr := ctrtenant.NewInviteCtr()
	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	v1RouterGroup.POST("/invites", inviteCtr.Create)
	v1RouterGroup.GET("/invites", inviteCtr.PageList)
	v1RouterGroup.DELETE("/invites/:inviteID", inviteCtr.Revoke)
}
