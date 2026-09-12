package svctenant

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/tenantadmin/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/tenantadmin/testutil"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/gcrypto"
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
// 用户必属部门：所有创建均携带 departmentIDs（t1→o1、t2→o2）。
func TestUserCreateFindOrCreatePerson(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.UserEntity{}, &model.PersonEntity{}, &model.DepartmentEntity{}, &model.DepartmentUserEntity{}, &model.RoleEntity{}, &model.UserRoleEntity{})
	svc := &userSvc{}

	// 部门（用户必属部门）：t1 拥有 o1，t2 拥有 o2
	for _, o := range []struct{ tenantID, id string }{{"t1", "o1"}, {"t2", "o2"}} {
		if err := db.Create(&model.DepartmentEntity{
			BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: o.id}},
			TenantID:   o.tenantID,
			DeptPath:   "/" + o.id,
			DeptDepth:  1,
			Name:       "部门" + o.id,
			Status:     model.DeptNodeStatusEnable,
		}).Error; err != nil {
			t.Fatalf("seed dept %s: %v", o.id, err)
		}
	}

	// 无 personID 且无登录标识：以姓名创建自然人并关联（person 始终存在）
	ginCtx := newAdminCtx(t, db, "t1", "op")
	resp, err := svc.Create(ginCtx, &dtotenant.UserCreateReq{Name: "仅姓名用户", PrimaryEmail: "nameonly@x.com", PrimaryDepartmentID: "o1"})
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

	// 提供 email：新建 person（姓名=req.Name，bcrypt 哈希），临时密码仅创建响应返回一次
	respB, err := svc.Create(ginCtx, &dtotenant.UserCreateReq{Name: "Bob", PrimaryEmail: "bob@x.com", PrimaryDepartmentID: "o1"})
	if err != nil {
		t.Fatalf("create user with new person: %v", err)
	}
	if respB.InitialPassword == "" {
		t.Fatalf("expected initial temporary password for newly created person")
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
	if bobPerson.PasswordEncrypted == "" || bobPerson.PasswordMethod != model.PasswordMethodBcrypt {
		t.Fatalf("expected bcrypt password on person")
	}
	if err := gcrypto.ComparePasswordHash(bobPerson.PasswordEncrypted, respB.InitialPassword); err != nil {
		t.Fatalf("initial password does not match stored hash: %v", err)
	}
	if !bobPerson.MustChangePassword {
		t.Fatalf("newly created person must be forced to change password on first login")
	}

	// 另一租户提供相同 email：find-or-create 命中已有 person 并关联（复用同一自然人）
	respC, err := svc.Create(newAdminCtx(t, db, "t2", "op2"), &dtotenant.UserCreateReq{Name: "Bob2", PrimaryEmail: "bob@x.com", PrimaryDepartmentID: "o2"})
	if err != nil {
		t.Fatalf("create user linking existing person: %v", err)
	}
	if respC.InitialPassword != "" {
		t.Fatalf("reusing an existing person must not return an initial password, got %q", respC.InitialPassword)
	}
	var bob2 model.UserEntity
	if err := db.First(&bob2, "id = ?", respC.UserID).Error; err != nil {
		t.Fatalf("query bob2: %v", err)
	}
	if bob2.PersonID != bob.PersonID {
		t.Fatalf("expected same person reused, got %s vs %s", bob2.PersonID, bob.PersonID)
	}

	// 同租户重复加入：拒绝
	if _, err := svc.Create(ginCtx, &dtotenant.UserCreateReq{Name: "Bob3", PrimaryEmail: "bob@x.com", PrimaryDepartmentID: "o1"}); err == nil {
		t.Fatalf("expected duplicate-in-tenant error")
	}

	// 指定 personID 直接关联；同一 person 在本租户已有 user 时拒绝
	personID := bob.PersonID
	if _, err := svc.Create(ginCtx, &dtotenant.UserCreateReq{Name: "Bob4", PersonID: personID, PrimaryEmail: "bob4@x.com", PrimaryDepartmentID: "o1"}); err == nil {
		t.Fatalf("expected error when linking person already in tenant")
	}

	// 仅有手机号（无邮箱）：同样创建自然人并返回临时密码
	respPwd, err := svc.Create(ginCtx, &dtotenant.UserCreateReq{Name: "NoID", PrimaryPhone: "13000000001", PrimaryDepartmentID: "o1"})
	if err != nil {
		t.Fatalf("create user with phone only: %v", err)
	}
	if respPwd.InitialPassword == "" {
		t.Fatalf("expected initial temporary password for newly created person")
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
	if err := gcrypto.ComparePasswordHash(noIDPerson.PasswordEncrypted, respPwd.InitialPassword); err != nil {
		t.Fatalf("initial password does not match stored hash: %v", err)
	}
}

// TestUserCreateRequiresDepartment 业务约束：创建用户必须从属于至少一个部门
// （departmentIDs 必传，缺失或为空一律拒绝；提供合法部门则正常创建）。
func TestUserCreateRequiresDepartment(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.UserEntity{}, &model.PersonEntity{}, &model.DepartmentEntity{}, &model.DepartmentUserEntity{}, &model.RoleEntity{}, &model.UserRoleEntity{})
	svc := &userSvc{}
	ginCtx := newAdminCtx(t, db, "t1", "op")

	if err := db.Create(&model.DepartmentEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "o1"}},
		TenantID:   "t1",
		DeptPath:   "/o1",
		DeptDepth:  1,
		Name:       "研发部",
		Status:     model.DeptNodeStatusEnable,
	}).Error; err != nil {
		t.Fatalf("seed dept: %v", err)
	}

	requiredErr := code.GetError(code.UserDepartmentRequiredError)
	for name, req := range map[string]*dtotenant.UserCreateReq{
		"missing": {Name: "张三"},
		"empty":   {Name: "李四", PrimaryDepartmentID: ""},
	} {
		if _, err := svc.Create(ginCtx, req); !errors.Is(err, requiredErr) {
			t.Fatalf("case %s: expected UserDepartmentRequiredError, got %v", name, err)
		}
	}

	// 邮箱、手机号二选一：都为空拒绝
	contactErr := code.GetError(code.UserContactRequiredError)
	if _, err := svc.Create(ginCtx, &dtotenant.UserCreateReq{Name: "无联系方式", PrimaryDepartmentID: "o1"}); !errors.Is(err, contactErr) {
		t.Fatalf("expected UserContactRequiredError, got %v", err)
	}
	// 仅手机号可创建
	if _, err := svc.Create(ginCtx, &dtotenant.UserCreateReq{Name: "仅手机", PrimaryPhone: "15011112222", PrimaryDepartmentID: "o1"}); err != nil {
		t.Fatalf("create with phone only should succeed: %v", err)
	}
	// 仅邮箱可创建
	if _, err := svc.Create(ginCtx, &dtotenant.UserCreateReq{Name: "仅邮箱", PrimaryEmail: "only@x.com", PrimaryDepartmentID: "o1"}); err != nil {
		t.Fatalf("create with email only should succeed: %v", err)
	}

	// 提供合法部门：正常创建
	if _, err := svc.Create(ginCtx, &dtotenant.UserCreateReq{Name: "王五", PrimaryEmail: "ww@x.com", PrimaryDepartmentID: "o1"}); err != nil {
		t.Fatalf("create user with valid dept: %v", err)
	}
}

