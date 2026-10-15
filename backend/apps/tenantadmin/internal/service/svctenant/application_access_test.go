package svctenant

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/tenantadmin/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/tenantadmin/testutil"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

// appAccessFixture 应用准入手工装配：应用/订阅/准入/角色/角色绑定。
// 不走 pkg/seed，避免测试依赖引导链路（引导本身由 pkg/seed 与 pkg/core/tenant 覆盖）。
type appAccessFixture struct {
	t   *testing.T
	db  *gorm.DB
	ctx *gin.Context
}

func newAppAccessFixtureDB(t *testing.T) *gorm.DB {
	t.Helper()
	return testutil.SetupSQLite(t,
		&model.TenantEntity{}, &model.UserEntity{}, &model.PersonEntity{},
		&model.ApplicationEntity{}, &model.TenantApplicationEntity{}, &model.UserApplicationEntity{},
		&model.RoleEntity{}, &model.UserRoleEntity{},
	)
}

// newAppAccessFixture 操作者绑定「内置管理员」角色（常规管理写路径）。
func newAppAccessFixture(t *testing.T, operatorID string) *appAccessFixture {
	t.Helper()
	db := newAppAccessFixtureDB(t)
	ctx := newAdminCtx(t, db, "t1", operatorID)
	return &appAccessFixture{t: t, db: db, ctx: ctx}
}

// newAppAccessFixtureCustomAdmin 操作者绑定「自定义来源」管理员角色：操作者自身不算
// 内置管理员持有者，用于隔离 I5（最后一个内置管理员）判定，避免夹具把判定结果搅浑。
func newAppAccessFixtureCustomAdmin(t *testing.T, operatorID string) *appAccessFixture {
	t.Helper()
	db := newAppAccessFixtureDB(t)
	seedTenantCustomAdminOperator(t, db, "t1", operatorID)
	ctx := newDeptGinCtx(t, "t1", operatorID)
	return &appAccessFixture{t: t, db: db, ctx: ctx}
}

// app 播种应用（默认启用）。
func (f *appAccessFixture) app(id, name string, status model.AppStatus) string {
	f.t.Helper()
	if err := f.db.Create(&model.ApplicationEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: id}},
		Code:       "code-" + id,
		Name:       name,
		Status:     status,
	}).Error; err != nil {
		f.t.Fatalf("seed app %s: %v", id, err)
	}
	return id
}

// appUnnamed 播种缺少名称字段的应用（列表装配用）。
func (f *appAccessFixture) appUnnamed(id string) string { return f.app(id, id, model.AppStatusEnable) }

func (f *appAccessFixture) subscribe(tenantID, appID string, status model.TenantApplicationStatus) {
	f.t.Helper()
	if err := f.db.Create(&model.TenantApplicationEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "sub-" + tenantID + "-" + appID}},
		TenantID:   tenantID,
		AppID:      appID,
		Status:     status,
	}).Error; err != nil {
		f.t.Fatalf("seed subscription %s/%s: %v", tenantID, appID, err)
	}
}

func (f *appAccessFixture) user(userID, tenantID, name string) {
	f.t.Helper()
	seedTestUser(f.t, f.db, tenantID, userID, name)
}

// userWithPerson 播种成员并关联自然人（用户名/邮箱只在 person 上，服务账号恒空）。
func (f *appAccessFixture) userWithPerson(userID, tenantID, personID, name, username, email string) {
	f.t.Helper()
	usernamePtr, emailPtr := username, email
	if err := f.db.Create(&model.PersonEntity{
		BaseEntity:   gormdao.BaseEntity{StringID: gormdao.StringID{ID: personID}},
		Username:     &usernamePtr,
		PrimaryEmail: &emailPtr,
		Name:         name,
		CreatedBy:    "t",
	}).Error; err != nil {
		f.t.Fatalf("seed person %s: %v", personID, err)
	}
	seedTestUserWithPerson(f.t, f.db, userID, tenantID, personID, name)
}

func (f *appAccessFixture) role(roleID, tenantID, appID string) {
	f.t.Helper()
	if err := f.db.Create(&model.RoleEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: roleID}},
		TenantID:   tenantID,
		AppID:      appID,
		Name:       roleID,
	}).Error; err != nil {
		f.t.Fatalf("seed role %s: %v", roleID, err)
	}
}

