package svctenant

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/tenantadmin/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/tenantadmin/testutil"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

func seedTestPerson(t *testing.T, db *gorm.DB, id, username, email string) *model.PersonEntity {
	t.Helper()
	entity := &model.PersonEntity{
		BaseEntity:        gormdao.BaseEntity{StringID: gormdao.StringID{ID: id}},
		Username:          model.StrPtr(username),
		PrimaryEmail:      model.StrPtr(email),
		PasswordEncrypted: "",
		PasswordMethod:    "",
		Name:              "张三",
		Profile:           json.RawMessage("{}"),
		CustomData:        json.RawMessage("{}"),
		CreatedBy:         "t",
	}
	if err := db.Create(entity).Error; err != nil {
		t.Fatalf("seed person: %v", err)
	}
	return entity
}

func seedTestUserWithPerson(t *testing.T, db *gorm.DB, userID, tenantID, personID, name string) {
	t.Helper()
	now := time.Now()
	if err := db.Create(&model.UserEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: userID}},
		TenantID:   tenantID,
		PersonID:   personID,
		Name:       name,
		Profile:    json.RawMessage("{}"),
		CustomData: json.RawMessage("{}"),
		JoinedAt:   &now,
		CreatedBy:  "t",
	}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
}

// TestUserCreateFindOrCreatePerson 覆盖 person find-or-create 全路径：
// 仅姓名→创建仅含姓名的自然人；提供标识→新建 person（姓名即自然人姓名）；标识命中已有 person→关联复用；同租户重复→拒绝。
// 用户必属部门：所有创建均携带 organizationIDs（t1→o1、t2→o2）。
func TestUserCreateFindOrCreatePerson(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.UserEntity{}, &model.PersonEntity{}, &model.OrganizationEntity{}, &model.OrganizationUserEntity{})
	svc := &userSvc{}

	// 部门（用户必属部门）：t1 拥有 o1，t2 拥有 o2
	for _, o := range []struct{ tenantID, id string }{{"t1", "o1"}, {"t2", "o2"}} {
		if err := db.Create(&model.OrganizationEntity{
			BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: o.id}},
			TenantID:   o.tenantID,
			OrgPath:    "/" + o.id,
			OrgDepth:   1,
			Name:       "组织" + o.id,
			Status:     "active",
		}).Error; err != nil {
			t.Fatalf("seed org %s: %v", o.id, err)
		}
	}

	// 无 personID 且无登录标识：以姓名创建自然人并关联（person 始终存在）
	ginCtx := newOrgGinCtx(t, "t1", "op")
	resp, err := svc.Create(ginCtx, &dtotenant.UserCreateReq{Name: "仅姓名用户", PrimaryEmail: "nameonly@x.com", OrganizationIDs: []string{"o1"}})
	if err != nil {
		t.Fatalf("create user with name-only person: %v", err)
	}
	var nameOnly model.UserEntity
	if err := db.First(&nameOnly, "id = ?", resp.UserID).Error; err != nil {
		t.Fatalf("query name-only user: %v", err)
	}
	if nameOnly.PersonID == "" {
		t.Fatalf("expected person created and linked")
	}
	var nameOnlyPerson model.PersonEntity
	if err := db.First(&nameOnlyPerson, "id = ?", nameOnly.PersonID).Error; err != nil {
		t.Fatalf("query name-only person: %v", err)
	}
	if nameOnlyPerson.Name != "仅姓名用户" {
		t.Fatalf("expected person name from 姓名, got %s", nameOnlyPerson.Name)
	}

	// 提供 email+password：新建 person（姓名=req.Name，bcrypt 哈希）
	respB, err := svc.Create(ginCtx, &dtotenant.UserCreateReq{Name: "Bob", PrimaryEmail: "bob@x.com", Password: "secret123", OrganizationIDs: []string{"o1"}})
	if err != nil {
		t.Fatalf("create user with new person: %v", err)
	}
	var bob model.UserEntity
	if err := db.First(&bob, "id = ?", respB.UserID).Error; err != nil {
		t.Fatalf("query bob: %v", err)
	}
	if bob.PersonID == "" {
		t.Fatalf("expected person created and linked")
	}
	var bobPerson model.PersonEntity
	if err := db.First(&bobPerson, "id = ?", bob.PersonID).Error; err != nil {
		t.Fatalf("query bob person: %v", err)
	}
	if bobPerson.PasswordEncrypted == "" || bobPerson.PasswordMethod != "bcrypt" {
		t.Fatalf("expected bcrypt password on person")
	}

	// 另一租户提供相同 email：find-or-create 命中已有 person 并关联（复用同一自然人）
	respC, err := svc.Create(newOrgGinCtx(t, "t2", "op2"), &dtotenant.UserCreateReq{Name: "Bob2", PrimaryEmail: "bob@x.com", OrganizationIDs: []string{"o2"}})
	if err != nil {
		t.Fatalf("create user linking existing person: %v", err)
	}
	var bob2 model.UserEntity
	if err := db.First(&bob2, "id = ?", respC.UserID).Error; err != nil {
		t.Fatalf("query bob2: %v", err)
	}
	if bob2.PersonID != bob.PersonID {
		t.Fatalf("expected same person reused, got %s vs %s", bob2.PersonID, bob.PersonID)
	}

	// 同租户重复加入：拒绝
	if _, err := svc.Create(ginCtx, &dtotenant.UserCreateReq{Name: "Bob3", PrimaryEmail: "bob@x.com", OrganizationIDs: []string{"o1"}}); err == nil {
		t.Fatalf("expected duplicate-in-tenant error")
	}

	// 指定 personID 直接关联；同一 person 在本租户已有 user 时拒绝
	personID := bob.PersonID
	if _, err := svc.Create(ginCtx, &dtotenant.UserCreateReq{Name: "Bob4", PersonID: personID, PrimaryEmail: "bob4@x.com", OrganizationIDs: []string{"o1"}}); err == nil {
		t.Fatalf("expected error when linking person already in tenant")
	}

	// 提供密码但无任何登录标识：仍创建自然人（姓名），密码哈希落库
	respPwd, err := svc.Create(ginCtx, &dtotenant.UserCreateReq{Name: "NoID", PrimaryPhone: "13000000001", Password: "x123456", OrganizationIDs: []string{"o1"}})
	if err != nil {
		t.Fatalf("create user with password only: %v", err)
	}
	var noID model.UserEntity
	if err := db.First(&noID, "id = ?", respPwd.UserID).Error; err != nil {
		t.Fatalf("query no-id user: %v", err)
	}
	var noIDPerson model.PersonEntity
	if err := db.First(&noIDPerson, "id = ?", noID.PersonID).Error; err != nil {
		t.Fatalf("query no-id person: %v", err)
	}
	if noIDPerson.PasswordEncrypted == "" {
		t.Fatalf("expected password hash on person")
	}
}

