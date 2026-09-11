package svctenant

import (
	"regexp"
	"testing"

	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/iam/password"
	"github.com/morehao/ark-iam/pkg/iam/tenant"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/platformadmin/testutil"
	"github.com/morehao/golib/gcrypto"
	"gorm.io/gorm"
)

// setupTenantCreateEnv 建租户链路的完整测试环境：租户/组织 + 管理员（person/user/组织关系）
// + 权限开通所需的全部表，并预置 tenant-admin 应用与其菜单（真实环境由 pkg/seed 写入）。
func setupTenantCreateEnv(t *testing.T) *gorm.DB {
	t.Helper()
	db := testutil.SetupSQLite(t,
		&model.TenantEntity{},
		&model.OrganizationEntity{},
		&model.PersonEntity{},
		&model.UserEntity{},
		&model.OrganizationUserEntity{},
		&model.ApplicationEntity{},
		&model.MenuEntity{},
		&model.RoleEntity{},
		&model.RoleMenuEntity{},
		&model.TenantApplicationEntity{},
		&model.UserRoleEntity{},
	)
	seedTenantAdminApp(t, db)
	return db
}

// seedTenantAdminApp 预置租户自服务应用与其内置菜单（编码取自 pkg/iam/tenant 的单一事实源）。
func seedTenantAdminApp(t *testing.T, db *gorm.DB) *model.ApplicationEntity {
	t.Helper()
	app := &model.ApplicationEntity{Code: tenant.ProvisionAppCode, Name: "租户自服务", Status: model.AppStatusEnable}
	if err := db.Create(app).Error; err != nil {
		t.Fatalf("seed application: %v", err)
	}
	for i, menuCode := range tenant.ProvisionMenuCodes {
		menu := &model.MenuEntity{
			AppID:      app.ID,
			Name:       menuCode,
			Code:       menuCode,
			Path:       "/" + menuCode,
			Sort:       i + 1,
			Type:       model.MenuTypeMenu,
			Visibility: model.MenuVisibilityAdmin,
			Status:     model.MenuStatusEnable,
		}
		if err := db.Create(menu).Error; err != nil {
			t.Fatalf("seed menu %s: %v", menuCode, err)
		}
	}
	return app
}

// newTenantCreateReq 构造带管理员的建租户入参（管理员现在必填）。
func newTenantCreateReq(name, adminEmail string) *dtotenant.TenantCreateReq {
	req := &dtotenant.TenantCreateReq{}
	req.Name = name
	req.Type = string(model.TenantTypeCustomer)
	req.Admin = &dtotenant.TenantAdminCreateReq{Name: name + "管理员", PrimaryEmail: adminEmail}
	return req
}

func countEntities(t *testing.T, db *gorm.DB, entity any, query string, args ...any) int64 {
	t.Helper()
	var count int64
	if err := db.Model(entity).Where(query, args...).Count(&count).Error; err != nil {
		t.Fatalf("count %T fail: %v", entity, err)
	}
	return count
}

// TestTenantPageListReturnsTimeFields 列表必须同时回传创建时间与更新时间
// （前端「创建时间」「更新时间」两列都读这两个字段，缺失则渲染为 "-"）。
func TestTenantPageListReturnsTimeFields(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.TenantEntity{})
	if err := db.Create(&model.TenantEntity{
		Code: "t_000000000001",
		Name: "Acme Corp",
		Type: model.TenantTypeCustomer,
	}).Error; err != nil {
		t.Fatalf("seed tenant: %v", err)
	}

	svc := &tenantSvc{}
	resp, err := svc.PageList(newTenantScopeGinCtx(""), &dtotenant.TenantPageListReq{})
	if err != nil {
		t.Fatalf("PageList failed: %v", err)
	}
	if len(resp.List) != 1 {
		t.Fatalf("PageList len = %d, want 1", len(resp.List))
	}
	item := resp.List[0]
	if item.CreatedAt <= 0 {
		t.Errorf("createdAt = %d, want > 0", item.CreatedAt)
	}
	if item.UpdatedAt <= 0 {
		t.Errorf("updatedAt = %d, want > 0", item.UpdatedAt)
	}
	if item.UpdatedAt < item.CreatedAt {
		t.Errorf("updatedAt(%d) < createdAt(%d)", item.UpdatedAt, item.CreatedAt)
	}
}

