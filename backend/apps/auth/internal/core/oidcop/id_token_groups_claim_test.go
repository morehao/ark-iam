package oidcop

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/testsetup"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/oidc/v3/pkg/op"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newRoleClaimTestStore 构造仅注入角色相关 DAO 的 persistent store（groups 声明专项测试用）。
// 与 newProtocolConformanceStore 同构，但不依赖 person/client 等其它表。
func newRoleClaimTestStore(t *testing.T) (*PersistentStore, *gorm.DB) {
	t.Helper()
	dsn := fmt.Sprintf("file:groups_claim_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.RoleEntity{}, &model.UserRoleEntity{}, &model.UserEntity{}, &model.PersonEntity{}, &model.ApplicationClientEntity{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	dbGetter := func(ctx context.Context) *gorm.DB { return db.WithContext(ctx) }
	ps := NewPersistentStore()
	ps.personDao = func(opts ...dao.DaoOption) *dao.PersonDao {
		return dao.NewPersonDao(dao.WithDBGetter(dbGetter))
	}
	ps.roleDao = func(opts ...dao.DaoOption) *dao.RoleDao {
		return dao.NewRoleDao(dao.WithDBGetter(dbGetter))
	}
	ps.userRoleDao = func(opts ...dao.DaoOption) *dao.UserRoleDao {
		return dao.NewUserRoleDao(dao.WithDBGetter(dbGetter))
	}
	ps.userDao = func(opts ...dao.DaoOption) *dao.UserDao {
		return dao.NewUserDao(dao.WithDBGetter(dbGetter))
	}
	// groups 作用域按 client → app 解析，故测试库必须能查 application_client。
	ps.applicationClientDao = func(opts ...dao.DaoOption) *dao.ApplicationClientDao {
		return dao.NewApplicationClientDao(dao.WithDBGetter(dbGetter))
	}
	ps.db = dbGetter
	return ps, db
}

// seedTenantMember 播种租户成员（tenant_user）：注意其主键与自然人 ID **不同**，
// user_role.user_id 引用的是成员主键——同 ID 的夹具会让「按 personID 查 user_role」的错误实现也能通过。
func seedTenantMember(t *testing.T, db *gorm.DB, id, tenantID, personID string) {
	t.Helper()
	if err := db.Create(&model.UserEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: id}},
		TenantID:   tenantID,
		PersonID:   personID,
		UserType:   model.UserTypeMember,
		Source:     model.UserSourceBuiltin,
		Name:       "成员-" + id,
		Profile:    json.RawMessage(`{}`),
		CustomData: json.RawMessage(`{}`),
	}).Error; err != nil {
		t.Fatalf("seed tenant_user: %v", err)
	}
}

// seedRoleWithCode 播种角色，appID 即该角色所属应用——groups 的作用域键：
// 只有与「请求客户端所属应用」一致的角色才会进入声明。
func seedRoleWithCode(t *testing.T, db *gorm.DB, id, tenantID, appID string, roleCode model.RoleCode) {
	t.Helper()
	if err := db.Create(&model.RoleEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: id}},
		TenantID:   tenantID,
		AppID:      appID,
		Code:       roleCode,
		Name:       "角色-" + id,
		CreatedBy:  "t",
	}).Error; err != nil {
		t.Fatalf("seed role: %v", err)
	}
}

// seedAppClient 播种 OIDC 客户端并绑定到 appID：groups 的作用域由 client → app 解析得到。
func seedAppClient(t *testing.T, db *gorm.DB, id, tenantID, appID, code string) {
	t.Helper()
	if err := db.Create(&model.ApplicationClientEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: id}},
		TenantID:   tenantID,
		AppID:      appID,
		Code:       code,
		Name:       "客户端-" + code,
		Source:     model.ApplicationClientSourceThirdParty,
		Status:     model.ApplicationClientStatusEnable,
	}).Error; err != nil {
		t.Fatalf("seed application_client: %v", err)
	}
}

func seedUserRoleBinding(t *testing.T, db *gorm.DB, id, tenantID, userID, roleID string) {
	t.Helper()
	if err := db.Create(&model.UserRoleEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: id}},
		TenantID:   tenantID,
		UserID:     userID,
		RoleID:     roleID,
		CreatedBy:  "t",
	}).Error; err != nil {
		t.Fatalf("seed user_role: %v", err)
	}
}