// TestUserCreateWithDepartments 创建用户时建立行政主部门(primary,至多1)与负责关系(leader,可多)。
func TestUserCreateWithDepartments(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.UserEntity{}, &model.PersonEntity{}, &model.DepartmentEntity{}, &model.DepartmentUserEntity{}, &model.RoleEntity{}, &model.UserRoleEntity{})
	svc := &userSvc{}

	now := time.Now()
	for i, deptID := range []string{"o1", "o2"} {
		if err := db.Create(&model.DepartmentEntity{
			BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: deptID}},
			TenantID:   "t1",
			DeptPath:   "/" + deptID,
			DeptDepth:  1,
			Name:       "部门" + deptID,
			Status:     model.DeptNodeStatusEnable,
		}).Error; err != nil {
			t.Fatalf("seed dept %d: %v", i, err)
		}
		_ = now
	}

	ginCtx := newAdminCtx(t, db, "t1", "op")
	// 创建用户：primary=o1（单个行政主部门）+ leader=o2
	resp, err := svc.Create(ginCtx, &dtotenant.UserCreateReq{Name: "张三", PrimaryEmail: "zs@x.com", PrimaryDepartmentID: "o1", LeaderDepartmentIDs: []string{"o2"}})
	if err != nil {
		t.Fatalf("create user with depts: %v", err)
	}
	var relations []model.DepartmentUserEntity
	if err := db.Where("tenant_id = ? AND user_id = ?", "t1", resp.UserID).Find(&relations).Error; err != nil {
		t.Fatalf("query relations: %v", err)
	}
	if len(relations) != 2 {
		t.Fatalf("expected 2 dept relations, got %d", len(relations))
	}
	byType := map[model.DeptUserRelationType]string{}
	for _, r := range relations {
		byType[r.RelationType] = r.DepartmentID
	}
	if byType[model.DeptUserRelationPrimary] != "o1" || byType[model.DeptUserRelationLeader] != "o2" {
		t.Fatalf("unexpected relations: %+v", relations)
	}

	// 主部门为单值入参：多主部门在类型层面已不可表达，改验"空主部门被拒"（防绕过 DTO 校验）
	if _, err := svc.Create(ginCtx, &dtotenant.UserCreateReq{Name: "李四", PrimaryEmail: "lis@x.com"}); !errors.Is(err, code.GetError(code.UserDepartmentRequiredError)) {
		t.Fatalf("expected primary department required, got %v", err)
	}

	// 非法部门（非本租户）拒绝
	if _, err := svc.Create(ginCtx, &dtotenant.UserCreateReq{Name: "王五", PrimaryPhone: "15000000002", PrimaryDepartmentID: "o-other"}); err == nil {
		t.Fatalf("expected invalid dept error")
	}
}

