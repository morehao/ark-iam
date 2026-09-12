package svctenant

import (
	"testing"

	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/tenantadmin/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/tenantadmin/testutil"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

// seedSubscribedApp 种子「应用 + 当前租户对其的启用订阅」；source 区分内置（builtin）与第一方/第三方应用；
// code 传真实种子编码（见 pkg/seed），因为菜单范围按控制台归属（code）判定。
func seedSubscribedApp(t *testing.T, db *gorm.DB, tenantID, appID, code, name string, source model.AppSource, sort int) {
	t.Helper()
	if err := db.Create(&model.ApplicationEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: appID}},
		Code:       code,
		Name:       name,
		Status:     model.AppStatusEnable,
		Source:     source,
		Sort:       sort,
	}).Error; err != nil {
		t.Fatalf("seed app: %v", err)
	}
	if err := db.Create(&model.TenantApplicationEntity{
		BaseEntity:   gormdao.BaseEntity{StringID: gormdao.StringID{ID: "ta-" + appID}},
		TenantID:     tenantID,
		AppID:        appID,
		Status:       model.TenantApplicationStatusEnable,
		Config:       []byte("{}"),
		GrantedScope: []byte("[]"),
	}).Error; err != nil {
		t.Fatalf("seed tenant application: %v", err)
	}
}

// TestTenantAppsIncludeSubscribedBuiltInApp 应用选项 = 租户订阅的启用应用：
// 内置应用（source=builtin，如管理后台）只要被订阅同样可选；未订阅的应用不出现；
// 角色创建与下拉同口径（订阅的内置应用可作为角色归属）。
func TestTenantAppsIncludeSubscribedBuiltInApp(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ApplicationEntity{}, &model.TenantApplicationEntity{},
		&model.RoleEntity{}, &model.UserRoleEntity{})
	seedTenantAdminOperator(t, db, "t1", "op")
	seedSubscribedApp(t, db, "t1", "app-admin", "platform_admin", "管理后台", model.AppSourceBuiltin, 0)
	seedSubscribedApp(t, db, "t1", "app-console", tenantAdminAppCode, "租户自服务", model.AppSourceBuiltin, 1)
	// 平台已建但该租户未订阅的应用：不可选
	if err := db.Create(&model.ApplicationEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "app-unsub"}},
		Code:       "code-app-unsub",
		Name:       "未订阅应用",
		Status:     model.AppStatusEnable,
	}).Error; err != nil {
		t.Fatalf("seed unsubscribed app: %v", err)
	}

	ginCtx := newDeptGinCtx(t, "t1", "op")
	resp, err := NewTenantMenuSvc().Apps(ginCtx)
	if err != nil {
		t.Fatalf("apps: %v", err)
	}
	got := make(map[string]string, len(resp.List))
	for _, item := range resp.List {
		got[item.AppID] = item.Name
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 subscribed apps (built-in app included), got %+v", resp.List)
	}
	if got["app-console"] != "租户自服务" || got["app-admin"] != "管理后台" {
		t.Fatalf("unexpected app options: %+v", resp.List)
	}
	// 顺序与 application.sort 一致
	if resp.List[0].AppID != "app-admin" || resp.List[1].AppID != "app-console" {
		t.Fatalf("app options must follow application.sort, got %+v", resp.List)
	}
	if _, ok := got["app-unsub"]; ok {
		t.Fatalf("unsubscribed app must not be selectable: %+v", resp.List)
	}

	// 订阅的内置应用可作为角色归属
	if _, err := NewRoleSvc().Create(ginCtx, &dtotenant.RoleCreateReq{AppID: "app-admin", Name: "后台管理员"}); err != nil {
		t.Fatalf("create role on subscribed built-in app: %v", err)
	}
}

// TestTenantAppsExcludeDisabledApplication 应用被平台停用（application.status=disable）后，
// 即便订阅关系仍是启用，也不再作为租户可选应用暴露（两道门槛：订阅启用 + 应用启用）。
func TestTenantAppsExcludeDisabledApplication(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ApplicationEntity{}, &model.TenantApplicationEntity{},
		&model.RoleEntity{}, &model.UserRoleEntity{})
	seedTenantAdminOperator(t, db, "t1", "op")
	seedSubscribedApp(t, db, "t1", "app-console", tenantAdminAppCode, "租户自服务", model.AppSourceBuiltin, 0)
	// 平台停用该应用（订阅关系保持 enable）
	if err := db.Model(&model.ApplicationEntity{}).Where("id = ?", "app-console").
		Update("status", model.AppStatusDisable).Error; err != nil {
		t.Fatalf("disable application: %v", err)
	}

	resp, err := NewTenantMenuSvc().Apps(newDeptGinCtx(t, "t1", "op"))
	if err != nil {
		t.Fatalf("apps: %v", err)
	}
	if len(resp.List) != 0 {
		t.Fatalf("disabled application must not be selectable, got %+v", resp.List)
	}
}

// TestConsoleMenuScopeExcludesBuiltInApp 租户控制台菜单范围排除「归属其它控制台的内置应用」：
// 管理后台（source=builtin，code=platform_admin）的菜单由其专属控制台呈现，不并入租户控制台侧边栏（避免串台页面）；
// 租户自服务同为 builtin，但它是本控制台菜单的载体（code=tenant_admin），必须留在范围内。
func TestConsoleMenuScopeExcludesBuiltInApp(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ApplicationEntity{}, &model.TenantApplicationEntity{},
		&model.MenuEntity{}, &model.RoleEntity{}, &model.UserRoleEntity{})
	seedTenantAdminOperator(t, db, "t1", "op")
	seedSubscribedApp(t, db, "t1", "app-admin", "platform_admin", "管理后台", model.AppSourceBuiltin, 0)
	seedSubscribedApp(t, db, "t1", "app-console", tenantAdminAppCode, "租户自服务", model.AppSourceBuiltin, 1)

	menus := []*model.MenuEntity{
		{
			BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "m-console"}},
			AppID:      "app-console",
			Name:       "部门架构",
			Code:       "department",
			Path:       "/department",
			Type:       model.MenuTypeMenu,
			Status:     model.MenuStatusEnable,
		},
		{
			BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "m-admin"}},
			AppID:      "app-admin",
			Name:       "租户管理",
			Code:       "tenant-manage",
			Path:       "/tenant",
			Type:       model.MenuTypeMenu,
			Status:     model.MenuStatusEnable,
		},
	}
	for _, m := range menus {
		if err := db.Create(m).Error; err != nil {
			t.Fatalf("seed menu %s: %v", m.Code, err)
		}
	}

	ginCtx := newDeptGinCtx(t, "t1", "op")
	tree, err := buildMyMenuTree(ginCtx)
	if err != nil {
		t.Fatalf("build my menu tree: %v", err)
	}
	if len(tree) != 1 || tree[0].MenuID != "m-console" {
		t.Fatalf("console menu tree must keep tenant console menus and drop platform console menus, got %+v", tree)
	}
}