func (f *appAccessFixture) bindRole(linkID, tenantID, userID, roleID string) {
	f.t.Helper()
	if err := f.db.Create(&model.UserRoleEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: linkID}},
		TenantID:   tenantID,
		UserID:     userID,
		RoleID:     roleID,
		CreatedBy:  "t",
	}).Error; err != nil {
		f.t.Fatalf("seed user_role %s: %v", linkID, err)
	}
}

func (f *appAccessFixture) grant(userID, appID string, status model.UserApplicationStatus, source model.UserApplicationSource) {
	f.t.Helper()
	if err := f.db.Create(&model.UserApplicationEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "grant-" + userID + "-" + appID}},
		TenantID:   "t1",
		UserID:     userID,
		AppID:      appID,
		Status:     status,
		Source:     source,
	}).Error; err != nil {
		f.t.Fatalf("seed grant %s/%s: %v", userID, appID, err)
	}
}

// grantOf 读取准入行（跨租户上下文，避免受被测上下文作用域影响）。
func (f *appAccessFixture) grantOf(userID, appID string) *model.UserApplicationEntity {
	f.t.Helper()
	entity := &model.UserApplicationEntity{}
	err := f.db.WithContext(f.ctx).Where("tenant_id = ? AND user_id = ? AND app_id = ?", "t1", userID, appID).First(entity).Error
	if err == gorm.ErrRecordNotFound {
		return nil
	}
	if err != nil {
		f.t.Fatalf("query grant %s/%s: %v", userID, appID, err)
	}
	return entity
}

func (f *appAccessFixture) activeRoleIDs(userID string) []string {
	f.t.Helper()
	var ids []string
	if err := f.db.WithContext(f.ctx).Model(&model.UserRoleEntity{}).
		Where("tenant_id = ? AND user_id = ?", "t1", userID).Pluck("role_id", &ids).Error; err != nil {
		f.t.Fatalf("query user_role: %v", err)
	}
	return ids
}

// TestListUserApplicationsView 清单是「订阅行 ∪ 准入行」的并集，并回传 roleCount：
//   - 订阅 + 准入：可用，status/source 取准入行；
//   - 订阅但未准入：呈现且 granted=false（UI 呈现全集与未选中态）；
//   - 订阅被停用但仍有准入行：如实呈现 subscribed=false（真相是订阅失效覆盖了准入）；
//   - 应用被停用：appStatus=disable（可用性由 appStatus 表达，不隐藏行）；
//   - 既未订阅也无准入：不呈现。
func TestListUserApplicationsView(t *testing.T) {
	f := newAppAccessFixture(t, "op")
	f.user("u1", "t1", "张三")
	f.app("app-on", "已订阅已准入", model.AppStatusEnable)
	f.app("app-un", "已订阅未准入", model.AppStatusEnable)
	f.app("app-off", "订阅已停用", model.AppStatusEnable)
	f.app("app-disabled", "应用已停用", model.AppStatusDisable)
	f.app("app-absent", "既未订阅也无准入", model.AppStatusEnable)
	f.subscribe("t1", "app-on", model.TenantApplicationStatusEnable)
	f.subscribe("t1", "app-un", model.TenantApplicationStatusEnable)
	f.subscribe("t1", "app-off", model.TenantApplicationStatusDisable)
	f.subscribe("t1", "app-disabled", model.TenantApplicationStatusEnable)
	f.grant("u1", "app-on", model.UserApplicationStatusEnable, model.UserApplicationSourceManual)
	f.grant("u1", "app-off", model.UserApplicationStatusEnable, model.UserApplicationSourceManual)
	f.grant("u1", "app-disabled", model.UserApplicationStatusEnable, model.UserApplicationSourceManual)
	// app-on 下 2 个角色 → roleCount=2；另一个应用的角色不计入
	f.role("r-on-1", "t1", "app-on")
	f.role("r-on-2", "t1", "app-on")
	f.role("r-un", "t1", "app-un")
	f.bindRole("ur-1", "t1", "u1", "r-on-1")
	f.bindRole("ur-2", "t1", "u1", "r-on-2")
	f.bindRole("ur-3", "t1", "u1", "r-un")

	resp, err := NewApplicationAccessSvc().ListUserApplications(f.ctx, &dtotenant.UserApplicationsListReq{UserID: "u1"})
	if err != nil {
		t.Fatalf("ListUserApplications: %v", err)
	}
	itemOf := map[string]dtotenant.UserApplicationItem{}
	for _, item := range resp.List {
		itemOf[item.AppID] = item
	}
	if len(itemOf) != 4 {
		t.Fatalf("清单应为订阅∪准入的 4 行，got %+v", resp.List)
	}
	on := itemOf["app-on"]
	if !on.Subscribed || !on.Granted || on.Status != model.UserApplicationStatusEnable ||
		on.Source != model.UserApplicationSourceManual || on.RoleCount != 2 || on.UpdatedAt <= 0 {
		t.Fatalf("app-on 行不符: %+v", on)
	}
	if on.AppStatus != model.AppStatusEnable {
		t.Fatalf("app-on 应用状态应回传 enable: %+v", on)
	}
	un := itemOf["app-un"]
	if !un.Subscribed || un.Granted || un.Status != "" || un.Source != "" {
		t.Fatalf("未准入的订阅应用应呈现 subscribed=true/granted=false: %+v", un)
	}
	// app-un 上确实挂了 1 个角色：roleCount 只按「该应用下的角色」计，与是否准入无关
	// （「有角色但无准入行」是回填前的合法存量态，管理端必须如实呈现）
	if un.RoleCount != 1 {
		t.Fatalf("app-un 的 roleCount 应为 1（该应用下的角色数，不含别处）: %+v", un)
	}
	off := itemOf["app-off"]
	if off.Subscribed || !off.Granted {
		t.Fatalf("订阅停用但仍有准入行时应如实呈现 subscribed=false/granted=true: %+v", off)
	}
	disabled := itemOf["app-disabled"]
	if disabled.AppStatus != model.AppStatusDisable {
		t.Fatalf("应用停用应回传 appStatus=disable: %+v", disabled)
	}
	if _, ok := itemOf["app-absent"]; ok {
		t.Fatalf("既未订阅也无准入的应用不得出现: %+v", resp.List)
	}
}