// TestTenantPageListNameKeyword 列表按租户名模糊搜索：name 入参映射到条件 Keyword。
func TestTenantPageListNameKeyword(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.TenantEntity{})
	for _, entity := range []*model.TenantEntity{
		{Code: "t_000000000002", Name: "Acme Corp", Type: model.TenantTypeCustomer},
		{Code: "t_000000000003", Name: "Globex", Type: model.TenantTypeCustomer},
	} {
		if err := db.Create(entity).Error; err != nil {
			t.Fatalf("seed tenant: %v", err)
		}
	}

	svc := &tenantSvc{}
	resp, err := svc.PageList(newTenantScopeGinCtx(""), &dtotenant.TenantPageListReq{Name: "cme"})
	if err != nil {
		t.Fatalf("PageList failed: %v", err)
	}
	if len(resp.List) != 1 || resp.List[0].Name != "Acme Corp" {
		t.Fatalf("PageList keyword result = %+v, want only Acme Corp", resp.List)
	}
}

// TestTenantCreateGeneratesCode 建租户时编码由服务端按规则自动生成，客户端传值被忽略。
func TestTenantCreateGeneratesCode(t *testing.T) {
	setupTenantCreateEnv(t)

	svc := &tenantSvc{}
	req := newTenantCreateReq("Acme Corp", "admin@acme.com")
	req.Code = "manual-code-should-be-ignored"

	resp, err := svc.Create(newTenantScopeGinCtx(""), req)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if resp.TenantID == "" {
		t.Fatal("Create returned empty tenantID")
	}

	stored, err := dao.NewTenantDao().GetByID(newTenantScopeGinCtx(""), resp.TenantID)
	if err != nil {
		t.Fatalf("dao GetByID failed: %v", err)
	}
	if stored == nil {
		t.Fatal("created tenant not found")
	}
	pattern := regexp.MustCompile(`^t_[0-9a-f]{12}$`)
	if !pattern.MatchString(stored.Code) {
		t.Errorf("stored code = %q, want match %s", stored.Code, pattern.String())
	}
	if stored.Code == req.Code {
		t.Errorf("stored code = %q, want server-generated code ignoring req.Code", stored.Code)
	}
}

// TestTenantCreateNormalizesStatus 租户状态是白名单枚举：非法值/缺省一律归一为 active，
// 绝不把脏值落库（脏值会让 IsActive 判定为不可用，导致租户整体无法登录）。
func TestTenantCreateNormalizesStatus(t *testing.T) {
	setupTenantCreateEnv(t)

	svc := &tenantSvc{}
	req := newTenantCreateReq("Acme Corp", "admin@acme.com")
	req.Status = model.TenantStatus("bogus")

	resp, err := svc.Create(newTenantScopeGinCtx(""), req)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	stored, err := dao.NewTenantDao().GetByID(newTenantScopeGinCtx(""), resp.TenantID)
	if err != nil {
		t.Fatalf("dao GetByID failed: %v", err)
	}
	if stored == nil {
		t.Fatal("created tenant not found")
	}
	if stored.Status != model.TenantStatusActive {
		t.Errorf("stored status = %q, want %q", stored.Status, model.TenantStatusActive)
	}
}