// seedPersonForUserinfo 播种自然人（userinfo 端点回查 person 以填充 profile/email 声明）。
func seedPersonForUserinfo(t *testing.T, db *gorm.DB, id string) {
	t.Helper()
	if err := db.Create(&model.PersonEntity{
		BaseEntity:        gormdao.BaseEntity{StringID: gormdao.StringID{ID: id}},
		Name:              "自然人-" + id,
		Username:          model.StrPtr("user-" + id),
		PrimaryEmail:      model.StrPtr(id + "@example.com"),
		Profile:           json.RawMessage(`{}`),
		CustomData:        json.RawMessage(`{}`),
		PasswordEncrypted: "hash",
	}).Error; err != nil {
		t.Fatalf("seed person: %v", err)
	}
}

func groupsClaimOf(t *testing.T, userinfo *oidc.UserInfo) []string {
	t.Helper()
	if userinfo.Claims == nil {
		return nil
	}
	raw, ok := userinfo.Claims[ClaimGroups]
	if !ok {
		return nil
	}
	codes, ok := raw.([]string)
	if !ok {
		t.Fatalf("groups claim must be []string, got %T", raw)
	}
	return codes
}

// TestAppendRoleGroupClaimsScopedByTenantProfileAndApp groups 是跨系统授权契约（下游按编码认策略名）：
// 只取「本次签发租户 × 请求客户端所属应用」的角色编码、只在 profile scope 下产出、空编码存量行跳过、
// 顺序稳定；**跨应用的角色一律不进声明**（否则任何应用的角色都能命中下游策略名）。
func TestAppendRoleGroupClaimsScopedByTenantProfileAndApp(t *testing.T) {
	ps, db := newRoleClaimTestStore(t)
	subject := BuildSubject("g1")

	// 同一自然人在两个租户各有角色；t1 内还含空编码存量行与一条历史重复编码角色。
	// 成员主键刻意与自然人 ID 不同：user_role.user_id 引用的是成员主键（tenant_user.id）。
	seedTenantMember(t, db, "u-t1-g1", "t1", "g1")
	seedTenantMember(t, db, "u-t2-g1", "t2", "g1")
	// t1 内两个应用各有角色：只有与请求客户端所属应用一致的那些才进声明。
	seedRoleWithCode(t, db, "role-t1-a", "t1", "app-store", model.RoleCodePlatformAdmin)
	seedRoleWithCode(t, db, "role-t1-b", "t1", "app-store", "storage_readonly")
	seedRoleWithCode(t, db, "role-t1-empty", "t1", "app-store", "")
	seedRoleWithCode(t, db, "role-t1-dup", "t1", "app-store", model.RoleCodePlatformAdmin) // 存量重码：声明里只能出现一次
	seedRoleWithCode(t, db, "role-t1-console", "t1", "app-console", "console_only")
	seedRoleWithCode(t, db, "role-t1-system", "t1", "", "unassigned_role") // 未归属应用：无下游命名空间
	seedRoleWithCode(t, db, "role-t2", "t2", "app-store", "other_tenant_role")
	seedAppClient(t, db, "cli-store", "t1", "app-store", "store_console")
	seedAppClient(t, db, "cli-console", "t1", "app-console", "console_web")
	seedUserRoleBinding(t, db, "ur1", "t1", "u-t1-g1", "role-t1-a")
	seedUserRoleBinding(t, db, "ur2", "t1", "u-t1-g1", "role-t1-b")
	seedUserRoleBinding(t, db, "ur3", "t1", "u-t1-g1", "role-t1-empty")
	seedUserRoleBinding(t, db, "ur4", "t1", "u-t1-g1", "role-t1-dup")
	seedUserRoleBinding(t, db, "ur5", "t2", "u-t2-g1", "role-t2")
	seedUserRoleBinding(t, db, "ur8", "t1", "u-t1-g1", "role-t1-console")
	seedUserRoleBinding(t, db, "ur9", "t1", "u-t1-g1", "role-t1-system")

	ctx := context.Background()

	// 只发请求客户端所属应用（app-store）的角色：app-console 的 console_only 与未归属应用的
	// unassigned_role 都不得出现——否则在无关应用里造一个同码角色即可命中下游策略（跨应用越权）。
	userinfo := &oidc.UserInfo{}
	if err := ps.appendRoleGroupClaims(ctx, userinfo, "t1", "store_console", subject, []string{oidc.ScopeOpenID, oidc.ScopeProfile}); err != nil {
		t.Fatalf("appendRoleGroupClaims failed: %v", err)
	}
	want := []string{string(model.RoleCodePlatformAdmin), "storage_readonly"} // 稳定升序
	if got := groupsClaimOf(t, userinfo); !reflect.DeepEqual(got, want) {
		t.Fatalf("expected groups %v, got %v", want, got)
	}

	// 换成另一个应用下的客户端：声明随请求方所属应用切换
	userinfo = &oidc.UserInfo{}
	if err := ps.appendRoleGroupClaims(ctx, userinfo, "t1", "console_web", subject, []string{oidc.ScopeOpenID, oidc.ScopeProfile}); err != nil {
		t.Fatalf("appendRoleGroupClaims failed: %v", err)
	}
	if got := groupsClaimOf(t, userinfo); !reflect.DeepEqual(got, []string{"console_only"}) {
		t.Fatalf("expected groups scoped to requested app, got %v", got)
	}

	// 客户端未知：作用域不可判定，fail-closed 报错（不签发作用域不明的声明）
	userinfo = &oidc.UserInfo{}
	if err := ps.appendRoleGroupClaims(ctx, userinfo, "t1", "no_such_client", subject, []string{oidc.ScopeOpenID, oidc.ScopeProfile}); err == nil {
		t.Fatal("expected error for unknown client (fail-closed)")
	}

	// 客户端为空：无命名空间，静默不产出（与「租户未定不猜租户」同构）
	userinfo = &oidc.UserInfo{}
	if err := ps.appendRoleGroupClaims(ctx, userinfo, "t1", "", subject, []string{oidc.ScopeOpenID, oidc.ScopeProfile}); err != nil {
		t.Fatalf("appendRoleGroupClaims failed: %v", err)
	}
	if got := groupsClaimOf(t, userinfo); got != nil {
		t.Fatalf("expected no groups without client, got %v", got)
	}

	// 未请求 profile：不产出 groups（授权声明随 profile scope 授权）
	userinfo = &oidc.UserInfo{}
	if err := ps.appendRoleGroupClaims(ctx, userinfo, "t1", "store_console", subject, []string{oidc.ScopeOpenID}); err != nil {
		t.Fatalf("appendRoleGroupClaims failed: %v", err)
	}
	if got := groupsClaimOf(t, userinfo); got != nil {
		t.Fatalf("expected no groups without profile scope, got %v", got)
	}

	// 租户未知：不猜租户、不跨租户兜底
	userinfo = &oidc.UserInfo{}
	if err := ps.appendRoleGroupClaims(ctx, userinfo, "", "store_console", subject, []string{oidc.ScopeOpenID, oidc.ScopeProfile}); err != nil {
		t.Fatalf("appendRoleGroupClaims failed: %v", err)
	}
	if got := groupsClaimOf(t, userinfo); got != nil {
		t.Fatalf("expected no groups without tenant, got %v", got)
	}

	// 该租户内无绑定：不产出空数组（空数组会让下游误判为"有 group 但无策略"）；
	// t3 里该自然人有成员行但没有角色绑定，覆盖「成员存在、角色为空」这一支。
	seedTenantMember(t, db, "u-t3-g1", "t3", "g1")
	seedUserRoleBinding(t, db, "ur6", "t3", "u-t3-g1", "role-t3-other")
	userinfo = &oidc.UserInfo{}
	if err := ps.appendRoleGroupClaims(ctx, userinfo, "t3", "store_console", subject, []string{oidc.ScopeOpenID, oidc.ScopeProfile}); err != nil {
		t.Fatalf("appendRoleGroupClaims failed: %v", err)
	}
	if got := groupsClaimOf(t, userinfo); got != nil {
		t.Fatalf("expected no groups for tenant without roles, got %v", got)
	}

	// 非该租户成员（无 tenant_user 行）：不产出 groups，也不跨租户兜底
	seedUserRoleBinding(t, db, "ur7", "t4", "u-t4-g1", "role-t1-a")
	userinfo = &oidc.UserInfo{}
	if err := ps.appendRoleGroupClaims(ctx, userinfo, "t4", "store_console", subject, []string{oidc.ScopeOpenID, oidc.ScopeProfile}); err != nil {
		t.Fatalf("appendRoleGroupClaims failed: %v", err)
	}
	if got := groupsClaimOf(t, userinfo); got != nil {
		t.Fatalf("expected no groups for non-member, got %v", got)
	}
}

