package menu

import (
	"fmt"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newMenuTestDB 内存 sqlite 注册为全局 iam 库：BuildAppMenuTree 内部直接 dao.NewMenuDao()，
// 因此必须走全局注册（与 apps/*/testutil.SetupSQLite 同做法）。
func newMenuTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:iam_menu_order_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.ApplicationEntity{}, &model.MenuEntity{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	dbclient.RegisterDBForTest(dbclient.ServiceNameIam, db)
	t.Cleanup(func() {
		dbclient.ClearDBForTest(dbclient.ServiceNameIam)
		if sqlDB, sqlErr := db.DB(); sqlErr == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func seedOrderTestApp(t *testing.T, db *gorm.DB) *model.ApplicationEntity {
	t.Helper()
	app := &model.ApplicationEntity{
		Code:   "order_app",
		Name:   "排序测试应用",
		Source: model.AppSourceBuiltin,
		Status: model.AppStatusEnable,
	}
	if err := db.Create(app).Error; err != nil {
		t.Fatalf("seed application: %v", err)
	}
	return app
}

// seedMenu 播种一条启用菜单：id 显式指定，便于断言顺序；播种顺序刻意与 sort 顺序不一致。
func seedMenu(t *testing.T, db *gorm.DB, id, appID, parentID, code, name string, sort int) {
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
	entity.ID = id
	if err := db.Create(entity).Error; err != nil {
		t.Fatalf("seed menu %s: %v", code, err)
	}
}

// TestBuildAppMenuTreeFollowsSort 侧边栏/授权树的菜单顺序必须是「排序」字段的函数：
// 查询本身必须显式排序（dao.MenuOrderBySort），否则数据库返回的是不保证的自然顺序——
// 回归背景：控制台「菜单管理」里的排序字段一度完全不生效，侧边栏顺序与 sort 无关（用户可见 bug）。
//
// 播种顺序刻意复刻问题现场：先插 sort=4 的 API密钥，再插 sort=1/2/3，且 sort 相同的场景用 code 兜底。
func TestBuildAppMenuTreeFollowsSort(t *testing.T) {
	db := newMenuTestDB(t)
	app := seedOrderTestApp(t, db)

	// 一级菜单：播种顺序 = api-key(4) → department(1) → user(2) → role(3)
	seedMenu(t, db, "m-api-key", app.ID, "", "tenant-api-key", "API密钥", 4)
	seedMenu(t, db, "m-department", app.ID, "", "department", "部门管理", 1)
	seedMenu(t, db, "m-user", app.ID, "", "tenant-user", "用户管理", 2)
	seedMenu(t, db, "m-role", app.ID, "", "tenant-role", "角色管理", 3)
	// 目录 + 子菜单：子级同样按 sort 排序（播种顺序与 sort 相反）
	seedMenu(t, db, "m-group", app.ID, "", "grp", "分组目录", 5)
	seedMenu(t, db, "m-group-b", app.ID, "m-group", "child-b", "子菜单B", 2)
	seedMenu(t, db, "m-group-a", app.ID, "m-group", "child-a", "子菜单A", 1)
	// sort 相同：按 code 升序兜底（同应用内 code 唯一，顺序确定）
	seedMenu(t, db, "m-tie-z", app.ID, "", "tie-z", "并列Z", 6)
	seedMenu(t, db, "m-tie-a", app.ID, "", "tie-a", "并列A", 6)

	ginCtx, _ := gin.CreateTestContext(nil)
	tree, err := BuildAppMenuTree(ginCtx, app.ID)
	if err != nil {
		t.Fatalf("BuildAppMenuTree: %v", err)
	}

	gotRoots := make([]string, 0, len(tree))
	for _, node := range tree {
		gotRoots = append(gotRoots, node.Code)
	}
	wantRoots := []string{"department", "tenant-user", "tenant-role", "tenant-api-key", "grp", "tie-a", "tie-z"}
	if fmt.Sprint(gotRoots) != fmt.Sprint(wantRoots) {
		t.Fatalf("一级菜单必须按 sort 升序（同 sort 按 code），got %v, want %v", gotRoots, wantRoots)
	}

	gotChildren := []string{}
	for _, node := range tree {
		if node.Code != "grp" {
			continue
		}
		for _, child := range node.Children {
			gotChildren = append(gotChildren, child.Code)
		}
	}
	if len(gotChildren) == 0 {
		t.Fatalf("目录 grp 必须出现在菜单树中: %+v", gotRoots)
	}
	if fmt.Sprint(gotChildren) != fmt.Sprint([]string{"child-a", "child-b"}) {
		t.Fatalf("子菜单必须按 sort 升序，got %v", gotChildren)
	}
}

// TestBuildAppMenuTreeOnlyEnableMenus 停用菜单不进树（顺序修复不得改变既有可见性语义）。
func TestBuildAppMenuTreeOnlyEnableMenus(t *testing.T) {
	db := newMenuTestDB(t)
	app := seedOrderTestApp(t, db)
	seedMenu(t, db, "m-on", app.ID, "", "on", "启用菜单", 1)
	seedMenu(t, db, "m-off", app.ID, "", "off", "停用菜单", 2)
	if err := db.Model(&model.MenuEntity{}).Where("id = ?", "m-off").
		Update("status", model.MenuStatusDisable).Error; err != nil {
		t.Fatalf("disable menu: %v", err)
	}

	ginCtx, _ := gin.CreateTestContext(nil)
	tree, err := BuildAppMenuTree(ginCtx, app.ID)
	if err != nil {
		t.Fatalf("BuildAppMenuTree: %v", err)
	}
	if len(tree) != 1 || tree[0].Code != "on" {
		t.Fatalf("停用菜单不得进树, got %+v", tree)
	}
}
