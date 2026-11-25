package svctenant

import (
	"github.com/morehao/ark-iam/pkg/dbclient"
	"testing"
	"time"

	"github.com/morehao/ark-iam/apps/tenantadmin/dto/dtotenant"
	"github.com/morehao/ark-iam/apps/tenantadmin/testutil"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

// setupUserSubResourceTestDB 建库并播种两个租户：t1 拥有 person p1 / user user1、p3 / user3，
// t2 拥有 p2 / user2。身份与登录日志均归属自然人，跨租户与跨用户用例需要同租户的第二个用户。
func setupUserSubResourceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := testutil.SetupSQLite(t,
		&model.UserEntity{},
		&model.PersonEntity{},
		&model.UserIdentityEntity{},
		&model.UserLoginLogEntity{},
		&model.RoleEntity{},
		&model.UserRoleEntity{},
	)
	seedTestPerson(t, db, "p1", "u1", "u1@example.com")
	seedTestPerson(t, db, "p2", "u2", "u2@example.com")
	seedTestPerson(t, db, "p3", "u3", "u3@example.com")
	seedTestUserWithPerson(t, db, "user1", "t1", "p1", "张三")
	seedTestUserWithPerson(t, db, "user2", "t2", "p2", "李四")
	seedTestUserWithPerson(t, db, "user3", "t1", "p3", "王五")
	// 身份绑定/解绑是管理写操作（Create/Delete 走 requireSystemAdmin）：给测试操作者 op1
	// 播种一个「自定义管理员」角色，使其具备系统管理能力但不算内置系统角色持有者。
	seedTestAdminOperator(t, db, "t1", "op1")
	return db
}

// seedTestAdminOperator 为操作用户绑定管理员角色（自定义来源），使其通过 requireSystemAdmin。
func seedTestAdminOperator(t *testing.T, db *gorm.DB, tenantID, userID string) {
	t.Helper()
	roleID := "test-admin-role-" + tenantID + "-" + userID
	if err := db.Create(&model.RoleEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: roleID}},
		TenantID:   tenantID,
		AppID:      "app-admin",
		Name:       "测试管理员角色",
		Source:     model.RoleSourceCustom,
		AdminType:  model.SysAdminTypeAdmin,
		CreatedBy:  "seed",
	}).Error; err != nil {
		t.Fatalf("seed admin role: %v", err)
	}
	if err := db.Create(&model.UserRoleEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "test-ur-" + roleID}},
		TenantID:   tenantID,
		UserID:     userID,
		RoleID:     roleID,
		CreatedBy:  "seed",
	}).Error; err != nil {
		t.Fatalf("seed user_role: %v", err)
	}
}

func seedTestIdentity(t *testing.T, db *gorm.DB, id, personID, issuer, subject string) {
	t.Helper()
	if err := db.Create(&model.UserIdentityEntity{
		BaseEntity:      gormdao.BaseEntity{StringID: gormdao.StringID{ID: id}},
		PersonID:        personID,
		Issuer:          issuer,
		ExternalSubject: subject,
		Detail:          model.UserIdentityDetail{Email: "seed@example.com"},
	}).Error; err != nil {
		t.Fatalf("seed identity %s: %v", id, err)
	}
}

func seedTestLoginLog(t *testing.T, db *gorm.DB, personID, tenantID, userID string, loginTime time.Time) {
	t.Helper()
	if err := db.Create(&model.UserLoginLogEntity{
		PersonID:  personID,
		TenantID:  tenantID,
		UserID:    userID,
		LoginType: model.LoginTypePassword,
		LoginIP:   "10.0.0.1",
		UserAgent: "test-agent",
		LoginTime: loginTime,
	}).Error; err != nil {
		t.Fatalf("seed login log: %v", err)
	}
}

