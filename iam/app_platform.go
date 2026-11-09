package iam

import (
	"fmt"

	"github.com/gin-gonic/gin"
	platrouter "github.com/morehao/ark-iam/iam/platformadmin/router"
)

// mountPlatformAdmin 按 App 选中的域，把 platformadmin 的业务域挂到 rg 上。
//
// platformadmin 目前承载的域：menu / tenant / application / oauth-client / domain /
// tenant-app / audit。这些路由挂在 /v1/platform 前缀下（与现网一致），
// 因此调用方传入的 rg 应是已经带 /v1/platform 的组；见 App.Mount 的约定。
func (a *App) mountPlatformAdmin(rg *gin.RouterGroup, keep map[Domain]bool) {
	domainKey := map[Domain]string{
		DomainMenu:        "menu",
		DomainTenant:      "tenant",
		DomainApplication: "application",
		DomainOAuthClient: "oauth-client",
		DomainDomain:      "domain",
		DomainTenantApp:   "tenant-app",
		DomainAudit:       "audit",
	}
	for domain, key := range domainKey {
		if !keep[domain] {
			continue
		}
		fn, ok := platrouter.Registry[key]
		if !ok {
			continue
		}
		fn(rg, a.dbGetter)
	}
}

var _ = fmt.Sprintf