// TestUserCreateRequiresOrganization 业务约束：创建用户必须从属于至少一个部门
// （organizationIDs 必传，缺失或为空一律拒绝；提供合法部门则正常创建）。
func TestUserCreateRequiresOrganization(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.UserEntity{}, &model.PersonEntity{}, &model.OrganizationEntity{}, &model.OrganizationUserEntity{})
	svc := &userSvc{}
	ginCtx := newOrgGinCtx(t, "t1", "op")

	if err := db.Create(&model.OrganizationEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "o1"}},
		TenantID:   "t1",
		OrgPath:    "/o1",
		OrgDepth:   1,
		Name:       "研发部",
		Status:     "active",
	}).Error; err != nil {
		t.Fatalf("seed org: %v", err)
	}

	requiredErr := code.GetError(code.UserOrganizationRequiredError)
	for name, req := range map[string]*dtotenant.UserCreateReq{
		"missing": {Name: "张三"},
		"empty":   {Name: "李四", OrganizationIDs: []string{}},
	} {
		if _, err := svc.Create(ginCtx, req); !errors.Is(err, requiredErr) {
			t.Fatalf("case %s: expected UserOrganizationRequiredError, got %v", name, err)
		}
	}

	// 邮箱、手机号二选一：都为空拒绝
	contactErr := code.GetError(code.UserContactRequiredError)
	if _, err := svc.Create(ginCtx, &dtotenant.UserCreateReq{Name: "无联系方式", OrganizationIDs: []string{"o1"}}); !errors.Is(err, contactErr) {
		t.Fatalf("expected UserContactRequiredError, got %v", err)
	}
	// 仅手机号可创建
	if _, err := svc.Create(ginCtx, &dtotenant.UserCreateReq{Name: "仅手机", PrimaryPhone: "15011112222", OrganizationIDs: []string{"o1"}}); err != nil {
		t.Fatalf("create with phone only should succeed: %v", err)
	}
	// 仅邮箱可创建
	if _, err := svc.Create(ginCtx, &dtotenant.UserCreateReq{Name: "仅邮箱", PrimaryEmail: "only@x.com", OrganizationIDs: []string{"o1"}}); err != nil {
		t.Fatalf("create with email only should succeed: %v", err)
	}

	// 提供合法部门：正常创建
	if _, err := svc.Create(ginCtx, &dtotenant.UserCreateReq{Name: "王五", PrimaryEmail: "ww@x.com", OrganizationIDs: []string{"o1"}}); err != nil {
		t.Fatalf("create user with valid org: %v", err)
	}
}

