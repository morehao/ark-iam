package server

import (
	"context"
	"crypto/rsa"
	"errors"
	"testing"

	"github.com/gin-gonic/gin"

	_ "github.com/morehao/golib/dbaccess/dbgorm/driver/sqlite"
	_ "github.com/morehao/golib/glog/driver/zap"

	pkconfig "github.com/morehao/ark-iam/pkg/config"
	"github.com/morehao/ark-iam/pkg/model"
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

func dumpRoutes(routes gin.RoutesInfo) []string {
	out := make([]string, 0, len(routes))
	for _, r := range routes {
		out = append(out, r.Method+" "+r.Path)
	}
	return out
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
	if !hasRoute(routes, "GET", "/iam/v1/platform/menus/tree") {
		t.Fatalf("menu tree route not mounted; routes=%v", dumpRoutes(routes))
	}
	if hasRoute(routes, "GET", "/iam/v1/platform/tenants") {
		t.Fatal("tenant route should NOT be mounted when domain not selected")
	}
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

// TestBackChannelLogoutDefaultPathMatchesSeedConstant 钉住：本进程真实挂载的 back-channel
// logout 接收端默认路径 = pkg/model 常量（与 L1 播种写进 application_client 的派生结果一致）。
//
// 两者不一致的后果是**静默的**：OP 登出时把令牌 POST 到 404 地址，全局登出整体失效。
func TestBackChannelLogoutDefaultPathMatchesSeedConstant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := &App{cfg: Config{}, keySource: stubKeySource{}}
	e := gin.New()
	app.registerBackChannelLogout(e.Group(""), &pkconfig.Config{}, model.SeedBackChannelLogoutPathPlatform, model.SeedBuiltinClientPlatformAdminWeb)

	want := "/oidc" + model.SeedBackChannelLogoutPathPlatform
	if !hasRoute(e.Routes(), "POST", want) {
		t.Fatalf("back-channel logout 接收端未挂在预期路径: want POST %s, 实际路由=%v", want, dumpRoutes(e.Routes()))
	}
}

// stubKeySource 是 KeySource 的空实现（仅用于挂载路由，不实际验签）。
type stubKeySource struct{}

func (stubKeySource) PublicKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	return nil, errors.New("stub key source: no key")
}
