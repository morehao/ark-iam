package svctenant

import (
	"testing"

	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/tenantadmin/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/tenantadmin/testutil"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

func seedTestApp(t *testing.T, db *gorm.DB, tenantID, appID string) {
	t.Helper()
	if err := db.Create(&model.ApplicationEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: appID}},
		Code:       "app-" + appID,
		Name:       "租户自服务",
		Status:     "enable",
	}).Error; err != nil {
		t.Fatalf("seed app: %v", err)
	}
	if err := db.Create(&model.TenantApplicationEntity{
		BaseEntity:  gormdao.BaseEntity{StringID: gormdao.StringID{ID: "ta-" + appID}},
		TenantID:    tenantID,
		AppID:       appID,
		Status:      "enable",
		Config:      []byte("{}"),
		GrantedScope: []byte("[]"),
	}).Error; err != nil {
		t.Fatalf("seed tenant application: %v", err)
	}
}

func seedTestRole(t *testing.T, db *gorm.DB, id, tenantID, appID, name, code string) {
	t.Helper()
	if err := db.Create(&model.RoleEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: id}},
		TenantID:   tenantID,
		AppID:      appID,
		Name:       name,
		Code:       code,
		Type:       "User",
		CreatedBy:  "t",
	}).Error; err != nil {
		t.Fatalf("seed role: %v", err)
	}
}

func seedTestMenuTree(t *testing.T, db *gorm.DB, tenantID string) (rootID, childID string) {
	t.Helper()
	seedTestApp(t, db, tenantID, "app1")
	root := &model.MenuEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "m1"}},
		AppID:      "app1",
		Name:       "组织架构",
		Code:       "organization",
		Path:       "/organization",
		Status:     "enable",
	}
	child := &model.MenuEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "m2"}},
		AppID:      "app1",
		ParentID:   "m1",
		Name:       "用户管理",
		Code:       "tenant-user",
		Path:       "/user",
		Status:     "enable",
	}
	for _, m := range []*model.MenuEntity{root, child} {
		if err := db.Create(m).Error; err != nil {
			t.Fatalf("seed menu: %v", err)
		}
	}
	return "m1", "m2"
}

// TestRoleCreateRequiresApp 角色从属于租户订阅的应用：非法应用拒绝、应用内编码唯一、跨应用同编码允许。
func TestRoleCreateRequiresApp(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.RoleEntity{}, &model.ApplicationEntity{}, &model.TenantApplicationEntity{})
	svc := &roleSvc{}
	seedTestApp(t, db, "t1", "app1")
	seedTestApp(t, db, "t2", "app2")

	ginCtx := newOrgGinCtx(t, "t1", "op")

	// 非法应用
	if _, err := svc.Create(ginCtx, &dtotenant.RoleCreateReq{AppID: "app-bad", Name: "管理员", Code: "admin"}); err == nil {
		t.Fatalf("expected invalid app error")
	}
	// 创建成功
	if _, err := svc.Create(ginCtx, &dtotenant.RoleCreateReq{AppID: "app1", Name: "管理员", Code: "admin"}); err != nil {
		t.Fatalf("create role: %v", err)
	}
	// 同应用编码唯一
	if _, err := svc.Create(ginCtx, &dtotenant.RoleCreateReq{AppID: "app1", Name: "重复", Code: "admin"}); err == nil {
		t.Fatalf("expected duplicate code error")
	}
	// 其他租户同编码不冲突
	if _, err := svc.Create(newOrgGinCtx(t, "t2", "op2"), &dtotenant.RoleCreateReq{AppID: "app2", Name: "管理员", Code: "admin"}); err != nil {
		t.Fatalf("cross-tenant same code should be allowed: %v", err)
	}
}