// TestSetUserinfoFromRequestCarriesGroupsClaim 走 op.CanSetUserinfoFromRequest 这条真实回调：
// 授权码流（*AuthRequest）与刷新流（*refreshTokenRequest）都要按各自的 client 裁剪出 groups，
// 且 sid 注入不被破坏。
func TestSetUserinfoFromRequestCarriesGroupsClaim(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)

	ps, db := newRoleClaimTestStore(t)
	seedTenantMember(t, db, "u-t1-g1", "t1", "g1")
	seedRoleWithCode(t, db, "role-t1", "t1", "app-store", model.RoleCodePlatformAdmin)
	seedRoleWithCode(t, db, "role-t1-b", "t1", "app-console", "console_only")
	seedAppClient(t, db, "cli-store", "t1", "app-store", "store_console")
	seedAppClient(t, db, "cli-console", "t1", "app-console", "console_web")
	seedUserRoleBinding(t, db, "ur1", "t1", "u-t1-g1", "role-t1")
	seedUserRoleBinding(t, db, "ur2", "t1", "u-t1-g1", "role-t1-b")

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	storage := NewOIDCStorage(NewRedisProtocolStateStore(), ps, privateKey, "test-key")
	ctx := context.Background()
	scopes := []string{oidc.ScopeOpenID, oidc.ScopeProfile}

	// 授权码流：client 决定作用域，store_console 只看得到 app-store 的角色
	authReq := &AuthRequest{ClientID: "store_console", TenantID: "t1", Subject: BuildSubject("g1"), SessionID: "sid-1"}
	userinfo := &oidc.UserInfo{}
	if err := storage.SetUserinfoFromRequest(ctx, userinfo, authReq, scopes); err != nil {
		t.Fatalf("SetUserinfoFromRequest(auth request) failed: %v", err)
	}
	if got := groupsClaimOf(t, userinfo); !reflect.DeepEqual(got, []string{string(model.RoleCodePlatformAdmin)}) {
		t.Fatalf("expected groups from auth request, got %v", got)
	}
	if userinfo.Claims["sid"] != "sid-1" {
		t.Fatalf("expected sid injected, got %v", userinfo.Claims["sid"])
	}

	// 刷新流：作用域同样跟着 refresh token 的 client 走（console_web → app-console）
	refreshReq := &refreshTokenRequest{subject: BuildSubject("g1"), tenantID: "t1", sessionID: "sid-2", clientID: "console_web"}
	userinfo = &oidc.UserInfo{}
	if err := storage.SetUserinfoFromRequest(ctx, userinfo, refreshReq, scopes); err != nil {
		t.Fatalf("SetUserinfoFromRequest(refresh request) failed: %v", err)
	}
	if got := groupsClaimOf(t, userinfo); !reflect.DeepEqual(got, []string{"console_only"}) {
		t.Fatalf("expected groups scoped to refresh client's app, got %v", got)
	}
	if userinfo.Claims["sid"] != "sid-2" {
		t.Fatalf("expected sid injected for refresh flow, got %v", userinfo.Claims["sid"])
	}

	// 无 sessionID 时不得因为 sid 为空而跳过后面的 groups 注入
	authReqNoSession := &AuthRequest{ClientID: "store_console", TenantID: "t1", Subject: BuildSubject("g1")}
	userinfo = &oidc.UserInfo{}
	if err := storage.SetUserinfoFromRequest(ctx, userinfo, authReqNoSession, scopes); err != nil {
		t.Fatalf("SetUserinfoFromRequest(no session) failed: %v", err)
	}
	if got := groupsClaimOf(t, userinfo); len(got) != 1 {
		t.Fatalf("expected groups even without session id, got %v", got)
	}
}