// TestTenantCreateProvisionsBuiltinAdmin 建租户必须同时产出"可用的租户管理员"：
// builtin 来源 + 归属租户根组织 + 初始临时密码可用且强制改密 + 应用订阅/内置角色/菜单授权/角色绑定齐备。
func TestTenantCreateProvisionsBuiltinAdmin(t *testing.T) {
	db := setupTenantCreateEnv(t)
	ctx := newTenantScopeGinCtx("")

	svc := &tenantSvc{}
	resp, err := svc.Create(ctx, newTenantCreateReq("Acme Corp", "admin@acme.com"))
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if resp.AdminUserID == "" {
		t.Fatal("Create returned empty adminUserID")
	}
	if resp.AdminInitialPassword == "" {
		t.Fatal("Create returned empty adminInitialPassword (new person must get a temporary password)")
	}
	if err := password.ValidateStrength(resp.AdminInitialPassword); err != nil {
		t.Errorf("adminInitialPassword %q fails strength: %v", resp.AdminInitialPassword, err)
	}

	// 管理员用户：builtin 来源、owner、member 类型
	adminUser, err := dao.NewUserDao().GetByID(ctx, resp.AdminUserID)
	if err != nil || adminUser == nil {
		t.Fatalf("load admin user fail, err:%v, user:%+v", err, adminUser)
	}
	if adminUser.Source != model.UserSourceBuiltin {
		t.Errorf("admin source = %q, want %q", adminUser.Source, model.UserSourceBuiltin)
	}
	if !adminUser.IsOwner {
		t.Error("admin should be tenant owner")
	}
	if adminUser.TenantID != resp.TenantID {
		t.Errorf("admin tenantID = %q, want %q", adminUser.TenantID, resp.TenantID)
	}

	// 密码哈希与返回的临时密码一致，且带强制改密标记
	person, err := dao.NewPersonDao().GetByID(ctx, adminUser.PersonID)
	if err != nil || person == nil {
		t.Fatalf("load admin person fail, err:%v, person:%+v", err, person)
	}
	if !person.MustChangePassword {
		t.Error("admin person must have must_change_password=true (temporary password)")
	}
	if err := gcrypto.ComparePasswordHash(person.PasswordEncrypted, resp.AdminInitialPassword); err != nil {
		t.Errorf("adminInitialPassword does not match stored hash: %v", err)
	}

	// 归属租户根组织（primary）
	rootOrg, err := dao.NewOrganizationDao().GetByCond(ctx, &dao.OrganizationCond{TenantID: resp.TenantID})
	if err != nil || rootOrg == nil {
		t.Fatalf("load root org fail, err:%v, org:%+v", err, rootOrg)
	}
	if got := countEntities(t, db, &model.OrganizationUserEntity{},
		"tenant_id = ? AND user_id = ? AND organization_id = ? AND relation_type = ?",
		resp.TenantID, resp.AdminUserID, rootOrg.ID, model.OrgUserRelationPrimary); got != 1 {
		t.Errorf("primary org relation count = %d, want 1", got)
	}

	// 权限开通：应用订阅 1 + 内置角色 1 + 菜单授权 4 + 管理员角色绑定 1
	if got := countEntities(t, db, &model.TenantApplicationEntity{}, "tenant_id = ?", resp.TenantID); got != 1 {
		t.Errorf("tenant_application count = %d, want 1", got)
	}
	role, err := dao.NewRoleDao().GetByCond(ctx, &dao.RoleCond{TenantID: resp.TenantID, Code: tenant.ProvisionRoleCode})
	if err != nil || role == nil {
		t.Fatalf("load builtin role fail, err:%v, role:%+v", err, role)
	}
	if !role.IsBuiltinAdmin() {
		t.Errorf("role should be builtin super admin, got source=%q adminLevel=%q", role.Source, role.AdminLevel)
	}
	if got := countEntities(t, db, &model.RoleMenuEntity{}, "tenant_id = ? AND role_id = ?", resp.TenantID, role.ID); got != int64(len(tenant.ProvisionMenuCodes)) {
		t.Errorf("role_menu count = %d, want %d", got, len(tenant.ProvisionMenuCodes))
	}
	if got := countEntities(t, db, &model.UserRoleEntity{}, "tenant_id = ? AND user_id = ? AND role_id = ?", resp.TenantID, resp.AdminUserID, role.ID); got != 1 {
		t.Errorf("user_role count = %d, want 1", got)
	}
}

// TestTenantCreateRejectsAdminWithoutContact 管理员必须至少有一个联系方式：
// 否则既无法识别自然人、也无从交接初始密码。
func TestTenantCreateRejectsAdminWithoutContact(t *testing.T) {
	setupTenantCreateEnv(t)

	svc := &tenantSvc{}
	req := newTenantCreateReq("Acme Corp", "")
	req.Admin.PrimaryPhone = ""
	_, err := svc.Create(newTenantScopeGinCtx(""), req)
	if err == nil || err.Error() != code.GetError(code.UserContactRequiredError).Error() {
		t.Fatalf("Create err = %v, want %v", err, code.GetError(code.UserContactRequiredError))
	}
}

