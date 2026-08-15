package svcpermission

import (
	"testing"

	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtopermission"
	"github.com/morehao/ark-iam/platformadmin/testutil"
)

func TestMenuDetailRejectsCrossTenantEntity(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.MenuEntity{})
	ctx := newGinCtx("21", "0")

	menu := &model.MenuEntity{Name: "menu-a"}
	if err := db.Create(menu).Error; err != nil {
		t.Fatalf("seed menu: %v", err)
	}

	// 菜单无租户维度，跨租户实体 detail 不拒绝
	svc := &menuSvc{}
	_, err := svc.Detail(ctx, &dtopermission.MenuDetailReq{MenuID: menu.ID})
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

	svc := &menuSvc{}
	resp, err := svc.PageList(ctx, &dtopermission.MenuPageListReq{AppID: "10"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Total != 1 {
		t.Fatalf("expected only application 10 menus, total=%d", resp.Total)
	}
	if len(resp.List) != 1 || resp.List[0].AppID != "10" {
		t.Fatalf("expected application 10 menu, got %+v", resp.List)
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

	svc := &menuSvc{}
	resp, err := svc.Tree(ctx, &dtopermission.MenuTreeReq{AppID: "10"})
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

func TestRoleDetailRejectsCrossTenantEntity(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.RoleEntity{})
	ctx := newGinCtx("31", "0")

	role := &model.RoleEntity{TenantID: "77", Name: "other-tenant-role"}
	if err := db.Create(role).Error; err != nil {
		t.Fatalf("seed role: %v", err)
	}

	svc := &roleSvc{}
	_, err := svc.Detail(ctx, &dtopermission.RoleDetailReq{RoleID: role.ID})
	if err == nil {
		t.Fatalf("expected cross-tenant role detail to fail")
	}
}

func TestRolePageListUsesContextTenant(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.RoleEntity{})
	ctx := newGinCtx("32", "0")

	if err := db.Create(&model.RoleEntity{TenantID: "32", Name: "r32"}).Error; err != nil {
		t.Fatalf("seed tenant32: %v", err)
	}
	if err := db.Create(&model.RoleEntity{TenantID: "99", Name: "r99"}).Error; err != nil {
		t.Fatalf("seed tenant99: %v", err)
	}

	svc := &roleSvc{}
	resp, err := svc.PageList(ctx, &dtopermission.RolePageListReq{TenantID: "99"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Total != 1 || len(resp.List) != 1 || resp.List[0].TenantID != "32" {
		t.Fatalf("expected tenant 32 from context, got %+v", resp.List)
	}
}

func TestUserRoleCreateRejectsCrossTenantRole(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.RoleEntity{}, &model.UserRoleEntity{})
	ctx := newGinCtx("33", "1001")

	role := &model.RoleEntity{TenantID: "44", Name: "other-tenant-role"}
	if err := db.Create(role).Error; err != nil {
		t.Fatalf("seed role: %v", err)
	}

	svc := &userRoleSvc{}
	_, err := svc.Create(ctx, &dtopermission.UserRoleCreateReq{TenantID: "33", UserID: "1", RoleID: role.ID})
	if err == nil {
		t.Fatalf("expected cross-tenant role reference to fail")
	}

	left, err := dao.NewUserRoleDao().GetListByCond(ctx, &dao.UserRoleCond{TenantID: "33"})
	if err != nil {
		t.Fatalf("GetListByCond: %v", err)
	}
	if len(left) != 0 {
		t.Fatalf("expected no insert for cross-tenant role, got %+v", left)
	}
}

func TestResourceDetailRejectsCrossTenantEntity(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ResourceEntity{})
	ctx := newGinCtx("41", "0")

	res := &model.ResourceEntity{TenantID: "66", Name: "other-tenant-resource"}
	if err := db.Create(res).Error; err != nil {
		t.Fatalf("seed resource: %v", err)
	}

	svc := &resourceSvc{}
	_, err := svc.Detail(ctx, &dtopermission.ResourceDetailReq{ResourceID: res.ID})
	if err == nil {
		t.Fatalf("expected cross-tenant resource detail to fail")
	}
}

func TestResourcePageListUsesContextTenant(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ResourceEntity{})
	ctx := newGinCtx("42", "0")

	if err := db.Create(&model.ResourceEntity{TenantID: "42", Name: "r42"}).Error; err != nil {
		t.Fatalf("seed tenant42: %v", err)
	}
	if err := db.Create(&model.ResourceEntity{TenantID: "99", Name: "r99"}).Error; err != nil {
		t.Fatalf("seed tenant99: %v", err)
	}

	svc := &resourceSvc{}
	resp, err := svc.PageList(ctx, &dtopermission.ResourcePageListReq{TenantID: "99"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Total != 1 || len(resp.List) != 1 || resp.List[0].TenantID != "42" {
		t.Fatalf("expected tenant 42 from context, got %+v", resp.List)
	}
}

func TestScopeDetailRejectsCrossTenantEntity(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ScopeEntity{})
	ctx := newGinCtx("51", "0")

	scope := &model.ScopeEntity{TenantID: "65", ResourceID: "1", Name: "other-tenant-scope"}
	if err := db.Create(scope).Error; err != nil {
		t.Fatalf("seed scope: %v", err)
	}

	svc := &scopeSvc{}
	_, err := svc.Detail(ctx, &dtopermission.ScopeDetailReq{ScopeID: scope.ID})
	if err == nil {
		t.Fatalf("expected cross-tenant scope detail to fail")
	}
}

func TestScopePageListUsesContextTenant(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ScopeEntity{})
	ctx := newGinCtx("52", "0")

	if err := db.Create(&model.ScopeEntity{TenantID: "52", ResourceID: "1", Name: "s52"}).Error; err != nil {
		t.Fatalf("seed tenant52: %v", err)
	}
	if err := db.Create(&model.ScopeEntity{TenantID: "99", ResourceID: "1", Name: "s99"}).Error; err != nil {
		t.Fatalf("seed tenant99: %v", err)
	}

	svc := &scopeSvc{}
	resp, err := svc.PageList(ctx, &dtopermission.ScopePageListReq{TenantID: "99"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Total != 1 || len(resp.List) != 1 || resp.List[0].TenantID != "52" {
		t.Fatalf("expected tenant 52 from context, got %+v", resp.List)
	}
}

func TestScopeCreateRejectsCrossTenantResource(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ResourceEntity{}, &model.ScopeEntity{})
	ctx := newGinCtx("53", "1002")

	res := &model.ResourceEntity{TenantID: "64", Name: "other-tenant-resource"}
	if err := db.Create(res).Error; err != nil {
		t.Fatalf("seed resource: %v", err)
	}

	svc := &scopeSvc{}
	_, err := svc.Create(ctx, &dtopermission.ScopeCreateReq{TenantID: "53", ResourceID: res.ID, Name: "read"})
	if err == nil {
		t.Fatalf("expected cross-tenant resource reference to fail")
	}

	left, err := dao.NewScopeDao().GetListByCond(ctx, &dao.ScopeCond{TenantID: "53"})
	if err != nil {
		t.Fatalf("GetListByCond: %v", err)
	}
	if len(left) != 0 {
		t.Fatalf("expected no insert for cross-tenant resource, got %+v", left)
	}
}
