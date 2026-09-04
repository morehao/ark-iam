package svctenant

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/iam/object/objtenant"
	"github.com/morehao/ark-iam/tenantadmin/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/tenantadmin/testutil"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

func newOrgGinCtx(t *testing.T, tenantID, userID string) *gin.Context {
	t.Helper()
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, tenantID)
	ginCtx.Set(gcontext.KeyUserID, userID)
	return ginCtx
}

func seedTestUser(t *testing.T, db *gorm.DB, tenantID, userID, name string) {
	t.Helper()
	now := time.Now()
	if err := db.Create(&model.UserEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: userID}},
		TenantID:   tenantID,
		Name:       name,
		Profile:    json.RawMessage("{}"),
		CustomData: json.RawMessage("{}"),
		JoinedAt:   &now,
	}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
}

// seedTenantSuperOperator 为指定租户的操作用户绑定内置超管角色（user_role/role），
// 使管理写接口的能力校验（requireSystemAdmin）通过；无需在 user 表播种操作者。
// 注意：调用方 SetupSQLite 需同时注册 &model.RoleEntity{} 与 &model.UserRoleEntity{}。
func seedTenantSuperOperator(t *testing.T, db *gorm.DB, tenantID, userID string) {
	t.Helper()
	roleID := "test-super-" + tenantID + "-" + userID
	seedBuiltinSystemRole(t, db, roleID, tenantID, "app-admin")
	seedUserRoleLink(t, db, "test-ur-"+roleID, tenantID, userID, roleID)
}

// seedTenantCustomSuperOperator 为操作用户绑定「自定义来源」的超管角色：
// 具备系统管理能力(admin_level=super)但不属于内置系统角色，避免被「最后一个内置管理员」
// 保护（仅统计内置系统角色持有者）误判为其他持有者。
func seedTenantCustomSuperOperator(t *testing.T, db *gorm.DB, tenantID, userID string) {
	t.Helper()
	roleID := "test-custom-super-" + tenantID + "-" + userID
	if err := db.Create(&model.RoleEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: roleID}},
		TenantID:   tenantID,
		AppID:      "app-admin",
		Name:       "测试自定义超管",
		Code:       "test-custom-super",
		Source:     string(model.RoleSourceCustom),
		AdminLevel: string(model.SysAdminLevelSuper),
		CreatedBy:  "t",
	}).Error; err != nil {
		t.Fatalf("seed custom super role: %v", err)
	}
	seedUserRoleLink(t, db, "test-cur-"+roleID, tenantID, userID, roleID)
}

// newSuperCtx 创建以「内置超管」身份执行管理写操作的 gin 上下文（自动绑定超管角色）。
// 前置条件：SetupSQLite 已注册 RoleEntity 与 UserRoleEntity 表。
func newSuperCtx(t *testing.T, db *gorm.DB, tenantID, userID string) *gin.Context {
	t.Helper()
	seedTenantSuperOperator(t, db, tenantID, userID)
	return newOrgGinCtx(t, tenantID, userID)
}

// newCustomSuperCtx 同 newSuperCtx，但绑定「自定义来源」的超管角色（见 seedTenantCustomSuperOperator）。
func newCustomSuperCtx(t *testing.T, db *gorm.DB, tenantID, userID string) *gin.Context {
	t.Helper()
	seedTenantCustomSuperOperator(t, db, tenantID, userID)
	return newOrgGinCtx(t, tenantID, userID)
}

func TestOrganizationCreateRootAndChildPaths(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.OrganizationEntity{}, &model.OrganizationUserEntity{}, &model.RoleEntity{}, &model.UserRoleEntity{})
	ginCtx := newOrgGinCtx(t, "41", "1001")
	seedTenantSuperOperator(t, db, "41", "1001")

	svc := &organizationSvc{}
	root, err := svc.Create(ginCtx, &dtotenant.OrganizationCreateReq{
		OrganizationBaseInfo: objtenant.OrganizationBaseInfo{Name: "总公司", Status: "active"},
	})
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	var rootEntity model.OrganizationEntity
	if err := db.First(&rootEntity, "id = ?", root.OrganizationID).Error; err != nil {
		t.Fatalf("query root: %v", err)
	}
	if rootEntity.OrgPath != "/"+root.OrganizationID || rootEntity.OrgDepth != 1 {
		t.Fatalf("unexpected root path: %s depth=%d", rootEntity.OrgPath, rootEntity.OrgDepth)
	}

	child, err := svc.Create(ginCtx, &dtotenant.OrganizationCreateReq{
		ParentID:             root.OrganizationID,
		OrganizationBaseInfo: objtenant.OrganizationBaseInfo{Name: "研发部"},
	})
	if err != nil {
		t.Fatalf("create child: %v", err)
	}
	var childEntity model.OrganizationEntity
	if err := db.First(&childEntity, "id = ?", child.OrganizationID).Error; err != nil {
		t.Fatalf("query child: %v", err)
	}
	if childEntity.OrgPath != "/"+root.OrganizationID+"/"+child.OrganizationID || childEntity.OrgDepth != 2 {
		t.Fatalf("unexpected child path: %s depth=%d", childEntity.OrgPath, childEntity.OrgDepth)
	}

	// 根节点唯一：再创建根应失败
	if _, err := svc.Create(ginCtx, &dtotenant.OrganizationCreateReq{
		OrganizationBaseInfo: objtenant.OrganizationBaseInfo{Name: "重复根"},
	}); err == nil {
		t.Fatalf("expected duplicate root create to fail")
	}
}

