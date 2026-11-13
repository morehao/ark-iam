package router

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/apps/platformadmin/controller/ctrtenantapplication"
	"github.com/morehao/golib/dbaccess/gormdao"
)

func tenantApplicationRouter(rg *gin.RouterGroup, db gormdao.DBGetter) {
	ctr := ctrtenantapplication.NewTenantApplicationCtr(db)
	rg.POST("/tenant-applications", ctr.Create)
	rg.GET("/tenant-applications", ctr.PageList)
	rg.GET("/tenant-applications/:tenantAppID", ctr.Detail)
	rg.PUT("/tenant-applications/:tenantAppID", ctr.Update)
	rg.DELETE("/tenant-applications/:tenantAppID", ctr.Delete)
}
