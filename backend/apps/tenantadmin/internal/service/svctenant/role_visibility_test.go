package svctenant

import (
	"testing"

	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/iam/object/objpermission"
	"github.com/morehao/ark-iam/tenantadmin/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/tenantadmin/testutil"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

func seedBuiltinRole(t *testing.T, db *gorm.DB, id, tenantID, appID string) {
	t.Helper()
	if err := db.Create(&model.RoleEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: id}},
		TenantID:   tenantID,
		AppID:      appID,
		Name:       "管理员",
		Code:       "admin",
		Source:     string(model.RoleSourceBuiltin),
		AdminLevel: string(model.SysAdminLevelSuper),
		CreatedBy:  "seed",
	}).Error; err != nil {
		t.Fatalf("seed builtin role: %v", err)
	}
}

// TestDeleteRejectsBuiltinRole 内置角色禁止删除。
func TestDeleteRejectsBuiltinRole(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.RoleEntity{}, &model.UserRoleEntity{}, &model.RoleMenuEntity{},
		&model.ApplicationEntity{}, &model.TenantApplicationEntity{})
	svc := &roleSvc{}
	seedTestApp(t, db, "t1", "app1")
	seedBuiltinRole(t, db, "r1", "t1", "app1")

	err := svc.Delete(newOrgGinCtx(t, "t1", "op"), &dtotenant.RoleDeleteReq{RoleID: "r1"})
	if err == nil {
		t.Fatalf("expected builtin delete forbidden error")
	}
	if err != code.GetError(code.RoleDeleteBuiltinForbiddenError) {
		t.Fatalf("expected RoleDeleteBuiltinForbiddenError, got %v", err)
	}
}

// TestUpdateRejectsBuiltinCoreChange 内置角色禁止改核心字段（编码/类别），名称与描述可改。
func TestUpdateRejectsBuiltinCoreChange(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.RoleEntity{}, &model.ApplicationEntity{}, &model.TenantApplicationEntity{})
	svc := &roleSvc{}
	seedTestApp(t, db, "t1", "app1")
	seedBuiltinRole(t, db, "r1", "t1", "app1")

	ginCtx := newOrgGinCtx(t, "t1", "op")

	// 改编码 → 拒绝
	err := svc.Update(ginCtx, &dtotenant.RoleUpdateReq{RoleID: "r1", Name: "管理员", Code: "super-admin"})
	if err == nil || err != code.GetError(code.RoleUpdateBuiltinForbiddenError) {
		t.Fatalf("expected builtin update forbidden, got %v", err)
	}

	// 改名 + 改描述 → 允许（核心字段不变）
	if err := svc.Update(ginCtx, &dtotenant.RoleUpdateReq{RoleID: "r1", Name: "超级管理员", Code: "admin", Description: "系统内置"}); err != nil {
		t.Fatalf("update name/desc should be allowed: %v", err)
	}
	var got model.RoleEntity
	if err := db.First(&got, "id = ?", "r1").Error; err != nil {
		t.Fatalf("load role: %v", err)
	}
	if got.Name != "超级管理员" || got.Code != "admin" {
		t.Fatalf("unexpected role after update: name=%s code=%s", got.Name, got.Code)
	}
}

