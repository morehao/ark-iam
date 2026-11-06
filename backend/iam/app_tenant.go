package iam

import (
	"github.com/gin-gonic/gin"
	tenantrouter "github.com/morehao/ark-iam/iam/tenantadmin/router"
)

// mountTenantAdmin 按 App 选中的域，把 tenantadmin 的业务域挂到 rg 上（/v1/tenant 前缀）。
func (a *App) mountTenantAdmin(rg *gin.RouterGroup, keep map[Domain]bool) {
	domainKey := map[Domain]string{
		DomainUser:        "user",
		DomainDepartment:  "department",
		DomainRole:        "role",
		DomainMenu:        "menu",   // tenantadmin 的菜单读树（菜单写仍在 platform，R3 说明）
		DomainMachineUser: "machine-user",
		DomainInvite:      "invite",
		DomainAPIKey:      "api-key",
		DomainApplication:  "application",
	}
	for domain, key := range domainKey {
		if !keep[domain] {
			continue
		}
		fn, ok := tenantrouter.Registry[key]
		if !ok {
			continue
		}
		fn(rg, a.dbGetter)
	}
}
