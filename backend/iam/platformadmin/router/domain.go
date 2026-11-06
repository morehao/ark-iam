package router

import (
	"github.com/morehao/ark-iam/iam/platformadmin/controller/ctrdomain"
	"github.com/gin-gonic/gin"
	"github.com/morehao/golib/dbaccess/gormdao"
)

func domainRouter(rg *gin.RouterGroup, db gormdao.DBGetter) {
	domainCtr := ctrdomain.NewDomainCtr(db)
	rg.POST("/domains", domainCtr.Create)
	rg.GET("/domains", domainCtr.PageList)
	rg.GET("/domains/:domainID", domainCtr.Detail)
	rg.PUT("/domains/:domainID", domainCtr.Update)
	rg.DELETE("/domains/:domainID", domainCtr.Delete)
}