// TestUserCreateWithOrganizations 创建用户时建立行政主部门(primary,至多1)与负责关系(leader,可多)。
func TestUserCreateWithOrganizations(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.UserEntity{}, &model.PersonEntity{}, &model.OrganizationEntity{}, &model.OrganizationUserEntity{})
	svc := &userSvc{}

	now := time.Now()
	for i, orgID := range []string{"o1", "o2"} {
		if err := db.Create(&model.OrganizationEntity{
			BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: orgID}},
			TenantID:   "t1",
			OrgPath:    "/" + orgID,
			OrgDepth:   1,
			Name:       "组织" + orgID,
			Status:     "active",
		}).Error; err != nil {
			t.Fatalf("seed org %d: %v", i, err)
		}
		_ = now
	}

	ginCtx := newOrgGinCtx(t, "t1", "op")
	// 创建用户：primary=o1（单个行政主部门）+ leader=o2
	resp, err := svc.Create(ginCtx, &dtotenant.UserCreateReq{Name: "张三", PrimaryEmail: "zs@x.com", OrganizationIDs: []string{"o1"}, LeaderOrgIDs: []string{"o2"}})
	if err != nil {
		t.Fatalf("create user with orgs: %v", err)
	}
	var relations []model.OrganizationUserEntity
	if err := db.Where("tenant_id = ? AND user_id = ?", "t1", resp.UserID).Find(&relations).Error; err != nil {
		t.Fatalf("query relations: %v", err)
	}
	if len(relations) != 2 {
		t.Fatalf("expected 2 org relations, got %d", len(relations))
	}
	byType := map[model.OrgUserRelationType]string{}
	for _, r := range relations {
		byType[r.RelationType] = r.OrganizationID
	}
	if byType[model.OrgUserRelationPrimary] != "o1" || byType[model.OrgUserRelationLeader] != "o2" {
		t.Fatalf("unexpected relations: %+v", relations)
	}

	// primary 至多 1 行：传多个行政主部门应拒绝
	if _, err := svc.Create(ginCtx, &dtotenant.UserCreateReq{Name: "李四", PrimaryEmail: "lis@x.com", OrganizationIDs: []string{"o1", "o2"}}); err == nil {
		t.Fatalf("expected multi-primary create to fail")
	}

	// 非法组织（非本租户）拒绝
	if _, err := svc.Create(ginCtx, &dtotenant.UserCreateReq{Name: "王五", PrimaryPhone: "15000000002", OrganizationIDs: []string{"o-other"}}); err == nil {
		t.Fatalf("expected invalid org error")
	}
}

// TestUserPageListKeyword 关键词过滤：姓名 / person 的 username / email 均命中。
func TestUserPageListKeyword(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.UserEntity{}, &model.PersonEntity{}, &model.OrganizationEntity{},
		&model.OrganizationUserEntity{}, &model.UserRoleEntity{})
	svc := &userSvc{}

	p1 := seedTestPerson(t, db, "p1", "zhangsan", "")
	seedTestUserWithPerson(t, db, "u1", "t1", p1.ID, "张三")
	seedTestUserWithPerson(t, db, "u2", "t1", "", "李四")
	p3 := seedTestPerson(t, db, "p3", "", "wang@x.com")
	seedTestUserWithPerson(t, db, "u3", "t1", p3.ID, "王五")

	ginCtx := newOrgGinCtx(t, "t1", "op")

	// 按 person username 命中
	resp, err := svc.PageList(ginCtx, &dtotenant.UserPageListReq{Page: 1, PageSize: 10, Keyword: "zhangsan"})
	if err != nil {
		t.Fatalf("page list: %v", err)
	}
	if len(resp.List) != 1 || resp.List[0].UserID != "u1" {
		t.Fatalf("expected only u1 matched by username, got %+v", resp.List)
	}

	// 按租户内姓名命中
	resp, err = svc.PageList(ginCtx, &dtotenant.UserPageListReq{Page: 1, PageSize: 10, Keyword: "李四"})
	if err != nil {
		t.Fatalf("page list: %v", err)
	}
	if len(resp.List) != 1 || resp.List[0].UserID != "u2" {
		t.Fatalf("expected only u2 matched by name, got %+v", resp.List)
	}

	// 按 person email 命中
	resp, err = svc.PageList(ginCtx, &dtotenant.UserPageListReq{Page: 1, PageSize: 10, Keyword: "wang@x"})
	if err != nil {
		t.Fatalf("page list: %v", err)
	}
	if len(resp.List) != 1 || resp.List[0].UserID != "u3" {
		t.Fatalf("expected only u3 matched by email, got %+v", resp.List)
	}

	// 空关键词返回全部
	resp, err = svc.PageList(ginCtx, &dtotenant.UserPageListReq{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("page list: %v", err)
	}
	if resp.Total != 3 {
		t.Fatalf("expected total 3, got %d", resp.Total)
	}
}