// TestUpdateUserApplicationsReplaceAndCascade 全量替换（PUT 集合语义）：
// 集合内 enable、集合外 disable，且集合外应用的角色绑定被级联撤销（I2）。
func TestUpdateUserApplicationsReplaceAndCascade(t *testing.T) {
	f := newAppAccessFixture(t, "op")
	f.user("u1", "t1", "张三")
	f.app("app-a", "应用A", model.AppStatusEnable)
	f.app("app-b", "应用B", model.AppStatusEnable)
	f.subscribe("t1", "app-a", model.TenantApplicationStatusEnable)
	f.subscribe("t1", "app-b", model.TenantApplicationStatusEnable)
	f.role("r-a", "t1", "app-a")
	f.role("r-b", "t1", "app-b")
	f.bindRole("ur-a", "t1", "u1", "r-a")
	f.bindRole("ur-b", "t1", "u1", "r-b")
	svc := NewApplicationAccessSvc()

	// 首次提交：{app-a, app-b}
	appIDs := []string{"app-a", "app-b"}
	if err := svc.UpdateUserApplications(f.ctx, &dtotenant.UserApplicationsUpdateReq{UserID: "u1", AppIDs: &appIDs}); err != nil {
		t.Fatalf("replace to {a,b}: %v", err)
	}
	for _, appID := range []string{"app-a", "app-b"} {
		if g := f.grantOf("u1", appID); g == nil || g.Status != model.UserApplicationStatusEnable {
			t.Fatalf("%s 应被准入, got %+v", appID, g)
		}
	}
	// 二次提交：仅 {app-a} → app-b 准入置 disable 且其角色绑定被级联撤销
	kept := []string{"app-a"}
	if err := svc.UpdateUserApplications(f.ctx, &dtotenant.UserApplicationsUpdateReq{UserID: "u1", AppIDs: &kept}); err != nil {
		t.Fatalf("replace to {a}: %v", err)
	}
	if g := f.grantOf("u1", "app-b"); g == nil || g.Status != model.UserApplicationStatusDisable {
		t.Fatalf("集合外应用应被置为 disable（保留行以留痕）, got %+v", g)
	}
	if g := f.grantOf("u1", "app-a"); g == nil || g.Status != model.UserApplicationStatusEnable {
		t.Fatalf("集合内应用应保持 enable, got %+v", g)
	}
	if ids := f.activeRoleIDs("u1"); len(ids) != 1 || ids[0] != "r-a" {
		t.Fatalf("app-b 的角色绑定应被级联撤销，仅留 r-a, got %v", ids)
	}

	// 幂等：重复提交同一集合结果一致
	if err := svc.UpdateUserApplications(f.ctx, &dtotenant.UserApplicationsUpdateReq{UserID: "u1", AppIDs: &kept}); err != nil {
		t.Fatalf("repeat replace: %v", err)
	}
	if g := f.grantOf("u1", "app-a"); g == nil || g.Status != model.UserApplicationStatusEnable {
		t.Fatalf("重复提交后 app-a 仍应 enable, got %+v", g)
	}

	// 清空：{}（appIDs=[]）撤销全部准入
	empty := []string{}
	if err := svc.UpdateUserApplications(f.ctx, &dtotenant.UserApplicationsUpdateReq{UserID: "u1", AppIDs: &empty}); err != nil {
		t.Fatalf("clear all: %v", err)
	}
	if g := f.grantOf("u1", "app-a"); g == nil || g.Status != model.UserApplicationStatusDisable {
		t.Fatalf("清空后 app-a 应 disable, got %+v", g)
	}
	if ids := f.activeRoleIDs("u1"); len(ids) != 0 {
		t.Fatalf("清空后角色绑定应全部撤销, got %v", ids)
	}
}