// TestHasSystemAdminCapability 系统管理能力判定（按角色显式 admin_level 标签）：
// admin 角色 admin_level=super 则具备能力；普通成员角色 admin_level=none 不具备。
func TestHasSystemAdminCapability(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.RoleEntity{}, &model.UserRoleEntity{},
		&model.ApplicationEntity{}, &model.TenantApplicationEntity{})
	seedTestApp(t, db, "t1", "app1")

	// 用户 u1 → 角色 r-admin（super）→ 具备
	if err := db.Create(&model.RoleEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "r-admin"}},
		TenantID:   "t1", AppID: "app1", Name: "管理员", Code: "admin",
		Source: string(model.RoleSourceCustom), AdminLevel: string(model.SysAdminLevelSuper),
		CreatedBy: "t",
	}).Error; err != nil {
		t.Fatalf("seed role: %v", err)
	}
	if err := db.Create(&model.UserRoleEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "ur1"}},
		TenantID:   "t1", UserID: "u1", RoleID: "r-admin", CreatedBy: "t",
	}).Error; err != nil {
		t.Fatalf("seed user_role: %v", err)
	}

	has, err := HasSystemAdminCapability(newOrgGinCtx(t, "t1", "u1"))
	if err != nil {
		t.Fatalf("hasadmin: %v", err)
	}
	if !has {
		t.Fatalf("admin user should have system admin capability")
	}

	// 用户 u2 → 角色 r-user（member）→ 不具备
	if err := db.Create(&model.RoleEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "r-user"}},
		TenantID:   "t1", AppID: "app1", Name: "成员", Code: "user",
		Source: string(model.RoleSourceCustom), AdminLevel: string(model.SysAdminLevelMember),
		CreatedBy: "t",
	}).Error; err != nil {
		t.Fatalf("seed role u2: %v", err)
	}
	if err := db.Create(&model.UserRoleEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "ur2"}},
		TenantID:   "t1", UserID: "u2", RoleID: "r-user", CreatedBy: "t",
	}).Error; err != nil {
		t.Fatalf("seed user_role u2: %v", err)
	}
	has, err = HasSystemAdminCapability(newOrgGinCtx(t, "t1", "u2"))
	if err != nil {
		t.Fatalf("hasadmin u2: %v", err)
	}
	if has {
		t.Fatalf("user with none admin_level role should NOT have system admin capability")
	}
}

func visItem(vis model.MenuVisibility, children ...dtotenant.MenuTreeItem) dtotenant.MenuTreeItem {
	return dtotenant.MenuTreeItem{
		MenuBaseInfo: objpermission.MenuBaseInfo{Visibility: vis},
		Children:     children,
	}
}

// sampleVisibilityTree 三级可见性样例树：public / member / admin。
func sampleVisibilityTree() []dtotenant.MenuTreeItem {
	return []dtotenant.MenuTreeItem{
		visItem(model.MenuVisibilityPublic),
		visItem(model.MenuVisibilityMember, visItem(model.MenuVisibilityAdmin), visItem(model.MenuVisibilityPublic)),
		visItem(model.MenuVisibilityAdmin),
	}
}

// TestPruneMenuTreeAdminSeesAll 管理员可见全部菜单。
func TestPruneMenuTreeAdminSeesAll(t *testing.T) {
	res := pruneMenuTree(sampleVisibilityTree(), model.MenuVisibilityAdmin.VisibilityRank())
	if len(res) != 3 {
		t.Fatalf("admin should see all 3 top-level menus, got %d", len(res))
	}
}

// TestPruneMenuTreeMemberHidesStandaloneAdmin 普通成员：独立 admin 菜单被隐藏；
// member 父菜单下 admin 子项被隐藏但保留 public 子项与该父壳。
func TestPruneMenuTreeMemberHidesStandaloneAdmin(t *testing.T) {
	res := pruneMenuTree(sampleVisibilityTree(), model.MenuVisibilityMember.VisibilityRank())
	if len(res) != 2 {
		t.Fatalf("member should see 2 (public + member父壳), got %d: %+v", len(res), res)
	}
	// 第一个独立 admin 项应被剪掉 → 现在仅剩 public 与 member
	if res[0].Visibility != model.MenuVisibilityPublic || res[1].Visibility != model.MenuVisibilityMember {
		t.Fatalf("unexpected top-level items: %+v", res)
	}
	// member 父壳下应只剩 public 子项（admin 子项被剪）
	if len(res[1].Children) != 1 || res[1].Children[0].Visibility != model.MenuVisibilityPublic {
		t.Fatalf("member parent should keep only public child, got %+v", res[1].Children)
	}
}