// TestUserPageListKeyword 关键词过滤：姓名 / person 的 username / email 均命中。
func TestUserPageListKeyword(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.UserEntity{}, &model.PersonEntity{}, &model.DepartmentEntity{},
		&model.DepartmentUserEntity{}, &model.UserRoleEntity{})
	svc := &userSvc{}

	p1 := seedTestPerson(t, db, "p1", "zhangsan", "")
	seedTestUserWithPerson(t, db, "u1", "t1", p1.ID, "张三")
	seedTestUserWithPerson(t, db, "u2", "t1", "", "李四")
	p3 := seedTestPerson(t, db, "p3", "", "wang@x.com")
	seedTestUserWithPerson(t, db, "u3", "t1", p3.ID, "王五")

	ginCtx := newDeptGinCtx(t, "t1", "op")

	// 按 person username 命中
	resp, err := svc.PageList(ginCtx, &dtotenant.UserPageListReq{Page: 1, PageSize: 10, Keyword: "zhangsan"})
	if err != nil {
		t.Fatalf("page list: %v", err)
	}
	if len(resp.List) != 1 || resp.List[0].UserID != "u1" {
		t.Fatalf("expected only u1 matched by username, got %+v", resp.List)
	}
	// 列表必须同时回传创建时间与更新时间（前端「创建时间」「更新时间」两列直读）
	if item := resp.List[0]; item.CreatedAt <= 0 || item.UpdatedAt <= 0 {
		t.Fatalf("createdAt/updatedAt not returned: %+v", item)
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

// TestUserDetailWithDepartmentsAndRoles 详情含部门归属与角色列表。
func TestUserDetailWithDepartmentsAndRoles(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.UserEntity{}, &model.PersonEntity{}, &model.DepartmentEntity{},
		&model.DepartmentUserEntity{}, &model.RoleEntity{}, &model.UserRoleEntity{},
		&model.ApplicationEntity{}, &model.TenantApplicationEntity{})
	svc := &userSvc{}

	now := time.Now()
	seedTestUserWithPerson(t, db, "u1", "t1", "", "张三")

	if err := db.Create(&model.DepartmentEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "o1"}},
		TenantID:   "t1",
		DeptPath:   "/o1",
		DeptDepth:  1,
		Name:       "研发部",
		Status:     model.DeptNodeStatusEnable,
	}).Error; err != nil {
		t.Fatalf("seed dept: %v", err)
	}
	if err := db.Create(&model.DepartmentUserEntity{
		BaseEntity:   gormdao.BaseEntity{StringID: gormdao.StringID{ID: "ou1"}},
		TenantID:     "t1",
		DepartmentID: "o1",
		UserID:       "u1",
		RelationType: model.DeptUserRelationPrimary,
	}).Error; err != nil {
		t.Fatalf("seed dept-user: %v", err)
	}
	if err := db.Create(&model.RoleEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "r1"}},
		TenantID:   "t1",
		Name:       "管理员",
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

	ginCtx := newAdminCtx(t, db, "t1", "op")
	resp, err := svc.Detail(ginCtx, &dtotenant.UserDetailReq{UserID: "u1"})
	if err != nil {
		t.Fatalf("detail: %v", err)
	}
	if len(resp.Departments) != 1 || resp.Departments[0].DepartmentName != "研发部" || resp.Departments[0].RelationType != model.DeptUserRelationPrimary {
		t.Fatalf("unexpected departments: %+v", resp.Departments)
	}
	if len(resp.Roles) != 1 || resp.Roles[0].Name != "管理员" {
		t.Fatalf("unexpected roles: %+v", resp.Roles)
	}
	_ = now
}

