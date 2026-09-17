package seed_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/morehao/ark-iam/pkg/model"
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
	assertCount("menu", 15)
	assertCount("person", 1)
	assertCount("tenant_user", 1)
	assertCount("application_client", 2)
	assertCount("user_role", 2)
	assertCount("role_menu", 15)
	assertCount("tenant_application", 2)
	assertCount("department", 1)
	assertCount("department_user", 1)

	// 内置角色：admin 属平台管理后台、tenant_admin 属租户管理后台（source=builtin、admin_type=admin）。
	// 角色无业务编码，按所属应用定位（每种内置角色在其应用内唯一）。
	appIDByName := map[string]string{}
	var apps []model.ApplicationEntity
	if err := db.Find(&apps).Error; err != nil {
		t.Fatalf("query applications: %v", err)
	}
	for _, a := range apps {
		appIDByName[a.Name] = a.ID
	}
	var roles []model.RoleEntity
	if err := db.Find(&roles).Error; err != nil {
		t.Fatalf("query roles: %v", err)
	}
	roleByAppName := map[string]*model.RoleEntity{}
	for i := range roles {
		if roles[i].Source != model.RoleSourceBuiltin {
			continue
		}
		for appName, appID := range appIDByName {
			if appID == roles[i].AppID {
				roleByAppName[appName] = &roles[i]
			}
		}
	}
	adminRole := roleByAppName["平台管理后台"]
	tenantAdminRole := roleByAppName["租户管理后台"]
	if adminRole == nil || adminRole.AdminType != model.SysAdminTypeAdmin || adminRole.Name != "管理员" {
		t.Fatalf("seed admin role mismatch: %+v", adminRole)
	}
	// 编码是跨系统授权契约值（OIDC groups 取值），跑通下游策略映射依赖它被播种
	if adminRole.Code != model.RoleCodePlatformAdmin {
		t.Fatalf("seed admin role code mismatch: %+v", adminRole)
	}
	if tenantAdminRole == nil || tenantAdminRole.AdminType != model.SysAdminTypeAdmin || tenantAdminRole.Name != "租户管理员" {
		t.Fatalf("seed tenant_admin role mismatch: %+v", tenantAdminRole)
	}
	if tenantAdminRole.Code != model.RoleCodeTenantAdmin {
		t.Fatalf("seed tenant_admin role code mismatch: %+v", tenantAdminRole)
	}

	// tenant_admin 预授权租户管理后台应用全部 4 个菜单
	wantMenuCodes := map[string]bool{"department": false, "tenant-user": false, "tenant-role": false, "tenant-api-key": false}
	menuIDByCode := map[string]string{}
	var menus []model.MenuEntity
	if err := db.Find(&menus).Error; err != nil {
		t.Fatalf("query menus: %v", err)
	}
	for _, m := range menus {
		menuIDByCode[m.Code] = m.ID
	}
	var tenantAdminMenus []model.RoleMenuEntity
	if err := db.Where("role_id = ?", tenantAdminRole.ID).Find(&tenantAdminMenus).Error; err != nil {
		t.Fatalf("query tenant_admin role_menu: %v", err)
	}
	if len(tenantAdminMenus) != len(wantMenuCodes) {
		t.Fatalf("tenant_admin role_menu count: want %d, got %d", len(wantMenuCodes), len(tenantAdminMenus))
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
	// 平台租户编码为固定值 t_platform（与自动生成规则 t_<12 位随机 hex> 同前缀）
	var tenant model.TenantEntity
	if err := db.Where("code = ?", "t_platform").First(&tenant).Error; err != nil {
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
		!roleIDsOfAdmin[adminRole.ID] ||
		!roleIDsOfAdmin[tenantAdminRole.ID] {
		t.Fatalf("admin user roles want {admin, tenant_admin}, got %v", roleIDsOfAdmin)
	}

	var rootDept model.DepartmentEntity
	if err := db.Where("tenant_id = ? AND parent_id = ?", tenant.ID, "").First(&rootDept).Error; err != nil {
		t.Fatalf("root department not found: %v", err)
	}
	var ou model.DepartmentUserEntity
	if err := db.Where("tenant_id = ? AND user_id = ? AND department_id = ? AND relation_type = ?",
		tenant.ID, adminUser.ID, rootDept.ID, model.DeptUserRelationPrimary).First(&ou).Error; err != nil {
		t.Fatalf("admin department relation not found: %v", err)
	}
}

// TestSeedPlatformMenuStructure 在全新库上验证平台管理后台菜单树终稿结构：
// 开发期按「全新项目」处理（可删库重建），故本测试直接锁定重建后的目标 IA，
// 防止后续调整 seed 时悄然漂移。
func TestSeedPlatformMenuStructure(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	if err := seed.SeedIam(ctx, db); err != nil {
		t.Fatalf("seed fail: %v", err)
	}

	var adminApp model.ApplicationEntity
	if err := db.Where("name = ?", "平台管理后台").First(&adminApp).Error; err != nil {
		t.Fatalf("admin app not found: %v", err)
	}

	// 目标 IA：code / name / parentCode / sort / type（全部为平台管理后台应用菜单）
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
		{code: "grp-platform", name: "平台管理", sort: 4, dir: true},
		{code: "menu", name: "菜单管理", parentCode: "grp-platform", sort: 1},
		{code: "log", name: "审计日志", parentCode: "grp-platform", sort: 2},
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

	// 一级菜单展示顺序（sort 升序）：工作台 → 租户中心 → 应用中心 → 平台管理
	var topMenus []model.MenuEntity
	if err := db.Where("app_id = ? AND parent_id = ?", adminApp.ID, "").Order("sort asc").Find(&topMenus).Error; err != nil {
		t.Fatalf("query top menus: %v", err)
	}
	wantOrder := []string{"dashboard", "grp-tenant", "grp-app", "grp-platform"}
	if len(topMenus) != len(wantOrder) {
		t.Fatalf("top-level menu count: want %d, got %d", len(wantOrder), len(topMenus))
	}
	for i, want := range wantOrder {
		if topMenus[i].Code != want {
			t.Errorf("top-level order[%d]: want %s, got %s", i, want, topMenus[i].Code)
		}
	}

	// 已下线模块不应再出现：system / grp-ops / 旧目录名 grp-org；
	// 用户与角色已无平台端入口（收敛到租户管理后台），平台应用不应再种子 grp-identity / user / role。
	for _, stale := range []string{"system", "grp-ops", "grp-org", "grp-identity", "user", "role"} {
		if byCode[stale] != nil {
			t.Errorf("retired menu %s should not be seeded", stale)
		}
	}

	// admin 角色菜单授权集合（平台应用菜单 + 租户管理后台菜单）
	var adminRole model.RoleEntity
	if err := db.Where("app_id = ? AND source = ?", adminApp.ID, model.RoleSourceBuiltin).First(&adminRole).Error; err != nil {
		t.Fatalf("admin role not found: %v", err)
	}
	var adminMenuLinks []model.RoleMenuEntity
	if err := db.Where("role_id = ?", adminRole.ID).Find(&adminMenuLinks).Error; err != nil {
		t.Fatalf("query admin role_menu: %v", err)
	}
	if len(adminMenuLinks) != 11 {
		t.Fatalf("admin role_menu count: want 11, got %d", len(adminMenuLinks))
	}
	// 授权集合涉及平台应用与租户管理后台两个应用的菜单，用全量映射解析
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
	for _, want := range []string{"dashboard", "menu", "tenant", "application", "tenant-application", "oauth-client", "domain", "log", "department", "tenant-user", "tenant-role"} {
		if !granted[want] {
			t.Errorf("admin role missing menu grant %s", want)
		}
	}
	if granted["system"] {
		t.Error("admin role should not grant retired menu system")
	}
}

// TestSeedIamPrunesRetiredMenus 存量库清理：历史版本种子写入、当前已下线的平台端菜单
// （api-key 与 身份中心 grp-identity + 其子菜单 user/role）及其 role_menu 授权绑定，
// 应在种子启动后被清除，且清理幂等。
//
// 为什么必须有这条回归：seedMenus 只做幂等 upsert、从不下线菜单，漏登记 retiredMenus
// 会让存量库持续渲染指向已删除页面的死链菜单——而「全新库菜单结构」测试测不出来，
// 因为新库根本不会创建这些菜单。
func TestSeedIamPrunesRetiredMenus(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	if err := seed.SeedIam(ctx, db); err != nil {
		t.Fatalf("seed fail: %v", err)
	}

	var adminApp model.ApplicationEntity
	if err := db.Where("code = ?", "platform_admin").First(&adminApp).Error; err != nil {
		t.Fatalf("admin app not found: %v", err)
	}
	var adminRole model.RoleEntity
	if err := db.Where("app_id = ? AND source = ?", adminApp.ID, model.RoleSourceBuiltin).First(&adminRole).Error; err != nil {
		t.Fatalf("admin role not found: %v", err)
	}
	var grpApp model.MenuEntity
	if err := db.Where("app_id = ? AND code = ?", adminApp.ID, "grp-app").First(&grpApp).Error; err != nil {
		t.Fatalf("grp-app menu not found: %v", err)
	}

	// 模拟存量库：补回历史版本的「身份中心」目录（子菜单需要其 ID），再补三个子/同级菜单
	legacyIdentity := &model.MenuEntity{
		AppID:      adminApp.ID,
		Name:       "身份中心",
		Code:       "grp-identity",
		Icon:       "team",
		Sort:       4,
		Type:       model.MenuTypeDirectory,
		Visibility: model.MenuVisibilityAdmin,
		Status:     model.MenuStatusEnable,
	}
	if err := db.Create(legacyIdentity).Error; err != nil {
		t.Fatalf("insert legacy grp-identity: %v", err)
	}
	legacyMenus := []*model.MenuEntity{
		{
			AppID:      adminApp.ID,
			ParentID:   grpApp.ID,
			Name:       "API密钥监督",
			Code:       "api-key",
			Path:       "/api-key",
			Icon:       "key",
			Sort:       3,
			Type:       model.MenuTypeMenu,
			Visibility: model.MenuVisibilityAdmin,
			Component:  "/apiKey/index",
			Status:     model.MenuStatusEnable,
		},
		{
			AppID:      adminApp.ID,
			ParentID:   legacyIdentity.ID,
			Name:       "用户管理",
			Code:       "user",
			Path:       "/user",
			Icon:       "user",
			Sort:       1,
			Type:       model.MenuTypeMenu,
			Visibility: model.MenuVisibilityAdmin,
			Component:  "/user/index",
			Status:     model.MenuStatusEnable,
		},
		{
			AppID:      adminApp.ID,
			ParentID:   legacyIdentity.ID,
			Name:       "角色管理",
			Code:       "role",
			Path:       "/role",
			Icon:       "role",
			Sort:       2,
			Type:       model.MenuTypeMenu,
			Visibility: model.MenuVisibilityAdmin,
			Component:  "/role/index",
			Status:     model.MenuStatusEnable,
		},
	}
	for _, legacy := range legacyMenus {
		if err := db.Create(legacy).Error; err != nil {
			t.Fatalf("insert legacy menu %s: %v", legacy.Code, err)
		}
	}
	// 存量库中这些菜单都被 admin 角色授权过，父目录同样有绑定
	allLegacy := append(append([]*model.MenuEntity{}, legacyMenus...), legacyIdentity)
	for _, legacy := range allLegacy {
		if err := db.Create(&model.RoleMenuEntity{
			TenantID: adminRole.TenantID,
			RoleID:   adminRole.ID,
			MenuID:   legacy.ID,
		}).Error; err != nil {
			t.Fatalf("insert legacy role_menu %s: %v", legacy.Code, err)
		}
	}

	// 再次种子启动：菜单行与授权绑定均应被清理；重复执行保持幂等
	for i := 0; i < 2; i++ {
		if err := seed.SeedIam(ctx, db); err != nil {
			t.Fatalf("seed (%d) fail: %v", i+2, err)
		}
	}

	for _, legacy := range allLegacy {
		var menuCount int64
		if err := db.Model(&model.MenuEntity{}).
			Where("app_id = ? AND code = ?", adminApp.ID, legacy.Code).Count(&menuCount).Error; err != nil {
			t.Fatalf("count retired menu %s: %v", legacy.Code, err)
		}
		if menuCount != 0 {
			t.Errorf("retired menu %s count = %d, want 0", legacy.Code, menuCount)
		}
		var linkCount int64
		if err := db.Model(&model.RoleMenuEntity{}).Where("menu_id = ?", legacy.ID).Count(&linkCount).Error; err != nil {
			t.Fatalf("count retired menu %s role_menu: %v", legacy.Code, err)
		}
		if linkCount != 0 {
			t.Errorf("retired menu %s role_menu count = %d, want 0", legacy.Code, linkCount)
		}
	}
}

// TestSeedIamMigratesLegacyNamesOnly 展示名的跨版本处理分两种语义，本用例锁定两者边界：
//   - 历史种子名（Default Tenant）走 migrate_once：值匹配才改写，租户与根部门同步到「平台运营中心」；
//   - 其余展示名（内置应用名/描述、内置客户端名）归运维（create_only）：控制台改过之后种子一律
//     不回写，也不再"回填"成种子定义值（reconcile 已收窄到定位键 + 安全不变式）。
func TestSeedIamMigratesLegacyNamesOnly(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	if err := seed.SeedIam(ctx, db); err != nil {
		t.Fatalf("seed fail: %v", err)
	}

	var tenant model.TenantEntity
	if err := db.Where("code = ?", "t_platform").First(&tenant).Error; err != nil {
		t.Fatalf("query tenant: %v", err)
	}
	// 模拟旧库遗留名：命中 migrate_once 的历史种子值，应被一次性改名为 平台运营中心
	if err := db.Model(&model.TenantEntity{}).Where("id = ?", tenant.ID).
		Update("name", "Default Tenant").Error; err != nil {
		t.Fatalf("degrade tenant name: %v", err)
	}
	if err := db.Model(&model.DepartmentEntity{}).Where("tenant_id = ? AND parent_id = ?", tenant.ID, "").
		Update("name", "Default Tenant").Error; err != nil {
		t.Fatalf("degrade root department name: %v", err)
	}
	// 模拟控制台改动：内置应用的展示名/描述（归运维）+ 启停/排序
	if err := db.Model(&model.ApplicationEntity{}).Where("code = ?", "platform_admin").
		Updates(map[string]any{
			"name":        "运维控制台",
			"description": "运维自定描述",
			"status":      model.AppStatusDisable, // 控制台改过的启停，种子不得覆盖
			"sort":        9,                      // 控制台改过的排序，种子不得覆盖
		}).Error; err != nil {
		t.Fatalf("degrade admin application: %v", err)
	}
	if err := db.Model(&model.ApplicationEntity{}).Where("code = ?", "tenant_admin").
		Updates(map[string]any{"name": "租户自服务", "description": "租户自服务控制台应用"}).Error; err != nil {
		t.Fatalf("degrade tenant admin application: %v", err)
	}
	if err := db.Model(&model.ApplicationClientEntity{}).Where("code = ?", "platform_admin_web").
		Update("name", "IAM管理平台").Error; err != nil {
		t.Fatalf("degrade platform client name: %v", err)
	}
	if err := db.Model(&model.ApplicationClientEntity{}).Where("code = ?", "tenant_admin_web").
		Update("name", "租户管理平台").Error; err != nil {
		t.Fatalf("degrade tenant client name: %v", err)
	}

	// 二次种子：一次性改名生效；三次种子：保持幂等
	if err := seed.SeedIam(ctx, db); err != nil {
		t.Fatalf("seed (2nd) fail: %v", err)
	}
	if err := seed.SeedIam(ctx, db); err != nil {
		t.Fatalf("seed (3rd) fail: %v", err)
	}

	if err := db.Where("code = ?", "t_platform").First(&tenant).Error; err != nil {
		t.Fatalf("query tenant after reseed: %v", err)
	}
	if tenant.Name != "平台运营中心" {
		t.Errorf("tenant name = %q, want %q（历史种子值应一次性改名）", tenant.Name, "平台运营中心")
	}
	if tenant.Status != model.TenantStatusActive {
		t.Errorf("tenant status = %q, want %q", tenant.Status, model.TenantStatusActive)
	}

	var rootDept model.DepartmentEntity
	if err := db.Where("tenant_id = ? AND parent_id = ?", tenant.ID, "").First(&rootDept).Error; err != nil {
		t.Fatalf("query root department: %v", err)
	}
	if rootDept.Name != tenant.Name {
		t.Errorf("root department name = %q, want %q (根部门随租户名一次性改名)", rootDept.Name, tenant.Name)
	}

	// 内置应用的名称/描述归运维：种子的值只是创建时初值，重启不得回写
	wantApps := map[string]struct{ name, desc string }{
		"platform_admin": {"运维控制台", "运维自定描述"},
		"tenant_admin":   {"租户自服务", "租户自服务控制台应用"},
	}
	for code, want := range wantApps {
		var app model.ApplicationEntity
		if err := db.Where("code = ?", code).First(&app).Error; err != nil {
			t.Fatalf("query application %s: %v", code, err)
		}
		if app.Name != want.name || app.Description != want.desc {
			t.Errorf("application %s = (%q, %q), want (%q, %q)（展示名归运维，种子不得回写）",
				code, app.Name, app.Description, want.name, want.desc)
		}
	}

	// 运行时编排字段保持控制台改动，种子不覆盖
	var adminApp model.ApplicationEntity
	if err := db.Where("code = ?", "platform_admin").First(&adminApp).Error; err != nil {
		t.Fatalf("query admin application: %v", err)
	}
	if adminApp.Status != model.AppStatusDisable {
		t.Errorf("application status = %q, want %q (种子不得覆盖控制台启停)", adminApp.Status, model.AppStatusDisable)
	}
	if adminApp.Sort != 9 {
		t.Errorf("application sort = %d, want 9 (种子不得覆盖控制台排序)", adminApp.Sort)
	}

	// 内置客户端名归运维：同样不得回写
	wantClients := map[string]string{
		"platform_admin_web": "IAM管理平台",
		"tenant_admin_web":   "租户管理平台",
	}
	for code, want := range wantClients {
		var client model.ApplicationClientEntity
		if err := db.Where("code = ?", code).First(&client).Error; err != nil {
			t.Fatalf("query application_client %s: %v", code, err)
		}
		if client.Name != want {
			t.Errorf("application_client %s name = %q, want %q（展示名归运维，种子不得回写）", code, client.Name, want)
		}
	}
}

// TestSeedIamKeepsOperatorMenuEdits 内置应用菜单的展示与结构字段归运维：在控制台改过之后
// （名称/路径/组件/图标/排序/类型/可见性/父级），重启种子不得回写——这是本次 reconcile 收窄的
// 核心诉求（此前菜单树整体不可编辑）。
func TestSeedIamKeepsOperatorMenuEdits(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	if err := seed.SeedIam(ctx, db); err != nil {
		t.Fatalf("seed fail: %v", err)
	}

	var adminApp model.ApplicationEntity
	if err := db.Where("code = ?", "platform_admin").First(&adminApp).Error; err != nil {
		t.Fatalf("query admin app: %v", err)
	}
	var editors []model.MenuEntity
	if err := db.Where("app_id = ? AND parent_id != ?", adminApp.ID, "").Find(&editors).Error; err != nil {
		t.Fatalf("query child menus: %v", err)
	}
	if len(editors) == 0 {
		t.Fatal("没有可验证的内置子菜单")
	}

	// 控制台改动：整体重命名 + 改结构（含把菜单挂到根节点，验证父级不再被收敛）
	custom := map[string]any{
		"name":       "运维自定菜单名",
		"path":       "/ops-custom",
		"component":  "pages/opsCustom",
		"icon":       "SettingOutlined",
		"sort":       99,
		"type":       model.MenuTypeMenu,
		"visibility": model.MenuVisibilityPublic,
		"parent_id":  "",
		"status":     model.MenuStatusDisable,
	}
	edited := editors[0]
	if err := db.Model(&model.MenuEntity{}).Where("id = ?", edited.ID).Updates(custom).Error; err != nil {
		t.Fatalf("degrade menu: %v", err)
	}

	if err := seed.SeedIam(ctx, db); err != nil {
		t.Fatalf("seed (2nd) fail: %v", err)
	}
	if err := seed.SeedIam(ctx, db); err != nil {
		t.Fatalf("seed (3rd) fail: %v", err)
	}

	var got model.MenuEntity
	if err := db.Where("id = ?", edited.ID).First(&got).Error; err != nil {
		t.Fatalf("query menu: %v", err)
	}
	if got.Name != "运维自定菜单名" || got.Path != "/ops-custom" || got.Component != "pages/opsCustom" ||
		got.Icon != "SettingOutlined" || got.Sort != 99 || got.ParentID != "" ||
		got.Visibility != model.MenuVisibilityPublic || got.Status != model.MenuStatusDisable {
		t.Errorf("内置菜单被种子回写: name=%q path=%q component=%q icon=%q sort=%d parent=%q visibility=%q status=%q",
			got.Name, got.Path, got.Component, got.Icon, got.Sort, got.ParentID, got.Visibility, got.Status)
	}
	if got.Code != edited.Code {
		t.Errorf("定位键 code 不得变化: %q -> %q", edited.Code, got.Code)
	}
	// 内置菜单自愈的边界：种子不再创建缺失行以外的动作，菜单总数保持不变（未因重命名重建）
	var total int64
	if err := db.Model(&model.MenuEntity{}).Where("app_id = ?", adminApp.ID).Count(&total).Error; err != nil {
		t.Fatalf("count menus: %v", err)
	}
	if total != 11 {
		t.Errorf("平台应用菜单数 = %d, want 11（改菜单不得触发种子重建行）", total)
	}
}

// TestSeedIamMigratesLegacyPlatformTenantCode 存量库平台租户编码为 "platform" 时，
// 种子启动应原地改名为 t_platform：保留主键、不重复建租户；同时回填租户名与状态。
func TestSeedIamMigratesLegacyPlatformTenantCode(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	legacy := &model.TenantEntity{
		Code:   "platform",
		Name:   "Default Tenant", // 旧库遗留名：种子启动时应回填为平台运营中心
		Type:   model.TenantTypePlatform,
		Status: model.TenantStatusSuspended, // 同时校验状态回填路径
	}
	if err := db.Create(legacy).Error; err != nil {
		t.Fatalf("seed legacy tenant: %v", err)
	}

	if err := seed.SeedIam(ctx, db); err != nil {
		t.Fatalf("seed fail: %v", err)
	}
	// 再跑一次：改名后必须幂等（不得因为找不到旧编码而新建租户）
	if err := seed.SeedIam(ctx, db); err != nil {
		t.Fatalf("seed (2nd) fail: %v", err)
	}

	var tenants []model.TenantEntity
	if err := db.Find(&tenants).Error; err != nil {
		t.Fatalf("query tenant: %v", err)
	}
	if len(tenants) != 1 {
		t.Fatalf("tenant count = %d, want 1 (legacy code must be renamed, not duplicated)", len(tenants))
	}
	if tenants[0].ID != legacy.ID {
		t.Errorf("tenant id = %s, want legacy id %s (rename must keep primary key)", tenants[0].ID, legacy.ID)
	}
	if tenants[0].Code != "t_platform" {
		t.Errorf("tenant code = %q, want %q", tenants[0].Code, "t_platform")
	}
	if tenants[0].Name != "平台运营中心" {
		t.Errorf("tenant name = %q, want %q (旧库遗留名必须随启动回填)", tenants[0].Name, "平台运营中心")
	}
	if tenants[0].Status != model.TenantStatusActive {
		t.Errorf("tenant status = %q, want %q", tenants[0].Status, model.TenantStatusActive)
	}
	// 改名后不得残留旧编码行
	var legacyCount int64
	if err := db.Model(&model.TenantEntity{}).Where("code = ?", "platform").Count(&legacyCount).Error; err != nil {
		t.Fatalf("count legacy tenant: %v", err)
	}
	if legacyCount != 0 {
		t.Errorf("legacy code row count = %d, want 0", legacyCount)
	}
}

// TestSeedIamBackfillsApplicationSource 存量库（source 由 AutoMigrate 补列时取列默认值
// `third_party`）在种子启动后必须被原地纠正：
// 两个种子应用（平台管理后台、租户管理后台）与种子 OAuth 客户端 → builtin（否则丢删除保护）。
// 回归背景：种子数据的两个应用都不是第三方接入，误判会同时污染删除保护与租户控制台菜单范围
// （见 docs/design/system-design.md §4.3、§4.5）。
func TestSeedIamBackfillsApplicationSource(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	// 首次种子：库由本次改造后的定义写入正确 source
	if err := seed.SeedIam(ctx, db); err != nil {
		t.Fatalf("seed fail: %v", err)
	}

	// 模拟存量库：两应用与种子客户端的 source 都被列默认值覆盖为 third_party
	// （等同旧库被 AutoMigrate 补列后的状态）
	if err := db.Model(&model.ApplicationEntity{}).Where("code = ?", "platform_admin").
		Update("source", model.AppSourceThirdParty).Error; err != nil {
		t.Fatalf("degrade platform-admin source: %v", err)
	}
	if err := db.Model(&model.ApplicationEntity{}).Where("code = ?", "tenant_admin").
		Update("source", model.AppSourceThirdParty).Error; err != nil {
		t.Fatalf("degrade tenant-admin source: %v", err)
	}
	if err := db.Model(&model.ApplicationClientEntity{}).Where("1 = 1").
		Update("source", model.ApplicationClientSourceThirdParty).Error; err != nil {
		t.Fatalf("degrade application_client source: %v", err)
	}

	// 二次种子：必须原地回填，不得新建记录
	if err := seed.SeedIam(ctx, db); err != nil {
		t.Fatalf("seed (2nd) fail: %v", err)
	}

	// 三次种子：回填后保持幂等
	if err := seed.SeedIam(ctx, db); err != nil {
		t.Fatalf("seed (3rd) fail: %v", err)
	}

	var appCount int64
	if err := db.Model(&model.ApplicationEntity{}).Count(&appCount).Error; err != nil {
		t.Fatalf("count application: %v", err)
	}
	if appCount != 2 {
		t.Fatalf("application count = %d, want 2 (回填不得新建应用)", appCount)
	}

	var adminApp model.ApplicationEntity
	if err := db.Where("code = ?", "platform_admin").First(&adminApp).Error; err != nil {
		t.Fatalf("query admin app: %v", err)
	}
	if adminApp.Source != model.AppSourceBuiltin {
		t.Errorf("平台管理后台 source = %q, want %q", adminApp.Source, model.AppSourceBuiltin)
	}

	var tenantAdminApp model.ApplicationEntity
	if err := db.Where("code = ?", "tenant_admin").First(&tenantAdminApp).Error; err != nil {
		t.Fatalf("query tenant-admin app: %v", err)
	}
	if tenantAdminApp.Source != model.AppSourceBuiltin {
		t.Errorf("租户管理后台 source = %q, want %q", tenantAdminApp.Source, model.AppSourceBuiltin)
	}

	// 种子客户端全部为内置客户端：存量库里不得残留 third_party
	var clients []model.ApplicationClientEntity
	if err := db.Find(&clients).Error; err != nil {
		t.Fatalf("query application_client: %v", err)
	}
	if len(clients) == 0 {
		t.Fatal("no seeded application_client found")
	}
	for _, c := range clients {
		if c.Source != model.ApplicationClientSourceBuiltin {
			t.Errorf("客户端 %s source = %q, want %q", c.Code, c.Source, model.ApplicationClientSourceBuiltin)
		}
	}
}

// TestSeedIamBindsBuiltinClientsToTheirApps 内置 OAuth 客户端必须挂在各自的控制台应用上：
// 平台管理后台客户端 → platform_admin，租户管理后台客户端 → tenant_admin。
// 回归背景：历史种子把两个客户端都挂在 platform_admin，后果有二：
//  1. 控制台「所属应用」列把租户管理后台客户端显示成"平台管理后台"（归属不可辨认）；
//  2. auth 侧 appAllowsPersonCreateTenant 按 client.AppID 解析应用策略，读到的是错误应用。
//
// app_id 已在字段权威矩阵声明为 reconcile：存量库的错误绑定在下次启动自愈，且不得新建客户端。
func TestSeedIamBindsBuiltinClientsToTheirApps(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	if err := seed.SeedIam(ctx, db); err != nil {
		t.Fatalf("seed fail: %v", err)
	}

	appIDByCode := map[string]string{}
	var apps []model.ApplicationEntity
	if err := db.Find(&apps).Error; err != nil {
		t.Fatalf("query applications: %v", err)
	}
	for _, a := range apps {
		appIDByCode[a.Code] = a.ID
	}
	wantAppCode := map[string]string{
		"platform_admin_web": "platform_admin",
		"tenant_admin_web":   "tenant_admin",
	}
	assertClientBinding := func(stage string) {
		t.Helper()
		for clientCode, appCode := range wantAppCode {
			var client model.ApplicationClientEntity
			if err := db.Where("code = ?", clientCode).First(&client).Error; err != nil {
				t.Fatalf("%s: query client %s: %v", stage, clientCode, err)
			}
			if client.AppID != appIDByCode[appCode] {
				t.Errorf("%s: 客户端 %s 挂在 app_id=%s, want %s(%s)",
					stage, clientCode, client.AppID, appCode, appIDByCode[appCode])
			}
		}
	}
	assertClientBinding("fresh seed")

	// 模拟存量库：两个客户端都被错误绑定到 platform_admin
	if err := db.Model(&model.ApplicationClientEntity{}).Where("1 = 1").
		Update("app_id", appIDByCode["platform_admin"]).Error; err != nil {
		t.Fatalf("degrade client app_id: %v", err)
	}
	if err := seed.SeedIam(ctx, db); err != nil {
		t.Fatalf("seed (2nd) fail: %v", err)
	}
	assertClientBinding("reconcile")

	var count int64
	if err := db.Model(&model.ApplicationClientEntity{}).Count(&count).Error; err != nil {
		t.Fatalf("count application_client: %v", err)
	}
	if count != 2 {
		t.Fatalf("application_client count = %d, want 2（收敛不得新建客户端）", count)
	}
}

// TestSeedIamMigratesLegacyClientCode 存量库的内置客户端编码为连字符形态（platform-admin-web /
// tenant-admin-web）时，种子启动必须原地改名为下划线形态（platform_admin_web / tenant_admin_web）：
// 保留主键，因此以客户端 id 为外键的 refresh_token / application_client_secret 不失联，
// 也不会重建出第二个内置客户端（code 是唯一键，重复建会直接撞唯一索引）。
// 回归背景：客户端编码统一为下划线口径（model.ClientCodePattern）后，若不迁移旧编码，
// 种子按新编码查不到旧行 → 另建两个客户端，库里会同时存在 4 行、控制台的 client_id 与网关
// audience 白名单也随之错配。
func TestSeedIamMigratesLegacyClientCode(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	legacyPlatform := &model.ApplicationClientEntity{
		TenantID: "t1", AppID: "a1", Code: "platform-admin-web", Name: "平台管理后台",
		Source: model.ApplicationClientSourceThirdParty, Status: model.ApplicationClientStatusEnable,
	}
	legacyTenant := &model.ApplicationClientEntity{
		TenantID: "t1", AppID: "a1", Code: "tenant-admin-web", Name: "租户管理后台",
		Source: model.ApplicationClientSourceThirdParty, Status: model.ApplicationClientStatusEnable,
	}
	for _, client := range []*model.ApplicationClientEntity{legacyPlatform, legacyTenant} {
		if err := db.Create(client).Error; err != nil {
			t.Fatalf("seed legacy client: %v", err)
		}
	}
	wantIDs := map[string]string{
		model.SeedBuiltinClientPlatformAdminWeb: legacyPlatform.ID,
		model.SeedBuiltinClientTenantAdminWeb:   legacyTenant.ID,
	}

	if err := seed.SeedIam(ctx, db); err != nil {
		t.Fatalf("seed fail: %v", err)
	}
	// 再跑一次：改名后必须幂等（不得因为旧编码缺席而新建客户端）
	if err := seed.SeedIam(ctx, db); err != nil {
		t.Fatalf("seed (2nd) fail: %v", err)
	}

	var clients []model.ApplicationClientEntity
	if err := db.Find(&clients).Error; err != nil {
		t.Fatalf("query application_client: %v", err)
	}
	if len(clients) != 2 {
		t.Fatalf("application_client count = %d, want 2（旧编码必须原地改名而非新建）", len(clients))
	}
	var legacyCount int64
	if err := db.Model(&model.ApplicationClientEntity{}).
		Where("code IN ?", []string{"platform-admin-web", "tenant-admin-web"}).Count(&legacyCount).Error; err != nil {
		t.Fatalf("count legacy clients: %v", err)
	}
	if legacyCount != 0 {
		t.Errorf("legacy client code row count = %d, want 0", legacyCount)
	}

	for code, wantID := range wantIDs {
		var got model.ApplicationClientEntity
		if err := db.Where("code = ?", code).First(&got).Error; err != nil {
			t.Fatalf("query migrated client %s: %v", code, err)
		}
		if got.ID != wantID {
			t.Errorf("client %s id = %s, want legacy id %s（改名必须保留主键）", code, got.ID, wantID)
		}
		if got.Source != model.ApplicationClientSourceBuiltin {
			t.Errorf("client %s source = %q, want %q（安全不变式必须收敛）", code, got.Source, model.ApplicationClientSourceBuiltin)
		}
	}
}

// TestSeedIamMigratesLegacyApplicationCode 存量库的应用编码为连字符形态（platform-admin /
// tenant-admin）时，种子启动必须原地改名为下划线形态（platform_admin / tenant_admin）：
// 保留主键，因此以 app_id 关联的菜单/订阅/角色全部随之迁移，不会重建出第二个内置应用。
// 回归背景：应用编码规则统一为下划线连接后，若不迁移旧编码，种子会按新编码再建一套应用，
// 而旧应用仍占着 platform-admin 这一唯一键，造成菜单/订阅/删除保护全部错位。
func TestSeedIamMigratesLegacyApplicationCode(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	legacy := &model.ApplicationEntity{
		Code:   "platform-admin",
		Name:   "管理后台",                    // 旧库遗留展示名：归运维，种子不回写
		Source: model.AppSourceThirdParty, // 旧库补列后的默认值，种子应一并纠正为 builtin
		Status: model.AppStatusEnable,
	}
	if err := db.Create(legacy).Error; err != nil {
		t.Fatalf("seed legacy application: %v", err)
	}
	// 旧编码应用名下已有一棵菜单：改名后必须仍挂在同一个 app_id 上
	legacyMenu := &model.MenuEntity{
		AppID:      legacy.ID,
		Name:       "工作台",
		Code:       "dashboard",
		Path:       "/dashboard",
		Type:       model.MenuTypeMenu,
		Visibility: model.MenuVisibilityMember,
		Status:     model.MenuStatusEnable,
	}
	if err := db.Create(legacyMenu).Error; err != nil {
		t.Fatalf("seed legacy menu: %v", err)
	}

	if err := seed.SeedIam(ctx, db); err != nil {
		t.Fatalf("seed fail: %v", err)
	}
	// 再跑一次：改名后必须幂等（不得因为旧编码缺席而新建应用）
	if err := seed.SeedIam(ctx, db); err != nil {
		t.Fatalf("seed (2nd) fail: %v", err)
	}

	var apps []model.ApplicationEntity
	if err := db.Find(&apps).Error; err != nil {
		t.Fatalf("query application: %v", err)
	}
	if len(apps) != 2 {
		t.Fatalf("application count = %d, want 2 (旧编码必须原地改名而非新建)", len(apps))
	}
	var legacyCount int64
	if err := db.Model(&model.ApplicationEntity{}).Where("code = ?", "platform-admin").Count(&legacyCount).Error; err != nil {
		t.Fatalf("count legacy application: %v", err)
	}
	if legacyCount != 0 {
		t.Errorf("legacy application code row count = %d, want 0", legacyCount)
	}

	var migrated model.ApplicationEntity
	if err := db.Where("code = ?", "platform_admin").First(&migrated).Error; err != nil {
		t.Fatalf("query migrated application: %v", err)
	}
	if migrated.ID != legacy.ID {
		t.Errorf("application id = %s, want legacy id %s (改名必须保留主键)", migrated.ID, legacy.ID)
	}
	if migrated.Source != model.AppSourceBuiltin {
		t.Errorf("application source = %q, want %q", migrated.Source, model.AppSourceBuiltin)
	}
	// 展示名归运维（create_only）：编码迁移不得顺带回填名称，旧库/控制台的值原样保留
	if migrated.Name != "管理后台" {
		t.Errorf("application name = %q, want %q (展示名归运维，种子不回写)", migrated.Name, "管理后台")
	}

	var menuCount int64
	if err := db.Model(&model.MenuEntity{}).
		Where("app_id = ? AND code = ?", legacy.ID, "dashboard").Count(&menuCount).Error; err != nil {
		t.Fatalf("count migrated menu: %v", err)
	}
	if menuCount != 1 {
		t.Errorf("menu rows under preserved app_id = %d, want 1", menuCount)
	}
}

// TestSeedIamRejectsConflictingApplicationCode 新旧编码并存（既有连字符内置应用，
// 又有用户自建的 platform_admin）时必须中断种子，而不是静默择一：无法判断哪一行才是内置应用，
// 若按新编码命中就回填 source，会把用户自建应用改写成内置（获得删除保护并接管菜单范围）。
func TestSeedIamRejectsConflictingApplicationCode(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	if err := db.Create(&model.ApplicationEntity{
		Code: "platform-admin", Name: "平台管理后台", Source: model.AppSourceBuiltin, Status: model.AppStatusEnable,
	}).Error; err != nil {
		t.Fatalf("seed legacy application: %v", err)
	}
	userApp := &model.ApplicationEntity{
		Code: "platform_admin", Name: "用户自建应用", Source: model.AppSourceThirdParty, Status: model.AppStatusEnable,
	}
	if err := db.Create(userApp).Error; err != nil {
		t.Fatalf("seed user application: %v", err)
	}

	if err := seed.SeedIam(ctx, db); err == nil {
		t.Fatal("expected conflict error when legacy and new application codes coexist")
	}

	// 冲突行不得被改写：用户自建应用仍为第三方，旧内置应用仍在且未被改名
	var got model.ApplicationEntity
	if err := db.Where("id = ?", userApp.ID).First(&got).Error; err != nil {
		t.Fatalf("query user application: %v", err)
	}
	if got.Source != model.AppSourceThirdParty {
		t.Errorf("user application source = %q, want %q (冲突时不得改写)", got.Source, model.AppSourceThirdParty)
	}
	var legacyCount int64
	if err := db.Model(&model.ApplicationEntity{}).Where("code = ?", "platform-admin").Count(&legacyCount).Error; err != nil {
		t.Fatalf("count legacy application: %v", err)
	}
	if legacyCount != 1 {
		t.Errorf("legacy application row count = %d, want 1 (冲突时不得改名)", legacyCount)
	}
}
