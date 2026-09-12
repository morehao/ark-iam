package svcpermission

import (
	"testing"

	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtopermission"
	"github.com/morehao/ark-iam/platformadmin/testutil"
	"github.com/morehao/golib/biz/gobject"
	"gorm.io/gorm"
)

// seedSortMenu 播种一条启用菜单并返回其主键（子菜单需按该主键挂父级）；播种顺序刻意与 sort 顺序不一致。
func seedSortMenu(t *testing.T, db *gorm.DB, appID, parentID, code, name string, sort int) string {
	t.Helper()
	entity := &model.MenuEntity{
		AppID:      appID,
		ParentID:   parentID,
		Name:       name,
		Code:       code,
		Path:       "/" + code,
		Sort:       sort,
		Type:       model.MenuTypeMenu,
		Visibility: model.MenuVisibilityAdmin,
		Status:     model.MenuStatusEnable,
	}
	if err := db.Create(entity).Error; err != nil {
		t.Fatalf("seed menu %s: %v", code, err)
	}
	return entity.ID
}

// TestMenuTreeFollowsSort 菜单管理页（Tree）的展示顺序必须是「排序」字段的函数：
// 运维在控制台里维护 sort 后，必须能在同一页面立刻看到新顺序，否则无法确认排序是否生效。
// 回归背景：菜单查询缺 ORDER BY，控制台里的排序字段一度完全不生效。
func TestMenuTreeFollowsSort(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.MenuEntity{}, &model.ApplicationEntity{})
	app := &model.ApplicationEntity{Code: "tenant_admin", Name: "租户管理后台", Source: model.AppSourceBuiltin, Status: model.AppStatusEnable}
	if err := db.Create(app).Error; err != nil {
		t.Fatalf("seed application: %v", err)
	}

	// 播种顺序 = 现场截图顺序（API密钥 4 → 部门 1 → 用户 2 → 角色 3）
	seedSortMenu(t, db, app.ID, "", "tenant-api-key", "API密钥", 4)
	seedSortMenu(t, db, app.ID, "", "department", "部门管理", 1)
	seedSortMenu(t, db, app.ID, "", "tenant-user", "用户管理", 2)
	seedSortMenu(t, db, app.ID, "", "tenant-role", "角色管理", 3)
	// 目录子级同样按 sort 升序（播种顺序与 sort 相反）
	groupID := seedSortMenu(t, db, app.ID, "", "grp", "分组目录", 5)
	seedSortMenu(t, db, app.ID, groupID, "child-b", "子菜单B", 2)
	seedSortMenu(t, db, app.ID, groupID, "child-a", "子菜单A", 1)
	// sort 相同：按 code 升序兜底
	seedSortMenu(t, db, app.ID, "", "tie-z", "并列Z", 6)
	seedSortMenu(t, db, app.ID, "", "tie-a", "并列A", 6)

	ctx := newGinCtx("1", "0")
	resp, err := NewMenuSvc().Tree(ctx, &dtopermission.MenuTreeReq{AppID: app.ID})
	if err != nil {
		t.Fatalf("menu tree: %v", err)
	}

	gotRoots := make([]string, 0, len(resp.List))
	for _, node := range resp.List {
		gotRoots = append(gotRoots, node.Code)
	}
	wantRoots := []string{"department", "tenant-user", "tenant-role", "tenant-api-key", "grp", "tie-a", "tie-z"}
	if len(gotRoots) != len(wantRoots) {
		t.Fatalf("菜单树一级节点数不符, got %v, want %v", gotRoots, wantRoots)
	}
	for i := range wantRoots {
		if gotRoots[i] != wantRoots[i] {
			t.Fatalf("菜单管理页必须按 sort 升序（同 sort 按 code），got %v, want %v", gotRoots, wantRoots)
		}
	}

	gotChildren := []string{}
	for _, node := range resp.List {
		if node.Code != "grp" {
			continue
		}
		for _, child := range node.Children {
			gotChildren = append(gotChildren, child.Code)
		}
	}
	if len(gotChildren) != 2 || gotChildren[0] != "child-a" || gotChildren[1] != "child-b" {
		t.Fatalf("子菜单必须按 sort 升序, got %v", gotChildren)
	}
}

// TestMenuPageListFollowsSort 分页列表同样必须显式排序：
// 缺 ORDER BY 时数据库返回顺序不稳定，翻页会出现重复行/漏行，且排序字段对运维不可验证。
func TestMenuPageListFollowsSort(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.MenuEntity{}, &model.ApplicationEntity{})
	app := &model.ApplicationEntity{Code: "tenant_admin", Name: "租户管理后台", Source: model.AppSourceBuiltin, Status: model.AppStatusEnable}
	if err := db.Create(app).Error; err != nil {
		t.Fatalf("seed application: %v", err)
	}
	seedSortMenu(t, db, app.ID, "", "tenant-api-key", "API密钥", 4)
	seedSortMenu(t, db, app.ID, "", "department", "部门管理", 1)
	seedSortMenu(t, db, app.ID, "", "tenant-user", "用户管理", 2)
	seedSortMenu(t, db, app.ID, "", "tenant-role", "角色管理", 3)

	ctx := newGinCtx("1", "0")
	resp, err := NewMenuSvc().PageList(ctx, &dtopermission.MenuPageListReq{
		PageQuery: gobject.PageQuery{Page: 1, PageSize: 2},
		AppID:     app.ID,
	})
	if err != nil {
		t.Fatalf("menu page list: %v", err)
	}
	if resp.Total != 4 || len(resp.List) != 2 {
		t.Fatalf("分页元数据不符, total=%d, len=%d", resp.Total, len(resp.List))
	}
	if resp.List[0].Code != "department" || resp.List[1].Code != "tenant-user" {
		t.Fatalf("第一页必须是最小两个 sort, got %s, %s", resp.List[0].Code, resp.List[1].Code)
	}

	// 第二页接着排，翻页不重复：缺 ORDER BY 时同一行可能出现在两页里
	page2, err := NewMenuSvc().PageList(ctx, &dtopermission.MenuPageListReq{
		PageQuery: gobject.PageQuery{Page: 2, PageSize: 2},
		AppID:     app.ID,
	})
	if err != nil {
		t.Fatalf("menu page list p2: %v", err)
	}
	if len(page2.List) != 2 || page2.List[0].Code != "tenant-role" || page2.List[1].Code != "tenant-api-key" {
		t.Fatalf("第二页顺序不符, got %+v", page2.List)
	}
}