// TestUserDetailWithOrganizationsAndRoles 详情含组织归属与角色列表。
func TestUserDetailWithOrganizationsAndRoles(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.UserEntity{}, &model.PersonEntity{}, &model.OrganizationEntity{},
		&model.OrganizationUserEntity{}, &model.RoleEntity{}, &model.UserRoleEntity{},
		&model.ApplicationEntity{}, &model.TenantApplicationEntity{})
	svc := &userSvc{}

	now := time.Now()
	seedTestUserWithPerson(t, db, "u1", "t1", "", "张三")

	if err := db.Create(&model.OrganizationEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "o1"}},
		TenantID:   "t1",
		OrgPath:    "/o1",
		OrgDepth:   1,
		Name:       "研发部",
		Status:     "active",
	}).Error; err != nil {
		t.Fatalf("seed org: %v", err)
	}
	if err := db.Create(&model.OrganizationUserEntity{
		BaseEntity:     gormdao.BaseEntity{StringID: gormdao.StringID{ID: "ou1"}},
		TenantID:       "t1",
		OrganizationID: "o1",
		UserID:         "u1",
		RelationType:   model.OrgUserRelationPrimary,
	}).Error; err != nil {
		t.Fatalf("seed org-user: %v", err)
	}
	if err := db.Create(&model.RoleEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "r1"}},
		TenantID:   "t1",
		Name:       "管理员",
		Code:       "admin",
	}).Error; err != nil {
		t.Fatalf("seed role: %v", err)
	}
	if err := db.Create(&model.UserRoleEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "ur1"}},
		TenantID:   "t1",
		UserID:     "u1",
		RoleID:     "r1",
		CreatedBy:  "t",
	}).Error; err != nil {
		t.Fatalf("seed user-role: %v", err)
	}

	ginCtx := newOrgGinCtx(t, "t1", "op")
	resp, err := svc.Detail(ginCtx, &dtotenant.UserDetailReq{UserID: "u1"})
	if err != nil {
		t.Fatalf("detail: %v", err)
	}
	if len(resp.Organizations) != 1 || resp.Organizations[0].OrganizationName != "研发部" || resp.Organizations[0].RelationType != model.OrgUserRelationPrimary {
		t.Fatalf("unexpected organizations: %+v", resp.Organizations)
	}
	if len(resp.Roles) != 1 || resp.Roles[0].Code != "admin" {
		t.Fatalf("unexpected roles: %+v", resp.Roles)
	}
	_ = now
}