// seedBuiltinSystemRole 种子一个「内置系统管理角色」。
func seedBuiltinSystemRole(t *testing.T, db *gorm.DB, id, tenantID, appID string) {
	t.Helper()
	if err := db.Create(&model.RoleEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: id}},
		TenantID:   tenantID,
		AppID:      appID,
		Name:       "管理员",
		Code:       "admin",
		Source:     string(model.RoleSourceBuiltin),
		AdminLevel: string(model.SysAdminLevelSuper),
		CreatedBy:  "seed",
	}).Error; err != nil {
		t.Fatalf("seed builtin system role: %v", err)
	}
}

func seedUserRoleLink(t *testing.T, db *gorm.DB, id, tenantID, userID, roleID string) {
	t.Helper()
	if err := db.Create(&model.UserRoleEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: id}},
		TenantID:   tenantID,
		UserID:     userID,
		RoleID:     roleID,
		CreatedBy:  "t",
	}).Error; err != nil {
		t.Fatalf("seed user_role: %v", err)
	}
}

// TestUpdateRolesRejectRemovingLastAdmin 移除唯一系统管理持有者的管理能力应被拒绝。
func TestUpdateRolesRejectRemovingLastAdmin(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.RoleEntity{}, &model.UserRoleEntity{}, &model.UserEntity{},
		&model.ApplicationEntity{}, &model.TenantApplicationEntity{})
	svc := &userSvc{}
	seedTestApp(t, db, "t1", "app1")
	seedTestUser(t, db, "t1", "u1", "admin")
	seedBuiltinSystemRole(t, db, "r-admin", "t1", "app1")
	seedUserRoleLink(t, db, "ur1", "t1", "u1", "r-admin")

	// u1 是唯一持有内置系统管理角色的人，尝试移除 → 拒绝
	err := svc.UpdateRoles(newOrgGinCtx(t, "t1", "op"), &dtotenant.UserRolesUpdateReq{UserID: "u1", RoleIDs: []string{}})
	if err == nil {
		t.Fatalf("expected last-admin removal forbidden")
	}
	if err != code.GetError(code.UserRoleRemoveLastAdminForbiddenError) {
		t.Fatalf("expected UserRoleRemoveLastAdminForbiddenError, got %v", err)
	}
}

// TestUpdateRolesAllowWhenKeepSystemRole 新列表仍含内置系统管理角色 → 允许。
func TestUpdateRolesAllowWhenKeepSystemRole(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.RoleEntity{}, &model.UserRoleEntity{}, &model.UserEntity{},
		&model.ApplicationEntity{}, &model.TenantApplicationEntity{})
	svc := &userSvc{}
	seedTestApp(t, db, "t1", "app1")
	seedTestUser(t, db, "t1", "u1", "admin")
	seedBuiltinSystemRole(t, db, "r-admin", "t1", "app1")
	seedBuiltinSystemRole(t, db, "r-sys2", "t1", "app1")
	seedUserRoleLink(t, db, "ur1", "t1", "u1", "r-admin")

	// 新列表仍保留一个内置系统管理角色 → 允许（权限不丢失）
	err := svc.UpdateRoles(newOrgGinCtx(t, "t1", "op"), &dtotenant.UserRolesUpdateReq{UserID: "u1", RoleIDs: []string{"r-sys2"}})
	if err != nil {
		t.Fatalf("update to another system role should be allowed: %v", err)
	}
}

// TestUpdateRolesAllowWhenOtherAdminExists 其他用户仍持有内置系统管理角色 → 允许移除目标用户的管理能力。
func TestUpdateRolesAllowWhenOtherAdminExists(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.RoleEntity{}, &model.UserRoleEntity{}, &model.UserEntity{},
		&model.ApplicationEntity{}, &model.TenantApplicationEntity{})
	svc := &userSvc{}
	seedTestApp(t, db, "t1", "app1")
	seedTestUser(t, db, "t1", "u1", "u1")
	seedTestUser(t, db, "t1", "u2", "u2")
	seedBuiltinSystemRole(t, db, "r-admin", "t1", "app1")
	seedUserRoleLink(t, db, "ur1", "t1", "u1", "r-admin")
	seedUserRoleLink(t, db, "ur2", "t1", "u2", "r-admin")

	// u2 仍持有内置系统管理角色 → u1 可释放，不锁死
	err := svc.UpdateRoles(newOrgGinCtx(t, "t1", "op"), &dtotenant.UserRolesUpdateReq{UserID: "u1", RoleIDs: []string{}})
	if err != nil {
		t.Fatalf("release when another admin remains should be allowed: %v", err)
	}
}

