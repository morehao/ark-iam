package platformadmin

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"

	pkgconfig "github.com/morehao/ark-iam/pkg/config"
	"github.com/morehao/ark-iam/pkg/model"
)

// back-channel logout 有两处必须一致的路径：
//
//	① 本进程真实挂载的接收端路由（本文件被测的 registerBackChannelLogout）
//	② L1 首次引导时写进 application_client.back_channel_logout_uri 的派生结果
//	   （pkg/seed，取自同一个 model 常量）
//
// 两者不一致的后果是**静默的**：OP 登出时把令牌 POST 到一个 404 地址，全局登出
// 整体失效，而控制台看不出任何异常。改造前两处各写一份字面量，只靠"碰巧相同"维系。
// 这里把"默认路径 = pkg/model 常量"钉成断言，使它成为机制而不是约定。
func TestBackChannelLogoutDefaultPathMatchesSeedConstant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	registerBackChannelLogout(engine, &pkgconfig.Config{}, nil)

	want := "/oidc" + model.SeedBackChannelLogoutPathPlatform
	if !routeMounted(engine, http.MethodPost, want) {
		t.Fatalf("back-channel logout 接收端未挂在预期路径: want POST %s, 实际路由=%v", want, routePaths(engine))
	}
}

// 分体部署（auth 与 platformadmin 不同源）时，issuer + 默认路径会算出一个不可达的接收端，
// 因此必须能显式覆盖——这条断言防的是"以后有人把默认值改成不可覆盖"。
func TestBackChannelLogoutPathOverridable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	registerBackChannelLogout(engine, &pkgconfig.Config{
		OIDC: pkgconfig.OIDC{BackChannelLogoutPath: "/bc-logout/custom"},
	}, nil)

	if !routeMounted(engine, http.MethodPost, "/oidc/bc-logout/custom") {
		t.Fatalf("分体部署的自定义路径未生效: 实际路由=%v", routePaths(engine))
	}
	// 覆盖生效时不应再挂载默认路径（否则等于埋了一个无人知晓的第二个接收端）
	if routeMounted(engine, http.MethodPost, "/oidc"+model.SeedBackChannelLogoutPathPlatform) {
		t.Fatalf("覆盖后仍挂载了默认路径: %v", routePaths(engine))
	}
}

// Conf 为 nil 时直接返回（独立部署/测试环境可能没有配置）
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
