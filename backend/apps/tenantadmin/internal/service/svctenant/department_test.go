package svctenant

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/object/objtenant"
	"github.com/morehao/ark-iam/tenantadmin/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/tenantadmin/testutil"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

func newDeptGinCtx(t *testing.T, tenantID, userID string) *gin.Context {
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

// seedTenantAdminOperator 为指定租户的操作用户绑定内置管理员角色（user_role/role），
// 使管理写接口的能力校验（requireSystemAdmin）通过；无需在 user 表播种操作者。
// 注意：调用方 SetupSQLite 需同时注册 &model.RoleEntity{} 与 &model.UserRoleEntity{}。
func seedTenantAdminOperator(t *testing.T, db *gorm.DB, tenantID, userID string) {
	t.Helper()
	roleID := "test-super-" + tenantID + "-" + userID
	seedBuiltinSystemRole(t, db, roleID, tenantID, "app-admin")
	seedUserRoleLink(t, db, "test-ur-"+roleID, tenantID, userID, roleID)
}

// seedTenantCustomAdminOperator 为操作用户绑定「自定义来源」的管理员角色：
// 具备系统管理能力(admin_type=admin)但不属于内置系统角色，避免被「最后一个内置管理员」
// 保护（仅统计内置系统角色持有者）误判为其他持有者。
func seedTenantCustomAdminOperator(t *testing.T, db *gorm.DB, tenantID, userID string) {
	t.Helper()
	roleID := "test-custom-super-" + tenantID + "-" + userID
	if err := db.Create(&model.RoleEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: roleID}},
		TenantID:   tenantID,
		AppID:      "app-admin",
		Name:       "测试自定义管理员角色",
		Source:     model.RoleSourceCustom,
		AdminType:  model.SysAdminTypeAdmin,
		CreatedBy:  "t",
	}).Error; err != nil {
		t.Fatalf("seed custom super role: %v", err)
	}
	seedUserRoleLink(t, db, "test-cur-"+roleID, tenantID, userID, roleID)
}

// newAdminCtx 创建以「内置管理员角色」身份执行管理写操作的 gin 上下文（自动绑定管理员角色）。
// 前置条件：SetupSQLite 已注册 RoleEntity 与 UserRoleEntity 表。
func newAdminCtx(t *testing.T, db *gorm.DB, tenantID, userID string) *gin.Context {
	t.Helper()
	seedTenantAdminOperator(t, db, tenantID, userID)
	return newDeptGinCtx(t, tenantID, userID)
}

// newCustomAdminCtx 同 newAdminCtx，但绑定「自定义来源」的管理员角色（见 seedTenantCustomAdminOperator）。
func newCustomAdminCtx(t *testing.T, db *gorm.DB, tenantID, userID string) *gin.Context {
	t.Helper()
	seedTenantCustomAdminOperator(t, db, tenantID, userID)
	return newDeptGinCtx(t, tenantID, userID)
}

func TestDepartmentCreateRootAndChildPaths(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.DepartmentEntity{}, &model.DepartmentUserEntity{}, &model.RoleEntity{}, &model.UserRoleEntity{})
	ginCtx := newDeptGinCtx(t, "41", "1001")
	seedTenantAdminOperator(t, db, "41", "1001")

	svc := &departmentSvc{}
	root, err := svc.Create(ginCtx, &dtotenant.DepartmentCreateReq{
		DepartmentBaseInfo: objtenant.DepartmentBaseInfo{Name: "总公司", Status: model.DeptNodeStatusEnable},
	})
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	var rootEntity model.DepartmentEntity
	if err := db.First(&rootEntity, "id = ?", root.DepartmentID).Error; err != nil {
		t.Fatalf("query root: %v", err)
	}
	if rootEntity.DeptPath != "/"+root.DepartmentID || rootEntity.DeptDepth != 1 {
		t.Fatalf("unexpected root path: %s depth=%d", rootEntity.DeptPath, rootEntity.DeptDepth)
	}

	child, err := svc.Create(ginCtx, &dtotenant.DepartmentCreateReq{
		ParentID:           root.DepartmentID,
		DepartmentBaseInfo: objtenant.DepartmentBaseInfo{Name: "研发部"},
	})
	if err != nil {
		t.Fatalf("create child: %v", err)
	}
	var childEntity model.DepartmentEntity
	if err := db.First(&childEntity, "id = ?", child.DepartmentID).Error; err != nil {
		t.Fatalf("query child: %v", err)
	}
	if childEntity.DeptPath != "/"+root.DepartmentID+"/"+child.DepartmentID || childEntity.DeptDepth != 2 {
		t.Fatalf("unexpected child path: %s depth=%d", childEntity.DeptPath, childEntity.DeptDepth)
	}

	// 根节点唯一：再创建根应失败
	if _, err := svc.Create(ginCtx, &dtotenant.DepartmentCreateReq{
		DepartmentBaseInfo: objtenant.DepartmentBaseInfo{Name: "重复根"},
	}); err == nil {
		t.Fatalf("expected duplicate root create to fail")
	}
}