// TestUserUpdateRolesFullReplace 全量替换用户角色。
func TestUserUpdateRolesFullReplace(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.UserEntity{}, &model.RoleEntity{}, &model.UserRoleEntity{})
	svc := &userSvc{}
	seedTestUserWithPerson(t, db, "u1", "t1", "", "张三")

	for _, r := range []struct{ id, name string }{{"r1", "管理员"}, {"r2", "成员"}, {"r3", "访客"}} {
		if err := db.Create(&model.RoleEntity{
			BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: r.id}},
			TenantID:   "t1",
			Name:       r.name,
		}).Error; err != nil {
			t.Fatalf("seed role %s: %v", r.id, err)
		}
	}
	// 另一租户的角色
	if err := db.Create(&model.RoleEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "r-other"}},
		TenantID:   "t2",
		Name:       "other",
	}).Error; err != nil {
		t.Fatalf("seed other role: %v", err)
	}

	ginCtx := newAdminCtx(t, db, "t1", "op")
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

// TestUserUpdateRolesScopedByApp 按应用粒度替换：仅替换目标应用下的角色，其它应用不受影响；
// 清空仅清空该应用；跨应用/跨租户授予被拒。
func TestUserUpdateRolesScopedByApp(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.UserEntity{}, &model.RoleEntity{}, &model.UserRoleEntity{})
	svc := &userSvc{}
	seedTestUserWithPerson(t, db, "u1", "t1", "", "张三")

	seedRole := func(id, tenantID, appID, name string) {
		t.Helper()
		if err := db.Create(&model.RoleEntity{
			BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: id}},
			TenantID:   tenantID,
			AppID:      appID,
			Name:       name,
		}).Error; err != nil {
			t.Fatalf("seed role %s: %v", id, err)
		}
	}
	seedRole("ra1", "t1", "app-a", "角色A1")
	seedRole("ra2", "t1", "app-a", "角色A2")
	seedRole("rb1", "t1", "app-b", "角色B1")
	seedRole("rc1", "t2", "app-c", "角色C1")

	seedUserRole := func(id, userID, roleID string) {
		t.Helper()
		if err := db.Create(&model.UserRoleEntity{
			BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: id}},
			TenantID:   "t1",
			UserID:     userID,
			RoleID:     roleID,
			CreatedBy:  "t",
		}).Error; err != nil {
			t.Fatalf("seed user_role %s: %v", id, err)
		}
	}
	seedUserRole("ur1", "u1", "ra1")
	seedUserRole("ur2", "u1", "rb1")

	queryRoleIDs := func() []string {
		var urList []model.UserRoleEntity
		if err := db.Where("tenant_id = ? AND user_id = ?", "t1", "u1").Find(&urList).Error; err != nil {
			t.Fatalf("query user_role: %v", err)
		}
		ids := make([]string, 0, len(urList))
		for _, ur := range urList {
			ids = append(ids, ur.RoleID)
		}
		return ids
	}
	hasRole := func(ids []string, id string) bool {
		for _, v := range ids {
			if v == id {
				return true
			}
		}
		return false
	}

	ginCtx := newAdminCtx(t, db, "t1", "op")

	// 仅替换 app-a：ra1 被移除、ra2 加入，app-b 的 rb1 必须保留
	if err := svc.UpdateRoles(ginCtx, &dtotenant.UserRolesUpdateReq{UserID: "u1", AppID: "app-a", RoleIDs: []string{"ra2"}}); err != nil {
		t.Fatalf("replace app-a roles: %v", err)
	}
	ids := queryRoleIDs()
	if len(ids) != 2 || !hasRole(ids, "ra2") || !hasRole(ids, "rb1") || hasRole(ids, "ra1") {
		t.Fatalf("expected [ra2 rb1] after app-a replace, got %+v", ids)
	}

	// 清空 app-b：仅移除 rb1，app-a 的 ra2 保留
	if err := svc.UpdateRoles(ginCtx, &dtotenant.UserRolesUpdateReq{UserID: "u1", AppID: "app-b", RoleIDs: []string{}}); err != nil {
		t.Fatalf("clear app-b roles: %v", err)
	}
	ids = queryRoleIDs()
	if len(ids) != 1 || ids[0] != "ra2" {
		t.Fatalf("expected only ra2 after clearing app-b, got %+v", ids)
	}

	// 跨应用授予：app-a 下塞 app-b 的角色 → 拒绝
	if err := svc.UpdateRoles(ginCtx, &dtotenant.UserRolesUpdateReq{UserID: "u1", AppID: "app-a", RoleIDs: []string{"rb1"}}); err != code.GetError(code.RoleNotExistError) {
		t.Fatalf("cross-app role grant: want RoleNotExist, got %v", err)
	}
	// 跨租户角色（即使 AppID 相同）→ 拒绝
	if err := svc.UpdateRoles(ginCtx, &dtotenant.UserRolesUpdateReq{UserID: "u1", AppID: "app-c", RoleIDs: []string{"rc1"}}); err != code.GetError(code.RoleNotExistError) {
		t.Fatalf("cross-tenant role grant: want RoleNotExist, got %v", err)
	}
}

