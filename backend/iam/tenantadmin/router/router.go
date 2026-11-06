package router

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/golib/dbaccess/gormdao"
)

// DomainFunc 是单个业务域的路由注册函数签名。
type DomainFunc func(rg *gin.RouterGroup, db gormdao.DBGetter)

// Registry 把 tenantadmin 的各业务域映射到注册函数，供 iam.App 按域选择性挂载。
//
// 域常量见 iam.Domain：DomainUser / DomainDepartment / DomainRole / DomainMenu /
// DomainMachineUser / DomainInvite / DomainAPIKey / DomainApplication。
var Registry = map[string]DomainFunc{
	"user":         userRouter,
	"department":   departmentRouter,
	"role":         roleRouter,
	"menu":         tenantMenuRouter,
	"machine-user": machineUserRouter,
	"invite":       inviteRouter,
	"api-key":      apiKeyRouter,
	"application":  applicationRouter,
}