func TestDepartmentMoveCascadesPathAndRejectsCycle(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.DepartmentEntity{}, &model.DepartmentUserEntity{}, &model.RoleEntity{}, &model.UserRoleEntity{})
	ginCtx := newDeptGinCtx(t, "41", "1001")
	seedTenantAdminOperator(t, db, "41", "1001")

	svc := &departmentSvc{}
	root, _ := svc.Create(ginCtx, &dtotenant.DepartmentCreateReq{
		DepartmentBaseInfo: objtenant.DepartmentBaseInfo{Name: "A"},
	})
	b, _ := svc.Create(ginCtx, &dtotenant.DepartmentCreateReq{
		ParentID:           root.DepartmentID,
		DepartmentBaseInfo: objtenant.DepartmentBaseInfo{Name: "B"},
	})
	c, _ := svc.Create(ginCtx, &dtotenant.DepartmentCreateReq{
		ParentID:           b.DepartmentID,
		DepartmentBaseInfo: objtenant.DepartmentBaseInfo{Name: "C"},
	})

	// 移动 C 到 A 下：C 的 path 应从 A/B/C 变为 A/C
	if err := svc.Update(ginCtx, &dtotenant.DepartmentUpdateReq{
		DepartmentID: c.DepartmentID,
		ParentID:     root.DepartmentID,
		DepartmentBaseInfo: objtenant.DepartmentBaseInfo{
			Name:   "C",
			Status: model.DeptNodeStatusEnable,
		},
	}); err != nil {
		t.Fatalf("move C under A: %v", err)
	}
	var cEntity model.DepartmentEntity
	if err := db.First(&cEntity, "id = ?", c.DepartmentID).Error; err != nil {
		t.Fatalf("query C: %v", err)
	}
	if cEntity.DeptPath != "/"+root.DepartmentID+"/"+c.DepartmentID || cEntity.DeptDepth != 2 {
		t.Fatalf("unexpected C path after move: %s depth=%d", cEntity.DeptPath, cEntity.DeptDepth)
	}

	// 环路：把 A 移到 C 下（C 是 A 的子孙）应拒绝
	err := svc.Update(ginCtx, &dtotenant.DepartmentUpdateReq{
		DepartmentID: root.DepartmentID,
		ParentID:     c.DepartmentID,
		DepartmentBaseInfo: objtenant.DepartmentBaseInfo{
			Name:   "A",
			Status: model.DeptNodeStatusEnable,
		},
	})
	if err == nil {
		t.Fatalf("expected cycle move to fail")
	}
	// 环路移动失败后 A 的路径不应被破坏
	var aEntity model.DepartmentEntity
	if err := db.First(&aEntity, "id = ?", root.DepartmentID).Error; err != nil {
		t.Fatalf("query A: %v", err)
	}
	if aEntity.DeptPath != "/"+root.DepartmentID {
		t.Fatalf("A path corrupted after rejected move: %s", aEntity.DeptPath)
	}
}

func TestDepartmentDeleteRejectsWithChildrenAndCascade(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.DepartmentEntity{}, &model.DepartmentUserEntity{}, &model.RoleEntity{}, &model.UserRoleEntity{})
	ginCtx := newDeptGinCtx(t, "41", "1001")
	seedTenantAdminOperator(t, db, "41", "1001")

	svc := &departmentSvc{}
	root, _ := svc.Create(ginCtx, &dtotenant.DepartmentCreateReq{
		DepartmentBaseInfo: objtenant.DepartmentBaseInfo{Name: "A"},
	})
	child, _ := svc.Create(ginCtx, &dtotenant.DepartmentCreateReq{
		ParentID:           root.DepartmentID,
		DepartmentBaseInfo: objtenant.DepartmentBaseInfo{Name: "B"},
	})

	// 有子节点默认拒绝
	if err := svc.Delete(ginCtx, &dtotenant.DepartmentDeleteReq{DepartmentID: root.DepartmentID}); err == nil {
		t.Fatalf("expected delete with children to fail")
	}
	// cascade 删除成功且子树软删
	if err := svc.Delete(ginCtx, &dtotenant.DepartmentDeleteReq{DepartmentID: root.DepartmentID, Cascade: true}); err != nil {
		t.Fatalf("cascade delete: %v", err)
	}
	var count int64
	if err := db.Model(&model.DepartmentEntity{}).Unscoped().Count(&count).Error; err != nil {
		t.Fatalf("count depts: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 soft-deleted rows, got %d", count)
	}
	var childEntity model.DepartmentEntity
	if err := db.First(&childEntity, "id = ?", child.DepartmentID).Error; err == nil {
		t.Fatalf("expected child soft-deleted")
	}
}