func TestOrganizationMoveCascadesPathAndRejectsCycle(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.OrganizationEntity{}, &model.OrganizationUserEntity{}, &model.RoleEntity{}, &model.UserRoleEntity{})
	ginCtx := newOrgGinCtx(t, "41", "1001")
	seedTenantSuperOperator(t, db, "41", "1001")

	svc := &organizationSvc{}
	root, _ := svc.Create(ginCtx, &dtotenant.OrganizationCreateReq{
		OrganizationBaseInfo: objtenant.OrganizationBaseInfo{Name: "A"},
	})
	b, _ := svc.Create(ginCtx, &dtotenant.OrganizationCreateReq{
		ParentID:             root.OrganizationID,
		OrganizationBaseInfo: objtenant.OrganizationBaseInfo{Name: "B"},
	})
	c, _ := svc.Create(ginCtx, &dtotenant.OrganizationCreateReq{
		ParentID:             b.OrganizationID,
		OrganizationBaseInfo: objtenant.OrganizationBaseInfo{Name: "C"},
	})

	// 移动 C 到 A 下：C 的 path 应从 A/B/C 变为 A/C
	if err := svc.Update(ginCtx, &dtotenant.OrganizationUpdateReq{
		OrganizationID: c.OrganizationID,
		ParentID:       root.OrganizationID,
		OrganizationBaseInfo: objtenant.OrganizationBaseInfo{
			Name:   "C",
			Status: "active",
		},
	}); err != nil {
		t.Fatalf("move C under A: %v", err)
	}
	var cEntity model.OrganizationEntity
	if err := db.First(&cEntity, "id = ?", c.OrganizationID).Error; err != nil {
		t.Fatalf("query C: %v", err)
	}
	if cEntity.OrgPath != "/"+root.OrganizationID+"/"+c.OrganizationID || cEntity.OrgDepth != 2 {
		t.Fatalf("unexpected C path after move: %s depth=%d", cEntity.OrgPath, cEntity.OrgDepth)
	}

	// 环路：把 A 移到 C 下（C 是 A 的子孙）应拒绝
	err := svc.Update(ginCtx, &dtotenant.OrganizationUpdateReq{
		OrganizationID: root.OrganizationID,
		ParentID:       c.OrganizationID,
		OrganizationBaseInfo: objtenant.OrganizationBaseInfo{
			Name:   "A",
			Status: "active",
		},
	})
	if err == nil {
		t.Fatalf("expected cycle move to fail")
	}
	// 环路移动失败后 A 的路径不应被破坏
	var aEntity model.OrganizationEntity
	if err := db.First(&aEntity, "id = ?", root.OrganizationID).Error; err != nil {
		t.Fatalf("query A: %v", err)
	}
	if aEntity.OrgPath != "/"+root.OrganizationID {
		t.Fatalf("A path corrupted after rejected move: %s", aEntity.OrgPath)
	}
}

func TestOrganizationDeleteRejectsWithChildrenAndCascade(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.OrganizationEntity{}, &model.OrganizationUserEntity{}, &model.RoleEntity{}, &model.UserRoleEntity{})
	ginCtx := newOrgGinCtx(t, "41", "1001")
	seedTenantSuperOperator(t, db, "41", "1001")

	svc := &organizationSvc{}
	root, _ := svc.Create(ginCtx, &dtotenant.OrganizationCreateReq{
		OrganizationBaseInfo: objtenant.OrganizationBaseInfo{Name: "A"},
	})
	child, _ := svc.Create(ginCtx, &dtotenant.OrganizationCreateReq{
		ParentID:             root.OrganizationID,
		OrganizationBaseInfo: objtenant.OrganizationBaseInfo{Name: "B"},
	})

	// 有子节点默认拒绝
	if err := svc.Delete(ginCtx, &dtotenant.OrganizationDeleteReq{OrganizationID: root.OrganizationID}); err == nil {
		t.Fatalf("expected delete with children to fail")
	}
	// cascade 删除成功且子树软删
	if err := svc.Delete(ginCtx, &dtotenant.OrganizationDeleteReq{OrganizationID: root.OrganizationID, Cascade: true}); err != nil {
		t.Fatalf("cascade delete: %v", err)
	}
	var count int64
	if err := db.Model(&model.OrganizationEntity{}).Unscoped().Count(&count).Error; err != nil {
		t.Fatalf("count orgs: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 soft-deleted rows, got %d", count)
	}
	var childEntity model.OrganizationEntity
	if err := db.First(&childEntity, "id = ?", child.OrganizationID).Error; err == nil {
		t.Fatalf("expected child soft-deleted")
	}
}