// TestTenantCreateReusesExistingPersonWithoutPassword 管理员的邮箱/手机命中已存在自然人时：
// 复用该自然人、不覆盖其密码、不置强制改密，且不回显初始密码（响应为空串）。
func TestTenantCreateReusesExistingPersonWithoutPassword(t *testing.T) {
	db := setupTenantCreateEnv(t)
	ctx := newTenantScopeGinCtx("")

	existing := &model.PersonEntity{
		PrimaryEmail:      model.StrPtr("admin@acme.com"),
		PasswordEncrypted: "existing-hash",
		PasswordMethod:    "bcrypt",
		Name:              "既有账号",
		Profile:           []byte(`{}`),
		CustomData:        []byte(`{}`),
	}
	if err := db.Create(existing).Error; err != nil {
		t.Fatalf("seed existing person: %v", err)
	}

	svc := &tenantSvc{}
	resp, err := svc.Create(ctx, newTenantCreateReq("Acme Corp", "admin@acme.com"))
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if resp.AdminInitialPassword != "" {
		t.Errorf("adminInitialPassword = %q, want empty when reusing an existing person", resp.AdminInitialPassword)
	}

	adminUser, err := dao.NewUserDao().GetByID(ctx, resp.AdminUserID)
	if err != nil || adminUser == nil {
		t.Fatalf("load admin user fail, err:%v", err)
	}
	if adminUser.PersonID != existing.ID {
		t.Errorf("admin personID = %q, want existing %q", adminUser.PersonID, existing.ID)
	}
	person, err := dao.NewPersonDao().GetByID(ctx, existing.ID)
	if err != nil || person == nil {
		t.Fatalf("load person fail, err:%v", err)
	}
	if person.PasswordEncrypted != "existing-hash" {
		t.Errorf("existing person password was overwritten: %q", person.PasswordEncrypted)
	}
	if person.MustChangePassword {
		t.Error("must_change_password must stay false when reusing an existing person")
	}
}

// TestResetAdminPasswordReissuesTemporaryPassword 兜底路径：重置内置管理员密码 → 返回新临时密码、
// 旧密码立即失效、强制改密标记置位。
func TestResetAdminPasswordReissuesTemporaryPassword(t *testing.T) {
	setupTenantCreateEnv(t)
	ctx := newTenantScopeGinCtx("")

	svc := &tenantSvc{}
	created, err := svc.Create(ctx, newTenantCreateReq("Acme Corp", "admin@acme.com"))
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	reset, err := svc.ResetAdminPassword(ctx, &dtotenant.TenantAdminResetPasswordReq{TenantID: created.TenantID})
	if err != nil {
		t.Fatalf("ResetAdminPassword failed: %v", err)
	}
	if reset.UserID != created.AdminUserID {
		t.Errorf("reset userID = %q, want builtin admin %q", reset.UserID, created.AdminUserID)
	}
	if reset.InitialPassword == "" {
		t.Fatal("ResetAdminPassword returned empty password")
	}
	if reset.InitialPassword == created.AdminInitialPassword {
		t.Error("reset must issue a new password, not reuse the previous one")
	}
	if err := password.ValidateStrength(reset.InitialPassword); err != nil {
		t.Errorf("reset password %q fails strength: %v", reset.InitialPassword, err)
	}

	adminUser, err := dao.NewUserDao().GetByID(ctx, created.AdminUserID)
	if err != nil || adminUser == nil {
		t.Fatalf("load admin user fail, err:%v", err)
	}
	person, err := dao.NewPersonDao().GetByID(ctx, adminUser.PersonID)
	if err != nil || person == nil {
		t.Fatalf("load admin person fail, err:%v", err)
	}
	if err := gcrypto.ComparePasswordHash(person.PasswordEncrypted, reset.InitialPassword); err != nil {
		t.Errorf("reset password does not match stored hash: %v", err)
	}
	if err := gcrypto.ComparePasswordHash(person.PasswordEncrypted, created.AdminInitialPassword); err == nil {
		t.Error("old temporary password must no longer be valid after reset")
	}
	if !person.MustChangePassword {
		t.Error("reset must set must_change_password=true")
	}
}

