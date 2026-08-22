package svcpermission

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtopermission"
	"github.com/morehao/ark-iam/platformadmin/testutil"
	"github.com/morehao/golib/biz/gcontext"
)

func newGinCtx(tenantID, userID string) *gin.Context {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, tenantID)
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
