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
	assertCount("menu", 17)
	assertCount("person", 1)
	assertCount("tenant_user", 1)
	assertCount("application_client", 2)
	assertCount("user_role", 2)
	assertCount("role_menu", 17)
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

// TestSeedPlatformMenuStructure 在全新库上验证平台管理控制台菜单树终稿结构：
// 开发期按「全新项目」处理（可删库重建），故本测试直接锁定重建后的目标 IA，
// 防止后续调整 seed 时悄然漂移。
func TestSeedPlatformMenuStructure(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	if err := seed.SeedIam(ctx, db); err != nil {
		t.Fatalf("seed fail: %v", err)
	}

	var adminApp model.ApplicationEntity
	if err := db.Where("name = ?", "管理后台").First(&adminApp).Error; err != nil {
		t.Fatalf("admin app not found: %v", err)
	}

	// 目标 IA：code / name / parentCode / sort / type（全部为管理后台应用菜单）
	type wantDef struct {
		code       string
		name       string
		parentCode string
		sort       int
		dir        bool
	}
	wantDefs := []wantDef{
		{code: "dashboard", name: "工作台", sort: 1},
		{code: "grp-tenant", name: "租户中心", sort: 2, dir: true},
		{code: "tenant", name: "租户管理", parentCode: "grp-tenant", sort: 1},
		{code: "tenant-application", name: "租户应用", parentCode: "grp-tenant", sort: 2},
		{code: "domain", name: "自定义域名", parentCode: "grp-tenant", sort: 3},
		{code: "grp-app", name: "应用中心", sort: 3, dir: true},
		{code: "application", name: "应用管理", parentCode: "grp-app", sort: 1},
		{code: "oauth-client", name: "OAuth客户端", parentCode: "grp-app", sort: 2},
		{code: "api-key", name: "API密钥", parentCode: "grp-app", sort: 3},
		{code: "grp-identity", name: "身份中心", sort: 4, dir: true},
		{code: "user", name: "用户管理", parentCode: "grp-identity", sort: 1},
		{code: "role", name: "角色管理", parentCode: "grp-identity", sort: 2},
		{code: "log", name: "审计日志", sort: 5},
		{code: "menu", name: "菜单管理", sort: 6},
	}

	var menus []model.MenuEntity
	if err := db.Where("app_id = ?", adminApp.ID).Find(&menus).Error; err != nil {
		t.Fatalf("query admin menus: %v", err)
	}
	if len(menus) != len(wantDefs) {
		t.Fatalf("admin app menu count: want %d, got %d", len(wantDefs), len(menus))
	}

	byCode := make(map[string]*model.MenuEntity, len(menus))
	codeOfID := make(map[string]string, len(menus)) // id -> code
	for i := range menus {
		byCode[menus[i].Code] = &menus[i]
		codeOfID[menus[i].ID] = menus[i].Code
	}
	parentCodeOf := make(map[string]string, len(menus)) // menu id -> parent code
	for i := range menus {
		if menus[i].ParentID != "" {
			parentCodeOf[menus[i].ID] = codeOfID[menus[i].ParentID]
		}
	}
	wantType := model.MenuTypeMenu
	wantTypeDir := model.MenuTypeDirectory
	for _, want := range wantDefs {
		got := byCode[want.code]
		if got == nil {
			t.Fatalf("menu %s not seeded", want.code)
		}
		if got.Name != want.name {
			t.Errorf("menu %s name: want %q, got %q", want.code, want.name, got.Name)
		}
		if got.Sort != want.sort {
			t.Errorf("menu %s sort: want %d, got %d", want.code, want.sort, got.Sort)
		}
		wantTyp := wantType
		if want.dir {
			wantTyp = wantTypeDir
		}
		if got.Type != wantTyp {
			t.Errorf("menu %s type: want %s, got %s", want.code, wantTyp, got.Type)
		}
		if want.parentCode == "" {
			if got.ParentID != "" {
				t.Errorf("menu %s should be top-level, got parent %s", want.code, got.ParentID)
			}
		} else if parentCodeOf[got.ID] != want.parentCode {
			t.Errorf("menu %s parent: want %s, got %s", want.code, want.parentCode, parentCodeOf[got.ID])
		}
	}

	// 一级菜单展示顺序（sort 升序）：工作台 → 租户中心 → 应用中心 → 身份中心 → 审计日志 → 菜单管理
	var topMenus []model.MenuEntity
	if err := db.Where("app_id = ? AND parent_id = ?", adminApp.ID, "").Order("sort asc").Find(&topMenus).Error; err != nil {
		t.Fatalf("query top menus: %v", err)
	}
	wantOrder := []string{"dashboard", "grp-tenant", "grp-app", "grp-identity", "log", "menu"}
	if len(topMenus) != len(wantOrder) {
		t.Fatalf("top-level menu count: want %d, got %d", len(wantOrder), len(topMenus))
	}
	for i, want := range wantOrder {
		if topMenus[i].Code != want {
			t.Errorf("top-level order[%d]: want %s, got %s", i, want, topMenus[i].Code)
		}
	}

	// 已下线模块不应再出现：system / grp-ops / 旧目录名 grp-org
	for _, stale := range []string{"system", "grp-ops", "grp-org"} {
		if byCode[stale] != nil {
			t.Errorf("retired menu %s should not be seeded", stale)
		}
	}

	// admin 角色菜单授权集合（平台应用菜单 + 租户自服务菜单）
	var adminRole model.RoleEntity
	if err := db.Where("app_id = ? AND code = ?", adminApp.ID, "admin").First(&adminRole).Error; err != nil {
		t.Fatalf("admin role not found: %v", err)
	}
	var adminMenuLinks []model.RoleMenuEntity
	if err := db.Where("role_id = ?", adminRole.ID).Find(&adminMenuLinks).Error; err != nil {
		t.Fatalf("query admin role_menu: %v", err)
	}
	if len(adminMenuLinks) != 14 {
		t.Fatalf("admin role_menu count: want 14, got %d", len(adminMenuLinks))
	}
	// 授权集合涉及平台应用与租户自服务两个应用的菜单，用全量映射解析
	var allMenus []model.MenuEntity
	if err := db.Find(&allMenus).Error; err != nil {
		t.Fatalf("query all menus: %v", err)
	}
	codeEntityAll := make(map[string]*model.MenuEntity, len(allMenus))
	for i := range allMenus {
		codeEntityAll[allMenus[i].Code] = &allMenus[i]
	}
	granted := make(map[string]bool, len(adminMenuLinks))
	for _, link := range adminMenuLinks {
		for code, m := range codeEntityAll {
			if m.ID == link.MenuID {
				granted[code] = true
			}
		}
	}
	for _, want := range []string{"dashboard", "user", "role", "menu", "tenant", "application", "tenant-application", "oauth-client", "api-key", "domain", "log", "organization", "tenant-user", "tenant-role"} {
		if !granted[want] {
			t.Errorf("admin role missing menu grant %s", want)
		}
	}
	if granted["system"] {
		t.Error("admin role should not grant retired menu system")
	}
}
