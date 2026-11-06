package router

import (
	"github.com/morehao/ark-iam/iam/tenantadmin/controller/ctrtenant"
	"github.com/gin-gonic/gin"
	"github.com/morehao/golib/dbaccess/gormdao"
)

func apiKeyRouter(rg *gin.RouterGroup, db gormdao.DBGetter) {
	apiKeyCtr := ctrtenant.NewApiKeyCtr(db)

	rg.POST("/api-keys", apiKeyCtr.Create)
	rg.GET("/api-keys", apiKeyCtr.PageList)
	rg.POST("/api-keys/:apiKeyID/revoke", apiKeyCtr.Revoke)
	rg.DELETE("/api-keys/:apiKeyID", apiKeyCtr.Delete)
}