// seedAdminVisibilityMenu 在指定应用下种子一个 visibility=admin 的启用菜单。
func seedAdminVisibilityMenu(t *testing.T, db *gorm.DB, id, appID string) {
	t.Helper()
	if err := db.Create(&model.MenuEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: id}},
		AppID:      appID,
		Name:       "菜单管理",
		Code:       "menu",
		Path:       "/menu",
		Type:       model.MenuTypeMenu,
		Visibility: model.MenuVisibilityAdmin,
		Status:     model.MenuStatusEnable,
	}).Error; err != nil {
		t.Fatalf("seed admin menu: %v", err)
	}
}

// TestUpdateMenusRejectsAdminVisibilityForNormalRole 授权约束：普通角色提交 visibility=admin 菜单被拒绝；
// 授权 member 可见性菜单允许。
func TestUpdateMenusRejectsAdminVisibilityForNormalRole(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.RoleEntity{}, &model.RoleMenuEntity{}, &model.MenuEntity{},
		&model.ApplicationEntity{}, &model.TenantApplicationEntity{})
	svc := &roleSvc{}
	seedTestApp(t, db, "t1", "app1")
	seedTestRole(t, db, "r1", "t1", "app1", "成员", "member")
	seedAdminVisibilityMenu(t, db, "m-admin", "app1")

	ginCtx := newOrgGinCtx(t, "t1", "op")

	// 普通角色授权 admin 可见性菜单 → 拒绝
	if err := svc.UpdateMenus(ginCtx, &dtotenant.RoleMenusUpdateReq{RoleID: "r1", MenuIDs: []string{"m-admin"}}); err == nil {
		t.Fatalf("expected admin-visibility rejection for normal role")
	} else if err != code.GetError(code.RoleMenuAdminVisibilityForbiddenError) {
		t.Fatalf("expected RoleMenuAdminVisibilityForbiddenError, got %v", err)
	}
}

// TestUpdateMenusAllowsAdminVisibilityForBuiltinAdmin 豁免：内置管理员（super&&builtin）角色可授权 admin 可见性菜单。
func TestUpdateMenusAllowsAdminVisibilityForBuiltinAdmin(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.RoleEntity{}, &model.RoleMenuEntity{}, &model.MenuEntity{},
		&model.ApplicationEntity{}, &model.TenantApplicationEntity{})
	svc := &roleSvc{}
	seedTestApp(t, db, "t1", "app1")
	seedBuiltinSystemRole(t, db, "r-admin", "t1", "app1")
	seedAdminVisibilityMenu(t, db, "m-admin", "app1")

	ginCtx := newOrgGinCtx(t, "t1", "op")
	if err := svc.UpdateMenus(ginCtx, &dtotenant.RoleMenusUpdateReq{RoleID: "r-admin", MenuIDs: []string{"m-admin"}}); err != nil {
		t.Fatalf("builtin admin should be allowed to grant admin menu: %v", err)
	}
}

