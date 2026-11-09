package iam

import (
	"testing"

	"github.com/gin-gonic/gin"

	_ "github.com/morehao/golib/dbaccess/dbgorm/driver/sqlite"
	_ "github.com/morehao/golib/glog/driver/zap"
)

// hasRoute 检查 engine 中是否注册了给定方法+路径的路由。
func hasRoute(routes gin.RoutesInfo, method, path string) bool {
	for _, r := range routes {
		if r.Method == method && r.Path == path {
			return true
		}
	}
	return false
}

// TestMountPlatformDomains 校验按域选择挂载：选中的 menu 域路由可达、未选的 tenant 域不可达。
func TestMountPlatformDomains(t *testing.T) {
	app, err := New(Config{
		DB:      DBConfig{DSN: "sqlite://:memory:"},
		Domains: []Domain{DomainMenu},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	if err := app.Mount(engine.Group("/iam")); err != nil {
		t.Fatalf("Mount: %v", err)
	}

	routes := engine.Routes()

	// 选中的 menu 域应可达（/iam/v1/platform/menus）
	if !hasRoute(routes, "GET", "/iam/v1/platform/menus/tree") {
		t.Fatalf("menu tree route not mounted; routes=%v", dumpRoutes(routes))
	}

	// 未选择的 tenant 域应不可达
	if hasRoute(routes, "GET", "/iam/v1/platform/tenants") {
		t.Fatal("tenant route should NOT be mounted when domain not selected")
	}
}

func dumpRoutes(routes gin.RoutesInfo) []string {
	out := make([]string, 0, len(routes))
	for _, r := range routes {
		out = append(out, r.Method+" "+r.Path)
	}
	return out
}

// TestMountTenantDomains 校验 tenantadmin 域按域挂载：选中的 user/department 可达，
// 未选的 machine-user 不可达。
func TestMountTenantDomains(t *testing.T) {
	app, err := New(Config{
		DB:      DBConfig{DSN: "sqlite://:memory:"},
		Domains: []Domain{DomainUser, DomainDepartment},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	if err := app.Mount(engine.Group("/iam")); err != nil {
		t.Fatalf("Mount: %v", err)
	}
	routes := engine.Routes()
	if !hasRoute(routes, "GET", "/iam/v1/tenant/users") {
		t.Fatalf("tenant user route not mounted; routes=%v", dumpRoutes(routes))
	}
	if !hasRoute(routes, "GET", "/iam/v1/tenant/departments/tree") {
		t.Fatalf("tenant department route not mounted")
	}
	if hasRoute(routes, "GET", "/iam/v1/tenant/machine-users") {
		t.Fatal("machine-user route should not be mounted")
	}
}