func TestDepartmentUserMemberSingletonAndValidTypes(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.DepartmentEntity{}, &model.DepartmentUserEntity{}, &model.UserEntity{}, &model.RoleEntity{}, &model.UserRoleEntity{})
	ginCtx := newDeptGinCtx(t, "41", "1001")
	seedTenantAdminOperator(t, db, "41", "1001")
	seedTestUser(t, db, "41", "u1", "用户一")
	seedTestUser(t, db, "41", "u2", "用户二")

	deptSvc := &departmentSvc{}
	root, _ := deptSvc.Create(ginCtx, &dtotenant.DepartmentCreateReq{
		DepartmentBaseInfo: objtenant.DepartmentBaseInfo{Name: "A"},
	})
	other, _ := deptSvc.Create(ginCtx, &dtotenant.DepartmentCreateReq{
		ParentID:           root.DepartmentID,
		DepartmentBaseInfo: objtenant.DepartmentBaseInfo{Name: "A1"},
	})

	svc := &departmentUserSvc{}
	// u1 行政归属 A
	if _, err := svc.Create(ginCtx, &dtotenant.DepartmentUserCreateReq{
		DepartmentID: root.DepartmentID,
		UserID:       "u1",
		RelationType: model.DeptUserRelationPrimary,
	}); err != nil {
		t.Fatalf("add member u1: %v", err)
	}
	// u1 再行政主部门 A1 → 应覆盖为 A1（primary 每用户至多 1 行）
	if _, err := svc.Create(ginCtx, &dtotenant.DepartmentUserCreateReq{
		DepartmentID: other.DepartmentID,
		UserID:       "u1",
		RelationType: model.DeptUserRelationPrimary,
	}); err != nil {
		t.Fatalf("reassign primary u1 to A1: %v", err)
	}
	var primaryCount int64
	if err := db.Model(&model.DepartmentUserEntity{}).
		Where("user_id = ? AND relation_type = ?", "u1", model.DeptUserRelationPrimary).
		Count(&primaryCount).Error; err != nil {
		t.Fatalf("count primary: %v", err)
	}
	if primaryCount != 1 {
		t.Fatalf("expected exactly 1 primary relation, got %d", primaryCount)
	}

	// 非法关系类型拒绝
	if _, err := svc.Create(ginCtx, &dtotenant.DepartmentUserCreateReq{
		DepartmentID: root.DepartmentID,
		UserID:       "u2",
		RelationType: "admin",
	}); err == nil {
		t.Fatalf("expected invalid relation type to fail")
	}
	// secondary 正常建立（u2 参与 A）
	if _, err := svc.Create(ginCtx, &dtotenant.DepartmentUserCreateReq{
		DepartmentID: root.DepartmentID,
		UserID:       "u2",
		RelationType: model.DeptUserRelationSecondary,
	}); err != nil {
		t.Fatalf("add secondary u2: %v", err)
	}
	// leader 不要求同时是成员：u2 负责 A1
	if _, err := svc.Create(ginCtx, &dtotenant.DepartmentUserCreateReq{
		DepartmentID: other.DepartmentID,
		UserID:       "u2",
		RelationType: model.DeptUserRelationLeader,
	}); err != nil {
		t.Fatalf("add leader u2: %v", err)
	}
	// 一个部门至多一个负责人：u2 已是 A1 的负责人，u1 再设 A1 负责人应冲突拒绝
	if _, err := svc.Create(ginCtx, &dtotenant.DepartmentUserCreateReq{
		DepartmentID: other.DepartmentID,
		UserID:       "u1",
		RelationType: model.DeptUserRelationLeader,
	}); err == nil {
		t.Fatalf("expected leader conflict to be rejected")
	}
	// 同一用户重复设为同一部门负责人：幂等成功
	if _, err := svc.Create(ginCtx, &dtotenant.DepartmentUserCreateReq{
		DepartmentID: other.DepartmentID,
		UserID:       "u2",
		RelationType: model.DeptUserRelationLeader,
	}); err != nil {
		t.Fatalf("re-set same leader should succeed: %v", err)
	}
}