// TestRoleMenuTreeHidesAdminForNormalRole 普通角色的可授权菜单树剔除 visibility=admin 节点；内置管理员可见全部。
func TestRoleMenuTreeHidesAdminForNormalRole(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.RoleEntity{}, &model.MenuEntity{}, &model.ApplicationEntity{},
		&model.TenantApplicationEntity{})
	svc := &roleSvc{}
	seedTestApp(t, db, "t1", "app1")
	seedAdminVisibilityMenu(t, db, "m-admin", "app1")

	ginCtx := newOrgGinCtx(t, "t1", "op")

	// 普通角色
	seedTestRole(t, db, "r1", "t1", "app1", "成员", "member")
	tree, err := svc.roleMenuTree(ginCtx, &model.RoleEntity{AppID: "app1"})
	if err != nil {
		t.Fatalf("roleMenuTree normal: %v", err)
	}
	if len(tree) != 0 || containAdminVisibilityMenus(tree) {
		t.Fatalf("normal role should not see admin menu, got %+v", tree)
	}

	// 内置管理员
	tree, err = svc.roleMenuTree(ginCtx, &model.RoleEntity{
		AppID:      "app1",
		Source:     string(model.RoleSourceBuiltin),
		AdminLevel: string(model.SysAdminLevelSuper),
	})
	if err != nil {
		t.Fatalf("roleMenuTree admin: %v", err)
	}
	if len(tree) != 1 || tree[0].MenuID != "m-admin" {
		t.Fatalf("builtin admin should see admin menu, got %+v", tree)
	}
}

// TestPruneMenuTreeByAuthed 按授权集合 + 可见等级剪枝：
// 只保留已授权且可见等级达标的节点，父未命中但子命中时保留父壳。
func TestPruneMenuTreeByAuthed(t *testing.T) {
	item := func(id string, vis model.MenuVisibility, children ...dtotenant.MenuTreeItem) dtotenant.MenuTreeItem {
		return dtotenant.MenuTreeItem{
			MenuID:       id,
			MenuBaseInfo: objpermission.MenuBaseInfo{Visibility: vis},
			Children:     children,
		}
	}
	tree := []dtotenant.MenuTreeItem{
		item("p1", model.MenuVisibilityPublic),
		item("p2", model.MenuVisibilityMember,
			item("c1", model.MenuVisibilityPublic),
			item("c2", model.MenuVisibilityAdmin)),
	}
	// 授权 p1 与 p2；p2 下仅 c1 授权
	authed := map[string]bool{"p1": true, "p2": true, "c1": true}
	res := pruneMenuTreeByAuthed(tree, authed, model.MenuVisibilityMember.VisibilityRank())
	if len(res) != 2 {
		t.Fatalf("expected p1 and p2(parent shell), got %+v", res)
	}
	if res[0].MenuID != "p1" {
		t.Fatalf("expected p1 first, got %+v", res)
	}
	// p2 未授权但 c1 已授权 → 保留 p2 父壳，仅含 c1（c2 admin 且未授权被剪）
	if res[1].MenuID != "p2" || len(res[1].Children) != 1 || res[1].Children[0].MenuID != "c1" {
		t.Fatalf("expected p2 shell with only c1 child, got %+v", res[1])
	}
}

// TestUserHoldsBuiltinAdmin 判定用户是否持有内置管理员角色：
// 持有 super&&builtin → true；仅持普通/自定义角色 → false。
func TestUserHoldsBuiltinAdmin(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.RoleEntity{}, &model.UserRoleEntity{},
		&model.ApplicationEntity{}, &model.TenantApplicationEntity{})
	seedTestApp(t, db, "t1", "app1")
	seedBuiltinSystemRole(t, db, "r-admin", "t1", "app1")
	seedUserRoleLink(t, db, "ur-admin", "t1", "u-admin", "r-admin")
	seedTestRole(t, db, "r-member", "t1", "app1", "成员", "member")
	seedUserRoleLink(t, db, "ur-member", "t1", "u-member", "r-member")

	admin, err := userHoldsBuiltinAdmin(newOrgGinCtx(t, "t1", "u-admin"))
	if err != nil {
		t.Fatalf("holds admin: %v", err)
	}
	if !admin {
		t.Fatalf("user with builtin super role should hold builtin admin")
	}

	mem, err := userHoldsBuiltinAdmin(newOrgGinCtx(t, "t1", "u-member"))
	if err != nil {
		t.Fatalf("holds member: %v", err)
	}
	if mem {
		t.Fatalf("member user should NOT hold builtin admin")
	}
}
