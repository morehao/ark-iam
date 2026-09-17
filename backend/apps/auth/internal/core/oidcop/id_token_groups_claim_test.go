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
	if err := db.AutoMigrate(&model.RoleEntity{}, &model.UserRoleEntity{}, &model.UserEntity{}, &model.PersonEntity{}); err != nil {
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

func seedRoleWithCode(t *testing.T, db *gorm.DB, id, tenantID string, roleCode model.RoleCode) {
	t.Helper()
	if err := db.Create(&model.RoleEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: id}},
		TenantID:   tenantID,
		Code:       roleCode,
		Name:       "角色-" + id,
		CreatedBy:  "t",
	}).Error; err != nil {
		t.Fatalf("seed role: %v", err)
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

// TestAppendRoleGroupClaimsScopedByTenantAndProfile groups 是跨系统授权契约（下游按编码认策略名）：
// 只取本次签发租户的角色编码、只在 profile scope 下产出、空编码存量行跳过、顺序稳定。
func TestAppendRoleGroupClaimsScopedByTenantAndProfile(t *testing.T) {
	ps, db := newRoleClaimTestStore(t)
	subject := BuildSubject("g1")

	// 同一自然人在两个租户各有角色；t1 内还含空编码存量行与一条历史重复编码角色。
	// 成员主键刻意与自然人 ID 不同：user_role.user_id 引用的是成员主键（tenant_user.id）。
	seedTenantMember(t, db, "u-t1-g1", "t1", "g1")
	seedTenantMember(t, db, "u-t2-g1", "t2", "g1")
	seedRoleWithCode(t, db, "role-t1-a", "t1", model.RoleCodePlatformAdmin)
	seedRoleWithCode(t, db, "role-t1-b", "t1", "storage_readonly")
	seedRoleWithCode(t, db, "role-t1-empty", "t1", "")
	seedRoleWithCode(t, db, "role-t1-dup", "t1", model.RoleCodePlatformAdmin) // 存量重码：声明里只能出现一次
	seedRoleWithCode(t, db, "role-t2", "t2", "other_tenant_role")
	seedUserRoleBinding(t, db, "ur1", "t1", "u-t1-g1", "role-t1-a")
	seedUserRoleBinding(t, db, "ur2", "t1", "u-t1-g1", "role-t1-b")
	seedUserRoleBinding(t, db, "ur3", "t1", "u-t1-g1", "role-t1-empty")
	seedUserRoleBinding(t, db, "ur4", "t1", "u-t1-g1", "role-t1-dup")
	seedUserRoleBinding(t, db, "ur5", "t2", "u-t2-g1", "role-t2")

	ctx := context.Background()

	userinfo := &oidc.UserInfo{}
	if err := ps.appendRoleGroupClaims(ctx, userinfo, "t1", subject, []string{oidc.ScopeOpenID, oidc.ScopeProfile}); err != nil {
		t.Fatalf("appendRoleGroupClaims failed: %v", err)
	}
	want := []string{string(model.RoleCodePlatformAdmin), "storage_readonly"} // 稳定升序
	if got := groupsClaimOf(t, userinfo); !reflect.DeepEqual(got, want) {
		t.Fatalf("expected groups %v, got %v", want, got)
	}

	// 未请求 profile：不产出 groups（授权声明随 profile scope 授权）
	userinfo = &oidc.UserInfo{}
	if err := ps.appendRoleGroupClaims(ctx, userinfo, "t1", subject, []string{oidc.ScopeOpenID}); err != nil {
		t.Fatalf("appendRoleGroupClaims failed: %v", err)
	}
	if got := groupsClaimOf(t, userinfo); got != nil {
		t.Fatalf("expected no groups without profile scope, got %v", got)
	}

	// 租户未知：不猜租户、不跨租户兜底
	userinfo = &oidc.UserInfo{}
	if err := ps.appendRoleGroupClaims(ctx, userinfo, "", subject, []string{oidc.ScopeOpenID, oidc.ScopeProfile}); err != nil {
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
	if err := ps.appendRoleGroupClaims(ctx, userinfo, "t3", subject, []string{oidc.ScopeOpenID, oidc.ScopeProfile}); err != nil {
		t.Fatalf("appendRoleGroupClaims failed: %v", err)
	}
	if got := groupsClaimOf(t, userinfo); got != nil {
		t.Fatalf("expected no groups for tenant without roles, got %v", got)
	}

	// 非该租户成员（无 tenant_user 行）：不产出 groups，也不跨租户兜底
	seedUserRoleBinding(t, db, "ur7", "t4", "u-t4-g1", "role-t1-a")
	userinfo = &oidc.UserInfo{}
	if err := ps.appendRoleGroupClaims(ctx, userinfo, "t4", subject, []string{oidc.ScopeOpenID, oidc.ScopeProfile}); err != nil {
		t.Fatalf("appendRoleGroupClaims failed: %v", err)
	}
	if got := groupsClaimOf(t, userinfo); got != nil {
		t.Fatalf("expected no groups for non-member, got %v", got)
	}
}

// TestSetUserinfoFromRequestCarriesGroupsClaim 走 op.CanSetUserinfoFromRequest 这条真实回调：
// 授权码流（*AuthRequest）与刷新流（*refreshTokenRequest）都要带上 groups，且 sid 注入不被破坏。
func TestSetUserinfoFromRequestCarriesGroupsClaim(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)

	ps, db := newRoleClaimTestStore(t)
	seedTenantMember(t, db, "u-t1-g1", "t1", "g1")
	seedRoleWithCode(t, db, "role-t1", "t1", model.RoleCodePlatformAdmin)
	seedUserRoleBinding(t, db, "ur1", "t1", "u-t1-g1", "role-t1")

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	storage := NewOIDCStorage(NewRedisProtocolStateStore(), ps, privateKey, "test-key")
	ctx := context.Background()
	scopes := []string{oidc.ScopeOpenID, oidc.ScopeProfile}

	// 授权码流
	authReq := &AuthRequest{TenantID: "t1", Subject: BuildSubject("g1"), SessionID: "sid-1"}
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

	// 刷新流
	refreshReq := &refreshTokenRequest{subject: BuildSubject("g1"), tenantID: "t1", sessionID: "sid-2"}
	userinfo = &oidc.UserInfo{}
	if err := storage.SetUserinfoFromRequest(ctx, userinfo, refreshReq, scopes); err != nil {
		t.Fatalf("SetUserinfoFromRequest(refresh request) failed: %v", err)
	}
	if got := groupsClaimOf(t, userinfo); !reflect.DeepEqual(got, []string{string(model.RoleCodePlatformAdmin)}) {
		t.Fatalf("expected groups from refresh request, got %v", got)
	}
	if userinfo.Claims["sid"] != "sid-2" {
		t.Fatalf("expected sid injected for refresh flow, got %v", userinfo.Claims["sid"])
	}

	// 无 sessionID 时不得因为 sid 为空而跳过后面的 groups 注入
	authReqNoSession := &AuthRequest{TenantID: "t1", Subject: BuildSubject("g1")}
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
// 端点没有「当前租户」，租户只能来自 access token 元数据——元数据缺租户即静默降权，必须由本测试守住。
func TestSetUserinfoFromTokenCarriesGroupsClaim(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)

	ps, db := newRoleClaimTestStore(t)
	seedPersonForUserinfo(t, db, "g1")
	seedTenantMember(t, db, "u-t1-g1", "t1", "g1")
	seedRoleWithCode(t, db, "role-t1", "t1", model.RoleCodePlatformAdmin)
	seedUserRoleBinding(t, db, "ur1", "t1", "u-t1-g1", "role-t1")

	ctx := context.Background()
	now := time.Now()

	for _, c := range []struct {
		name     string
		tenantID string
		want     []string
	}{
		{"元数据带租户", "t1", []string{string(model.RoleCodePlatformAdmin)}},
		{"元数据缺租户", "", nil},
	} {
		t.Run(c.name, func(t *testing.T) {
			tokenID := "at-" + c.tenantID + "x"
			storeAccessTokenMeta(ctx, tokenID, accessTokenMeta{
				Subject:   BuildSubject("g1"),
				ClientID:  "rustfs_console",
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
