package router

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/apps/platformadmin/controller/ctrdomain"
	"github.com/morehao/golib/dbaccess/gormdao"
)

func domainRouter(rg *gin.RouterGroup, db gormdao.DBGetter) {
	domainCtr := ctrdomain.NewDomainCtr(db)
	rg.POST("/domains", domainCtr.Create)
	rg.GET("/domains", domainCtr.PageList)
	rg.GET("/domains/:domainID", domainCtr.Detail)
	rg.DELETE("/domains/:domainID", domainCtr.Delete)
	// 动作端点（状态流转只走这里，无通用 PUT 改 status/role/usage）
	rg.POST("/domains/:domainID/verify", domainCtr.Verify)
	rg.POST("/domains/:domainID/tls-ready", domainCtr.TLSReady)
	rg.POST("/domains/:domainID/activate", domainCtr.Activate)
	rg.POST("/domains/:domainID/disable", domainCtr.Disable)
	rg.POST("/domains/:domainID/enable", domainCtr.Enable)
	rg.POST("/domains/:domainID/retire", domainCtr.Retire)
	rg.POST("/domains/:domainID/release", domainCtr.Release)
}