func TestOrganizationUserMemberSingletonAndValidTypes(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.OrganizationEntity{}, &model.OrganizationUserEntity{}, &model.UserEntity{}, &model.RoleEntity{}, &model.UserRoleEntity{})
	ginCtx := newOrgGinCtx(t, "41", "1001")
	seedTenantSuperOperator(t, db, "41", "1001")
	seedTestUser(t, db, "41", "u1", "用户一")
	seedTestUser(t, db, "41", "u2", "用户二")

	orgSvc := &organizationSvc{}
	root, _ := orgSvc.Create(ginCtx, &dtotenant.OrganizationCreateReq{
		OrganizationBaseInfo: objtenant.OrganizationBaseInfo{Name: "A"},
	})
	other, _ := orgSvc.Create(ginCtx, &dtotenant.OrganizationCreateReq{
		ParentID:             root.OrganizationID,
		OrganizationBaseInfo: objtenant.OrganizationBaseInfo{Name: "A1"},
	})

	svc := &organizationUserSvc{}
	// u1 行政归属 A
	if _, err := svc.Create(ginCtx, &dtotenant.OrganizationUserCreateReq{
		OrganizationID: root.OrganizationID,
		UserID:         "u1",
		RelationType:   model.OrgUserRelationPrimary,
	}); err != nil {
		t.Fatalf("add member u1: %v", err)
	}
	// u1 再行政主部门 A1 → 应覆盖为 A1（primary 每用户至多 1 行）
	if _, err := svc.Create(ginCtx, &dtotenant.OrganizationUserCreateReq{
		OrganizationID: other.OrganizationID,
		UserID:         "u1",
		RelationType:   model.OrgUserRelationPrimary,
	}); err != nil {
		t.Fatalf("reassign primary u1 to A1: %v", err)
	}
	var primaryCount int64
	if err := db.Model(&model.OrganizationUserEntity{}).
		Where("user_id = ? AND relation_type = ?", "u1", model.OrgUserRelationPrimary).
		Count(&primaryCount).Error; err != nil {
		t.Fatalf("count primary: %v", err)
	}
	if primaryCount != 1 {
		t.Fatalf("expected exactly 1 primary relation, got %d", primaryCount)
	}

	// 非法关系类型拒绝
	if _, err := svc.Create(ginCtx, &dtotenant.OrganizationUserCreateReq{
		OrganizationID: root.OrganizationID,
		UserID:         "u2",
		RelationType:   "admin",
	}); err == nil {
		t.Fatalf("expected invalid relation type to fail")
	}
	// secondary 正常建立（u2 参与 A）
	if _, err := svc.Create(ginCtx, &dtotenant.OrganizationUserCreateReq{
		OrganizationID: root.OrganizationID,
		UserID:         "u2",
		RelationType:   model.OrgUserRelationSecondary,
	}); err != nil {
		t.Fatalf("add secondary u2: %v", err)
	}
	// leader 不要求同时是成员：u2 负责 A1
	if _, err := svc.Create(ginCtx, &dtotenant.OrganizationUserCreateReq{
		OrganizationID: other.OrganizationID,
		UserID:         "u2",
		RelationType:   model.OrgUserRelationLeader,
	}); err != nil {
		t.Fatalf("add leader u2: %v", err)
	}
	// 一个部门至多一个负责人：u2 已是 A1 的负责人，u1 再设 A1 负责人应冲突拒绝
	if _, err := svc.Create(ginCtx, &dtotenant.OrganizationUserCreateReq{
		OrganizationID: other.OrganizationID,
		UserID:         "u1",
		RelationType:   model.OrgUserRelationLeader,
	}); err == nil {
		t.Fatalf("expected leader conflict to be rejected")
	}
	// 同一用户重复设为同一部门负责人：幂等成功
	if _, err := svc.Create(ginCtx, &dtotenant.OrganizationUserCreateReq{
		OrganizationID: other.OrganizationID,
		UserID:         "u2",
		RelationType:   model.OrgUserRelationLeader,
	}); err != nil {
		t.Fatalf("re-set same leader should succeed: %v", err)
	}
}

