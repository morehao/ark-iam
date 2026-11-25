package svcpermission

import (
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/apps/platformadmin/dto/dtopermission"
	"github.com/morehao/ark-iam/apps/platformadmin/testutil"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/object/objpermission"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/dbaccess/gormdao"
)

func newGinCtx(tenantID, userID string) *gin.Context {
	ginCtx, _ := gin.CreateTestContext(nil)
	if ginCtx.Request == nil {
		ginCtx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	}
	ginCtx.Set(gcontext.KeyTenantID, tenantID)
	gincontext.SetTenantScope(ginCtx, gcontext.CurrentScope(tenantID))
	ginCtx.Set(gcontext.KeyUserID, userID)
	return ginCtx
}

func TestMenuDetailRejectsCrossTenantEntity(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.MenuEntity{})
	ctx := newGinCtx("21", "0")

	menu := &model.MenuEntity{Name: "menu-a"}
	if err := db.Create(menu).Error; err != nil {
		t.Fatalf("seed menu: %v", err)
	}

	// 菜单无租户维度，跨租户实体 detail 不拒绝
	svc := &menuSvc{db: dbclient.IamDB}
	_, err := svc.Detail(requestCtx(ctx), &dtopermission.MenuDetailReq{MenuID: menu.ID})
	if err != nil {
		t.Fatalf("unexpected error for menu detail: %v", err)
	}
}

func TestMenuPageListUsesAppID(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.MenuEntity{})
	ctx := newGinCtx("22", "0")

	if err := db.Create(&model.MenuEntity{AppID: "10", Name: "m10"}).Error; err != nil {
		t.Fatalf("seed app10: %v", err)
	}
	if err := db.Create(&model.MenuEntity{AppID: "20", Name: "m20"}).Error; err != nil {
		t.Fatalf("seed app20: %v", err)
	}

	svc := &menuSvc{db: dbclient.IamDB}
	resp, err := svc.PageList(requestCtx(ctx), &dtopermission.MenuPageListReq{AppID: "10"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Total != 1 {
		t.Fatalf("expected only application 10 menus, total=%d", resp.Total)
	}
	if len(resp.List) != 1 || resp.List[0].AppID != "10" {
		t.Fatalf("expected application 10 menu, got %+v", resp.List)
	}
	// 列表必须同时回传创建时间与更新时间（前端「创建时间」「更新时间」两列直读）
	if item := resp.List[0]; item.CreatedAt <= 0 || item.UpdatedAt <= 0 {
		t.Fatalf("menu createdAt/updatedAt not returned: %+v", item)
	}
}

func TestMenuTreeUsesAppID(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.MenuEntity{})
	ctx := newGinCtx("23", "0")

	if err := db.Create(&model.MenuEntity{AppID: "10", Name: "root10", ParentID: ""}).Error; err != nil {
		t.Fatalf("seed app10 root: %v", err)
	}
	if err := db.Create(&model.MenuEntity{AppID: "20", Name: "root20", ParentID: ""}).Error; err != nil {
		t.Fatalf("seed app20 root: %v", err)
	}

	svc := &menuSvc{db: dbclient.IamDB}
	resp, err := svc.Tree(requestCtx(ctx), &dtopermission.MenuTreeReq{AppID: "10"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.List) != 1 {
		t.Fatalf("expected only application 10 root menu, got %+v", resp.List)
	}
	if resp.List[0].AppID != "10" {
		t.Fatalf("expected application 10, got %+v", resp.List[0])
	}
}

func TestMenuCreatePersistsVisibilityAndStatusAndType(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.MenuEntity{})
	ctx := newGinCtx("20", "0")

	svc := &menuSvc{db: dbclient.IamDB}
	resp, err := svc.Create(requestCtx(ctx), &dtopermission.MenuCreateReq{
		MenuBaseInfo: objpermission.MenuBaseInfo{
			AppID:      "10",
			Name:       "工作台",
			Code:       "dashboard",
			Type:       model.MenuTypeMenu,
			Visibility: model.MenuVisibilityMember,
			Status:     model.MenuStatusEnable,
		},

		Operator: operatorOf(ctx)})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	var got model.MenuEntity
	if err := db.First(&got, "id = ?", resp.MenuID).Error; err != nil {
		t.Fatalf("load menu: %v", err)
	}
	if got.Visibility != model.MenuVisibilityMember {
		t.Fatalf("expected visibility member, got %q", got.Visibility)
	}
	if got.Type != model.MenuTypeMenu {
		t.Fatalf("expected type menu, got %q", got.Type)
	}
	if got.Status != model.MenuStatusEnable {
		t.Fatalf("expected status enable, got %q", got.Status)
	}
}