// TestUserIdentityCreateBindsPersonOfUser 校验身份绑定以「用户在租户内的自然人」为准：
// 提交里的 userID 是路径参数，落库 person_id 必须取自该用户，而不是请求体里的任何值。
func TestUserIdentityCreateBindsPersonOfUser(t *testing.T) {
	db := setupUserSubResourceTestDB(t)
	svc := NewUserIdentitySvc(nil)

	opCtx := newTestTenantCtx("t1", "op1")
	resp, err := svc.Create(requestCtx(opCtx), &dtotenant.UserIdentityCreateReq{
		UserID:     "user1",
		Issuer:     "https://accounts.example.com",
		IdentityID: "sub-1",
		Detail:     model.UserIdentityDetail{Email: "u1@example.com"},
		Operator:   operatorOf(opCtx),
	})
	if err != nil {
		t.Fatalf("create identity: %v", err)
	}

	var got model.UserIdentityEntity
	if err := db.First(&got, "id = ?", resp.UserIdentityID).Error; err != nil {
		t.Fatalf("load identity: %v", err)
	}
	if got.PersonID != "p1" {
		t.Fatalf("expected person p1 from user, got %q", got.PersonID)
	}
	if got.Issuer != "https://accounts.example.com" || got.ExternalSubject != "sub-1" {
		t.Fatalf("unexpected identity fields: %+v", got)
	}

	list, err := svc.ListByUser(requestCtx(newTestTenantCtx("t1", "op1")), &dtotenant.UserIdentityListReq{UserID: "user1"})
	if err != nil {
		t.Fatalf("list identity: %v", err)
	}
	if list.Total != 1 || len(list.List) != 1 {
		t.Fatalf("expected 1 identity, got %+v", list)
	}
	if list.List[0].IdentityID != "sub-1" || list.List[0].Detail.Email != "u1@example.com" {
		t.Fatalf("expected identity subject and parsed detail, got %+v", list.List[0])
	}
}

// TestUserIdentityRejectsCrossTenantUser 校验身份子资源按租户隔离：
// 用 t1 的登录上下文访问 t2 的用户一律按「用户不存在」拒绝，不泄露他租户数据。
func TestUserIdentityRejectsCrossTenantUser(t *testing.T) {
	db := setupUserSubResourceTestDB(t)
	svc := NewUserIdentitySvc(nil)
	seedTestIdentity(t, db, "id-other", "p2", "https://accounts.example.com", "sub-2")

	ctx := newTestTenantCtx("t1", "op1")

	if _, err := svc.ListByUser(requestCtx(ctx), &dtotenant.UserIdentityListReq{UserID: "user2"}); err != code.GetError(code.UserNotExistError) {
		t.Fatalf("cross-tenant list: want user not exist, got %v", err)
	}
	if _, err := svc.Create(requestCtx(ctx), &dtotenant.UserIdentityCreateReq{
		UserID: "user2", Issuer: "https://accounts.example.com", IdentityID: "sub-3",

		Operator: operatorOf(ctx)}); err != code.GetError(code.UserNotExistError) {
		t.Fatalf("cross-tenant create: want user not exist, got %v", err)
	}
	if err := svc.Delete(requestCtx(ctx), &dtotenant.UserIdentityDeleteReq{UserID: "user2", UserIdentityID: "id-other",
		Operator: operatorOf(ctx),
	}); err != code.GetError(code.UserNotExistError) {
		t.Fatalf("cross-tenant delete: want user not exist, got %v", err)
	}

	var count int64
	if err := db.Model(&model.UserIdentityEntity{}).Where("id = ?", "id-other").Count(&count).Error; err != nil {
		t.Fatalf("count identity: %v", err)
	}
	if count != 1 {
		t.Fatalf("cross-tenant delete must not remove identity, count=%d", count)
	}
}

// TestUserIdentityDeleteRejectsMismatchedUser 校验解绑必须「用户 ↔ 身份」对应：
// 路径上的 userID 与本租户其他用户互换也不能解绑（否则同租户内任意管理员可越权解绑他人身份）。
func TestUserIdentityDeleteRejectsMismatchedUser(t *testing.T) {
	db := setupUserSubResourceTestDB(t)
	svc := NewUserIdentitySvc(nil)
	seedTestIdentity(t, db, "id-1", "p1", "https://accounts.example.com", "sub-1")

	// user1 的身份，用同租户 user3 的路径解绑 → 拒绝
	opCtx := newTestTenantCtx("t1", "op1")
	if err := svc.Delete(requestCtx(opCtx), &dtotenant.UserIdentityDeleteReq{
		UserID: "user3", UserIdentityID: "id-1",
		Operator: operatorOf(opCtx),
	}); err != code.GetError(code.UserIdentityNotExistError) {
		t.Fatalf("mismatched user delete: want identity not exist, got %v", err)
	}

	var count int64
	if err := db.Model(&model.UserIdentityEntity{}).Where("id = ?", "id-1").Count(&count).Error; err != nil {
		t.Fatalf("count identity: %v", err)
	}
	if count != 1 {
		t.Fatalf("mismatched delete must not remove identity, count=%d", count)
	}
}

