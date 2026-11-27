package appassembly

// 双 Host 端到端回归（W8 e2e）：不是浏览器 Playwright，而是真实装配栈 + 真实 Gin 引擎的进程内 e2e，
// 可在 CI 无浏览器依赖下运行。覆盖 W2/W4 的 Host→binding 路由核心：
//   - 平台 Host（kind=platform binding）与 D Host（kind=tenant binding）各自解析出不同 issuer；
//   - /config.json 按 Host 返回对应 binding 的 issuer/clientId；
//   - D Host 访问 /v1/platform/* 被 Host 白名单 404 拒绝，平台 Host 放行；
//   - 未知 Host → 404（不回退中央）。
//
// 通过 New → RunMigrations → install.Initialize（引导平台租户 + 平台 binding/console client）建立真实环境，
// 再手工 seed 一个 D 租户 + active primary 域 + 启用 tenant binding，模拟 D onboarding 完成态。

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	_ "github.com/morehao/golib/dbaccess/dbgorm/driver/sqlite"

	authconfig "github.com/morehao/ark-iam/apps/auth/config"
	"github.com/morehao/ark-iam/apps/auth/dto/dtoinstall"
	"github.com/morehao/ark-iam/apps/auth/service/svcdb"
	"github.com/morehao/ark-iam/apps/auth/service/svcinstall"
	pkconfig "github.com/morehao/ark-iam/pkg/config"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
)

const (
	e2ePlatformIssuer = "http://platform.test/oidc"
	e2eDHost          = "login.acme.test"
	e2eDIssuer        = "https://" + e2eDHost + "/oidc"
)

// bootDualHostApp 建立真实装配栈（sqlite 内存），install 引导后 seed 一个 D 租户/域/binding。
func bootDualHostApp(t *testing.T) *App {
	t.Helper()
	raw := &pkconfig.Config{
		OIDC: pkconfig.OIDC{Issuer: e2ePlatformIssuer, FrontendLoginURL: e2ePlatformIssuer + "/login"},
	}
	app, err := New(Config{
		Env:     "dev",
		DB:      DBConfig{DSN: "sqlite://:memory:"},
		Domains: DefaultConsoleDomains,
		Raw:     raw,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = app.Close() })
	authconfig.Set(raw)
	svcdb.Set(app.DBGetter())

	ctx := context.Background()
	if err := app.RunMigrations(ctx); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	if _, err := svcinstall.NewInstallSvc(app.DBGetter()).Initialize(ctx, &dtoinstall.InstallationInitializeReq{
		AdminUsername: "admin", AdminPassword: "Adm1n@Strong!Pass", AdminEmail: "admin@example.com", TenantName: "Acme",
	}); err != nil {
		t.Fatalf("install.Initialize: %v", err)
	}

	// seed D 租户（dedicated/active）+ active primary 域 + enable tenant binding。
	seedDedicatedTenant(t, app, ctx)
	return app
}

func seedDedicatedTenant(t *testing.T, app *App, ctx context.Context) {
	t.Helper()
	db := app.DBGetter()(ctx)
	cross := dbclient.CrossTenantContext(ctx)

	tn := &model.TenantEntity{Status: model.TenantStatusActive, IsolationMode: model.TenantIsolationDedicated, Issuer: e2eDIssuer}
	tn.ID = "td_e2e_1"
	if err := db.WithContext(cross).Create(tn).Error; err != nil {
		t.Fatalf("seed D tenant: %v", err)
	}
	// primary 域（active）
	dom := &model.DomainEntity{
		TenantID: tn.ID, Domain: e2eDHost, DomainRole: model.DomainRolePrimary,
		DomainUsage: model.DomainUsageLogin, Status: model.DomainStatusActive,
	}
	dom.ID = "dom_e2e_1"
	if err := db.WithContext(dbclient.ExplicitTenantContext(ctx, tn.ID)).Create(dom).Error; err != nil {
		t.Fatalf("seed D primary: %v", err)
	}
	// enable 的 tenant binding
	cb := &model.ConsoleBindingEntity{
		Host: e2eDHost, TenantID: tn.ID, Issuer: e2eDIssuer, ClientID: "tenant_console",
		Audience: "tenant_admin_web", Kind: model.ConsoleBindingKindTenant, Status: model.ConsoleBindingStatusEnable,
	}
	if err := dao.NewConsoleBindingDao(dao.WithDBGetter(app.DBGetter())).Create(ctx, db, cb); err != nil {
		t.Fatalf("seed D binding: %v", err)
	}
}