// TestUserUpdateRolesFullReplace 全量替换用户角色。
func TestUserUpdateRolesFullReplace(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.UserEntity{}, &model.RoleEntity{}, &model.UserRoleEntity{})
	svc := &userSvc{}
	seedTestUserWithPerson(t, db, "u1", "t1", "", "张三")

	for _, r := range []struct{ id, code string }{{"r1", "admin"}, {"r2", "user"}, {"r3", "guest"}} {
		if err := db.Create(&model.RoleEntity{
			BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: r.id}},
			TenantID:   "t1",
			Name:       r.code,
			Code:       r.code,
		}).Error; err != nil {
			t.Fatalf("seed role %s: %v", r.id, err)
		}
	}
	// 另一租户的角色
	if err := db.Create(&model.RoleEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "r-other"}},
		TenantID:   "t2",
		Name:       "other",
		Code:       "other",
	}).Error; err != nil {
		t.Fatalf("seed other role: %v", err)
	}

	ginCtx := newOrgGinCtx(t, "t1", "op")
	if err := svc.UpdateRoles(ginCtx, &dtotenant.UserRolesUpdateReq{UserID: "u1", RoleIDs: []string{"r1", "r2"}}); err != nil {
		t.Fatalf("update roles: %v", err)
	}
	var urList []model.UserRoleEntity
	if err := db.Where("tenant_id = ? AND user_id = ?", "t1", "u1").Find(&urList).Error; err != nil {
		t.Fatalf("query user_role: %v", err)
	}
	if len(urList) != 2 {
		t.Fatalf("expected 2 relations, got %d", len(urList))
	}

	// 全量替换为单个
	if err := svc.UpdateRoles(ginCtx, &dtotenant.UserRolesUpdateReq{UserID: "u1", RoleIDs: []string{"r2"}}); err != nil {
		t.Fatalf("update roles: %v", err)
	}
	if err := db.Where("tenant_id = ? AND user_id = ?", "t1", "u1").Find(&urList).Error; err != nil {
		t.Fatalf("query user_role: %v", err)
	}
	if len(urList) != 1 || urList[0].RoleID != "r2" {
		t.Fatalf("expected only r2, got %+v", urList)
	}

	// 跨租户角色应拒绝
	if err := svc.UpdateRoles(ginCtx, &dtotenant.UserRolesUpdateReq{UserID: "u1", RoleIDs: []string{"r-other"}}); err == nil {
		t.Fatalf("expected cross-tenant role error")
	}
}

// TestUserCreateWithLeaderOrgs 创建用户同时建立行政主部门(primary)、参与部门(secondary)与负责部门(leader)。
func TestUserCreateWithLeaderOrgs(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.UserEntity{}, &model.PersonEntity{}, &model.OrganizationEntity{}, &model.OrganizationUserEntity{})
	svc := &userSvc{}

	for _, orgID := range []string{"o1", "o2", "o3"} {
		if err := db.Create(&model.OrganizationEntity{
			BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: orgID}},
			TenantID:   "t1",
			OrgPath:    "/" + orgID,
			OrgDepth:   1,
			Name:       "组织" + orgID,
			Status:     "active",
		}).Error; err != nil {
			t.Fatalf("seed org %s: %v", orgID, err)
		}
	}

	ginCtx := newOrgGinCtx(t, "t1", "op")
	resp, err := svc.Create(ginCtx, &dtotenant.UserCreateReq{
		Name:            "张三",
		PrimaryEmail:    "zs2@x.com",
		OrganizationIDs: []string{"o1"},
		SecondaryOrgIDs: []string{"o2"},
		LeaderOrgIDs:    []string{"o3"},
	})
	if err != nil {
		t.Fatalf("create user with leader orgs: %v", err)
	}
	var relations []model.OrganizationUserEntity
	if err := db.Where("tenant_id = ? AND user_id = ?", "t1", resp.UserID).Find(&relations).Error; err != nil {
		t.Fatalf("query relations: %v", err)
	}
	if len(relations) != 3 {
		t.Fatalf("expected 3 relations (1 primary + 1 secondary + 1 leader), got %d", len(relations))
	}
	var primaryCount, secondaryCount, leaderCount int
	for _, r := range relations {
		switch r.RelationType {
		case model.OrgUserRelationPrimary:
			primaryCount++
		case model.OrgUserRelationSecondary:
			secondaryCount++
		case model.OrgUserRelationLeader:
			leaderCount++
		}
	}
	if primaryCount != 1 || secondaryCount != 1 || leaderCount != 1 {
		t.Fatalf("unexpected relation distribution, primary:%d secondary:%d leaders:%d", primaryCount, secondaryCount, leaderCount)
	}

	// 一个部门至多一个负责人：新用户再负责 o3 应拒绝
	if _, err := svc.Create(ginCtx, &dtotenant.UserCreateReq{Name: "李四", PrimaryPhone: "15000000003", OrganizationIDs: []string{"o1"}, LeaderOrgIDs: []string{"o3"}}); err == nil {
		t.Fatalf("expected leader conflict to be rejected")
	}

	// 负责部门非法（非本租户）拒绝
	if _, err := svc.Create(ginCtx, &dtotenant.UserCreateReq{Name: "王五", PrimaryEmail: "ww5@x.com", OrganizationIDs: []string{"o1"}, LeaderOrgIDs: []string{"o-other"}}); err == nil {
		t.Fatalf("expected invalid leader org error")
	}
}