// TestUserIdentityDeleteInTenant 校验本租户内解绑成功。
func TestUserIdentityDeleteInTenant(t *testing.T) {
	db := setupUserSubResourceTestDB(t)
	svc := NewUserIdentitySvc(nil)
	seedTestIdentity(t, db, "id-1", "p1", "https://accounts.example.com", "sub-1")

	opCtx := newTestTenantCtx("t1", "op1")
	if err := svc.Delete(requestCtx(opCtx), &dtotenant.UserIdentityDeleteReq{
		UserID: "user1", UserIdentityID: "id-1",
		Operator: operatorOf(opCtx),
	}); err != nil {
		t.Fatalf("delete identity: %v", err)
	}

	list, err := svc.ListByUser(requestCtx(newTestTenantCtx("t1", "op1")), &dtotenant.UserIdentityListReq{UserID: "user1"})
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if list.Total != 0 {
		t.Fatalf("expected identity removed, got %+v", list)
	}

	if err := svc.Delete(requestCtx(opCtx), &dtotenant.UserIdentityDeleteReq{
		UserID: "user1", UserIdentityID: "id-1",
		Operator: operatorOf(opCtx),
	}); err != code.GetError(code.UserIdentityNotExistError) {
		t.Fatalf("delete missing identity: want not exist, got %v", err)
	}
}

// TestUserLoginLogsScopeToTenantAndUser 校验登录日志以「租户 + 用户」双重条件取数。
func TestUserLoginLogsScopeToTenantAndUser(t *testing.T) {
	db := setupUserSubResourceTestDB(t)
	svc := &userSvc{db: dbclient.IamDB}

	now := time.Now()
	seedTestLoginLog(t, db, "p1", "t1", "user1", now.Add(-2*time.Hour))
	seedTestLoginLog(t, db, "p1", "t1", "user1", now.Add(-1*time.Hour))
	seedTestLoginLog(t, db, "p1", "t1", "user1", now)
	seedTestLoginLog(t, db, "p2", "t2", "user2", now)

	resp, err := svc.ListLoginLogs(requestCtx(newTestTenantCtx("t1", "op1")), &dtotenant.UserLoginLogListReq{UserID: "user1"})
	if err != nil {
		t.Fatalf("list login logs: %v", err)
	}
	if resp.Total != 3 || len(resp.List) != 3 {
		t.Fatalf("expected 3 tenant-scoped logs, got %+v", resp)
	}
	for _, item := range resp.List {
		if item.UserLoginLogID == "" || item.LoginTime == 0 {
			t.Fatalf("expected id and unix login time, got %+v", item)
		}
	}

	if _, err := svc.ListLoginLogs(requestCtx(newTestTenantCtx("t1", "op1")), &dtotenant.UserLoginLogListReq{UserID: "user2"}); err != code.GetError(code.UserNotExistError) {
		t.Fatalf("cross-tenant list login logs: want user not exist, got %v", err)
	}
}

// TestUserIdentityWriteRequiresSystemAdmin 校验身份绑定/解绑是管理写操作：
// 不具备系统管理能力(admin_type=normal)的普通成员调用必须被拒（UserSystemAdminRequiredError），
// 否则可把外部身份绑到他人账号实施接管。回归 P0-1。
func TestUserIdentityWriteRequiresSystemAdmin(t *testing.T) {
	db := setupUserSubResourceTestDB(t)
	svc := NewUserIdentitySvc(nil)
	seedTestIdentity(t, db, "id-1", "p1", "https://accounts.example.com", "sub-1")

	// op-normal 是 t1 内无管理员角色的普通成员。
	normalCtx := newTestTenantCtx("t1", "op-normal")

	if _, err := svc.Create(requestCtx(normalCtx), &dtotenant.UserIdentityCreateReq{
		UserID: "user1", Issuer: "https://evil.example.com", IdentityID: "attacker-sub",
		Operator: operatorOf(normalCtx),
	}); err != code.GetError(code.UserSystemAdminRequiredError) {
		t.Fatalf("non-admin create: want UserSystemAdminRequiredError, got %v", err)
	}

	if err := svc.Delete(requestCtx(normalCtx), &dtotenant.UserIdentityDeleteReq{
		UserID: "user1", UserIdentityID: "id-1",
		Operator: operatorOf(normalCtx),
	}); err != code.GetError(code.UserSystemAdminRequiredError) {
		t.Fatalf("non-admin delete: want UserSystemAdminRequiredError, got %v", err)
	}

	// 确认非管理员的两步操作都未落库改动：原身份仍在。
	var count int64
	if err := db.Model(&model.UserIdentityEntity{}).Where("id = ?", "id-1").Count(&count).Error; err != nil {
		t.Fatalf("count identity: %v", err)
	}
	if count != 1 {
		t.Fatalf("non-admin write must not touch identity, count=%d", count)
	}
}
