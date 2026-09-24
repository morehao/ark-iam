package seed_test

import (
	"context"
	"sort"
	"testing"

	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/seed"
	"gorm.io/gorm"
)

// 设计 §3.3「数据清单」的可执行形态：L1 一次引导后，12 张表 / 43 行的逐项内容。
//
// 这是整个改造的黄金用例——P1（参数化）、P3（删机制）、P4（回调地址配置化）都不得让它变红，
// 它同时也是"字段权威矩阵 immut/ create_only 归属"的实测依据。
// 基线采集见 .dsh/docs/notes/2026-09-21-seed-golden-baseline.md。
func TestBootstrap_GoldenDataManifest(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	if _, status, err := seed.Bootstrap(ctx, db, testDefinition(t)); err != nil {
		t.Fatalf("bootstrap: %v", err)
	} else if status != seed.StatusCreated {
		t.Fatalf("status = %q, want %q", status, seed.StatusCreated)
	}

	// ---- 行数：12 张表，合计 43 行 ----
	wantCounts := []struct {
		table string
		want  int64
	}{
		{model.TableNameTenant, 1},
		{model.TableNameDepartment, 1},
		{model.TableNameApplication, 2},
		{model.TableNameRole, 2},
		{model.TableNameMenu, 14},
		{model.TableNameRoleMenu, 14},
		{model.TableNameTenantApplication, 2},
		{model.TableNamePerson, 1},
		{model.TableNameUser, 1},
		{model.TableNameDepartmentUser, 1},
		{model.TableNameUserRole, 2},
		{model.TableNameApplicationClient, 2},
	}
	var total int64
	for _, c := range wantCounts {
		got := countRows(t, db, c.table)
		if got != c.want {
			t.Errorf("表 %s 行数 = %d, want %d", c.table, got, c.want)
		}
		total += got
	}
	if total != 43 {
		t.Errorf("12 张表合计 = %d 行, want 43", total)
	}

	// ---- 租户与根部门 ----
	var tenant model.TenantEntity
	if err := db.Where("code = ?", model.SeedPlatformTenantCode).First(&tenant).Error; err != nil {
		t.Fatalf("query tenant: %v", err)
	}
	if tenant.Type != model.TenantTypePlatform || tenant.Status != model.TenantStatusActive {
		t.Errorf("tenant type/status = %q/%q", tenant.Type, tenant.Status)
	}
	if tenant.Tag != "default" || tenant.DbUser != "default_user" {
		t.Errorf("tenant tag/db_user = %q/%q", tenant.Tag, tenant.DbUser)
	}
	var rootDept model.DepartmentEntity
	if err := db.Where("tenant_id = ? AND parent_id = ?", tenant.ID, "").First(&rootDept).Error; err != nil {
		t.Fatalf("query root department: %v", err)
	}
	if rootDept.Status != model.DeptNodeStatusEnable {
		t.Errorf("root dept status = %q", rootDept.Status)
	}
	if rootDept.DeptDepth != 1 || rootDept.DeptPath != "/"+rootDept.ID {
		t.Errorf("root dept path/depth = %q/%d, want /%s/1", rootDept.DeptPath, rootDept.DeptDepth, rootDept.ID)
	}

	// ---- 应用：两个内置应用，编码与归属应用排序固定 ----
	apps := map[string]model.ApplicationEntity{}
	var appList []model.ApplicationEntity
	if err := db.Order("sort asc").Find(&appList).Error; err != nil {
		t.Fatalf("query applications: %v", err)
	}
	for _, app := range appList {
		apps[app.Code] = app
		if app.Source != model.AppSourceBuiltin {
			t.Errorf("应用 %s source = %q, want builtin", app.Code, app.Source)
		}
	}
	for _, code := range []string{"platform_admin", "tenant_admin"} {
		if _, ok := apps[code]; !ok {
			t.Fatalf("内置应用 %s 缺失（现有 %v）", code, keysOf(apps))
		}
	}
	if apps["platform_admin"].Sort != 0 || apps["tenant_admin"].Sort != 1 {
		t.Errorf("应用 sort = %d/%d, want 0/1", apps["platform_admin"].Sort, apps["tenant_admin"].Sort)
	}

	// ---- 角色：平台 admin 角色 code/admin_type/source 固定 ----
	var platformRole model.RoleEntity
	if err := db.Where("app_id = ?", apps["platform_admin"].ID).First(&platformRole).Error; err != nil {
		t.Fatalf("query platform admin role: %v", err)
	}
	if platformRole.Code != "platform_admin" || platformRole.Name != "管理员" {
		t.Errorf("平台角色 code/name = %q/%q, want platform_admin/管理员", platformRole.Code, platformRole.Name)
	}
	if platformRole.AdminType != model.SysAdminTypeAdmin || platformRole.Source != model.RoleSourceBuiltin {
		t.Errorf("平台角色 admin_type/source = %q/%q", platformRole.AdminType, platformRole.Source)
	}
	var tenantRole model.RoleEntity
	if err := db.Where("app_id = ?", apps["tenant_admin"].ID).First(&tenantRole).Error; err != nil {
		t.Fatalf("query tenant admin role: %v", err)
	}
	if tenantRole.AdminType != model.SysAdminTypeAdmin || tenantRole.Source != model.RoleSourceBuiltin {
		t.Errorf("租户角色 admin_type/source = %q/%q", tenantRole.AdminType, tenantRole.Source)
	}

	// ---- 菜单：14 个 seed_key，父级关系与类型/可见性 ----
	// 列顺序即「平台管理后台（3 目录 + 7 页）→ 租户管理后台（4 页）」。
	wantMenus := []struct {
		seedKey    string
		appCode    string
		parentCode string
		menuType   model.MenuType
		visibility model.MenuVisibility
	}{
		{"dashboard", "platform_admin", "", model.MenuTypeMenu, model.MenuVisibilityMember},
		{"grp-tenant", "platform_admin", "", model.MenuTypeDirectory, model.MenuVisibilityAdmin},
		{"tenant", "platform_admin", "grp-tenant", model.MenuTypeMenu, model.MenuVisibilityAdmin},
		{"tenant-application", "platform_admin", "grp-tenant", model.MenuTypeMenu, model.MenuVisibilityAdmin},
		{"domain", "platform_admin", "grp-tenant", model.MenuTypeMenu, model.MenuVisibilityAdmin},
		{"grp-app", "platform_admin", "", model.MenuTypeDirectory, model.MenuVisibilityAdmin},
		{"application", "platform_admin", "grp-app", model.MenuTypeMenu, model.MenuVisibilityAdmin},
		{"oauth-client", "platform_admin", "grp-app", model.MenuTypeMenu, model.MenuVisibilityAdmin},
		{"grp-platform", "platform_admin", "", model.MenuTypeDirectory, model.MenuVisibilityAdmin},
		{"menu", "platform_admin", "grp-platform", model.MenuTypeMenu, model.MenuVisibilityAdmin},
		{"department", "tenant_admin", "", model.MenuTypeMenu, model.MenuVisibilityAdmin},
		{"tenant-user", "tenant_admin", "", model.MenuTypeMenu, model.MenuVisibilityAdmin},
		{"tenant-role", "tenant_admin", "", model.MenuTypeMenu, model.MenuVisibilityAdmin},
		{"tenant-api-key", "tenant_admin", "", model.MenuTypeMenu, model.MenuVisibilityAdmin},
	}
	menusBySeedKey := map[string]model.MenuEntity{}
	var menuList []model.MenuEntity
	if err := db.Find(&menuList).Error; err != nil {
		t.Fatalf("query menus: %v", err)
	}
	for _, m := range menuList {
		if m.SeedKey == "" {
			t.Errorf("菜单 %s 缺 seed_key：种子身份键是认行的唯一依据", m.Code)
		}
		menusBySeedKey[m.SeedKey] = m
	}
	if len(menusBySeedKey) != len(wantMenus) {
		t.Errorf("菜单 seed_key 数 = %d, want %d（现有 %v）", len(menusBySeedKey), len(wantMenus), keysOf(menusBySeedKey))
	}
	appIDByCode := map[string]string{"platform_admin": apps["platform_admin"].ID, "tenant_admin": apps["tenant_admin"].ID}
	for _, want := range wantMenus {
		got, ok := menusBySeedKey[want.seedKey]
		if !ok {
			t.Errorf("菜单 seed_key %s 缺失", want.seedKey)
			continue
		}
		if got.AppID != appIDByCode[want.appCode] {
			t.Errorf("菜单 %s 归属应用 = %s, want %s", want.seedKey, got.AppID, want.appCode)
		}
		if got.Type != want.menuType {
			t.Errorf("菜单 %s type = %q, want %q", want.seedKey, got.Type, want.menuType)
		}
		if got.Visibility != want.visibility {
			t.Errorf("菜单 %s visibility = %q, want %q", want.seedKey, got.Visibility, want.visibility)
		}
		// 父级：按 seed_key 解析（顶层为 ""）
		wantParentID := ""
		if want.parentCode != "" {
			parent, ok := menusBySeedKey[want.parentCode]
			if !ok {
				t.Errorf("菜单 %s 的父级 %s 缺失", want.seedKey, want.parentCode)
				continue
			}
			wantParentID = parent.ID
		}
		if got.ParentID != wantParentID {
			t.Errorf("菜单 %s parent_id = %q, want %q（父 seed_key=%q）", want.seedKey, got.ParentID, wantParentID, want.parentCode)
		}
		// 5 个未由种子携带、走列默认值的字段：菜单管理页可填全，但种子不写
		if got.Redirect != "" {
			t.Errorf("菜单 %s redirect = %q, want 空（列默认）", want.seedKey, got.Redirect)
		}
		if got.Hidden != model.MenuHiddenFlagDisable {
			t.Errorf("菜单 %s hidden = %q, want disable", want.seedKey, got.Hidden)
		}
		if got.ExternalLink != model.MenuExternalLinkFlagDisable {
			t.Errorf("菜单 %s external_link = %q, want disable", want.seedKey, got.ExternalLink)
		}
		if got.KeepAlive != model.MenuKeepAliveFlagDisable {
			t.Errorf("菜单 %s keep_alive = %q, want disable", want.seedKey, got.KeepAlive)
		}
		if got.Status != model.MenuStatusEnable {
			t.Errorf("菜单 %s status = %q, want enable", want.seedKey, got.Status)
		}
	}

	// ---- role_menu：平台 admin 角色 10 条 + 租户 admin 角色 4 条 = 14 ----
	// 目录菜单从不授权；平台角色的 10 条含 3 个 tenant_admin 应用菜单（department/tenant-user/tenant-role），
	// 这是现状（便于平台侧改后无需重新授权），改造中不得"顺手纠正"。
	wantPlatformRoleMenus := []string{
		"dashboard", "menu", "tenant", "application", "tenant-application",
		"oauth-client", "domain", "department", "tenant-user", "tenant-role",
	}
	assertRoleMenuSeedKeys(t, db, tenant.ID, platformRole.ID, menusBySeedKey, wantPlatformRoleMenus)
	assertRoleMenuSeedKeys(t, db, tenant.ID, tenantRole.ID, menusBySeedKey,
		[]string{"department", "tenant-user", "tenant-role", "tenant-api-key"})

	// ---- 内置 OIDC 客户端 ----
	wantClients := []struct {
		code                 string
		appCode              string
		redirectURI          string
		postLogoutURI        string
		backChannelLogoutURI string
	}{
		{
			code: model.SeedBuiltinClientPlatformAdminWeb, appCode: "platform_admin",
			redirectURI:          "http://localhost:4001/auth/callback",
			postLogoutURI:        "http://localhost:4001/login",
			backChannelLogoutURI: "http://localhost:8100/oidc/bc-logout/platform",
		},
		{
			code: model.SeedBuiltinClientTenantAdminWeb, appCode: "tenant_admin",
			redirectURI:          "http://localhost:4002/auth/callback",
			postLogoutURI:        "http://localhost:4002/login",
			backChannelLogoutURI: "http://localhost:8100/oidc/bc-logout/tenant",
		},
	}
	for _, want := range wantClients {
		var client model.ApplicationClientEntity
		if err := db.Where("code = ?", want.code).First(&client).Error; err != nil {
			t.Fatalf("query client %s: %v", want.code, err)
		}
		if client.AppID != appIDByCode[want.appCode] {
			t.Errorf("客户端 %s app_id = %s, want %s", want.code, client.AppID, want.appCode)
		}
		if len(client.RedirectURIs) != 1 || client.RedirectURIs[0] != want.redirectURI {
			t.Errorf("客户端 %s redirect_uris = %v, want [%s]", want.code, client.RedirectURIs, want.redirectURI)
		}
		if len(client.PostLogoutRedirectURIs) != 1 || client.PostLogoutRedirectURIs[0] != want.postLogoutURI {
			t.Errorf("客户端 %s post_logout_redirect_uris = %v, want [%s]", want.code, client.PostLogoutRedirectURIs, want.postLogoutURI)
		}
		if client.BackChannelLogoutURI != want.backChannelLogoutURI {
			t.Errorf("客户端 %s back_channel_logout_uri = %q, want %q", want.code, client.BackChannelLogoutURI, want.backChannelLogoutURI)
		}
		if client.Source != model.ApplicationClientSourceBuiltin {
			t.Errorf("客户端 %s source = %q, want builtin", want.code, client.Source)
		}
		if client.RequirePKCE != model.ClientPKCEPolicyEnable {
			t.Errorf("客户端 %s require_pkce = %q, want enable（PKCE 未强制意味着授权码可被重放）", want.code, client.RequirePKCE)
		}
		// 内置客户端是纯浏览器 SPA：必须登记为公共客户端（none），不得持有/要求客户端密钥
		// （RFC 6749 §10.1、RFC 10017 §6.3.3.1）；登记成 basic/post 会让该控制台登录当场不可用。
		if client.TokenEndpointAuthMethod != model.TokenEndpointAuthMethodNone {
			t.Errorf("客户端 %s token_endpoint_auth_method = %q, want none（浏览器客户端不得登记为机密客户端）",
				want.code, client.TokenEndpointAuthMethod)
		}
	}

	// ---- 订阅：平台租户订阅两个应用 ----
	for _, code := range []string{"platform_admin", "tenant_admin"} {
		var n int64
		if err := db.Model(&model.TenantApplicationEntity{}).
			Where("tenant_id = ? AND app_id = ?", tenant.ID, appIDByCode[code]).Count(&n).Error; err != nil {
			t.Fatalf("count tenant_application %s: %v", code, err)
		}
		if n != 1 {
			t.Errorf("平台租户订阅 %s 的行数 = %d, want 1", code, n)
		}
	}

	// ---- 管理员：person/user/部门归属/两个角色 ----
	var person model.PersonEntity
	if err := db.Where("username = ?", "admin").First(&person).Error; err != nil {
		t.Fatalf("query admin person: %v", err)
	}
	var adminUser model.UserEntity
	if err := db.Where("tenant_id = ? AND person_id = ?", tenant.ID, person.ID).First(&adminUser).Error; err != nil {
		t.Fatalf("query admin user: %v", err)
	}
	if adminUser.OwnerType != model.OwnerTypeOwner || adminUser.Source != model.UserSourceBuiltin {
		t.Errorf("管理员 owner_type/source = %q/%q", adminUser.OwnerType, adminUser.Source)
	}
	if adminUser.Status != model.UserStatusActive || adminUser.JoinedAt == nil {
		t.Errorf("管理员 status/joined_at = %q/%v", adminUser.Status, adminUser.JoinedAt)
	}
	var deptUser model.DepartmentUserEntity
	if err := db.Where("tenant_id = ? AND user_id = ? AND department_id = ?", tenant.ID, adminUser.ID, rootDept.ID).
		First(&deptUser).Error; err != nil {
		t.Fatalf("query admin department relation: %v", err)
	}
	if deptUser.RelationType != model.DeptUserRelationPrimary {
		t.Errorf("管理员部门关系 = %q, want primary", deptUser.RelationType)
	}
	for _, roleID := range []string{platformRole.ID, tenantRole.ID} {
		var n int64
		if err := db.Model(&model.UserRoleEntity{}).
			Where("tenant_id = ? AND user_id = ? AND role_id = ?", tenant.ID, adminUser.ID, roleID).Count(&n).Error; err != nil {
			t.Fatalf("count user_role: %v", err)
		}
		if n != 1 {
			t.Errorf("管理员与角色 %s 的 user_role 行数 = %d, want 1", roleID, n)
		}
	}
}

