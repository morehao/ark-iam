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
		BaseEntity:       gormdao.BaseEntity{StringID: gormdao.StringID{ID: id}},
		Username:         model.StrPtr(username),
		PrimaryEmail:     model.StrPtr(email),
		PasswordEncrypted: "",
		PasswordMethod:   "",
		Name:             "张三",
		Profile:          json.RawMessage("{}"),
		CustomData:       json.RawMessage("{}"),
		CreatedBy:        "t",
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
	resp, err := svc.Create(ginCtx, &dtotenant.UserCreateReq{Name: "仅姓名用户", OrganizationIDs: []string{"o1"}})
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
	if _, err := svc.Create(ginCtx, &dtotenant.UserCreateReq{Name: "Bob4", PersonID: personID, OrganizationIDs: []string{"o1"}}); err == nil {
		t.Fatalf("expected error when linking person already in tenant")
	}

	// 提供密码但无任何登录标识：仍创建自然人（姓名），密码哈希落库
	respPwd, err := svc.Create(ginCtx, &dtotenant.UserCreateReq{Name: "NoID", Password: "x123456", OrganizationIDs: []string{"o1"}})
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

	// 提供合法部门：正常创建
	if _, err := svc.Create(ginCtx, &dtotenant.UserCreateReq{Name: "王五", OrganizationIDs: []string{"o1"}}); err != nil {
		t.Fatalf("create user with valid org: %v", err)
	}
}

// TestUserCreateWithOrganizations 创建用户同时建立组织归属（首个为主组织）。
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
	resp, err := svc.Create(ginCtx, &dtotenant.UserCreateReq{Name: "张三", OrganizationIDs: []string{"o1", "o2"}})
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
	if !relations[0].IsPrimary || relations[1].IsPrimary {
		t.Fatalf("expected first relation primary, got %+v", relations)
	}

	// 非法组织（非本租户）拒绝
	if _, err := svc.Create(ginCtx, &dtotenant.UserCreateReq{Name: "李四", OrganizationIDs: []string{"o-other"}}); err == nil {
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
		RelationType:   "member",
		IsPrimary:      true,
	}).Error; err != nil {
		t.Fatalf("seed org-user: %v", err)
	}
	if err := db.Create(&model.RoleEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "r1"}},
		TenantID:   "t1",
		Name:       "管理员",
		Code:       "admin",
		Type:       "Admin",
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
	if len(resp.Organizations) != 1 || resp.Organizations[0].OrganizationName != "研发部" || !resp.Organizations[0].IsPrimary {
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
			Type:       "User",
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