// dualHostEngine 构造：HostResolveMiddleware + /config.json + 占位 /v1/platform 与 /v1/auth 路由。
func dualHostEngine(t *testing.T, app *App) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	e := gin.New()
	e.Use(app.HostResolveMiddleware())
	if err := app.MountConfig(e, ""); err != nil {
		t.Fatalf("MountConfig: %v", err)
	}
	e.GET("/v1/platform/ping", func(c *gin.Context) { c.String(http.StatusOK, "platform-ok") })
	e.GET("/v1/auth/ping", func(c *gin.Context) { c.String(http.StatusOK, "auth-ok") })
	return e
}

func doReq(e *gin.Engine, host, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "http://"+host+path, nil)
	req.Host = host
	w := httptest.NewRecorder()
	e.ServeHTTP(w, req)
	return w
}

// TestE2EDualHostConfigIssuerPerHost：/config.json 按 Host 返回各自 binding 的 issuer。
func TestE2EDualHostConfigIssuerPerHost(t *testing.T) {
	app := bootDualHostApp(t)
	e := dualHostEngine(t, app)

	// 平台 Host：host 取平台 binding（seed 用 URL(issuer).Host = platform.test）
	w := doReq(e, "platform.test", "/config.json")
	if w.Code != http.StatusOK {
		t.Fatalf("platform /config.json status=%d body=%s", w.Code, w.Body.String())
	}
	var pcfg map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &pcfg)
	if got := pcfg["issuer"]; got != e2ePlatformIssuer {
		t.Fatalf("platform issuer = %v, want %s", got, e2ePlatformIssuer)
	}

	// D Host：host binding 的 issuer
	w = doReq(e, e2eDHost, "/config.json")
	if w.Code != http.StatusOK {
		t.Fatalf("D /config.json status=%d body=%s", w.Code, w.Body.String())
	}
	var dcfg map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &dcfg)
	if got := dcfg["issuer"]; got != e2eDIssuer {
		t.Fatalf("D issuer = %v, want %s", got, e2eDIssuer)
	}
	if got := dcfg["clientId"]; got != "tenant_console" {
		t.Fatalf("D clientId = %v, want tenant_console", got)
	}
}

// TestE2EDualHostPlatformPathWhitelist：D Host 访问 /v1/platform/* → 404；平台 Host 放行。
func TestE2EDualHostPlatformPathWhitelist(t *testing.T) {
	app := bootDualHostApp(t)
	e := dualHostEngine(t, app)

	if w := doReq(e, e2eDHost, "/v1/platform/ping"); w.Code != http.StatusNotFound {
		t.Fatalf("D host /v1/platform/* should be 404, got %d", w.Code)
	}
	if w := doReq(e, "platform.test", "/v1/platform/ping"); w.Code != http.StatusOK {
		t.Fatalf("platform host /v1/platform/* should pass, got %d body=%s", w.Code, w.Body.String())
	}
	// D Host 的 /v1/auth/* 允许
	if w := doReq(e, e2eDHost, "/v1/auth/ping"); w.Code != http.StatusOK {
		t.Fatalf("D host /v1/auth/* should pass, got %d", w.Code)
	}
}

// TestE2EDualHostUnknown404：未知 Host → 404（不回退中央）。
func TestE2EDualHostUnknown404(t *testing.T) {
	app := bootDualHostApp(t)
	e := dualHostEngine(t, app)
	if w := doReq(e, "unknown.nowhere.test", "/config.json"); w.Code != http.StatusNotFound {
		t.Fatalf("unknown host should 404, got %d", w.Code)
	}
}