// assertRoleMenuSeedKeys 断言某角色被授权的菜单 seed_key 集合恰好等于 want。
func assertRoleMenuSeedKeys(t *testing.T, db *gorm.DB, tenantID, roleID string, menus map[string]model.MenuEntity, want []string) {
	t.Helper()
	var rows []model.RoleMenuEntity
	if err := db.Where("tenant_id = ? AND role_id = ?", tenantID, roleID).Find(&rows).Error; err != nil {
		t.Fatalf("query role_menu: %v", err)
	}
	seedKeyByMenuID := make(map[string]string, len(menus))
	for seedKey, m := range menus {
		seedKeyByMenuID[m.ID] = seedKey
	}
	got := make([]string, 0, len(rows))
	for _, row := range rows {
		seedKey, ok := seedKeyByMenuID[row.MenuID]
		if !ok {
			t.Errorf("role_menu 指向未知菜单 id=%s", row.MenuID)
			continue
		}
		got = append(got, seedKey)
	}
	sort.Strings(got)
	sortedWant := append([]string(nil), want...)
	sort.Strings(sortedWant)
	if len(got) != len(sortedWant) {
		t.Fatalf("角色 %s 授权菜单数 = %d, want %d（got=%v want=%v）", roleID, len(got), len(sortedWant), got, sortedWant)
	}
	for i := range got {
		if got[i] != sortedWant[i] {
			t.Errorf("角色 %s 授权菜单[%d] = %s, want %s（got=%v want=%v）", roleID, i, got[i], sortedWant[i], got, sortedWant)
		}
	}
}

// keysOf 返回 map 的键（升序），用于让失败信息可读。
func keysOf[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