// TestUserCreateWithLeaderDepts 创建用户同时建立行政主部门(primary)、参与部门(secondary)与负责部门(leader)。
func TestUserCreateWithLeaderDepts(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.UserEntity{}, &model.PersonEntity{}, &model.DepartmentEntity{}, &model.DepartmentUserEntity{}, &model.RoleEntity{}, &model.UserRoleEntity{})
	svc := &userSvc{}

	for _, deptID := range []string{"o1", "o2", "o3"} {
		if err := db.Create(&model.DepartmentEntity{
			BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: deptID}},
			TenantID:   "t1",
			DeptPath:   "/" + deptID,
			DeptDepth:  1,
			Name:       "部门" + deptID,
			Status:     model.DeptNodeStatusEnable,
		}).Error; err != nil {
			t.Fatalf("seed dept %s: %v", deptID, err)
		}
	}

	ginCtx := newAdminCtx(t, db, "t1", "op")
	resp, err := svc.Create(ginCtx, &dtotenant.UserCreateReq{
		Name:                   "张三",
		PrimaryEmail:           "zs2@x.com",
		PrimaryDepartmentID:    "o1",
		SecondaryDepartmentIDs: []string{"o2"},
		LeaderDepartmentIDs:    []string{"o3"},
	})
	if err != nil {
		t.Fatalf("create user with leader depts: %v", err)
	}
	var relations []model.DepartmentUserEntity
	if err := db.Where("tenant_id = ? AND user_id = ?", "t1", resp.UserID).Find(&relations).Error; err != nil {
		t.Fatalf("query relations: %v", err)
	}
	if len(relations) != 3 {
		t.Fatalf("expected 3 relations (1 primary + 1 secondary + 1 leader), got %d", len(relations))
	}
	var primaryCount, secondaryCount, leaderCount int
	for _, r := range relations {
		switch r.RelationType {
		case model.DeptUserRelationPrimary:
			primaryCount++
		case model.DeptUserRelationSecondary:
			secondaryCount++
		case model.DeptUserRelationLeader:
			leaderCount++
		}
	}
	if primaryCount != 1 || secondaryCount != 1 || leaderCount != 1 {
		t.Fatalf("unexpected relation distribution, primary:%d secondary:%d leaders:%d", primaryCount, secondaryCount, leaderCount)
	}

	// 一个部门至多一个负责人：新用户再负责 o3 应拒绝
	if _, err := svc.Create(ginCtx, &dtotenant.UserCreateReq{Name: "李四", PrimaryPhone: "15000000003", PrimaryDepartmentID: "o1", LeaderDepartmentIDs: []string{"o3"}}); err == nil {
		t.Fatalf("expected leader conflict to be rejected")
	}

	// 负责部门非法（非本租户）拒绝
	if _, err := svc.Create(ginCtx, &dtotenant.UserCreateReq{Name: "王五", PrimaryEmail: "ww5@x.com", PrimaryDepartmentID: "o1", LeaderDepartmentIDs: []string{"o-other"}}); err == nil {
		t.Fatalf("expected invalid leader dept error")
	}
}