// TestResetAdminPasswordNeverTargetsManualMember 内置管理员被删/租户只有手工成员时，
// 重置接口不得"顺手"重置某个 manual 成员（授权边界：平台不管理租户内部成员）。
func TestResetAdminPasswordNeverTargetsManualMember(t *testing.T) {
	db := setupTenantCreateEnv(t)
	ctx := newTenantScopeGinCtx("")

	svc := &tenantSvc{}
	created, err := svc.Create(ctx, newTenantCreateReq("Acme Corp", "admin@acme.com"))
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// 该租户内再插入一个手工成员
	manualPerson := &model.PersonEntity{
		PrimaryEmail:      model.StrPtr("member@acme.com"),
		PasswordEncrypted: "member-hash",
		PasswordMethod:    "bcrypt",
		Name:              "成员",
		Profile:           []byte(`{}`),
		CustomData:        []byte(`{}`),
	}
	if err := db.Create(manualPerson).Error; err != nil {
		t.Fatalf("seed manual person: %v", err)
	}
	manualUser := &model.UserEntity{
		TenantID: created.TenantID, PersonID: manualPerson.ID, Source: model.UserSourceManual,
		UserType: model.UserTypeMember, Name: "成员", Profile: []byte(`{}`), CustomData: []byte(`{}`),
	}
	if err := db.Create(manualUser).Error; err != nil {
		t.Fatalf("seed manual user: %v", err)
	}

	// 删除内置管理员后，租户只剩 manual 成员 → 必须拒绝，且不得改动 manual 成员
	if err := db.Where("tenant_id = ? AND source = ?", created.TenantID, model.UserSourceBuiltin).
		Delete(&model.UserEntity{}).Error; err != nil {
		t.Fatalf("delete builtin admin: %v", err)
	}
	_, err = svc.ResetAdminPassword(ctx, &dtotenant.TenantAdminResetPasswordReq{TenantID: created.TenantID})
	if err == nil || err.Error() != code.GetError(code.UserNotExistError).Error() {
		t.Fatalf("ResetAdminPassword err = %v, want %v", err, code.GetError(code.UserNotExistError))
	}
	storedManual, err := dao.NewPersonDao().GetByID(ctx, manualPerson.ID)
	if err != nil || storedManual == nil {
		t.Fatalf("load manual person fail, err:%v", err)
	}
	if storedManual.PasswordEncrypted != "member-hash" {
		t.Error("manual member password must not be touched by the platform reset endpoint")
	}
}

// TestResetAdminPasswordTenantNotFound 租户不存在时返回 TenantNotExistError（区别于"无内置管理员"）。
func TestResetAdminPasswordTenantNotFound(t *testing.T) {
	setupTenantCreateEnv(t)

	svc := &tenantSvc{}
	_, err := svc.ResetAdminPassword(newTenantScopeGinCtx(""), &dtotenant.TenantAdminResetPasswordReq{TenantID: "not-exist"})
	if err == nil || err.Error() != code.GetError(code.TenantNotExistError).Error() {
		t.Fatalf("ResetAdminPassword err = %v, want %v", err, code.GetError(code.TenantNotExistError))
	}
}

// TestTenantUpdateSuspendsAndListsStatus 挂起他租户：状态落库并在列表出参回传
// （前端「状态」列读 status，不再读已废弃的 isSuspended）。
func TestTenantUpdateSuspendsAndListsStatus(t *testing.T) {
	setupTenantCreateEnv(t)

	svc := &tenantSvc{}
	created, err := svc.Create(newTenantScopeGinCtx(""), newTenantCreateReq("Acme Corp", "admin@acme.com"))
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	updateReq := &dtotenant.TenantUpdateReq{TenantID: created.TenantID}
	updateReq.Name = "Acme Corp"
	updateReq.Type = string(model.TenantTypeCustomer)
	updateReq.Status = model.TenantStatusSuspended
	// 操作者所在租户是另一个租户，允许挂起
	if err := svc.Update(newTenantScopeGinCtx("other-tenant"), updateReq); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	stored, err := dao.NewTenantDao().GetByID(newTenantScopeGinCtx(""), created.TenantID)
	if err != nil {
		t.Fatalf("dao GetByID failed: %v", err)
	}
	if stored == nil || stored.Status != model.TenantStatusSuspended {
		t.Fatalf("stored status = %+v, want %q", stored, model.TenantStatusSuspended)
	}

	listResp, err := svc.PageList(newTenantScopeGinCtx(""), &dtotenant.TenantPageListReq{})
	if err != nil {
		t.Fatalf("PageList failed: %v", err)
	}
	var found bool
	for _, item := range listResp.List {
		if item.TenantID == created.TenantID {
			found = true
			if item.Status != model.TenantStatusSuspended {
				t.Errorf("list status = %q, want %q", item.Status, model.TenantStatusSuspended)
			}
		}
	}
	if !found {
		t.Fatalf("suspended tenant not found in list: %+v", listResp.List)
	}
}

