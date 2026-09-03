package seed_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/seed"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupDB 打开独立内存 SQLite 并 AutoMigrate 全部 IAM 表（seed 不依赖全局 iam 库注册）。
func setupDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:seed_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := model.AutoMigrateAll(db); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

// TestSeedIamSQLite 在内存 SQLite 上验证种子数据：首次写入 + 二次幂等，
// 并断言管理员用户从属顶级部门（primary 行政主部门）。
func TestSeedIamSQLite(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	// 第一次：种子写入
	if err := seed.SeedIam(ctx, db); err != nil {
		t.Fatalf("seed fail: %v", err)
	}
	// 第二次：幂等性验证（不应报错、不应重复插入）
	if err := seed.SeedIam(ctx, db); err != nil {
		t.Fatalf("seed (2nd) fail: %v", err)
	}

	assertCount := func(tbl string, want int64) {
		t.Helper()
		var n int64
		if err := db.Table(tbl).Count(&n).Error; err != nil {
			t.Fatalf("count %s: %v", tbl, err)
		}
		if n != want {
			t.Fatalf("table %s: want %d rows, got %d", tbl, want, n)
		}
	}
	assertCount("tenant", 1)
	assertCount("application", 2)
	assertCount("role", 2)
	assertCount("menu", 19)
	assertCount("person", 1)
	assertCount("tenant_user", 1)
	assertCount("application_client", 2)
	assertCount("user_role", 2)
	assertCount("role_menu", 18)
	assertCount("tenant_application", 2)
	assertCount("organization", 1)
	assertCount("organization_user", 1)

	// 内置唯一角色 admin 的 admin_level = super（种子显式能力标签，非 scope 推导）
	adminLevelByCode := map[string]string{}
	var roles []model.RoleEntity
	if err := db.Find(&roles).Error; err != nil {
		t.Fatalf("query roles: %v", err)
	}
	for _, r := range roles {
		adminLevelByCode[r.Code] = r.AdminLevel
	}
	wantLevels := map[string]string{
		"admin":        string(model.SysAdminLevelSuper),
		"tenant_admin": string(model.SysAdminLevelSuper),
	}
	for code, want := range wantLevels {
		if adminLevelByCode[code] != want {
			t.Fatalf("role %s admin_level = %q, want %q", code, adminLevelByCode[code], want)
		}
	}

	// 内置角色归属应用：admin 属管理后台、tenant_admin 属租户自服务（source=builtin）
	appIDByName := map[string]string{}
	var apps []model.ApplicationEntity
	if err := db.Find(&apps).Error; err != nil {
		t.Fatalf("query applications: %v", err)
	}
	for _, a := range apps {
		appIDByName[a.Name] = a.ID
	}
	roleByCode := map[string]*model.RoleEntity{}
	for i := range roles {
		roleByCode[roles[i].Code] = &roles[i]
	}
	if got := roleByCode["admin"]; got.Source != string(model.RoleSourceBuiltin) || got.AppID != appIDByName["管理后台"] {
		t.Fatalf("seed admin role source/appID mismatch: source=%s appID=%s", got.Source, got.AppID)
	}
	if got := roleByCode["tenant_admin"]; got.Source != string(model.RoleSourceBuiltin) || got.AppID != appIDByName["租户自服务"] {
		t.Fatalf("seed tenant_admin role source/appID mismatch: source=%s appID=%s", got.Source, got.AppID)
	}

	// tenant_admin 预授权租户自服务应用全部 3 个菜单
	wantMenuCodes := map[string]bool{"organization": false, "tenant-user": false, "tenant-role": false}
	menuIDByCode := map[string]string{}
	var menus []model.MenuEntity
	if err := db.Find(&menus).Error; err != nil {
		t.Fatalf("query menus: %v", err)
	}
	for _, m := range menus {
		menuIDByCode[m.Code] = m.ID
	}
	var tenantAdminMenus []model.RoleMenuEntity
	if err := db.Where("role_id = ?", roleByCode["tenant_admin"].ID).Find(&tenantAdminMenus).Error; err != nil {
		t.Fatalf("query tenant_admin role_menu: %v", err)
	}
	for _, rm := range tenantAdminMenus {
		for code, id := range menuIDByCode {
			if rm.MenuID == id {
				wantMenuCodes[code] = true
			}
		}
	}
	for code, found := range wantMenuCodes {
		if !found {
			t.Fatalf("tenant_admin role_menu missing menu %s", code)
		}
	}

	// 管理员从属顶级部门（primary 行政主部门）
	var tenant model.TenantEntity
	if err := db.Where("code = ?", "platform").First(&tenant).Error; err != nil {
		t.Fatalf("platform tenant not found: %v", err)
	}
	var adminUser model.UserEntity
	if err := db.Where("tenant_id = ? AND is_owner = ?", tenant.ID, true).First(&adminUser).Error; err != nil {
		t.Fatalf("admin user not found: %v", err)
	}

	// 默认管理员用户额外多绑 tenant_admin（拥有 admin + tenant_admin 两角色，非替换）
	roleIDsOfAdmin := map[string]bool{}
	var adminRoles []model.UserRoleEntity
	if err := db.Where("user_id = ?", adminUser.ID).Find(&adminRoles).Error; err != nil {
		t.Fatalf("query admin user_role: %v", err)
	}
	for _, ur := range adminRoles {
		roleIDsOfAdmin[ur.RoleID] = true
	}
	if len(roleIDsOfAdmin) != 2 ||
		!roleIDsOfAdmin[roleByCode["admin"].ID] ||
		!roleIDsOfAdmin[roleByCode["tenant_admin"].ID] {
		t.Fatalf("admin user roles want {admin, tenant_admin}, got %v", roleIDsOfAdmin)
	}

	var rootOrg model.OrganizationEntity
	if err := db.Where("tenant_id = ? AND parent_id = ?", tenant.ID, "").First(&rootOrg).Error; err != nil {
		t.Fatalf("root organization not found: %v", err)
	}
	var ou model.OrganizationUserEntity
	if err := db.Where("tenant_id = ? AND user_id = ? AND organization_id = ? AND relation_type = ?",
		tenant.ID, adminUser.ID, rootOrg.ID, model.OrgUserRelationPrimary).First(&ou).Error; err != nil {
		t.Fatalf("admin organization relation not found: %v", err)
	}
}