// TestUserUpdateDepartments 编辑用户时更新主/参与/负责部门：
// primary 替换、secondary/leader 全量替换、leader 冲突拒绝。
func TestUserUpdateDepartments(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.UserEntity{}, &model.PersonEntity{}, &model.DepartmentEntity{}, &model.DepartmentUserEntity{}, &model.RoleEntity{}, &model.UserRoleEntity{})
	svc := &userSvc{}

	for _, deptID := range []string{"o1", "o2", "o3"} {
		if err := db.Create(&model.DepartmentEntity{
			BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: deptID}},
			TenantID:   "t1",
			DeptPath:   "/" + deptID,
			DeptDepth:  1,
			Name:       "部门" + deptID,
			Status:     model.DeptNodeStatusEnable,
		}).Error; err != nil {
			t.Fatalf("seed dept %s: %v", deptID, err)
		}
	}

	ginCtx := newAdminCtx(t, db, "t1", "op")
	u1, err := svc.Create(ginCtx, &dtotenant.UserCreateReq{
		Name:                   "张三",
		PrimaryEmail:           "u1@x.com",
		PrimaryDepartmentID:    "o1",
		SecondaryDepartmentIDs: []string{"o2"},
		LeaderDepartmentIDs:    []string{"o3"},
	})
	if err != nil {
		t.Fatalf("create u1: %v", err)
	}

	countByType := func(userID string) map[model.DeptUserRelationType]string {
		var rows []model.DepartmentUserEntity
		if err := db.Where("tenant_id = ? AND user_id = ?", "t1", userID).Find(&rows).Error; err != nil {
			t.Fatalf("query relations: %v", err)
		}
		m := map[model.DeptUserRelationType]string{}
		for _, r := range rows {
			m[r.RelationType] = r.DepartmentID
		}
		return m
	}

	// 替换主部门 o1 -> o2
	prim := "o2"
	if err := svc.Update(ginCtx, &dtotenant.UserUpdateReq{UserID: u1.UserID, PrimaryDepartmentID: &prim}); err != nil {
		t.Fatalf("update primary: %v", err)
	}
	m := countByType(u1.UserID)
	if m[model.DeptUserRelationPrimary] != "o2" {
		t.Fatalf("expected primary o2, got %+v", m)
	}

	// 全量替换参与部门 o2 -> [o3, o1]
	sec := []string{"o3", "o1"}
	if err := svc.Update(ginCtx, &dtotenant.UserUpdateReq{UserID: u1.UserID, SecondaryDepartmentIDs: &sec}); err != nil {
		t.Fatalf("update secondary: %v", err)
	}
	var secCount int64
	if err := db.Model(&model.DepartmentUserEntity{}).
		Where("tenant_id = ? AND user_id = ? AND relation_type = ?", "t1", u1.UserID, model.DeptUserRelationSecondary).
		Count(&secCount).Error; err != nil {
		t.Fatalf("count secondary: %v", err)
	}
	if secCount != 2 {
		t.Fatalf("expected 2 secondary relations, got %d", secCount)
	}

	// 全量替换负责部门 o3 -> [o1]
	lead := []string{"o1"}
	if err := svc.Update(ginCtx, &dtotenant.UserUpdateReq{UserID: u1.UserID, LeaderDepartmentIDs: &lead}); err != nil {
		t.Fatalf("update leader: %v", err)
	}
	m = countByType(u1.UserID)
	if m[model.DeptUserRelationLeader] != "o1" {
		t.Fatalf("expected leader o1, got %+v", m)
	}

	// leader 冲突：另建用户 u2 负责 o1（已被 u1 负责）应拒绝
	u2, err := svc.Create(ginCtx, &dtotenant.UserCreateReq{Name: "李四", PrimaryPhone: "15000000004", PrimaryDepartmentID: "o1"})
	if err != nil {
		t.Fatalf("create u2: %v", err)
	}
	conflictLead := []string{"o1"}
	if err := svc.Update(ginCtx, &dtotenant.UserUpdateReq{UserID: u2.UserID, LeaderDepartmentIDs: &conflictLead}); err == nil {
		t.Fatalf("expected leader conflict rejected")
	}
}

// TestUserUpdateContact 编辑成员联系方式：更新 person 的邮箱/手机号/用户名，
// 校验邮箱与手机号二选一、以及全局唯一性冲突。
func TestUserUpdateContact(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.UserEntity{}, &model.PersonEntity{}, &model.DepartmentEntity{}, &model.DepartmentUserEntity{}, &model.RoleEntity{}, &model.UserRoleEntity{})
	svc := &userSvc{}

	if err := db.Create(&model.DepartmentEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "o1"}},
		TenantID:   "t1",
		DeptPath:   "/o1",
		DeptDepth:  1,
		Name:       "研发部",
		Status:     model.DeptNodeStatusEnable,
	}).Error; err != nil {
		t.Fatalf("seed dept: %v", err)
	}

	ginCtx := newAdminCtx(t, db, "t1", "op")
	u1, err := svc.Create(ginCtx, &dtotenant.UserCreateReq{Name: "张三", PrimaryEmail: "zs@x.com", PrimaryDepartmentID: "o1"})
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
	u2, err := svc.Create(ginCtx, &dtotenant.UserCreateReq{Name: "李四", PrimaryPhone: "15000000002", PrimaryDepartmentID: "o1"})
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