// TestUserUpdateOrganizations 编辑用户时更新主/参与/负责部门：
// primary 替换、secondary/leader 全量替换、leader 冲突拒绝。
func TestUserUpdateOrganizations(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.UserEntity{}, &model.PersonEntity{}, &model.OrganizationEntity{}, &model.OrganizationUserEntity{})
	svc := &userSvc{}

	for _, orgID := range []string{"o1", "o2", "o3"} {
		if err := db.Create(&model.OrganizationEntity{
			BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: orgID}},
			TenantID:   "t1",
			OrgPath:    "/" + orgID,
			OrgDepth:   1,
			Name:       "组织" + orgID,
			Status:     "active",
		}).Error; err != nil {
			t.Fatalf("seed org %s: %v", orgID, err)
		}
	}

	ginCtx := newOrgGinCtx(t, "t1", "op")
	u1, err := svc.Create(ginCtx, &dtotenant.UserCreateReq{
		Name:            "张三",
		PrimaryEmail:    "u1@x.com",
		OrganizationIDs: []string{"o1"},
		SecondaryOrgIDs: []string{"o2"},
		LeaderOrgIDs:    []string{"o3"},
	})
	if err != nil {
		t.Fatalf("create u1: %v", err)
	}

	countByType := func(userID string) map[model.OrgUserRelationType]string {
		var rows []model.OrganizationUserEntity
		if err := db.Where("tenant_id = ? AND user_id = ?", "t1", userID).Find(&rows).Error; err != nil {
			t.Fatalf("query relations: %v", err)
		}
		m := map[model.OrgUserRelationType]string{}
		for _, r := range rows {
			m[r.RelationType] = r.OrganizationID
		}
		return m
	}

	// 替换主部门 o1 -> o2
	prim := "o2"
	if err := svc.Update(ginCtx, &dtotenant.UserUpdateReq{UserID: u1.UserID, PrimaryOrgID: &prim}); err != nil {
		t.Fatalf("update primary: %v", err)
	}
	m := countByType(u1.UserID)
	if m[model.OrgUserRelationPrimary] != "o2" {
		t.Fatalf("expected primary o2, got %+v", m)
	}

	// 全量替换参与部门 o2 -> [o3, o1]
	sec := []string{"o3", "o1"}
	if err := svc.Update(ginCtx, &dtotenant.UserUpdateReq{UserID: u1.UserID, SecondaryOrgIDs: &sec}); err != nil {
		t.Fatalf("update secondary: %v", err)
	}
	var secCount int64
	if err := db.Model(&model.OrganizationUserEntity{}).
		Where("tenant_id = ? AND user_id = ? AND relation_type = ?", "t1", u1.UserID, model.OrgUserRelationSecondary).
		Count(&secCount).Error; err != nil {
		t.Fatalf("count secondary: %v", err)
	}
	if secCount != 2 {
		t.Fatalf("expected 2 secondary relations, got %d", secCount)
	}

	// 全量替换负责部门 o3 -> [o1]
	lead := []string{"o1"}
	if err := svc.Update(ginCtx, &dtotenant.UserUpdateReq{UserID: u1.UserID, LeaderOrgIDs: &lead}); err != nil {
		t.Fatalf("update leader: %v", err)
	}
	m = countByType(u1.UserID)
	if m[model.OrgUserRelationLeader] != "o1" {
		t.Fatalf("expected leader o1, got %+v", m)
	}

	// leader 冲突：另建用户 u2 负责 o1（已被 u1 负责）应拒绝
	u2, err := svc.Create(ginCtx, &dtotenant.UserCreateReq{Name: "李四", PrimaryPhone: "15000000004", OrganizationIDs: []string{"o1"}})
	if err != nil {
		t.Fatalf("create u2: %v", err)
	}
	conflictLead := []string{"o1"}
	if err := svc.Update(ginCtx, &dtotenant.UserUpdateReq{UserID: u2.UserID, LeaderOrgIDs: &conflictLead}); err == nil {
		t.Fatalf("expected leader conflict rejected")
	}
}

