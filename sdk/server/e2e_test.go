package server

import (
	"context"
	"testing"

	_ "github.com/morehao/golib/dbaccess/dbgorm/driver/sqlite"

	authconfig "github.com/morehao/ark-iam/auth/config"
	"github.com/morehao/ark-iam/auth/dto/dtoinstall"
	"github.com/morehao/ark-iam/auth/service/svcauth"
	"github.com/morehao/ark-iam/auth/service/svcdb"
	"github.com/morehao/ark-iam/auth/service/svcinstall"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/object/objidentity"
	pkconfig "github.com/morehao/ark-iam/pkg/config"
)

// TestEndToEndBootstrapAndLogin 端到端：New → RunMigrations（真实建表）→ install.Initialize
// （引导建平台管理员）→ 校验管理员落库 → AuthenticatePassword 登录成功。
//
// 覆盖「门面装配 + 真实 DDL + seed 引导 + 认证」的核心闭环。
func TestEndToEndBootstrapAndLogin(t *testing.T) {
	app, err := New(Config{
		Env:     "dev",
		DB:      DBConfig{DSN: "sqlite://:memory:"},
		Domains: DefaultConsoleDomains,
		Raw: &pkconfig.Config{
			OIDC: pkconfig.OIDC{
				Issuer:           "http://localhost:8099/oidc",
				FrontendLoginURL: "http://localhost:8099/login",
			},
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	// 注入 auth 域配置与共享 DBGetter（与 App.mountAuth 一致）。
	authconfig.Set(app.cfg.Raw)
	svcdb.Set(app.DBGetter())

	ctx := context.Background()
	if err := app.RunMigrations(ctx); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	// 引导：创建平台管理员。
	installSvc := svcinstall.NewInstallSvc(app.DBGetter())
	_, err = installSvc.Initialize(ctx, &dtoinstall.InstallationInitializeReq{
		AdminUsername: "admin",
		AdminPassword: "Adm1n@Strong!Pass",
		AdminEmail:    "admin@example.com",
		TenantName:    "Acme",
	})
	if err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	// 校验管理员已落库（跨租户查询）。
	cross := dbclient.CrossTenantContext(ctx)
	var count int64
	if err := app.DBGetter()(cross).Model(&model.PersonEntity{}).Count(&count).Error; err != nil {
		t.Fatalf("count persons: %v", err)
	}
	if count == 0 {
		t.Fatal("expected at least one person after bootstrap")
	}

	// 登录：用户名+口令鉴权。
	authSvc := svcauth.NewAuthSvc(app.DBGetter())
	person, user, _, err := authSvc.AuthenticatePassword(cross, objidentity.Operator{}, "admin", "Adm1n@Strong!Pass")
	if err != nil {
		t.Fatalf("AuthenticatePassword: %v", err)
	}
	if person == nil || person.ID == "" {
		t.Fatal("login returned nil person")
	}
	if user == nil || user.ID == "" {
		t.Fatal("login returned nil user")
	}
}