// TestUserPageListDepartmentFilter 用户目录：支持按"恰在该部门"过滤（primary/secondary/leader 任一关系命中，不含子部门）、
// 关键词命中租户内姓名；跨租户用户不可见；主部门名聚合来自 primary 行政主部门关系。
func TestUserPageListDepartmentFilter(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.UserEntity{}, &model.PersonEntity{}, &model.DepartmentEntity{},
		&model.DepartmentUserEntity{}, &model.UserRoleEntity{})
	svc := &userSvc{}

	// 部门
	for _, deptID := range []string{"o1", "o2"} {
		if err := db.Create(&model.DepartmentEntity{
			BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: deptID}},
			TenantID:   "t1",
			DeptPath:   "/" + deptID,
			DeptDepth:  1,
			Name:       "部门" + deptID,
			Status:     model.DeptNodeStatusEnable,
		}).Error; err != nil {
			t.Fatalf("seed dept %s: %v", deptID, err)
		}
	}

	// 用户：u1 行政归属 o1；u2 负责 o2
	seedTestUserWithPerson(t, db, "u1", "t1", "", "张三")
	seedTestUserWithPerson(t, db, "u2", "t1", "", "李四")
	// 其他租户用户（不应出现）
	seedTestUserWithPerson(t, db, "u-other", "t2", "", "外人")

	if err := db.Create(&model.DepartmentUserEntity{
		BaseEntity:   gormdao.BaseEntity{StringID: gormdao.StringID{ID: "ou1"}},
		TenantID:     "t1",
		DepartmentID: "o1",
		UserID:       "u1",
		RelationType: model.DeptUserRelationPrimary,
	}).Error; err != nil {
		t.Fatalf("seed rel ou1: %v", err)
	}
	if err := db.Create(&model.DepartmentUserEntity{
		BaseEntity:   gormdao.BaseEntity{StringID: gormdao.StringID{ID: "ou2"}},
		TenantID:     "t1",
		DepartmentID: "o2",
		UserID:       "u2",
		RelationType: model.DeptUserRelationLeader,
	}).Error; err != nil {
		t.Fatalf("seed rel ou2: %v", err)
	}

	ginCtx := newDeptGinCtx(t, "t1", "op")

	// 全量（不含其他租户）：主部门名仅聚合 primary 关系
	resp, err := svc.PageList(ginCtx, &dtotenant.UserPageListReq{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("user page list: %v", err)
	}
	if resp.Total != 2 {
		t.Fatalf("expected total 2, got %d", resp.Total)
	}
	var u1Item, u2Item *dtotenant.UserPageListItem
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
	if u1Item.PrimaryDepartmentName != "部门o1" {
		t.Fatalf("expected u1 primary dept name 部门o1, got %s", u1Item.PrimaryDepartmentName)
	}
	if u2Item.PrimaryDepartmentName != "" {
		t.Fatalf("expected u2 empty primary dept name (leader only), got %s", u2Item.PrimaryDepartmentName)
	}

	// 按部门过滤：o1 命中 u1；o2 命中 u2（leader 也算该部门用户）
	resp, err = svc.PageList(ginCtx, &dtotenant.UserPageListReq{Page: 1, PageSize: 10, DepartmentID: "o1"})
	if err != nil {
		t.Fatalf("user page list by dept: %v", err)
	}
	if resp.Total != 1 || resp.List[0].UserID != "u1" {
		t.Fatalf("expected only u1 for o1, got %+v", resp.List)
	}

	resp, err = svc.PageList(ginCtx, &dtotenant.UserPageListReq{Page: 1, PageSize: 10, DepartmentID: "o2"})
	if err != nil {
		t.Fatalf("user page list by dept: %v", err)
	}
	if resp.Total != 1 || resp.List[0].UserID != "u2" {
		t.Fatalf("expected only u2 for o2, got %+v", resp.List)
	}

	// 关键词命中租户内姓名
	resp, err = svc.PageList(ginCtx, &dtotenant.UserPageListReq{Page: 1, PageSize: 10, Keyword: "张三"})
	if err != nil {
		t.Fatalf("user page list by keyword: %v", err)
	}
	if resp.Total != 1 || resp.List[0].UserID != "u1" {
		t.Fatalf("expected u1 by keyword, got %+v", resp.List)
	}

	// 无匹配部门返回空
	resp, err = svc.PageList(ginCtx, &dtotenant.UserPageListReq{Page: 1, PageSize: 10, DepartmentID: "o-none"})
	if err != nil {
		t.Fatalf("user page list by missing dept: %v", err)
	}
	if resp.Total != 0 || len(resp.List) != 0 {
		t.Fatalf("expected empty list for missing dept, got %+v", resp.List)
	}
}