// TestUserUpdateContact 编辑成员联系方式：更新 person 的邮箱/手机号/用户名，
// 校验邮箱与手机号二选一、以及全局唯一性冲突。
func TestUserUpdateContact(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.UserEntity{}, &model.PersonEntity{}, &model.OrganizationEntity{}, &model.OrganizationUserEntity{})
	svc := &userSvc{}

	if err := db.Create(&model.OrganizationEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "o1"}},
		TenantID:   "t1",
		OrgPath:    "/o1",
		OrgDepth:   1,
		Name:       "研发部",
		Status:     "active",
	}).Error; err != nil {
		t.Fatalf("seed org: %v", err)
	}

	ginCtx := newOrgGinCtx(t, "t1", "op")
	u1, err := svc.Create(ginCtx, &dtotenant.UserCreateReq{Name: "张三", PrimaryEmail: "zs@x.com", OrganizationIDs: []string{"o1"}})
	if err != nil {
		t.Fatalf("create u1: %v", err)
	}

	loadPerson := func() *model.PersonEntity {
		var user model.UserEntity
		if err := db.First(&user, "id = ?", u1.UserID).Error; err != nil {
			t.Fatalf("query user: %v", err)
		}
		var p model.PersonEntity
		if err := db.First(&p, "id = ?", user.PersonID).Error; err != nil {
			t.Fatalf("query person: %v", err)
		}
		return &p
	}

	// 更新手机号（原来只有邮箱）：编辑后邮箱+手机都非空
	phone := "15000000001"
	if err := svc.Update(ginCtx, &dtotenant.UserUpdateReq{UserID: u1.UserID, PrimaryPhone: &phone}); err != nil {
		t.Fatalf("update phone: %v", err)
	}
	p := loadPerson()
	if model.DerefStr(p.PrimaryPhone) != phone || model.DerefStr(p.PrimaryEmail) != "zs@x.com" {
		t.Fatalf("unexpected person after phone update: %+v", p)
	}

	// 清空邮箱只留手机号（编辑双方联系方式且至少一个非空）：email 清空成功
	empty := ""
	if err := svc.Update(ginCtx, &dtotenant.UserUpdateReq{UserID: u1.UserID, PrimaryEmail: &empty}); err != nil {
		t.Fatalf("clear email with phone kept: %v", err)
	}
	p = loadPerson()
	if model.DerefStr(p.PrimaryEmail) != "" || model.DerefStr(p.PrimaryPhone) != phone {
		t.Fatalf("unexpected person after clear email: %+v", p)
	}

	// 邮箱、手机都清空：二选一拒绝
	if err := svc.Update(ginCtx, &dtotenant.UserUpdateReq{UserID: u1.UserID, PrimaryEmail: &empty, PrimaryPhone: &empty}); err == nil {
		t.Fatalf("expected contact required error when both cleared")
	}

	// 唯一性：另建用户占 u1 的手机号，编辑 u1 撞库应拒绝
	u2, err := svc.Create(ginCtx, &dtotenant.UserCreateReq{Name: "李四", PrimaryPhone: "15000000002", OrganizationIDs: []string{"o1"}})
	if err != nil {
		t.Fatalf("create u2: %v", err)
	}
	_ = u2
	conflict := "15000000002"
	if err := svc.Update(ginCtx, &dtotenant.UserUpdateReq{UserID: u1.UserID, PrimaryPhone: &conflict}); err == nil {
		t.Fatalf("expected phone conflict error")
	}

	// 更新用户名
	uname := "zs_new"
	if err := svc.Update(ginCtx, &dtotenant.UserUpdateReq{UserID: u1.UserID, Username: &uname}); err != nil {
		t.Fatalf("update username: %v", err)
	}
	p = loadPerson()
	if model.DerefStr(p.Username) != uname {
		t.Fatalf("unexpected username after update: %+v", p)
	}
}

