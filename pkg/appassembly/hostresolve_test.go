package appassembly

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	dbgormsqlite "github.com/morehao/golib/dbaccess/dbgorm/driver/sqlite"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

func newResolveTestApp(t *testing.T) (*App, *gorm.DB) {
	t.Helper()
	dsn := fmt.Sprintf("sqlite://file:hostresolve_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(dbgormsqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.ConsoleBindingEntity{}, &model.DomainEntity{}, &model.TenantEntity{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	dbclient.RegisterDBForTest(dbclient.ServiceNameIam, db)
	t.Cleanup(func() {
		dbclient.ClearDBForTest(dbclient.ServiceNameIam)
		if sqlDB, _ := db.DB(); sqlDB != nil {
			_ = sqlDB.Close()
		}
	})
	app := &App{dbGetter: gormdao.DBGetter(func(context.Context) *gorm.DB { return db })}
	return app, db
}

func seedBinding(t *testing.T, db *gorm.DB, host string, kind model.ConsoleBindingKind, status model.ConsoleBindingStatus, tenantID, issuer string) {
	t.Helper()
	b := &model.ConsoleBindingEntity{Host: host, TenantID: tenantID, Issuer: issuer, ClientID: "tenant_console", Audience: "tenant_admin_web", Kind: kind, Status: status}
	if err := db.WithContext(dbclient.CrossTenantContext(context.Background())).Create(b).Error; err != nil {
		t.Fatalf("seed binding: %v", err)
	}
}

// TestHostResolvePlatformBinding：平台 binding enable → 解析出 platform issuer/audience。
func TestHostResolvePlatformBinding(t *testing.T) {
	app, db := newResolveTestApp(t)
	seedBinding(t, db, "platform.example.com", model.ConsoleBindingKindPlatform, model.ConsoleBindingStatusEnable, "tp", "https://platform.example.com/oidc")

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(app.HostResolveMiddleware())
	r.GET("/v1/platform/ping", func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest(http.MethodGet, "http://platform.example.com/v1/platform/ping", nil)
	req.Host = "platform.example.com"
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("platform binding should allow /v1/platform, got %d", rec.Code)
	}
}

// TestHostResolveDHostRejectsPlatformPath：D Host（tenant）访问 /v1/platform → 404。
func TestHostResolveDHostRejectsPlatformPath(t *testing.T) {
	app, db := newResolveTestApp(t)
	seedBinding(t, db, "login.acme.com", model.ConsoleBindingKindTenant, model.ConsoleBindingStatusEnable, "t1", "https://login.acme.com/oidc")

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(app.HostResolveMiddleware())
	r.GET("/v1/platform/ping", func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest(http.MethodGet, "http://login.acme.com/v1/platform/ping", nil)
	req.Host = "login.acme.com"
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != 404 {
		t.Fatalf("D host should reject platform path, got %d", rec.Code)
	}
}

// TestHostResolveUnknown404：未知 Host → 404（不回退中央）。
func TestHostResolveUnknown404(t *testing.T) {
	app, _ := newResolveTestApp(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(app.HostResolveMiddleware())
	r.GET("/v1/auth/x", func(c *gin.Context) { c.String(200, "ok") })
	req := httptest.NewRequest(http.MethodGet, "http://unknown.example.com/v1/auth/x", nil)
	req.Host = "unknown.example.com"
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != 404 {
		t.Fatalf("unknown host should be 404, got %d", rec.Code)
	}
}

// TestHostResolveAlias302：active alias（无 binding）→ GET 302 到 primary，保留 path+query。
func TestHostResolveAlias302(t *testing.T) {
	app, db := newResolveTestApp(t)
	// primary 行（供 alias target 解析）
	p := &model.DomainEntity{TenantID: "t1", Domain: "login.acme.com", DomainRole: model.DomainRolePrimary, DomainUsage: model.DomainUsageLogin, Status: model.DomainStatusActive}
	if err := db.WithContext(dbclient.ExplicitTenantContext(context.Background(), "t1")).Create(p).Error; err != nil {
		t.Fatalf("seed primary: %v", err)
	}
	a := &model.DomainEntity{TenantID: "t1", Domain: "sso.acme.com", DomainRole: model.DomainRoleAlias, DomainUsage: model.DomainUsageLogin, TargetDomainID: p.ID, Status: model.DomainStatusActive}
	if err := db.WithContext(dbclient.ExplicitTenantContext(context.Background(), "t1")).Create(a).Error; err != nil {
		t.Fatalf("seed alias: %v", err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(app.HostResolveMiddleware())
	r.GET("/x", func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest(http.MethodGet, "http://sso.acme.com/x?a=1", nil)
	req.Host = "sso.acme.com"
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("alias should 302, got %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "https://login.acme.com/x?a=1" {
		t.Fatalf("alias Location = %q", loc)
	}
}

var _ = dao.NewConsoleBindingDao