// TestIDTokenUserinfoClaimsAssertionEnabled ID token 必须合并 userinfo 声明：
// 为 false 时 zitadel 会把 profile/email/phone 从 ID token 的 scope 里裁掉（removeUserinfoScopes），
// 只读 ID token 的 RP（如对象存储控制台）将拿不到 groups/name/preferred_username。
func TestIDTokenUserinfoClaimsAssertionEnabled(t *testing.T) {
	client := &OIDCClient{}
	if !client.IDTokenUserinfoClaimsAssertion() {
		t.Fatal("IDTokenUserinfoClaimsAssertion 必须为 true，否则 profile/groups 声明会被裁掉")
	}
}

// TestTenantIDFromRequestCoversUserFlows access token 元数据的 tenant_id 必须覆盖「人」的三条签发路径
// （授权码 / 刷新 / client_credentials）：userinfo 的 groups 与 introspection 的 tenant_id 都只从元数据取租户，
// 只认 client_credentials 会让授权码流的 token 在 userinfo 上恒缺 groups（已由端到端实测暴露）。
func TestTenantIDFromRequestCoversUserFlows(t *testing.T) {
	cases := []struct {
		name    string
		request op.TokenRequest
		want    string
	}{
		{"授权码流", &AuthRequest{TenantID: "t1", Subject: BuildSubject("g1")}, "t1"},
		{"刷新流", &refreshTokenRequest{subject: BuildSubject("g1"), tenantID: "t2"}, "t2"},
		{"客户端凭证（API Key 所属租户）", &clientCredentialsTokenRequest{ownerTenantID: "t3"}, "t3"},
		// 未定租户的请求不得猜租户：产出空 tenant_id 由调用方按 fail-closed 处理
		{"未定租户的授权码流", &AuthRequest{Subject: BuildSubject("g1")}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := tenantIDFromRequest(c.request); got != c.want {
				t.Fatalf("expected tenantID %q, got %q", c.want, got)
			}
		})
	}
}

