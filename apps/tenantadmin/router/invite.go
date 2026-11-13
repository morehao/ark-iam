package router

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/apps/tenantadmin/controller/ctrtenant"
	"github.com/morehao/golib/dbaccess/gormdao"
)

func inviteRouter(rg *gin.RouterGroup, db gormdao.DBGetter) {
	inviteCtr := ctrtenant.NewInviteCtr(db)
	rg.POST("/invites", inviteCtr.Create)
	rg.GET("/invites", inviteCtr.PageList)
	rg.DELETE("/invites/:inviteID", inviteCtr.Revoke)
}