// TestUpdateUserApplicationsRejectsUnsubscribed 未订阅/订阅停用的应用不可准入（100786），
// 且整体拒绝、零写入（全量替换是原子操作，不做部分成功）。
func TestUpdateUserApplicationsRejectsUnsubscribed(t *testing.T) {
	f := newAppAccessFixture(t, "op")
	f.user("u1", "t1", "张三")
	f.app("app-a", "应用A", model.AppStatusEnable)
	f.app("app-un", "未订阅", model.AppStatusEnable)
	f.app("app-off", "订阅停用", model.AppStatusEnable)
	f.subscribe("t1", "app-a", model.TenantApplicationStatusEnable)
	f.subscribe("t1", "app-off", model.TenantApplicationStatusDisable)
	f.grant("u1", "app-a", model.UserApplicationStatusEnable, model.UserApplicationSourceManual)
	f.role("r-a", "t1", "app-a")
	f.bindRole("ur-a", "t1", "u1", "r-a")
	svc := NewApplicationAccessSvc()

	for _, appID := range []string{"app-un", "app-off", "app-missing"} {
		appIDs := []string{"app-a", appID}
		err := svc.UpdateUserApplications(f.ctx, &dtotenant.UserApplicationsUpdateReq{UserID: "u1", AppIDs: &appIDs})
		if got := iamCodeOf(err); got != code.UserApplicationNotSubscribedError {
			t.Fatalf("准入 %s: want 100786, got %v", appID, err)
		}
		// 整体拒绝：原有准入与角色绑定均不受影响
		if g := f.grantOf("u1", "app-a"); g == nil || g.Status != model.UserApplicationStatusEnable {
			t.Fatalf("拒绝后 app-a 准入应保持不变, got %+v", g)
		}
		if ids := f.activeRoleIDs("u1"); len(ids) != 1 || ids[0] != "r-a" {
			t.Fatalf("拒绝后角色绑定应保持不变, got %v", ids)
		}
		if g := f.grantOf("u1", appID); g != nil {
			t.Fatalf("拒绝时不得写入 %s 的准入行, got %+v", appID, g)
		}
	}
}

// TestUpdateUserApplicationsGuards 前置门槛：非系统管理员(权限)、跨租户/不存在成员、服务账号。
func TestUpdateUserApplicationsGuards(t *testing.T) {
	f := newAppAccessFixture(t, "op")
	f.user("u1", "t1", "张三")
	f.user("u2", "t2", "他租户成员")
	f.app("app-a", "应用A", model.AppStatusEnable)
	f.subscribe("t1", "app-a", model.TenantApplicationStatusEnable)
	svc := NewApplicationAccessSvc()
	appIDs := []string{"app-a"}

	// 非系统管理员（不带任何管理员角色）
	plainCtx := newDeptGinCtx(t, "t1", "plain")
	err := svc.UpdateUserApplications(plainCtx, &dtotenant.UserApplicationsUpdateReq{UserID: "u1", AppIDs: &appIDs})
	if err != code.GetError(code.UserSystemAdminRequiredError) {
		t.Fatalf("非管理员: want UserSystemAdminRequired, got %v", err)
	}
	// 跨租户成员：不得跨租户准入
	err = svc.UpdateUserApplications(f.ctx, &dtotenant.UserApplicationsUpdateReq{UserID: "u2", AppIDs: &appIDs})
	if err != code.GetError(code.UserNotExistError) {
		t.Fatalf("跨租户成员: want UserNotExist, got %v", err)
	}
	// 不存在的成员
	err = svc.UpdateUserApplications(f.ctx, &dtotenant.UserApplicationsUpdateReq{UserID: "ghost", AppIDs: &appIDs})
	if err != code.GetError(code.UserNotExistError) {
		t.Fatalf("不存在成员: want UserNotExist, got %v", err)
	}
	// 服务账号走 /machine-users 入口
	f.user("m1", "t1", "服务账号")
	if err := f.db.WithContext(f.ctx).Model(&model.UserEntity{}).Where("id = ?", "m1").
		Update("user_type", model.UserTypeMachine).Error; err != nil {
		t.Fatalf("mark machine: %v", err)
	}
	err = svc.UpdateUserApplications(f.ctx, &dtotenant.UserApplicationsUpdateReq{UserID: "m1", AppIDs: &appIDs})
	if err != code.GetError(code.UserMemberOperationOnlyError) {
		t.Fatalf("服务账号: want UserMemberOperationOnly, got %v", err)
	}
	if g := f.grantOf("u1", "app-a"); g != nil {
		t.Fatalf("被拒场景不得写入准入: %+v", g)
	}
}

