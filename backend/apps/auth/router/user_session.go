package router

import (
	"github.com/morehao/ark-iam/backend/apps/auth/controller/ctrsession"
	"github.com/gin-gonic/gin"
	"github.com/morehao/golib/dbaccess/gormdao"
)

func userSessionRouter(rg *gin.RouterGroup, db gormdao.DBGetter) {
	sessionCtr := ctrsession.NewSessionCtr(db)
	rg.GET("/me/sessions", sessionCtr.List)
	rg.DELETE("/me/sessions", sessionCtr.RevokeAll)
	rg.DELETE("/me/sessions/:sessionID", sessionCtr.Revoke)
}