// TestMemberPageList 成员总表：以人为维度返回全部成员，含部门关系数组；
// 支持按"恰在该部门"(member/leader) 过滤、关键词命中租户内姓名。
func TestMemberPageList(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.UserEntity{}, &model.PersonEntity{}, &model.OrganizationEntity{},
		&model.OrganizationUserEntity{}, &model.UserRoleEntity{})
	svc := &userSvc{}

	// 组织
	for _, orgID := range []string{"o1", "o2"} {
		if err := db.Create(&model.OrganizationEntity{
			BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: orgID}},
			TenantID:   "t1",
			OrgPath:    "/" + orgID,
			OrgDepth:   1,
			Name:       "组织" + orgID,
			Status:     "active",
		}).Error; err != nil {
			t.Fatalf("seed org %s: %v", orgID, err)
		}
	}

	// 用户：u1 行政归属 o1；u2 负责 o2
	seedTestUserWithPerson(t, db, "u1", "t1", "", "张三")
	seedTestUserWithPerson(t, db, "u2", "t1", "", "李四")
	// 其他租户用户（不应出现）
	seedTestUserWithPerson(t, db, "u-other", "t2", "", "外人")

	if err := db.Create(&model.OrganizationUserEntity{
		BaseEntity:     gormdao.BaseEntity{StringID: gormdao.StringID{ID: "ou1"}},
		TenantID:       "t1",
		OrganizationID: "o1",
		UserID:         "u1",
		RelationType:   model.OrgUserRelationPrimary,
	}).Error; err != nil {
		t.Fatalf("seed rel ou1: %v", err)
	}
	if err := db.Create(&model.OrganizationUserEntity{
		BaseEntity:     gormdao.BaseEntity{StringID: gormdao.StringID{ID: "ou2"}},
		TenantID:       "t1",
		OrganizationID: "o2",
		UserID:         "u2",
		RelationType:   model.OrgUserRelationLeader,
	}).Error; err != nil {
		t.Fatalf("seed rel ou2: %v", err)
	}

	ginCtx := newOrgGinCtx(t, "t1", "op")

	// 全部成员（不含其他租户）
	resp, err := svc.MemberPageList(ginCtx, &dtotenant.MemberPageListReq{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("member page list: %v", err)
	}
	if resp.Total != 2 {
		t.Fatalf("expected total 2, got %d", resp.Total)
	}
	var u1Item, u2Item *dtotenant.MemberPageListItem
	for i := range resp.List {
		if resp.List[i].UserID == "u1" {
			u1Item = &resp.List[i]
		}
		if resp.List[i].UserID == "u2" {
			u2Item = &resp.List[i]
		}
	}
	if u1Item == nil || u2Item == nil {
		t.Fatalf("expected u1/u2 in list, got %+v", resp.List)
	}
	if u1Item.PrimaryOrgID != "o1" {
		t.Fatalf("expected u1 primary o1, got %s", u1Item.PrimaryOrgID)
	}
	if len(u1Item.Organizations) != 1 || u1Item.Organizations[0].OrganizationName != "组织o1" || u1Item.Organizations[0].RelationType != model.OrgUserRelationPrimary {
		t.Fatalf("unexpected u1 organizations: %+v", u1Item.Organizations)
	}
	if len(u2Item.Organizations) != 1 || u2Item.Organizations[0].RelationType != model.OrgUserRelationLeader {
		t.Fatalf("unexpected u2 organizations: %+v", u2Item.Organizations)
	}

	// 按部门过滤：o1 命中 u1；o2 命中 u2（leader 也算成员）
	resp, err = svc.MemberPageList(ginCtx, &dtotenant.MemberPageListReq{Page: 1, PageSize: 10, OrganizationID: "o1"})
	if err != nil {
		t.Fatalf("member page list by org: %v", err)
	}
	if resp.Total != 1 || resp.List[0].UserID != "u1" {
		t.Fatalf("expected only u1 for o1, got %+v", resp.List)
	}

	resp, err = svc.MemberPageList(ginCtx, &dtotenant.MemberPageListReq{Page: 1, PageSize: 10, OrganizationID: "o2"})
	if err != nil {
		t.Fatalf("member page list by org: %v", err)
	}
	if resp.Total != 1 || resp.List[0].UserID != "u2" {
		t.Fatalf("expected only u2 for o2, got %+v", resp.List)
	}

	// 关键词命中租户内姓名
	resp, err = svc.MemberPageList(ginCtx, &dtotenant.MemberPageListReq{Page: 1, PageSize: 10, Keyword: "张三"})
	if err != nil {
		t.Fatalf("member page list by keyword: %v", err)
	}
	if resp.Total != 1 || resp.List[0].UserID != "u1" {
		t.Fatalf("expected u1 by keyword, got %+v", resp.List)
	}

	// 部门过滤 + 无任何关系用户 u2 未在其他部门：查询无匹配部门(如空部门)返回空
	resp, err = svc.MemberPageList(ginCtx, &dtotenant.MemberPageListReq{Page: 1, PageSize: 10, OrganizationID: "o-none"})
	if err != nil {
		t.Fatalf("member page list by missing org: %v", err)
	}
	if resp.Total != 0 || len(resp.List) != 0 {
		t.Fatalf("expected empty list for missing org, got %+v", resp.List)
	}
}
