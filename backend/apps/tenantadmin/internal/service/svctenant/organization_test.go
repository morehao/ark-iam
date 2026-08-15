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

func TestOrganizationCreateRootAndChildPaths(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.OrganizationEntity{}, &model.OrganizationUserEntity{})
	ginCtx := newOrgGinCtx(t, "41", "1001")

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
		ParentID: root.OrganizationID,
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
	db := testutil.SetupSQLite(t, &model.OrganizationEntity{}, &model.OrganizationUserEntity{})
	ginCtx := newOrgGinCtx(t, "41", "1001")

	svc := &organizationSvc{}
	root, _ := svc.Create(ginCtx, &dtotenant.OrganizationCreateReq{
		OrganizationBaseInfo: objtenant.OrganizationBaseInfo{Name: "A"},
	})
	b, _ := svc.Create(ginCtx, &dtotenant.OrganizationCreateReq{
		ParentID: root.OrganizationID,
		OrganizationBaseInfo: objtenant.OrganizationBaseInfo{Name: "B"},
	})
	c, _ := svc.Create(ginCtx, &dtotenant.OrganizationCreateReq{
		ParentID: b.OrganizationID,
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
	db := testutil.SetupSQLite(t, &model.OrganizationEntity{}, &model.OrganizationUserEntity{})
	ginCtx := newOrgGinCtx(t, "41", "1001")

	svc := &organizationSvc{}
	root, _ := svc.Create(ginCtx, &dtotenant.OrganizationCreateReq{
		OrganizationBaseInfo: objtenant.OrganizationBaseInfo{Name: "A"},
	})
	child, _ := svc.Create(ginCtx, &dtotenant.OrganizationCreateReq{
		ParentID: root.OrganizationID,
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

func TestOrganizationUserRelationPrimaryAndLeaderGuard(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.OrganizationEntity{}, &model.OrganizationUserEntity{}, &model.UserEntity{})
	ginCtx := newOrgGinCtx(t, "41", "1001")
	seedTestUser(t, db, "41", "u1", "用户一")
	seedTestUser(t, db, "41", "u2", "用户二")

	orgSvc := &organizationSvc{}
	root, _ := orgSvc.Create(ginCtx, &dtotenant.OrganizationCreateReq{
		OrganizationBaseInfo: objtenant.OrganizationBaseInfo{Name: "A"},
	})
	other, _ := orgSvc.Create(ginCtx, &dtotenant.OrganizationCreateReq{
		ParentID: root.OrganizationID,
		OrganizationBaseInfo: objtenant.OrganizationBaseInfo{Name: "A1"},
	})

	svc := &organizationUserSvc{}
	// u1 归属 A（主）
	if _, err := svc.Create(ginCtx, &dtotenant.OrganizationUserCreateReq{
		OrganizationID: root.OrganizationID,
		UserID:         "u1",
		RelationType:   string(model.OrgUserRelationMember),
		IsPrimary:      true,
	}); err != nil {
		t.Fatalf("add member u1: %v", err)
	}
	// u1 再归属 A1（主）→ A 的主标记应被清
	if _, err := svc.Create(ginCtx, &dtotenant.OrganizationUserCreateReq{
		OrganizationID: other.OrganizationID,
		UserID:         "u1",
		RelationType:   string(model.OrgUserRelationMember),
		IsPrimary:      true,
	}); err != nil {
		t.Fatalf("add member u1 to A1: %v", err)
	}
	var primaryCount int64
	if err := db.Model(&model.OrganizationUserEntity{}).Where("user_id = ? AND is_primary = ?", "u1", true).Count(&primaryCount).Error; err != nil {
		t.Fatalf("count primary: %v", err)
	}
	if primaryCount != 1 {
		t.Fatalf("expected exactly 1 primary relation, got %d", primaryCount)
	}

	// leader 关系不能置主
	if _, err := svc.Create(ginCtx, &dtotenant.OrganizationUserCreateReq{
		OrganizationID: root.OrganizationID,
		UserID:         "u2",
		RelationType:   string(model.OrgUserRelationLeader),
		IsPrimary:      true,
	}); err == nil {
		t.Fatalf("expected leader-with-primary to fail")
	}
	// leader 关系正常建立（不要求是成员）
	if _, err := svc.Create(ginCtx, &dtotenant.OrganizationUserCreateReq{
		OrganizationID: root.OrganizationID,
		UserID:         "u2",
		RelationType:   string(model.OrgUserRelationLeader),
	}); err != nil {
		t.Fatalf("add leader u2: %v", err)
	}
}

func TestOrganizationUserCrossTenantRejected(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.OrganizationEntity{}, &model.OrganizationUserEntity{}, &model.UserEntity{})
	ginCtx := newOrgGinCtx(t, "41", "1001")
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
		RelationType:   string(model.OrgUserRelationMember),
	}); err == nil {
		t.Fatalf("expected cross-tenant relation create to fail")
	}
}

func TestUpdateUserOrganizationsReplace(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.OrganizationEntity{}, &model.OrganizationUserEntity{}, &model.UserEntity{})
	ginCtx := newOrgGinCtx(t, "41", "1001")
	seedTestUser(t, db, "41", "u1", "用户一")

	orgSvc := &organizationSvc{}
	root, _ := orgSvc.Create(ginCtx, &dtotenant.OrganizationCreateReq{
		OrganizationBaseInfo: objtenant.OrganizationBaseInfo{Name: "A"},
	})
	org2, _ := orgSvc.Create(ginCtx, &dtotenant.OrganizationCreateReq{
		ParentID: root.OrganizationID,
		OrganizationBaseInfo: objtenant.OrganizationBaseInfo{Name: "A2"},
	})

	svc := &organizationUserSvc{}
	// 全量替换为 [A, A2]
	if err := svc.UpdateUserOrganizations(ginCtx, &dtotenant.UserOrganizationsUpdateReq{
		UserID:          "u1",
		OrganizationIDs: []string{root.OrganizationID, org2.OrganizationID},
	}); err != nil {
		t.Fatalf("replace orgs: %v", err)
	}
	resp, err := svc.GetUserOrganizations(ginCtx, &dtotenant.UserOrganizationListReq{UserID: "u1"})
	if err != nil {
		t.Fatalf("get user orgs: %v", err)
	}
	if len(resp.List) != 2 {
		t.Fatalf("expected 2 relations, got %d", len(resp.List))
	}
	if !resp.List[0].IsPrimary || resp.List[1].IsPrimary {
		t.Fatalf("expected only first org primary, got %+v", resp.List)
	}

	// 全量替换为 [A2] → 只剩 1 条且为主
	if err := svc.UpdateUserOrganizations(ginCtx, &dtotenant.UserOrganizationsUpdateReq{
		UserID:          "u1",
		OrganizationIDs: []string{org2.OrganizationID},
	}); err != nil {
		t.Fatalf("replace orgs again: %v", err)
	}
	resp, err = svc.GetUserOrganizations(ginCtx, &dtotenant.UserOrganizationListReq{UserID: "u1"})
	if err != nil {
		t.Fatalf("get user orgs: %v", err)
	}
	if len(resp.List) != 1 || !resp.List[0].IsPrimary || resp.List[0].OrganizationID != org2.OrganizationID {
		t.Fatalf("unexpected relations after second replace: %+v", resp.List)
	}
}