func TestDepartmentUserCrossTenantRejected(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.DepartmentEntity{}, &model.DepartmentUserEntity{}, &model.UserEntity{}, &model.RoleEntity{}, &model.UserRoleEntity{})
	ginCtx := newDeptGinCtx(t, "41", "1001")
	seedTenantAdminOperator(t, db, "41", "1001")
	seedTestUser(t, db, "41", "u1", "用户一")

	deptSvc := &departmentSvc{}
	root, _ := deptSvc.Create(ginCtx, &dtotenant.DepartmentCreateReq{
		DepartmentBaseInfo: objtenant.DepartmentBaseInfo{Name: "A"},
	})

	// 切换到其他租户：部门不可见 → 添加关系失败
	otherCtx := newDeptGinCtx(t, "99", "2001")
	svc := &departmentUserSvc{}
	if _, err := svc.Create(otherCtx, &dtotenant.DepartmentUserCreateReq{
		DepartmentID: root.DepartmentID,
		UserID:       "u1",
		RelationType: model.DeptUserRelationPrimary,
	}); err == nil {
		t.Fatalf("expected cross-tenant relation create to fail")
	}
}

func TestDepartmentChildrenPageAndHasChildren(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.DepartmentEntity{}, &model.DepartmentUserEntity{}, &model.RoleEntity{}, &model.UserRoleEntity{})
	ginCtx := newDeptGinCtx(t, "41", "1001")
	seedTenantAdminOperator(t, db, "41", "1001")

	svc := &departmentSvc{}
	root, err := svc.Create(ginCtx, &dtotenant.DepartmentCreateReq{
		DepartmentBaseInfo: objtenant.DepartmentBaseInfo{Name: "总公司", Status: model.DeptNodeStatusEnable},
	})
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	a, _ := svc.Create(ginCtx, &dtotenant.DepartmentCreateReq{
		ParentID:           root.DepartmentID,
		DepartmentBaseInfo: objtenant.DepartmentBaseInfo{Name: "A", Status: model.DeptNodeStatusEnable},
	})
	if _, err := svc.Create(ginCtx, &dtotenant.DepartmentCreateReq{
		ParentID:           root.DepartmentID,
		DepartmentBaseInfo: objtenant.DepartmentBaseInfo{Name: "B", Status: model.DeptNodeStatusDisable},
	}); err != nil {
		t.Fatalf("create B: %v", err)
	}
	// A 下挂一个深层子级，验证 hasChildren
	if _, err := svc.Create(ginCtx, &dtotenant.DepartmentCreateReq{
		ParentID:           a.DepartmentID,
		DepartmentBaseInfo: objtenant.DepartmentBaseInfo{Name: "A1", Status: model.DeptNodeStatusEnable},
	}); err != nil {
		t.Fatalf("create A1: %v", err)
	}

	// 直属子级：应只有 A、B 两项
	resp, err := svc.Children(ginCtx, &dtotenant.DepartmentChildrenReq{
		DepartmentID: root.DepartmentID,
		Page:         1,
		PageSize:     10,
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
	// 列表必须同时回传创建时间与更新时间（前端「创建时间」「更新时间」两列直读）
	for _, item := range resp.List {
		if item.CreatedAt <= 0 || item.UpdatedAt <= 0 {
			t.Fatalf("children item missing time fields: %+v", item)
		}
	}

	// 状态筛选：只返回启用的 A
	resp, err = svc.Children(ginCtx, &dtotenant.DepartmentChildrenReq{
		DepartmentID: root.DepartmentID,
		Status:       model.DeptNodeStatusEnable,
		Page:         1,
		PageSize:     10,
	})
	if err != nil {
		t.Fatalf("children status filter: %v", err)
	}
	if resp.Total != 1 || resp.List[0].Name != "A" {
		t.Fatalf("expected only A, got total=%d names=%+v", resp.Total, resp.List)
	}

	// 分页：pageSize=1 时 total 仍为 2
	resp, err = svc.Children(ginCtx, &dtotenant.DepartmentChildrenReq{
		DepartmentID: root.DepartmentID,
		Page:         1,
		PageSize:     1,
	})
	if err != nil {
		t.Fatalf("children paging: %v", err)
	}
	if resp.Total != 2 || len(resp.List) != 1 {
		t.Fatalf("expected total=2 len=1, got total=%d len=%d", resp.Total, len(resp.List))
	}

	// 跨租户：另一租户查询该部门子级应失败
	otherCtx := newDeptGinCtx(t, "42", "1002")
	if _, err := svc.Children(otherCtx, &dtotenant.DepartmentChildrenReq{
		DepartmentID: root.DepartmentID,
		Page:         1,
		PageSize:     10,
	}); err == nil {
		t.Fatalf("expected cross-tenant children to fail")
	}
}