// TestSetUserinfoFromTokenCarriesGroupsClaim userinfo 端点（Bearer access token 回查）同样要产出 groups：
// 端点没有「当前租户」，租户只能来自 access token 元数据，作用域还要叠加元数据里的 client → app——
// 元数据缺租户即静默降权，必须由本测试守住；换客户端则声明随之切换。
func TestSetUserinfoFromTokenCarriesGroupsClaim(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)

	ps, db := newRoleClaimTestStore(t)
	seedPersonForUserinfo(t, db, "g1")
	seedTenantMember(t, db, "u-t1-g1", "t1", "g1")
	seedRoleWithCode(t, db, "role-t1", "t1", "app-store", model.RoleCodePlatformAdmin)
	seedRoleWithCode(t, db, "role-t1-console", "t1", "app-console", "console_only")
	seedUserRoleBinding(t, db, "ur1", "t1", "u-t1-g1", "role-t1")
	seedUserRoleBinding(t, db, "ur2", "t1", "u-t1-g1", "role-t1-console")
	// rustfs_console 属 app-store；console_web 属 app-console；app-empty 下无角色。
	seedAppClient(t, db, "cli-store", "t1", "app-store", "rustfs_console")
	seedAppClient(t, db, "cli-console", "t1", "app-console", "console_web")
	seedAppClient(t, db, "cli-empty", "t1", "app-empty", "empty_console")

	ctx := context.Background()
	now := time.Now()

	for _, c := range []struct {
		name     string
		tenantID string
		clientID string
		want     []string
	}{
		{"元数据带租户+客户端所属应用", "t1", "rustfs_console", []string{string(model.RoleCodePlatformAdmin)}},
		{"换应用下的客户端：只发该应用的角色", "t1", "console_web", []string{"console_only"}},
		{"客户端所属应用无角色：不产出空数组", "t1", "empty_console", nil},
		{"元数据缺租户", "", "rustfs_console", nil},
	} {
		t.Run(c.name, func(t *testing.T) {
			tokenID := "at-" + c.tenantID + "-" + c.clientID
			storeAccessTokenMeta(ctx, tokenID, accessTokenMeta{
				Subject:   BuildSubject("g1"),
				ClientID:  c.clientID,
				Scopes:    []string{oidc.ScopeOpenID, oidc.ScopeProfile},
				IssuedAt:  now,
				ExpiresAt: now.Add(time.Hour),
				TenantID:  c.tenantID,
			})
			userinfo := &oidc.UserInfo{}
			if err := ps.SetUserinfoFromToken(ctx, userinfo, tokenID, BuildSubject("g1"), ""); err != nil {
				t.Fatalf("SetUserinfoFromToken failed: %v", err)
			}
			if got := groupsClaimOf(t, userinfo); !reflect.DeepEqual(got, c.want) {
				t.Fatalf("expected groups %v, got %v", c.want, got)
			}
		})
	}
}