// TestUpdateUserApplicationsLastAdminForbidden 防自锁（I5）：
// 撤销最后一个内置管理员的控制台准入会使其无法进入本控制台，整体拒绝且零写入。
func TestUpdateUserApplicationsLastAdminForbidden(t *testing.T) {
	f := newAppAccessFixtureCustomAdmin(t, "op")
	// 目标成员持内置管理员角色（app-admin）+ 该应用准入
	f.user("u1", "t1", "内置管理员")
	f.app("app-admin", "租户管理后台", model.AppStatusEnable)
	f.subscribe("t1", "app-admin", model.TenantApplicationStatusEnable)
	seedBuiltinSystemRole(t, f.db, "role-admin", "t1", "app-admin")
	seedUserRoleLink(t, f.db, "ur-admin", "t1", "u1", "role-admin")
	f.grant("u1", "app-admin", model.UserApplicationStatusEnable, model.UserApplicationSourceProvision)
	svc := NewApplicationAccessSvc()

	empty := []string{}
	err := svc.UpdateUserApplications(f.ctx, &dtotenant.UserApplicationsUpdateReq{UserID: "u1", AppIDs: &empty})
	if got := iamCodeOf(err); got != code.UserApplicationLastAdminForbiddenError {
		t.Fatalf("撤销最后一个内置管理员准入: want 100787, got %v", err)
	}
	if g := f.grantOf("u1", "app-admin"); g == nil || g.Status != model.UserApplicationStatusEnable {
		t.Fatalf("被拒后准入应保持不变, got %+v", g)
	}
	if ids := f.activeRoleIDs("u1"); len(ids) != 1 || ids[0] != "role-admin" {
		t.Fatalf("被拒后角色绑定应保持不变, got %v", ids)
	}

	// 存在第二个内置管理员持有者时放行（保护只针对"最后一个"）
	f.user("u2", "t1", "第二管理员")
	seedUserRoleLink(t, f.db, "ur-admin-2", "t1", "u2", "role-admin")
	if err := svc.UpdateUserApplications(f.ctx, &dtotenant.UserApplicationsUpdateReq{UserID: "u1", AppIDs: &empty}); err != nil {
		t.Fatalf("存在其他内置管理员时应放行: %v", err)
	}
	if g := f.grantOf("u1", "app-admin"); g == nil || g.Status != model.UserApplicationStatusDisable {
		t.Fatalf("放行后准入应被撤销, got %+v", g)
	}
}