// TestRolePageListWithCounts 分页列表带成员数/菜单数聚合。
func TestRolePageListWithCounts(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.RoleEntity{}, &model.UserRoleEntity{}, &model.RoleMenuEntity{},
		&model.OrganizationEntity{}, &model.OrganizationUserEntity{}, &model.ApplicationEntity{}, &model.TenantApplicationEntity{})
	svc := &roleSvc{}
	seedTestApp(t, db, "t1", "app1")
	seedTestRole(t, db, "r1", "t1", "app1", "管理员", "admin")

	if err := db.Create(&model.UserRoleEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "ur1"}},
		TenantID:   "t1",
		UserID:     "u1",
		RoleID:     "r1",
		CreatedBy:  "t",
	}).Error; err != nil {
		t.Fatalf("seed user_role: %v", err)
	}
	if err := db.Create(&model.RoleMenuEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "rm1"}},
		TenantID:   "t1",
		RoleID:     "r1",
		MenuID:     "m1",
		CreatedBy:  "t",
	}).Error; err != nil {
		t.Fatalf("seed role_menu: %v", err)
	}

	resp, err := svc.PageList(newOrgGinCtx(t, "t1", "op"), &dtotenant.RolePageListReq{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("page list: %v", err)
	}
	if resp.Total != 1 {
		t.Fatalf("expected total 1, got %d", resp.Total)
	}
	item := resp.List[0]
	if item.MemberCount != 1 || item.MenuCount != 1 {
		t.Fatalf("unexpected counts: member=%d menu=%d", item.MemberCount, item.MenuCount)
	}
	if item.AppID != "app1" {
		t.Fatalf("expected appID app1, got %s", item.AppID)
	}
}

// TestRoleMenusUpdateAndGet 角色菜单授权：按角色所属应用菜单授权 + 回显 + 非法菜单拒绝。
func TestRoleMenusUpdateAndGet(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.RoleEntity{}, &model.RoleMenuEntity{}, &model.MenuEntity{},
		&model.ApplicationEntity{}, &model.TenantApplicationEntity{})
	svc := &roleSvc{}
	seedTestRole(t, db, "r1", "t1", "app1", "管理员", "admin")
	rootID, childID := seedTestMenuTree(t, db, "t1")

	ginCtx := newOrgGinCtx(t, "t1", "op")

	// 非法菜单（不在角色所属应用菜单集合内）
	if err := svc.UpdateMenus(ginCtx, &dtotenant.RoleMenusUpdateReq{RoleID: "r1", MenuIDs: []string{"m-bad"}}); err == nil {
		t.Fatalf("expected invalid menu error")
	}

	// 全量替换
	if err := svc.UpdateMenus(ginCtx, &dtotenant.RoleMenusUpdateReq{RoleID: "r1", MenuIDs: []string{rootID, childID}}); err != nil {
		t.Fatalf("update menus: %v", err)
	}

	// 回显
	resp, err := svc.GetMenus(ginCtx, &dtotenant.RoleDetailReq{RoleID: "r1"})
	if err != nil {
		t.Fatalf("get menus: %v", err)
	}
	if len(resp.List) != 1 || len(resp.List[0].Children) != 1 {
		t.Fatalf("unexpected tree: %+v", resp.List)
	}
	if len(resp.MenuIDs) != 2 {
		t.Fatalf("expected 2 authorized menu ids, got %+v", resp.MenuIDs)
	}

	// 全量替换为空（撤销全部授权）
	if err := svc.UpdateMenus(ginCtx, &dtotenant.RoleMenusUpdateReq{RoleID: "r1", MenuIDs: []string{}}); err != nil {
		t.Fatalf("clear menus: %v", err)
	}
	resp, err = svc.GetMenus(ginCtx, &dtotenant.RoleDetailReq{RoleID: "r1"})
	if err != nil {
		t.Fatalf("get menus: %v", err)
	}
	if len(resp.MenuIDs) != 0 {
		t.Fatalf("expected cleared menu ids, got %+v", resp.MenuIDs)
	}
}