// TestUserResetPasswordIssuesTemporaryPassword 重置成员密码（D7）：
// 服务端生成新临时密码并仅返回一次、落库哈希与之匹配、强制下次登录改密、旧密码立即失效。
func TestUserResetPasswordIssuesTemporaryPassword(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.UserEntity{}, &model.PersonEntity{}, &model.DepartmentEntity{},
		&model.DepartmentUserEntity{}, &model.RoleEntity{}, &model.UserRoleEntity{})
	svc := &userSvc{}

	if err := db.Create(&model.DepartmentEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "o1"}},
		TenantID:   "t1", DeptPath: "/o1", DeptDepth: 1, Name: "部门o1", Status: model.DeptNodeStatusEnable,
	}).Error; err != nil {
		t.Fatalf("seed dept: %v", err)
	}
	ginCtx := newAdminCtx(t, db, "t1", "op1")

	created, err := svc.Create(ginCtx, &dtotenant.UserCreateReq{Name: "张三", PrimaryEmail: "zs@x.com", PrimaryDepartmentID: "o1"})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	var createdUser model.UserEntity
	if err := db.First(&createdUser, "id = ?", created.UserID).Error; err != nil {
		t.Fatalf("query user: %v", err)
	}
	var createdPerson model.PersonEntity
	if err := db.First(&createdPerson, "id = ?", createdUser.PersonID).Error; err != nil {
		t.Fatalf("query person: %v", err)
	}
	oldHash := createdPerson.PasswordEncrypted

	reset, err := svc.ResetPassword(ginCtx, &dtotenant.UserResetPasswordReq{UserID: created.UserID})
	if err != nil {
		t.Fatalf("reset password: %v", err)
	}
	if reset.UserID != created.UserID {
		t.Fatalf("reset userID = %q, want %q", reset.UserID, created.UserID)
	}
	if reset.InitialPassword == "" {
		t.Fatalf("expected temporary password in reset response")
	}
	if reset.InitialPassword == created.InitialPassword {
		t.Fatalf("reset must issue a new password, not reuse the initial one")
	}

	var personAfterReset model.PersonEntity
	if err := db.First(&personAfterReset, "id = ?", createdUser.PersonID).Error; err != nil {
		t.Fatalf("query person after reset: %v", err)
	}
	if err := gcrypto.ComparePasswordHash(personAfterReset.PasswordEncrypted, reset.InitialPassword); err != nil {
		t.Fatalf("reset password does not match stored hash: %v", err)
	}
	if personAfterReset.PasswordEncrypted == oldHash {
		t.Fatalf("password hash must change after reset")
	}
	if err := gcrypto.ComparePasswordHash(personAfterReset.PasswordEncrypted, created.InitialPassword); err == nil {
		t.Fatalf("old temporary password must no longer be valid after reset")
	}
	if !personAfterReset.MustChangePassword {
		t.Fatalf("reset must set must_change_password=true")
	}
}

// TestUserResetPasswordRejectsMachineUser 服务账号无口令语义：即使 userID 存在也必须拒绝，
// 且绝不动其关联 person（若有）的密码。
func TestUserResetPasswordRejectsMachineUser(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.UserEntity{}, &model.PersonEntity{}, &model.DepartmentEntity{},
		&model.DepartmentUserEntity{}, &model.RoleEntity{}, &model.UserRoleEntity{})
	svc := &userSvc{}
	ginCtx := newAdminCtx(t, db, "t1", "op1")

	machinePerson := seedTestPerson(t, db, "pm1", "sa", "sa@x.com")
	if err := db.Model(&model.PersonEntity{}).Where("id = ?", machinePerson.ID).
		Update("password_encrypted", "machine-hash").Error; err != nil {
		t.Fatalf("seed machine person password: %v", err)
	}
	seedTestUserWithPerson(t, db, "um1", "t1", machinePerson.ID, "服务账号")
	if err := db.Model(&model.UserEntity{}).Where("id = ?", "um1").
		Update("user_type", model.UserTypeMachine).Error; err != nil {
		t.Fatalf("mark machine user: %v", err)
	}

	_, err := svc.ResetPassword(ginCtx, &dtotenant.UserResetPasswordReq{UserID: "um1"})
	if err == nil || err.Error() != code.GetError(code.UserNotExistError).Error() {
		t.Fatalf("ResetPassword err = %v, want %v", err, code.GetError(code.UserNotExistError))
	}
	var stored model.PersonEntity
	if err := db.First(&stored, "id = ?", machinePerson.ID).Error; err != nil {
		t.Fatalf("query machine person: %v", err)
	}
	if stored.PasswordEncrypted != "machine-hash" {
		t.Fatalf("machine user password must not be touched, got %q", stored.PasswordEncrypted)
	}
}
