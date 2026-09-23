package tenantadmin

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"

	pkgconfig "github.com/morehao/ark-iam/pkg/config"
	"github.com/morehao/ark-iam/pkg/model"
)

// 同 platformadmin/app_test.go：接收端挂载路径必须与 L1 播种写进
// application_client.back_channel_logout_uri 的路径一致，否则全局登出静默失效。
// 两个应用各钉一条，因为它们是两份独立的默认值（platform / tenant 子路径）。
func TestBackChannelLogoutDefaultPathMatchesSeedConstant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	registerBackChannelLogout(engine, &pkgconfig.Config{}, nil)

	want := "/oidc" + model.SeedBackChannelLogoutPathTenant
	if !routeMounted(engine, http.MethodPost, want) {
		t.Fatalf("back-channel logout 接收端未挂在预期路径: want POST %s, 实际路由=%v", want, routePaths(engine))
	}
	// 防御性：租户端不能挂到平台端的子路径上（gateway 聚合部署时两者同进程，
	// 撞路径会让其中一个应用的路由被另一个覆盖）
	if platformPath := "/oidc" + model.SeedBackChannelLogoutPathPlatform; platformPath == want {
		t.Fatalf("平台与租户的 back-channel logout 子路径不应相同: %s", platformPath)
	}
}

func TestBackChannelLogoutPathOverridable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	registerBackChannelLogout(engine, &pkgconfig.Config{
		OIDC: pkgconfig.OIDC{BackChannelLogoutPath: "/bc-logout/custom"},
	}, nil)

	if !routeMounted(engine, http.MethodPost, "/oidc/bc-logout/custom") {
		t.Fatalf("分体部署的自定义路径未生效: 实际路由=%v", routePaths(engine))
	}
	if routeMounted(engine, http.MethodPost, "/oidc"+model.SeedBackChannelLogoutPathTenant) {
		t.Fatalf("覆盖后仍挂载了默认路径: %v", routePaths(engine))
	}
}

func TestBackChannelLogoutNilConfigIsNoop(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	registerBackChannelLogout(engine, nil, nil)
	if len(engine.Routes()) != 0 {
		t.Fatalf("Conf 为 nil 时不应注册任何路由: %v", routePaths(engine))
	}
}

func routeMounted(engine *gin.Engine, method, path string) bool {
	for _, route := range engine.Routes() {
		if route.Method == method && route.Path == path {
			return true
		}
	}
	return false
}

func routePaths(engine *gin.Engine) []string {
	paths := make([]string, 0, len(engine.Routes()))
	for _, route := range engine.Routes() {
		paths = append(paths, route.Method+" "+route.Path)
	}
	return paths
}
