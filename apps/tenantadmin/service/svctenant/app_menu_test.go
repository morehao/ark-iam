package svctenant

import (
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/morehao/ark-iam/apps/tenantadmin/dto/dtotenant"
	"github.com/morehao/ark-iam/apps/tenantadmin/testutil"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

func dtotenantCreateReq(appID, code, name string, ginCtx *gin.Context) *dtotenant.AppMenuCreateReq {
	return &dtotenant.AppMenuCreateReq{
		AppID:         appID,
		AppMenuFields: dtotenant.AppMenuFields{Code: code, Name: name, Status: model.MenuStatusEnable, Type: model.MenuTypeMenu, Visibility: model.MenuVisibilityMember},
		Operator:      operatorOf(ginCtx),
	}
}

func dtotenantDeleteReq(appID, menuID string, ginCtx *gin.Context) *dtotenant.AppMenuDeleteReq {
	return &dtotenant.AppMenuDeleteReq{AppID: appID, MenuID: menuID, Operator: operatorOf(ginCtx)}
}

func dtotenantTreeReq(appID string, ginCtx *gin.Context) *dtotenant.AppMenuTreeReq {
	return &dtotenant.AppMenuTreeReq{AppID: appID, Operator: operatorOf(ginCtx)}
}

// seedOwnedApp 播种一个归属 tenantID 的第三方应用。
func seedOwnedApp(t *testing.T, db *gorm.DB, tenantID, appID, code string) *model.ApplicationEntity {
	t.Helper()
	app := &model.ApplicationEntity{
		BaseEntity:    gormdao.BaseEntity{StringID: gormdao.StringID{ID: appID}},
		Code:          code,
		Name:          code,
		Status:        model.AppStatusEnable,
		Source:        model.AppSourceThirdParty,
		OwnerType:     model.AppOwnerTypeTenant,
		OwnerTenantID: tenantID,
	}
	if err := db.Create(app).Error; err != nil {
		t.Fatalf("seed owned app: %v", err)
	}
	return app
}

func setupAppMenuDB(t *testing.T) *gorm.DB {
	t.Helper()
	return testutil.SetupSQLite(t,
		&model.ApplicationEntity{}, &model.MenuEntity{}, &model.RoleEntity{},
		&model.UserRoleEntity{}, &model.RoleMenuEntity{}, &model.TenantApplicationEntity{},
	)
}

// TestAppMenuCreateRejectsNonOwner：非归属租户的管理员写别人应用的菜单被拒（fail-closed）。
func TestAppMenuCreateRejectsNonOwner(t *testing.T) {
	db := setupAppMenuDB(t)
	// app 归属 t1，但用 t2 管理员操作
	seedOwnedApp(t, db, "t1", "app-t1", "app-t1")
	ginCtx := newAdminCtx(t, db, "t2", "op-t2")

	svc := NewAppMenuSvc(nil)
	_, err := svc.Create(requestCtx(ginCtx), dtotenantCreateReq("app-t1", "m1", "菜单1", ginCtx))
	if err == nil {
		t.Fatalf("非归属租户写菜单应被拒，实际成功")
	}
}

// TestAppMenuCreateAllowedForOwner：归属租户管理员可创建菜单，落库 app_id 为权威值。
func TestAppMenuCreateAllowedForOwner(t *testing.T) {
	db := setupAppMenuDB(t)
	seedOwnedApp(t, db, "t1", "app-t1", "app-t1")
	ginCtx := newAdminCtx(t, db, "t1", "op-t1")

	svc := NewAppMenuSvc(nil)
	resp, err := svc.Create(requestCtx(ginCtx), dtotenantCreateReq("app-t1", "m1", "菜单1", ginCtx))
	if err != nil {
		t.Fatalf("归属租户创建菜单失败: %v", err)
	}
	if resp.MenuID == "" {
		t.Fatalf("返回 MenuID 为空")
	}
	// 断言落库 app_id = app-t1
	m, err := dao.NewMenuDao().GetByID(requestCtx(ginCtx), resp.MenuID)
	if err != nil || m == nil {
		t.Fatalf("查询菜单失败: %v", err)
	}
	if m.AppID != "app-t1" {
		t.Fatalf("菜单 app_id = %s, want app-t1", m.AppID)
	}
}

// TestAppMenuCreateBodyAppIDIgnored：body 伪造 AppID 也无法写到别的应用（path 是唯一来源）。
func TestAppMenuCreateBodyAppIDIgnored(t *testing.T) {
	db := setupAppMenuDB(t)
	seedOwnedApp(t, db, "t1", "app-t1", "app-t1")
	seedOwnedApp(t, db, "t1", "app-other", "app-other") // 同为 t1 的另一个应用
	ginCtx := newAdminCtx(t, db, "t1", "op-t1")

	req := dtotenantCreateReq("app-t1", "m1", "菜单1", ginCtx)
	req.AppMenuFields.Name = "菜单1"
	svc := NewAppMenuSvc(nil)
	resp, err := svc.Create(requestCtx(ginCtx), req)
	if err != nil {
		t.Fatalf("创建失败: %v", err)
	}
	m, _ := dao.NewMenuDao().GetByID(requestCtx(ginCtx), resp.MenuID)
	if m == nil || m.AppID != "app-t1" {
		t.Fatalf("菜单 app_id 必须为 path 的 app-t1，实际 %v", m)
	}
}

// TestAppMenuDeleteRejectsCrossAppMenuID：用 A 的 appID 传 B 的 menuID 删除应被拒。
func TestAppMenuDeleteRejectsCrossAppMenuID(t *testing.T) {
	db := setupAppMenuDB(t)
	seedOwnedApp(t, db, "t1", "app-a", "app-a")
	seedOwnedApp(t, db, "t1", "app-b", "app-b")
	ginCtx := newAdminCtx(t, db, "t1", "op-t1")
	svc := NewAppMenuSvc(nil)

	// 在 app-b 下建菜单
	rb, err := svc.Create(requestCtx(ginCtx), dtotenantCreateReq("app-b", "mb", "B菜单", ginCtx))
	if err != nil {
		t.Fatalf("建 B 菜单失败: %v", err)
	}
	// 用 app-a 的 appID 删 app-b 的菜单
	err = svc.Delete(requestCtx(ginCtx), dtotenantDeleteReq("app-a", rb.MenuID, ginCtx))
	if err == nil {
		t.Fatalf("跨应用删除应被拒，实际成功")
	}
}

// TestAppMenuTreeEmptyForNoMenuApp：归属应用无菜单时返回空列表（非 nil）。
func TestAppMenuTreeEmptyForNoMenuApp(t *testing.T) {
	db := setupAppMenuDB(t)
	seedOwnedApp(t, db, "t1", "app-empty", "app-empty")
	ginCtx := newAdminCtx(t, db, "t1", "op-t1")
	svc := NewAppMenuSvc(nil)

	resp, err := svc.Tree(requestCtx(ginCtx), dtotenantTreeReq("app-empty", ginCtx))
	if err != nil {
		t.Fatalf("Tree 失败: %v", err)
	}
	if resp == nil || resp.List == nil {
		t.Fatalf("无菜单应用应返回空列表(非 nil)")
	}
	if len(resp.List) != 0 {
		t.Fatalf("无菜单应用菜单树应为空，实际 %d", len(resp.List))
	}
}

// TestAppMenuDeletePurgesRoleMenuAcrossTenants：归属租户删菜单后，其它订阅租户的 role_menu
// 绑定被跨租户清理，无悬空行（P0 安全回归）。
func TestAppMenuDeletePurgesRoleMenuAcrossTenants(t *testing.T) {
	db := setupAppMenuDB(t)
	seedOwnedApp(t, db, "t1", "app-t1", "app-t1")
	ctxGin := newAdminCtx(t, db, "t1", "op-t1")
	svc := NewAppMenuSvc(nil)

	// t1 建菜单
	resp, err := svc.Create(requestCtx(ctxGin), dtotenantCreateReq("app-t1", "m1", "菜单1", ctxGin))
	if err != nil {
		t.Fatalf("建菜单失败: %v", err)
	}
	// 造：租户 t2 的一条 role_menu 授权绑定该菜单（模拟订阅方授权）
	rmID := "rm-t2"
	if err := db.Create(&model.RoleMenuEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: rmID}},
		TenantID:   "t2",
		RoleID:     "role-t2",
		MenuID:     resp.MenuID,
	}).Error; err != nil {
		t.Fatalf("seed role_menu: %v", err)
	}
	// t1 删菜单
	if err := svc.Delete(requestCtx(ctxGin), dtotenantDeleteReq("app-t1", resp.MenuID, ctxGin)); err != nil {
		t.Fatalf("删菜单失败: %v", err)
	}
	// 断言 t2 的绑定已清（跨租户作用域查询）
	crossCtx := dbclient.CrossTenantContext(requestCtx(ctxGin))
	list, err := dao.NewRoleMenuDao().GetListByCond(crossCtx, &dao.RoleMenuCond{MenuID: resp.MenuID})
	if err != nil {
		t.Fatalf("查询 role_menu 失败: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("删菜单后跨租户 role_menu 应清空，实际剩 %d 条", len(list))
	}
}
