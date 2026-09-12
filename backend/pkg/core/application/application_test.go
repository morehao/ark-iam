package application

import (
	"fmt"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newTestDB 内存 SQLite 注册为全局 iam 库——GetByClientID 内部直接 dao.NewXxxDao()，
// 因此必须走全局注册（同 apps/*/testutil.SetupSQLite 的做法）。
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:iam_application_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.ApplicationEntity{}, &model.ApplicationClientEntity{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	dbclient.RegisterDBForTest(dbclient.ServiceNameIam, db)
	t.Cleanup(func() {
		dbclient.ClearDBForTest(dbclient.ServiceNameIam)
		if sqlDB, sqlErr := db.DB(); sqlErr == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func seedAppWithClient(t *testing.T, db *gorm.DB, appCode, clientCode string, allowJoin *bool) {
	t.Helper()
	appEntity := &model.ApplicationEntity{Code: appCode, AllowJoinByInvite: allowJoin}
	if err := db.Create(appEntity).Error; err != nil {
		t.Fatalf("seed application: %v", err)
	}
	client := &model.ApplicationClientEntity{Code: clientCode, AppID: appEntity.ID}
	client.ID = client.AppID
	client.RedirectURIs = datatypes.JSON(`[]`)
	client.PostLogoutRedirectURIs = datatypes.JSON(`[]`)
	client.GrantTypes = datatypes.JSON(`["authorization_code"]`)
	client.ResponseTypes = datatypes.JSON(`["code"]`)
	client.AllowedOrigins = datatypes.JSON(`[]`)
	client.DefaultScopes = datatypes.JSON(`["openid"]`)
	if err := db.Create(client).Error; err != nil {
		t.Fatalf("seed application client: %v", err)
	}
}

// TestGetByClientID 按 client_id 解析归属应用：命中返回应用，其余可预期边界一律 (nil, nil)。
func TestGetByClientID(t *testing.T) {
	db := newTestDB(t)
	seedAppWithClient(t, db, "app_join", "join_client", model.BoolPtr(true))

	cases := []struct {
		name     string
		clientID string
		wantCode string
	}{
		{name: "命中客户端", clientID: "join_client", wantCode: "app_join"},
		{name: "空 client_id", clientID: "", wantCode: ""},
		{name: "客户端不存在", clientID: "no_such_client", wantCode: ""},
	}
	ginCtx, _ := gin.CreateTestContext(nil)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			app, err := GetByClientID(ginCtx, c.clientID)
			if err != nil {
				t.Fatalf("GetByClientID(%q) unexpected err: %v", c.clientID, err)
			}
			got := ""
			if app != nil {
				got = app.Code
			}
			if got != c.wantCode {
				t.Fatalf("GetByClientID(%q) = %q, want %q", c.clientID, got, c.wantCode)
			}
		})
	}
}

// TestPolicyReaders 两个入口策略读取器：nil 应用、NULL 字段、false 一律 false，只有显式 true 才放行。
func TestPolicyReaders(t *testing.T) {
	cases := []struct {
		name string
		app  *model.ApplicationEntity
		want bool
	}{
		{name: "nil 应用", app: nil, want: false},
		{name: "字段为 NULL", app: &model.ApplicationEntity{}, want: false},
		{name: "显式 false", app: &model.ApplicationEntity{
			AllowJoinByInvite:       model.BoolPtr(false),
			AllowPersonCreateTenant: model.BoolPtr(false),
		}, want: false},
		{name: "显式 true", app: &model.ApplicationEntity{
			AllowJoinByInvite:       model.BoolPtr(true),
			AllowPersonCreateTenant: model.BoolPtr(true),
		}, want: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := AllowsJoinByInvite(c.app); got != c.want {
				t.Fatalf("AllowsJoinByInvite = %v, want %v", got, c.want)
			}
			if got := AllowsPersonCreateTenant(c.app); got != c.want {
				t.Fatalf("AllowsPersonCreateTenant = %v, want %v", got, c.want)
			}
		})
	}
}

// TestPolicyReadersAreIndependent 两个开关互相独立，不得串味。
func TestPolicyReadersAreIndependent(t *testing.T) {
	app := &model.ApplicationEntity{
		AllowJoinByInvite:       model.BoolPtr(true),
		AllowPersonCreateTenant: model.BoolPtr(false),
	}
	if !AllowsJoinByInvite(app) {
		t.Fatal("AllowsJoinByInvite 应为 true")
	}
	if AllowsPersonCreateTenant(app) {
		t.Fatal("AllowsPersonCreateTenant 应为 false")
	}
}
