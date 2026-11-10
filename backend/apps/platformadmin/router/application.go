package router

import (
	"github.com/morehao/ark-iam/platformadmin/controller/ctrapplication"
	"github.com/gin-gonic/gin"
	"github.com/morehao/golib/dbaccess/gormdao"
)

func applicationRouter(rg *gin.RouterGroup, db gormdao.DBGetter) {
	ctr := ctrapplication.NewApplicationCtr(db)
	rg.POST("/applications", ctr.Create)
	rg.GET("/applications", ctr.PageList)
	rg.GET("/applications/:appID", ctr.Detail)
	rg.PUT("/applications/:appID", ctr.Update)
	rg.DELETE("/applications/:appID", ctr.Delete)
}
