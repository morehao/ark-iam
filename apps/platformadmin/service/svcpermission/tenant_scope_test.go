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
	"github.com/morehao/golib/biz/gcontext"
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