func TestOrganizationUserCrossTenantRejected(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.OrganizationEntity{}, &model.OrganizationUserEntity{}, &model.UserEntity{}, &model.RoleEntity{}, &model.UserRoleEntity{})
	ginCtx := newOrgGinCtx(t, "41", "1001")
	seedTenantSuperOperator(t, db, "41", "1001")
	seedTestUser(t, db, "41", "u1", "用户一")

	orgSvc := &organizationSvc{}
	root, _ := orgSvc.Create(ginCtx, &dtotenant.OrganizationCreateReq{
		OrganizationBaseInfo: objtenant.OrganizationBaseInfo{Name: "A"},
	})

	// 切换到其他租户：组织不可见 → 添加关系失败
	otherCtx := newOrgGinCtx(t, "99", "2001")
	svc := &organizationUserSvc{}
	if _, err := svc.Create(otherCtx, &dtotenant.OrganizationUserCreateReq{
		OrganizationID: root.OrganizationID,
		UserID:         "u1",
		RelationType:   model.OrgUserRelationPrimary,
	}); err == nil {
		t.Fatalf("expected cross-tenant relation create to fail")
	}
}

func TestOrganizationChildrenPageAndHasChildren(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.OrganizationEntity{}, &model.OrganizationUserEntity{}, &model.RoleEntity{}, &model.UserRoleEntity{})
	ginCtx := newOrgGinCtx(t, "41", "1001")
	seedTenantSuperOperator(t, db, "41", "1001")

	svc := &organizationSvc{}
	root, err := svc.Create(ginCtx, &dtotenant.OrganizationCreateReq{
		OrganizationBaseInfo: objtenant.OrganizationBaseInfo{Name: "总公司", Status: "active"},
	})
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	a, _ := svc.Create(ginCtx, &dtotenant.OrganizationCreateReq{
		ParentID:             root.OrganizationID,
		OrganizationBaseInfo: objtenant.OrganizationBaseInfo{Name: "A", Status: "active"},
	})
	svc.Create(ginCtx, &dtotenant.OrganizationCreateReq{
		ParentID:             root.OrganizationID,
		OrganizationBaseInfo: objtenant.OrganizationBaseInfo{Name: "B", Status: "inactive"},
	})
	// A 下挂一个深层子级，验证 hasChildren
	svc.Create(ginCtx, &dtotenant.OrganizationCreateReq{
		ParentID:             a.OrganizationID,
		OrganizationBaseInfo: objtenant.OrganizationBaseInfo{Name: "A1", Status: "active"},
	})

	// 直属子级：应只有 A、B 两项
	resp, err := svc.Children(ginCtx, &dtotenant.OrganizationChildrenReq{
		OrganizationID: root.OrganizationID,
		Page:           1,
		PageSize:       10,
	})
	if err != nil {
		t.Fatalf("children: %v", err)
	}
	if resp.Total != 2 || len(resp.List) != 2 {
		t.Fatalf("expected 2 children, got total=%d len=%d", resp.Total, len(resp.List))
	}
	// A 有下级，B 无下级
	has := map[string]bool{}
	for _, item := range resp.List {
		has[item.Name] = item.HasChildren
	}
	if !has["A"] {
		t.Fatalf("expected A to have children, got %+v", has)
	}
	if has["B"] {
		t.Fatalf("expected B to have no children, got %+v", has)
	}

	// 状态筛选：只返回启用的 A
	resp, err = svc.Children(ginCtx, &dtotenant.OrganizationChildrenReq{
		OrganizationID: root.OrganizationID,
		Status:         "active",
		Page:           1,
		PageSize:       10,
	})
	if err != nil {
		t.Fatalf("children status filter: %v", err)
	}
	if resp.Total != 1 || resp.List[0].Name != "A" {
		t.Fatalf("expected only A, got total=%d names=%+v", resp.Total, resp.List)
	}

	// 分页：pageSize=1 时 total 仍为 2
	resp, err = svc.Children(ginCtx, &dtotenant.OrganizationChildrenReq{
		OrganizationID: root.OrganizationID,
		Page:           1,
		PageSize:       1,
	})
	if err != nil {
		t.Fatalf("children paging: %v", err)
	}
	if resp.Total != 2 || len(resp.List) != 1 {
		t.Fatalf("expected total=2 len=1, got total=%d len=%d", resp.Total, len(resp.List))
	}

	// 跨租户：另一租户查询该部门子级应失败
	otherCtx := newOrgGinCtx(t, "42", "1002")
	if _, err := svc.Children(otherCtx, &dtotenant.OrganizationChildrenReq{
		OrganizationID: root.OrganizationID,
		Page:           1,
		PageSize:       10,
	}); err == nil {
		t.Fatalf("expected cross-tenant children to fail")
	}
}