// TestTenantUpdateRefusesSuspendOwnTenant 禁止挂起操作者自己所在的租户：
// 挂起后该租户无法登录、平台控制台随之失联且无恢复路径（不可逆自锁）。
func TestTenantUpdateRefusesSuspendOwnTenant(t *testing.T) {
	setupTenantCreateEnv(t)

	svc := &tenantSvc{}
	created, err := svc.Create(newTenantScopeGinCtx(""), newTenantCreateReq("Acme Corp", "admin@acme.com"))
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	updateReq := &dtotenant.TenantUpdateReq{TenantID: created.TenantID}
	updateReq.Name = "Acme Corp"
	updateReq.Type = string(model.TenantTypeCustomer)
	updateReq.Status = model.TenantStatusSuspended
	err = svc.Update(newTenantScopeGinCtx(created.TenantID), updateReq)
	if err == nil || err.Error() != code.GetError(code.TenantSuspendSelfForbiddenError).Error() {
		t.Fatalf("Update err = %v, want %v", err, code.GetError(code.TenantSuspendSelfForbiddenError))
	}

	// 拒绝后状态不得被改写
	stored, err := dao.NewTenantDao().GetByID(newTenantScopeGinCtx(""), created.TenantID)
	if err != nil {
		t.Fatalf("dao GetByID failed: %v", err)
	}
	if stored == nil || stored.Status != model.TenantStatusActive {
		t.Fatalf("stored status = %+v, want %q", stored, model.TenantStatusActive)
	}
}

// TestTenantPageListStatusFilter 状态筛选：不传表示不筛选，传 active/suspended 精确过滤。
func TestTenantPageListStatusFilter(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.TenantEntity{})
	for _, entity := range []*model.TenantEntity{
		{Code: "t_000000000011", Name: "Active Co", Type: model.TenantTypeCustomer, Status: model.TenantStatusActive},
		{Code: "t_000000000012", Name: "Suspended Co", Type: model.TenantTypeCustomer, Status: model.TenantStatusSuspended},
	} {
		if err := db.Create(entity).Error; err != nil {
			t.Fatalf("seed tenant: %v", err)
		}
	}

	svc := &tenantSvc{}
	all, err := svc.PageList(newTenantScopeGinCtx(""), &dtotenant.TenantPageListReq{})
	if err != nil {
		t.Fatalf("PageList(no filter) failed: %v", err)
	}
	if len(all.List) != 2 {
		t.Fatalf("PageList(no filter) len = %d, want 2", len(all.List))
	}

	suspended, err := svc.PageList(newTenantScopeGinCtx(""), &dtotenant.TenantPageListReq{Status: model.TenantStatusSuspended})
	if err != nil {
		t.Fatalf("PageList(suspended) failed: %v", err)
	}
	if len(suspended.List) != 1 || suspended.List[0].Name != "Suspended Co" {
		t.Fatalf("PageList(suspended) = %+v, want only Suspended Co", suspended.List)
	}
	if suspended.List[0].Status != model.TenantStatusSuspended {
		t.Errorf("filtered status = %q, want %q", suspended.List[0].Status, model.TenantStatusSuspended)
	}

	active, err := svc.PageList(newTenantScopeGinCtx(""), &dtotenant.TenantPageListReq{Status: model.TenantStatusActive})
	if err != nil {
		t.Fatalf("PageList(active) failed: %v", err)
	}
	if len(active.List) != 1 || active.List[0].Name != "Active Co" {
		t.Fatalf("PageList(active) = %+v, want only Active Co", active.List)
	}
}

// TestTenantPageListRejectsInvalidStatus 非法状态筛选值必须报错，而不是静默返回全部/空集
// （静默处理会让调用方以为"筛选成功但没数据"）。
func TestTenantPageListRejectsInvalidStatus(t *testing.T) {
	testutil.SetupSQLite(t, &model.TenantEntity{})

	svc := &tenantSvc{}
	_, err := svc.PageList(newTenantScopeGinCtx(""), &dtotenant.TenantPageListReq{Status: model.TenantStatus("bogus")})
	if err == nil || err.Error() != code.GetError(code.TenantPageListStatusInvalidError).Error() {
		t.Fatalf("PageList err = %v, want %v", err, code.GetError(code.TenantPageListStatusInvalidError))
	}
}