// TestListApplicationUsers 应用视角：只列被准入的成员，回传成员摘要与 roleCount。
func TestListApplicationUsers(t *testing.T) {
	f := newAppAccessFixture(t, "op")
	f.userWithPerson("u1", "t1", "p1", "张三", "zhangsan", "zhangsan@example.com")
	f.user("u2", "t1", "李四")
	f.user("u3", "t1", "未准入成员")
	f.app("app-a", "应用A", model.AppStatusEnable)
	f.subscribe("t1", "app-a", model.TenantApplicationStatusEnable)
	f.grant("u1", "app-a", model.UserApplicationStatusEnable, model.UserApplicationSourceRole)
	f.grant("u2", "app-a", model.UserApplicationStatusEnable, model.UserApplicationSourceManual)
	f.grant("u3", "app-a", model.UserApplicationStatusDisable, model.UserApplicationSourceManual)
	f.role("r-a", "t1", "app-a")
	f.bindRole("ur-1", "t1", "u1", "r-a")
	svc := NewApplicationAccessSvc()

	resp, err := svc.ListApplicationUsers(f.ctx, &dtotenant.ApplicationUsersListReq{AppID: "app-a"})
	if err != nil {
		t.Fatalf("ListApplicationUsers: %v", err)
	}
	if resp.Total != 2 || len(resp.List) != 2 {
		t.Fatalf("应只列 2 名被准入成员, got total=%d list=%+v", resp.Total, resp.List)
	}
	byUser := map[string]dtotenant.ApplicationUserItem{}
	for _, item := range resp.List {
		byUser[item.UserID] = item
	}
	u1 := byUser["u1"]
	if u1.Name != "张三" || u1.Username != "zhangsan" || u1.PrimaryEmail != "zhangsan@example.com" ||
		u1.Source != model.UserApplicationSourceRole || u1.RoleCount != 1 || u1.UpdatedAt <= 0 {
		t.Fatalf("u1 行不符（用户名/邮箱须由 person 回填）: %+v", u1)
	}
	if u1.UserType != model.UserTypeMember || u1.Status != model.UserStatusActive {
		t.Fatalf("u1 行账号类型/状态不符: %+v", u1)
	}
	if u2 := byUser["u2"]; u2.Source != model.UserApplicationSourceManual || u2.RoleCount != 0 {
		t.Fatalf("u2 行不符: %+v", u2)
	}
	if _, ok := byUser["u3"]; ok {
		t.Fatalf("被撤销准入的成员不得出现: %+v", resp.List)
	}

	// 无准入的应用：空列表而非报错
	f.app("app-b", "应用B", model.AppStatusEnable)
	f.subscribe("t1", "app-b", model.TenantApplicationStatusEnable)
	empty, err := svc.ListApplicationUsers(f.ctx, &dtotenant.ApplicationUsersListReq{AppID: "app-b"})
	if err != nil {
		t.Fatalf("empty app list: %v", err)
	}
	if empty.Total != 0 || len(empty.List) != 0 {
		t.Fatalf("无准入成员应返回空列表, got %+v", empty)
	}

	// 不存在的应用：明确报"应用不存在"
	_, err = svc.ListApplicationUsers(f.ctx, &dtotenant.ApplicationUsersListReq{AppID: "ghost"})
	if err != code.GetError(code.ApplicationNotExistError) {
		t.Fatalf("不存在应用: want ApplicationNotExist, got %v", err)
	}
}

// TestListApplicationUsersTenantIsolation 跨租户隔离：他租户的准入行与成员不得出现。
func TestListApplicationUsersTenantIsolation(t *testing.T) {
	f := newAppAccessFixture(t, "op")
	f.user("u1", "t1", "本租户成员")
	f.app("app-a", "应用A", model.AppStatusEnable)
	f.subscribe("t1", "app-a", model.TenantApplicationStatusEnable)
	f.grant("u1", "app-a", model.UserApplicationStatusEnable, model.UserApplicationSourceManual)
	// 他租户同应用的准入行
	if err := f.db.Create(&model.UserApplicationEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "grant-t2"}},
		TenantID:   "t2", UserID: "x1", AppID: "app-a",
		Status: model.UserApplicationStatusEnable, Source: model.UserApplicationSourceManual,
	}).Error; err != nil {
		t.Fatalf("seed cross-tenant grant: %v", err)
	}

	resp, err := NewApplicationAccessSvc().ListApplicationUsers(f.ctx, &dtotenant.ApplicationUsersListReq{AppID: "app-a"})
	if err != nil {
		t.Fatalf("ListApplicationUsers: %v", err)
	}
	if resp.Total != 1 || resp.List[0].UserID != "u1" {
		t.Fatalf("不得带出他租户准入行, got %+v", resp.List)
	}
}

// iamCodeOf 取错误的业务码。**不要**用 `err != code.GetError(c)` 或比较 `.Error()` 字符串：
// 那两种写法在"错误码未登记"时两边同为零值/空串，断言恒真（曾因此漏掉
// 100784–100787 未登记、对外返回 code:0 的问题）。
func iamCodeOf(err error) int {
	if err == nil {
		return 0
	}
	if e, ok := err.(interface{ GetCode() int }); ok {
		return e.GetCode()
	}
	return -1
}