func TestMenuCreateRejectsInvalidEnums(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.MenuEntity{})
	ctx := newGinCtx("21", "0")

	svc := &menuSvc{db: dbclient.IamDB}
	_, err := svc.Create(requestCtx(ctx), &dtopermission.MenuCreateReq{
		MenuBaseInfo: objpermission.MenuBaseInfo{
			AppID:      "10",
			Name:       "x",
			Code:       "x",
			Type:       model.MenuTypeButton,
			Visibility: model.MenuVisibility("illegal"),
			Status:     model.MenuStatusEnable,
		},

		Operator: operatorOf(ctx)})
	if err == nil {
		t.Fatalf("expected invalid visibility to be rejected")
	}
	var count int64
	if err := db.Model(&model.MenuEntity{}).Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 rows created, got %d", count)
	}
}

func TestMenuUpdatePersistsVisibility(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.MenuEntity{})
	ctx := newGinCtx("22", "0")

	menu := &model.MenuEntity{
		AppID:      "10",
		Name:       "m",
		Code:       "m",
		Type:       model.MenuTypeMenu,
		Visibility: model.MenuVisibilityPublic,
		Status:     model.MenuStatusEnable,
	}
	if err := db.Create(menu).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	svc := &menuSvc{db: dbclient.IamDB}
	err := svc.Update(requestCtx(ctx), &dtopermission.MenuUpdateReq{
		MenuID: menu.ID,
		MenuBaseInfo: objpermission.MenuBaseInfo{
			AppID:      "10",
			Name:       "m",
			Code:       "m",
			Type:       model.MenuTypeMenu,
			Visibility: model.MenuVisibilityAdmin,
			Status:     model.MenuStatusEnable,
		},

		Operator: operatorOf(ctx)})
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	var got model.MenuEntity
	if err := db.First(&got, "id = ?", menu.ID).Error; err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.Visibility != model.MenuVisibilityAdmin {
		t.Fatalf("expected visibility admin after update, got %q", got.Visibility)
	}
}

// TestMenuDeletePurgesRoleMenuAcrossTenants 回归 P0-3：
// menu 是全局表、role_menu 是租户级表。删除一个已被多租户角色授权的全局菜单时，
// 必须在**全部租户**范围内清除 role_menu 绑定，否则其它业务租户会残留指向已删菜单的悬空授权。
func TestMenuDeletePurgesRoleMenuAcrossTenants(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.MenuEntity{}, &model.RoleMenuEntity{})
	platformTenant, businessTenant := "21", "99"
	ctx := newGinCtx(platformTenant, "0")
	svc := NewMenuSvc(dbclient.IamDB)

	menu := &model.MenuEntity{AppID: "10", Name: "shared-menu", ParentID: ""}
	if err := db.Create(menu).Error; err != nil {
		t.Fatalf("seed menu: %v", err)
	}
	// 两个租户都给该菜单绑定了角色授权。
	for _, tid := range []string{platformTenant, businessTenant} {
		if err := db.Create(&model.RoleMenuEntity{
			BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "rm-" + tid}},
			TenantID:   tid, RoleID: "role-" + tid, MenuID: menu.ID,
		}).Error; err != nil {
			t.Fatalf("seed role_menu %s: %v", tid, err)
		}
	}

	if err := svc.Delete(requestCtx(ctx), &dtopermission.MenuDeleteReq{MenuID: menu.ID}); err != nil {
		t.Fatalf("delete menu: %v", err)
	}

	// 两个租户的绑定都应被清除（软删，deleted_at 非空）——按主键直接查原始行校验。
	// 校验查询跨全部租户，需显式全租户作用域。
	var alive int64
	if err := db.WithContext(dbclient.CrossTenantContext(requestCtx(ctx))).Model(&model.RoleMenuEntity{}).
		Where("menu_id = ? AND deleted_at IS NULL", menu.ID).Count(&alive).Error; err != nil {
		t.Fatalf("count role_menu: %v", err)
	}
	if alive != 0 {
		t.Fatalf("menu delete must purge role_menu bindings in ALL tenants, %d still alive", alive)
	}
}
