package application

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newAccessTestDB 内存 SQLite 注册为全局 iam 库（同 RegisterDBForTest，会挂租户隔离插件）：
// 本文件的用例要顺带验证"准入查询都显式声明了租户作用域"，因此必须走带插件的注册路径。
func newAccessTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:iam_access_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.ApplicationEntity{},
		&model.TenantApplicationEntity{},
		&model.UserApplicationEntity{},
		&model.RoleEntity{},
		&model.UserRoleEntity{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	dbclient.RegisterDBForTest(dbclient.ServiceNameIam, db)
	t.Cleanup(func() {
		dbclient.ClearDBForTest(dbclient.ServiceNameIam)
		if sqlDB, sqlErr := db.DB(); sqlErr == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

// accessFixture 装配「租户 + 成员 + 应用/订阅/准入/角色」的测试数据。
type accessFixture struct {
	t        *testing.T
	db       *gorm.DB
	base     context.Context
	tenantID string
	userID   string
}

func newAccessFixture(t *testing.T, tenantID, userID string) *accessFixture {
	t.Helper()
	return &accessFixture{t: t, db: newAccessTestDB(t), base: context.Background(), tenantID: tenantID, userID: userID}
}

func (f *accessFixture) scoped() context.Context {
	return dbclient.ExplicitTenantContext(f.base, f.tenantID)
}

func (f *accessFixture) app(code string, status model.AppStatus) string {
	f.t.Helper()
	entity := &model.ApplicationEntity{
		Code:         code,
		Name:         code,
		Source:       model.AppSourceThirdParty,
		Status:       status,
		RoleTemplate: model.RoleTemplateItemList{}, // not null JSON 列须显式播种
	}
	if err := dao.NewApplicationDao().Insert(f.scoped(), entity); err != nil {
		f.t.Fatalf("seed application %s: %v", code, err)
	}
	return entity.ID
}

func (f *accessFixture) subscribe(appID string, status model.TenantApplicationStatus) {
	f.t.Helper()
	if err := dao.NewTenantApplicationDao().Insert(f.scoped(), &model.TenantApplicationEntity{
		TenantID: f.tenantID, AppID: appID, Status: status,
	}); err != nil {
		f.t.Fatalf("seed subscription: %v", err)
	}
}

func (f *accessFixture) grant(appID string, status model.UserApplicationStatus, source model.UserApplicationSource) {
	f.t.Helper()
	if err := dao.NewUserApplicationDao().Insert(f.scoped(), &model.UserApplicationEntity{
		TenantID: f.tenantID, UserID: f.userID, AppID: appID, Status: status, Source: source,
	}); err != nil {
		f.t.Fatalf("seed grant: %v", err)
	}
}

func (f *accessFixture) role(appID string, source model.RoleSource) string {
	f.t.Helper()
	entity := &model.RoleEntity{
		TenantID: f.tenantID, AppID: appID, Name: "role-" + appID,
		Source: source, AdminType: model.SysAdminTypeNormal,
	}
	if err := dao.NewRoleDao().Insert(f.scoped(), entity); err != nil {
		f.t.Fatalf("seed role: %v", err)
	}
	return entity.ID
}

func (f *accessFixture) bindRole(roleID string) {
	f.t.Helper()
	if err := dao.NewUserRoleDao().Insert(f.scoped(), &model.UserRoleEntity{
		TenantID: f.tenantID, UserID: f.userID, RoleID: roleID,
	}); err != nil {
		f.t.Fatalf("seed user_role: %v", err)
	}
}

func (f *accessFixture) grantOf(appID string) *model.UserApplicationEntity {
	f.t.Helper()
	entity, err := dao.NewUserApplicationDao().GetByCond(f.scoped(), &dao.UserApplicationCond{
		TenantID: f.tenantID, UserID: f.userID, AppID: appID,
	})
	if err != nil {
		f.t.Fatalf("load grant: %v", err)
	}
	return entity
}

func (f *accessFixture) activeRoleIDs() []string {
	f.t.Helper()
	userRoles, err := dao.NewUserRoleDao().GetListByCond(f.scoped(), &dao.UserRoleCond{
		TenantID: f.tenantID, UserID: f.userID,
	})
	if err != nil {
		f.t.Fatalf("load user_role: %v", err)
	}
	ids := make([]string, 0, len(userRoles))
	for i := range userRoles {
		ids = append(ids, userRoles[i].RoleID)
	}
	return ids
}

// stateSnapshot 用于断言"预检失败时零写入"。
func (f *accessFixture) stateSnapshot() string {
	f.t.Helper()
	grants, err := dao.NewUserApplicationDao().GetListByCond(f.scoped(), &dao.UserApplicationCond{
		TenantID: f.tenantID, UserID: f.userID,
	})
	if err != nil {
		f.t.Fatalf("snapshot grants: %v", err)
	}
	roles, err := dao.NewUserRoleDao().GetListByCond(f.scoped(), &dao.UserRoleCond{
		TenantID: f.tenantID, UserID: f.userID,
	})
	if err != nil {
		f.t.Fatalf("snapshot roles: %v", err)
	}
	describe := func(appID string, status model.UserApplicationStatus, source model.UserApplicationSource) string {
		return fmt.Sprintf("%s|%s|%s", appID, status, source)
	}
	parts := make([]string, 0, len(grants)+len(roles))
	for i := range grants {
		parts = append(parts, describe(grants[i].AppID, grants[i].Status, grants[i].Source))
	}
	for i := range roles {
		parts = append(parts, "role|"+roles[i].RoleID)
	}
	return fmt.Sprint(parts)
}

// TestUsableExpression 判定表达式本身：三者取交集，任一不启用即不可用。
func TestUsableExpression(t *testing.T) {
	enabledApp := &model.ApplicationEntity{Status: model.AppStatusEnable}
	enabledSub := &model.TenantApplicationEntity{Status: model.TenantApplicationStatusEnable}
	enabledGrant := &model.UserApplicationEntity{Status: model.UserApplicationStatusEnable}

	cases := []struct {
		name  string
		app   *model.ApplicationEntity
		sub   *model.TenantApplicationEntity
		grant *model.UserApplicationEntity
		want  bool
	}{
		{name: "三者齐备", app: enabledApp, sub: enabledSub, grant: enabledGrant, want: true},
		{name: "缺应用", app: nil, sub: enabledSub, grant: enabledGrant, want: false},
		{name: "缺订阅", app: enabledApp, sub: nil, grant: enabledGrant, want: false},
		{name: "缺准入", app: enabledApp, sub: enabledSub, grant: nil, want: false},
		{name: "应用停用", app: &model.ApplicationEntity{Status: model.AppStatusDisable}, sub: enabledSub, grant: enabledGrant, want: false},
		{name: "订阅停用", app: enabledApp, sub: &model.TenantApplicationEntity{Status: model.TenantApplicationStatusDisable}, grant: enabledGrant, want: false},
		{name: "准入撤销", app: enabledApp, sub: enabledSub, grant: &model.UserApplicationEntity{Status: model.UserApplicationStatusDisable}, want: false},
		{name: "零值实体（未配置）", app: &model.ApplicationEntity{}, sub: &model.TenantApplicationEntity{}, grant: &model.UserApplicationEntity{}, want: false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Usable(c.app, c.sub, c.grant); got != c.want {
				t.Fatalf("Usable = %v, want %v", got, c.want)
			}
		})
	}
}

// TestCheckUsableMatrix 点查版判定：六种边界 + 缺参 fail-closed。
func TestCheckUsableMatrix(t *testing.T) {
	f := newAccessFixture(t, "tenant-1", "user-1")
	ctx := f.base

	// 全链路可用
	okApp := f.app("app_ok", model.AppStatusEnable)
	f.subscribe(okApp, model.TenantApplicationStatusEnable)
	f.grant(okApp, model.UserApplicationStatusEnable, model.UserApplicationSourceManual)

	// 无准入
	noGrantApp := f.app("app_no_grant", model.AppStatusEnable)
	f.subscribe(noGrantApp, model.TenantApplicationStatusEnable)

	// 准入被撤销
	revokedApp := f.app("app_revoked", model.AppStatusEnable)
	f.subscribe(revokedApp, model.TenantApplicationStatusEnable)
	f.grant(revokedApp, model.UserApplicationStatusDisable, model.UserApplicationSourceManual)

	// 未订阅
	noSubApp := f.app("app_no_sub", model.AppStatusEnable)

	// 订阅停用
	subDisabledApp := f.app("app_sub_disabled", model.AppStatusEnable)
	f.subscribe(subDisabledApp, model.TenantApplicationStatusDisable)
	f.grant(subDisabledApp, model.UserApplicationStatusEnable, model.UserApplicationSourceManual)

	// 应用停用
	appDisabled := f.app("app_disabled", model.AppStatusDisable)
	f.subscribe(appDisabled, model.TenantApplicationStatusEnable)
	f.grant(appDisabled, model.UserApplicationStatusEnable, model.UserApplicationSourceManual)

	cases := []struct {
		name     string
		tenantID string
		userID   string
		appID    string
		want     bool
	}{
		{name: "准入+订阅+应用均启用", tenantID: f.tenantID, userID: f.userID, appID: okApp, want: true},
		{name: "无准入行", tenantID: f.tenantID, userID: f.userID, appID: noGrantApp, want: false},
		{name: "准入已撤销", tenantID: f.tenantID, userID: f.userID, appID: revokedApp, want: false},
		{name: "租户未订阅", tenantID: f.tenantID, userID: f.userID, appID: noSubApp, want: false},
		{name: "订阅已停用", tenantID: f.tenantID, userID: f.userID, appID: subDisabledApp, want: false},
		{name: "应用已停用", tenantID: f.tenantID, userID: f.userID, appID: appDisabled, want: false},
		{name: "缺租户", tenantID: "", userID: f.userID, appID: okApp, want: false},
		{name: "缺成员", tenantID: f.tenantID, userID: "", appID: okApp, want: false},
		{name: "缺应用", tenantID: f.tenantID, userID: f.userID, appID: "", want: false},
		{name: "应用不存在", tenantID: f.tenantID, userID: f.userID, appID: "no_such_app", want: false},
		{name: "成员不存在", tenantID: f.tenantID, userID: "no_such_user", appID: okApp, want: false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := CheckUsable(ctx, c.tenantID, c.userID, c.appID)
			if err != nil {
				t.Fatalf("CheckUsable unexpected err: %v", err)
			}
			if got != c.want {
				t.Fatalf("CheckUsable = %v, want %v", got, c.want)
			}
		})
	}
}

// TestCheckUsableTenantIsolation 租户作用域真的生效：A 租户查不到 B 租户的准入。
func TestCheckUsableTenantIsolation(t *testing.T) {
	f := newAccessFixture(t, "tenant-a", "user-1")
	appID := f.app("app_shared", model.AppStatusEnable)
	f.subscribe(appID, model.TenantApplicationStatusEnable)
	f.grant(appID, model.UserApplicationStatusEnable, model.UserApplicationSourceManual)

	got, err := CheckUsable(f.base, f.tenantID, f.userID, appID)
	if err != nil {
		t.Fatalf("CheckUsable(A) err: %v", err)
	}
	if !got {
		t.Fatal("A 租户应可用")
	}

	// B 租户既不共享该订阅也不共享该准入行
	got, err = CheckUsable(f.base, "tenant-b", f.userID, appID)
	if err != nil {
		t.Fatalf("CheckUsable(B) err: %v", err)
	}
	if got {
		t.Fatal("B 租户不得因 A 租户的订阅/准入而放行")
	}
}

// TestEnsureGrantIdempotentAndRevive 不变式 I1 的写入原语：幂等、可复活、空应用不落行。
func TestEnsureGrantIdempotentAndRevive(t *testing.T) {
	f := newAccessFixture(t, "tenant-1", "user-1")
	appID := f.app("app_a", model.AppStatusEnable)
	txErr := dbclient.IamDB(f.base).Transaction(func(tx *gorm.DB) error {
		return EnsureGrant(f.base, tx, GrantParams{
			TenantID: f.tenantID, UserID: f.userID, AppID: appID,
			Source: model.UserApplicationSourceRole, Operator: "op-1",
		})
	})
	if txErr != nil {
		t.Fatalf("EnsureGrant(first) err: %v", txErr)
	}
	first := f.grantOf(appID)
	if first == nil || first.Status != model.UserApplicationStatusEnable || first.Source != model.UserApplicationSourceRole {
		t.Fatalf("首次授权应写入 enable/role，got %+v", first)
	}

	// 幂等：同 source 重复调用不产生第二行、不改状态
	txErr = dbclient.IamDB(f.base).Transaction(func(tx *gorm.DB) error {
		return EnsureGrant(f.base, tx, GrantParams{
			TenantID: f.tenantID, UserID: f.userID, AppID: appID,
			Source: model.UserApplicationSourceRole, Operator: "op-2",
		})
	})
	if txErr != nil {
		t.Fatalf("EnsureGrant(idempotent) err: %v", txErr)
	}
	again := f.grantOf(appID)
	if again.ID != first.ID || again.Status != model.UserApplicationStatusEnable {
		t.Fatalf("重复授权应命中同一行且保持 enable，got %+v", again)
	}

	// 被撤销后再次授权：翻回 enable 并刷新来源（保留原行，便于审计"撤销又恢复"）
	if err := dao.UpdateFields(f.scoped(), dao.NewUserApplicationDao().Dao, first.ID, &model.UserApplicationEntity{
		Status: model.UserApplicationStatusDisable,
	}, "status"); err != nil {
		t.Fatalf("manual revoke: %v", err)
	}
	txErr = dbclient.IamDB(f.base).Transaction(func(tx *gorm.DB) error {
		return EnsureGrant(f.base, tx, GrantParams{
			TenantID: f.tenantID, UserID: f.userID, AppID: appID,
			Source: model.UserApplicationSourceManual, Operator: "op-3",
		})
	})
	if txErr != nil {
		t.Fatalf("EnsureGrant(revive) err: %v", txErr)
	}
	revived := f.grantOf(appID)
	if revived.ID != first.ID || revived.Status != model.UserApplicationStatusEnable || revived.Source != model.UserApplicationSourceManual {
		t.Fatalf("复活应复用原行并置 enable/manual，got %+v", revived)
	}

	// 空 appID（未归属应用组）不得产生准入行
	txErr = dbclient.IamDB(f.base).Transaction(func(tx *gorm.DB) error {
		return EnsureGrant(f.base, tx, GrantParams{
			TenantID: f.tenantID, UserID: f.userID, AppID: "",
			Source: model.UserApplicationSourceRole, Operator: "op-4",
		})
	})
	if txErr != nil {
		t.Fatalf("EnsureGrant(empty app) err: %v", txErr)
	}
	grants, err := dao.NewUserApplicationDao().GetListByCond(f.scoped(), &dao.UserApplicationCond{
		TenantID: f.tenantID, UserID: f.userID,
	})
	if err != nil {
		t.Fatalf("list grants: %v", err)
	}
	if len(grants) != 1 {
		t.Fatalf("空 appID 不应落行，期望仍只有 1 行，got %d", len(grants))
	}
}

// TestReplaceRevokesAndCascades 不变式 I2：撤销准入的应用同步级联撤销该应用下的角色绑定，
// 其他应用的角色绑定与角色行本身不受影响；新增应用以 source=manual 落准入。
func TestReplaceRevokesAndCascades(t *testing.T) {
	f := newAccessFixture(t, "tenant-1", "user-1")
	appA := f.app("app_a", model.AppStatusEnable)
	appB := f.app("app_b", model.AppStatusEnable)
	appC := f.app("app_c", model.AppStatusEnable)
	for _, appID := range []string{appA, appB, appC} {
		f.subscribe(appID, model.TenantApplicationStatusEnable)
	}
	f.grant(appA, model.UserApplicationStatusEnable, model.UserApplicationSourceRole)
	f.grant(appB, model.UserApplicationStatusEnable, model.UserApplicationSourceRole)

	roleA := f.role(appA, model.RoleSourceCustom)
	roleB := f.role(appB, model.RoleSourceCustom)
	f.bindRole(roleA)
	f.bindRole(roleB)

	txErr := dbclient.IamDB(f.base).Transaction(func(tx *gorm.DB) error {
		return Replace(f.base, tx, ReplaceParams{
			TenantID: f.tenantID, UserID: f.userID,
			AppIDs: []string{appB, appC}, Operator: "op-1",
		})
	})
	if txErr != nil {
		t.Fatalf("Replace err: %v", txErr)
	}

	if got := f.grantOf(appA); got.Status != model.UserApplicationStatusDisable {
		t.Fatalf("集合外的 appA 应被置 disable，got %q", got.Status)
	}
	if got := f.grantOf(appB); got.Status != model.UserApplicationStatusEnable {
		t.Fatalf("集合内的 appB 应保持 enable，got %q", got.Status)
	}
	gotC := f.grantOf(appC)
	if gotC == nil || gotC.Status != model.UserApplicationStatusEnable || gotC.Source != model.UserApplicationSourceManual {
		t.Fatalf("集合内新应用应以 enable/manual 落准入，got %+v", gotC)
	}

	// 级联：appA 的角色绑定被撤，appB 的保留
	kept := f.activeRoleIDs()
	if !reflect.DeepEqual(kept, []string{roleB}) {
		t.Fatalf("应只保留 appB 的角色绑定，got %v", kept)
	}
	// 角色行本身不删（准入撤销不销毁角色定义）
	roleEntity, err := dao.NewRoleDao().GetByID(f.scoped(), roleA)
	if err != nil {
		t.Fatalf("load roleA err: %v", err)
	}
	if roleEntity == nil {
		t.Fatal("角色行不应随准入撤销被删除")
	}
}

// TestReplacePreservesDisabledGrantOfKeptApp 集合内应用若此前被撤销（disable 行），应翻回 enable。
func TestReplacePreservesDisabledGrantOfKeptApp(t *testing.T) {
	f := newAccessFixture(t, "tenant-1", "user-1")
	appA := f.app("app_a", model.AppStatusEnable)
	f.subscribe(appA, model.TenantApplicationStatusEnable)
	f.grant(appA, model.UserApplicationStatusDisable, model.UserApplicationSourceRole)

	txErr := dbclient.IamDB(f.base).Transaction(func(tx *gorm.DB) error {
		return Replace(f.base, tx, ReplaceParams{
			TenantID: f.tenantID, UserID: f.userID, AppIDs: []string{appA}, Operator: "op-1",
		})
	})
	if txErr != nil {
		t.Fatalf("Replace err: %v", txErr)
	}
	got := f.grantOf(appA)
	if got.Status != model.UserApplicationStatusEnable || got.Source != model.UserApplicationSourceManual {
		t.Fatalf("集合内应用应翻回 enable/manual，got %+v", got)
	}
}

// TestReplacePrecheckRejectsUnsubscribedWithoutWrites 未订阅的应用混入集合 → 整体拒绝且零写入。
func TestReplacePrecheckRejectsUnsubscribedWithoutWrites(t *testing.T) {
	f := newAccessFixture(t, "tenant-1", "user-1")
	appA := f.app("app_a", model.AppStatusEnable)
	appB := f.app("app_b", model.AppStatusEnable)
	f.subscribe(appA, model.TenantApplicationStatusEnable)
	// appB 未订阅
	f.grant(appA, model.UserApplicationStatusEnable, model.UserApplicationSourceRole)
	f.bindRole(f.role(appA, model.RoleSourceCustom))

	before := f.stateSnapshot()
	txErr := dbclient.IamDB(f.base).Transaction(func(tx *gorm.DB) error {
		return Replace(f.base, tx, ReplaceParams{
			TenantID: f.tenantID, UserID: f.userID,
			AppIDs: []string{appA, appB}, Operator: "op-1",
		})
	})
	if !errors.Is(txErr, ErrAppNotSubscribed) {
		t.Fatalf("期望 ErrAppNotSubscribed，got %v", txErr)
	}
	if after := f.stateSnapshot(); after != before {
		t.Fatalf("预检失败必须零写入\nbefore: %s\nafter:  %s", before, after)
	}
}

// TestReplaceLastAdminForbiddenWithoutWrites 防自锁（I5）：回调判定为最后管理员 → 整体拒绝且零写入。
func TestReplaceLastAdminForbiddenWithoutWrites(t *testing.T) {
	f := newAccessFixture(t, "tenant-1", "user-1")
	appA := f.app("app_a", model.AppStatusEnable)
	f.subscribe(appA, model.TenantApplicationStatusEnable)
	f.grant(appA, model.UserApplicationStatusEnable, model.UserApplicationSourceProvision)
	roleA := f.role(appA, model.RoleSourceCustom)
	f.bindRole(roleA)

	before := f.stateSnapshot()
	var keptSeen []string
	txErr := dbclient.IamDB(f.base).Transaction(func(tx *gorm.DB) error {
		return Replace(f.base, tx, ReplaceParams{
			TenantID: f.tenantID, UserID: f.userID,
			AppIDs: nil, Operator: "op-1",
			CheckLastAdmin: func(_ context.Context, keptRoleIDs []string) (bool, error) {
				keptSeen = keptRoleIDs
				return true, nil // 模拟"撤掉这个应用后就没人能进控制台了"
			},
		})
	})
	if !errors.Is(txErr, ErrLastAdminForbidden) {
		t.Fatalf("期望 ErrLastAdminForbidden，got %v", txErr)
	}
	if len(keptSeen) != 0 {
		t.Fatalf("撤销 appA 后不应保留任何角色，回调收到 %v", keptSeen)
	}
	if after := f.stateSnapshot(); after != before {
		t.Fatalf("防自锁命中必须零写入\nbefore: %s\nafter:  %s", before, after)
	}
}

// TestReplaceCallbackSeesKeptRoles 回调拿到的"保留角色集合"必须已排除随撤销被删的角色。
func TestReplaceCallbackSeesKeptRoles(t *testing.T) {
	f := newAccessFixture(t, "tenant-1", "user-1")
	appA := f.app("app_a", model.AppStatusEnable)
	appB := f.app("app_b", model.AppStatusEnable)
	f.subscribe(appA, model.TenantApplicationStatusEnable)
	f.subscribe(appB, model.TenantApplicationStatusEnable)
	f.grant(appA, model.UserApplicationStatusEnable, model.UserApplicationSourceRole)
	f.grant(appB, model.UserApplicationStatusEnable, model.UserApplicationSourceRole)
	roleA := f.role(appA, model.RoleSourceCustom)
	roleB := f.role(appB, model.RoleSourceCustom)
	f.bindRole(roleA)
	f.bindRole(roleB)

	var keptSeen []string
	txErr := dbclient.IamDB(f.base).Transaction(func(tx *gorm.DB) error {
		return Replace(f.base, tx, ReplaceParams{
			TenantID: f.tenantID, UserID: f.userID,
			AppIDs: []string{appB}, Operator: "op-1",
			CheckLastAdmin: func(_ context.Context, keptRoleIDs []string) (bool, error) {
				keptSeen = keptRoleIDs
				return false, nil
			},
		})
	})
	if txErr != nil {
		t.Fatalf("Replace err: %v", txErr)
	}
	if !reflect.DeepEqual(keptSeen, []string{roleB}) {
		t.Fatalf("回调应只看到保留的 roleB，got %v", keptSeen)
	}
}

// TestListAccessUnionView 管理端视图：订阅与准入的并集，且如实标注可用性。
func TestListAccessUnionView(t *testing.T) {
	f := newAccessFixture(t, "tenant-1", "user-1")
	usableApp := f.app("app_usable", model.AppStatusEnable)
	subOnlyApp := f.app("app_sub_only", model.AppStatusEnable)
	unsubscribedApp := f.app("app_unsubscribed", model.AppStatusEnable)

	f.subscribe(usableApp, model.TenantApplicationStatusEnable)
	f.subscribe(subOnlyApp, model.TenantApplicationStatusEnable)
	f.grant(usableApp, model.UserApplicationStatusEnable, model.UserApplicationSourceManual)
	f.grant(unsubscribedApp, model.UserApplicationStatusEnable, model.UserApplicationSourceRole)

	rows, err := ListAccess(f.base, f.tenantID, f.userID)
	if err != nil {
		t.Fatalf("ListAccess err: %v", err)
	}
	byApp := make(map[string]Access, len(rows))
	for _, row := range rows {
		byApp[row.App.ID] = row
	}
	if len(byApp) != 3 {
		t.Fatalf("期望订阅 2 + 有准入但已无订阅 1 = 3 行，got %d", len(byApp))
	}
	if !byApp[usableApp].Usable() {
		t.Fatal("usableApp 应可用")
	}
	if byApp[subOnlyApp].Usable() {
		t.Fatal("只有订阅没有准入的应用不可用（无角色应用也必须有准入行）")
	}
	if byApp[unsubscribedApp].Usable() {
		t.Fatal("订阅已消失时，残留准入行不得判为可用")
	}
	if byApp[unsubscribedApp].Grant == nil {
		t.Fatal("残留准入行必须如实呈现，否则管理端看不到真相")
	}
}

// TestListAccessEmptyInput 缺参返回空表而非报错（可预期边界）。
func TestListAccessEmptyInput(t *testing.T) {
	f := newAccessFixture(t, "tenant-1", "user-1")
	rows, err := ListAccess(f.base, "", f.userID)
	if err != nil {
		t.Fatalf("ListAccess err: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("缺租户应返回空表，got %d", len(rows))
	}
}
