package svcoidc

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	appconfig "github.com/morehao/ark-iam/auth/config"
	"github.com/morehao/ark-iam/auth/internal/core/oidcop"
	"github.com/morehao/ark-iam/auth/internal/dto/dtooidc"
	"github.com/morehao/ark-iam/pkg/code"
	pkgconfig "github.com/morehao/ark-iam/pkg/config"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/object/objauth"
	"github.com/morehao/ark-iam/pkg/testsetup"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/stretchr/testify/require"
)

// 登录期准入读路径（AC-3/AC-4/AC-5）跑在真实资源上（同包既有用例口径：
// testsetup 初始化 Postgres/Redis，SetupOIDCProvider 按配置装配持久化存储）。
// 门禁开关经配置项 oidc.enforceAppAccess 注入，测试因此同时覆盖 P5-2 的装配。

// appAccessGateEnv 门禁开启的登录测试环境。
type appAccessGateEnv struct {
	provider *OIDCProvider
	ctx      *gin.Context
}

// newAppAccessGateEnv 初始化环境并返回**门禁开启**的 provider。
func newAppAccessGateEnv(t *testing.T) *appAccessGateEnv {
	t.Helper()
	testsetup.Initialize(testsetup.AppNameAuth)
	t.Cleanup(func() { testsetup.Done(testsetup.AppNameAuth) })
	return newAppAccessGateEnvWithFlag(t, true)
}

// newAppAccessGateEnvWithFlag 按指定开关值重建 provider（开关只在装配时读取一次）。
func newAppAccessGateEnvWithFlag(t *testing.T, enforce bool) *appAccessGateEnv {
	t.Helper()
	// 门禁用例跑在真实库上（同包既有用例口径）。首次跑到的库可能还没有 user_application 表
	// （该表随本需求新增，dev 库多为升级前建的），故显式补一次 AutoMigrate——
	// 与生产启动同一条路径（model.AutoMigrateAll），只增不删且幂等。
	migrateCtx := dbclient.CrossTenantContext(context.Background())
	require.NoError(t, model.AutoMigrateAll(dbclient.IamDB(migrateCtx).WithContext(migrateCtx)))
	appconfig.Conf = &pkgconfig.Config{
		JWT: pkgconfig.JWT{SignKey: "test-sign-key"},
		OIDC: pkgconfig.OIDC{
			Issuer:           "http://localhost:8099/oidc",
			FrontendLoginURL: "http://localhost:4000/oidc/login",
			AllowInsecure:    true,
			EnforceAppAccess: enforce,
		},
	}
	provider, err := SetupOIDCProvider()
	require.NoError(t, err)
	return &appAccessGateEnv{provider: provider, ctx: testsetup.NewCtx()}
}

// appAccessSeed 一套「应用 + 客户端 + 租户 + 成员 + 订阅」夹具的标识。
type appAccessSeed struct {
	appID      string
	clientCode string
	tenantID   string
	personID   string
	memberID   string
}

// seedApp 建一个应用及其客户端（= 本次登录的客户端），并物化一个订阅占位租户无关的基线。
// client code 必须满足 model.ClientCodePattern（仅小写字母与下划线），故用随机字母保证唯一。
func (e *appAccessGateEnv) seedApp(t *testing.T) (appID, clientCode string) {
	t.Helper()
	suffix := randomLetters(8)
	appID = "app-appaccess-" + suffix
	clientCode = "appaccess_client_" + suffix
	db := crossTenantDB(e.ctx)
	require.NoError(t, db.Create(&model.ApplicationEntity{
		BaseEntity:   gormdao.BaseEntity{StringID: gormdao.StringID{ID: appID}},
		Code:         "app_appaccess_" + suffix,
		Name:         "准入用例应用-" + suffix,
		Status:       model.AppStatusEnable,
		RoleTemplate: model.RoleTemplateItemList{},
	}).Error)
	require.NoError(t, db.Create(&model.ApplicationClientEntity{
		BaseEntity:             gormdao.BaseEntity{StringID: gormdao.StringID{ID: "client-appaccess-" + suffix}},
		AppID:                  appID,
		Code:                   clientCode,
		Name:                   "准入用例客户端-" + suffix,
		RedirectURIs:           model.RedirectURIList{"https://client.example.com/callback"},
		PostLogoutRedirectURIs: model.PostLogoutRedirectURIList{},
		GrantTypes:             model.GrantTypeList{model.GrantTypeAuthorizationCode},
		ResponseTypes:          model.ResponseTypeList{model.ResponseTypeCode},
		AllowedOrigins:         model.AllowedOriginList{},
		DefaultScopes:          model.DefaultScopeList{model.ScopeOpenID},
		Status:                 model.ApplicationClientStatusEnable,
	}).Error)
	return appID, clientCode
}

// seedMemberTenant 建一个 active 租户、其成员，并订阅 appID；grant 为空表示不写准入行。
func (e *appAccessGateEnv) seedMemberTenant(t *testing.T, appID string, grant model.UserApplicationStatus) appAccessSeed {
	t.Helper()
	suffix := randomLetters(8)
	seed := appAccessSeed{
		appID:    appID,
		tenantID: "t-appaccess-" + suffix,
		personID: "p-appaccess-" + suffix,
		memberID: "u-appaccess-" + suffix,
	}
	db := crossTenantDB(e.ctx)
	require.NoError(t, db.Create(&model.TenantEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: seed.tenantID}},
		Code:       "t_appaccess_" + suffix,
		Name:       "准入用例租户-" + suffix,
		Tag:        "t_appaccess_" + suffix,
		Status:     model.TenantStatusActive,
	}).Error)
	require.NoError(t, db.Create(&model.UserEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: seed.memberID}},
		TenantID:   seed.tenantID,
		PersonID:   seed.personID,
		UserType:   model.UserTypeMember,
		Name:       "准入用例成员-" + suffix,
	}).Error)
	require.NoError(t, db.Create(&model.TenantApplicationEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "ta-appaccess-" + suffix}},
		TenantID:   seed.tenantID,
		AppID:      appID,
		Status:     model.TenantApplicationStatusEnable,
	}).Error)
	if grant != "" {
		require.NoError(t, db.Create(&model.UserApplicationEntity{
			TenantID: seed.tenantID,
			UserID:   seed.memberID,
			AppID:    appID,
			Status:   grant,
			Source:   model.UserApplicationSourceManual,
		}).Error)
	}
	return seed
}

// crossTenantDB 返回显式跨租户作用域的测试库句柄（夹具写入不依赖「ctx 恰好没有作用域」）。
func crossTenantDB(ctx *gin.Context) *gorm.DB {
	scoped := dbclient.CrossTenantContext(ctx)
	return dbclient.IamDB(scoped).WithContext(scoped)
}

// randomLetters 生成仅小写字母的随机串（客户端编码不允许数字）。
func randomLetters(n int) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyz"
	buf := make([]byte, n)
	for i := range buf {
		buf[i] = alphabet[rand.IntN(len(alphabet))]
	}
	return string(buf)
}

// TestFilterTenantsByClientApp 租户列表过滤（AC-4）：
// 只保留「该成员在本客户端所属应用下已被准入」的租户，避免用户选到必然失败的租户；
// 开关关闭时不过滤（等价现状，AC-13）。
func TestFilterTenantsByClientApp(t *testing.T) {
	env := newAppAccessGateEnv(t)
	appID, clientCode := env.seedApp(t)
	withGrant := env.seedMemberTenant(t, appID, model.UserApplicationStatusEnable)
	withoutGrant := env.seedMemberTenant(t, appID, "")
	// 已撤销准入（disable）同样不可用
	revoked := env.seedMemberTenant(t, appID, model.UserApplicationStatusDisable)

	tenants := []objauth.TenantOption{
		{TenantID: withoutGrant.tenantID, Name: "no-grant", UserID: withoutGrant.memberID},
		{TenantID: revoked.tenantID, Name: "revoked", UserID: revoked.memberID},
		{TenantID: withGrant.tenantID, Name: "granted", UserID: withGrant.memberID},
	}

	svc := &oidcAuthSvc{provider: env.provider}
	filtered, err := svc.filterTenantsByClientApp(requestCtx(env.ctx), clientCode, tenants)
	require.NoError(t, err)
	require.Len(t, filtered, 1)
	require.Equal(t, withGrant.tenantID, filtered[0].TenantID)

	// 开关关闭：原样返回（不查库、不过滤）
	offEnv := newAppAccessGateEnvWithFlag(t, false)
	offSvc := &oidcAuthSvc{provider: offEnv.provider}
	unfiltered, err := offSvc.filterTenantsByClientApp(offEnv.ctx, clientCode, tenants)
	require.NoError(t, err)
	require.Len(t, unfiltered, 3)
}

// TestCompleteLoginRejectsTenantWithoutAppAccess 登录期早拒（AC-3）：
// 单租户且该成员无准入时返回 110016，且不完成授权（done=false，不发 code、不建 SSO 会话）。
func TestCompleteLoginRejectsTenantWithoutAppAccess(t *testing.T) {
	env := newAppAccessGateEnv(t)
	appID, clientCode := env.seedApp(t)
	seed := env.seedMemberTenant(t, appID, "")

	authReq := newAuthReq(t, env.provider, clientCode)
	svc := newAppAccessLoginSvc(env, seed)

	_, err := svc.CompleteLogin(requestCtx(env.ctx), &dtooidc.OIDCLoginReq{
		AuthRequestID: authReq.GetID(),
		Identifier:    "member",
		Password:      "secret",

		Operator: operatorOf(env.ctx)})
	require.Error(t, err)
	require.Equal(t, code.GetError(code.AuthAppAccessDeniedError).Error(), err.Error())

	// 不发 code：授权票据必须仍未完成
	updated, aErr := env.provider.Storage.AuthRequestByID(env.ctx, authReq.GetID())
	require.NoError(t, aErr)
	stored, ok := updated.(*oidcop.AuthRequest)
	require.True(t, ok)
	require.False(t, stored.Done())

	// AC-4 的"审计"要求：拒绝必须留下可查的失败审计（同时是灰度 P3"准入拒绝次数"指标的来源）
	assertAppAccessDeniedAudit(t, env, fmt.Sprintf("personID:%s", seed.personID))
}

// assertAppAccessDeniedAudit 断言库中存在一条"因应用准入被拒"的登录失败审计。
// detail 前缀用于区分同一用例交叉命中的其它拒绝路径（如 hint 失效）。
func assertAppAccessDeniedAudit(t *testing.T, env *appAccessGateEnv, detailPrefix string) {
	t.Helper()
	readCtx := dbclient.CrossTenantContext(context.Background())
	db := dbclient.IamDB(readCtx).WithContext(readCtx)
	var rows []model.AuditLogEntity
	require.NoError(t, db.Where("action = ? AND result = ?", model.AuditActionLogin, model.AuditResultFailure).
		Order("created_at desc").Limit(10).Find(&rows).Error)
	for _, row := range rows {
		if strings.HasPrefix(row.Detail, detailPrefix) && strings.Contains(row.Detail, "reason:no app access") {
			require.Equal(t, model.AuditTargetTypePerson, row.TargetType)
			return
		}
	}
	t.Fatalf("未找到准入拒绝审计，detail 前缀:%s，最近失败审计:%+v", detailPrefix, rows)
}

// TestCompleteLoginAllowsMemberWithAppAccess 准入正常时必须放行（门禁不能误伤主路径）。
func TestCompleteLoginAllowsMemberWithAppAccess(t *testing.T) {
	env := newAppAccessGateEnv(t)
	appID, clientCode := env.seedApp(t)
	seed := env.seedMemberTenant(t, appID, model.UserApplicationStatusEnable)

	authReq := newAuthReq(t, env.provider, clientCode)
	svc := newAppAccessLoginSvc(env, seed)

	res, err := svc.CompleteLogin(requestCtx(env.ctx), &dtooidc.OIDCLoginReq{
		AuthRequestID: authReq.GetID(),
		Identifier:    "member",
		Password:      "secret",

		Operator: operatorOf(env.ctx)})
	require.NoError(t, err)
	require.Equal(t, seed.tenantID, res.TenantID)
	require.NotEmpty(t, res.ContinueURL)
}

// TestSilentLoginPicksTenantWithAppAccess 静默登录（AC-5 正向）：
// 租户选择改为「本客户端应用下第一个有准入的租户」，hint 指向无准入租户时被忽略。
func TestSilentLoginPicksTenantWithAppAccess(t *testing.T) {
	env := newAppAccessGateEnv(t)
	appID, clientCode := env.seedApp(t)
	withoutGrant := env.seedMemberTenant(t, appID, "")
	withGrant := env.seedMemberTenant(t, appID, model.UserApplicationStatusEnable)

	authReq := newAuthReq(t, env.provider, clientCode)
	// 把 hint 指向无准入的租户：必须被忽略，落到有准入的租户
	require.NoError(t, env.provider.Storage.CompleteAuthRequest(env.ctx, authReq.GetID(),
		authReq.GetSubject(), authReq.GetAuthTime(), []string{"pwd"}, "", withoutGrant.tenantID, false))

	svc := newAppAccessSilentLoginSvc(env, withoutGrant.personID, []objauth.TenantOption{
		{TenantID: withoutGrant.tenantID, Name: "no-grant", UserID: withoutGrant.memberID},
		{TenantID: withGrant.tenantID, Name: "granted", UserID: withGrant.memberID},
	})

	continueURL, err := svc.CompleteLoginBySession(requestCtx(env.ctx), operatorOf(env.ctx), authReq.GetID(), "session-appaccess")
	require.NoError(t, err)
	require.NotEmpty(t, continueURL)

	updated, aErr := env.provider.Storage.AuthRequestByID(env.ctx, authReq.GetID())
	require.NoError(t, aErr)
	stored, ok := updated.(*oidcop.AuthRequest)
	require.True(t, ok)
	require.True(t, stored.Done())
	require.Equal(t, withGrant.tenantID, stored.TenantID)
}

// TestSilentLoginWithoutAnyAppAccessIsRejected 静默登录（AC-5 反向）：
// 一个可用租户都没有时返回专用哨兵 errSilentLoginNoAppAccess（**不得**复用
// errSilentLoginNotAllowed，否则 SSOLogin 会清 cookie 回登录页形成死循环），且不完成授权。
func TestSilentLoginWithoutAnyAppAccessIsRejected(t *testing.T) {
	env := newAppAccessGateEnv(t)
	appID, clientCode := env.seedApp(t)
	first := env.seedMemberTenant(t, appID, "")
	second := env.seedMemberTenant(t, appID, model.UserApplicationStatusDisable)

	authReq := newAuthReq(t, env.provider, clientCode)
	svc := newAppAccessSilentLoginSvc(env, first.personID, []objauth.TenantOption{
		{TenantID: first.tenantID, Name: "no-grant", UserID: first.memberID},
		{TenantID: second.tenantID, Name: "revoked", UserID: second.memberID},
	})

	_, err := svc.CompleteLoginBySession(requestCtx(env.ctx), operatorOf(env.ctx), authReq.GetID(), "session-appaccess")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrSilentLoginNoAppAccess)
	// 必须是**另一个**哨兵：混用 errSilentLoginNotAllowed 会让 SSOLogin 清 cookie 回登录页（死循环）
	require.NotErrorIs(t, err, errSilentLoginNotAllowed)

	updated, aErr := env.provider.Storage.AuthRequestByID(env.ctx, authReq.GetID())
	require.NoError(t, aErr)
	stored, ok := updated.(*oidcop.AuthRequest)
	require.True(t, ok)
	require.False(t, stored.Done())

	// 静默登录被拒同样必须留审计（灰度期"准入拒绝次数"不区分登录入口）
	assertAppAccessDeniedAudit(t, env, fmt.Sprintf("personID:%s", first.personID))
}

// TestSelectTenantRejectsTenantWithoutAppAccess 租户选择页提交（AC-3 第二落点）：
// 选中一个「是本人成员但未被准入该应用」的租户 → 110016。
func TestSelectTenantRejectsTenantWithoutAppAccess(t *testing.T) {
	env := newAppAccessGateEnv(t)
	appID, clientCode := env.seedApp(t)
	seed := env.seedMemberTenant(t, appID, "")

	authReq := newAuthReq(t, env.provider, clientCode)
	// 多租户登录的中间态：主体已绑定但未 done（SelectTenant 从票据取自然人）
	require.NoError(t, env.provider.Storage.CompleteAuthRequest(env.ctx, authReq.GetID(),
		oidcop.BuildSubject(seed.personID), authReq.GetAuthTime(), []string{"pwd"}, "", "", false))
	svc := newAppAccessSilentLoginSvc(env, seed.personID, []objauth.TenantOption{
		{TenantID: seed.tenantID, Name: "no-grant", UserID: seed.memberID},
	})

	_, err := svc.SelectTenant(requestCtx(env.ctx), operatorOf(env.ctx), authReq.GetID(), seed.tenantID)
	require.Error(t, err)
	require.Equal(t, code.GetError(code.AuthAppAccessDeniedError).Error(), err.Error())

	assertAppAccessDeniedAudit(t, env, fmt.Sprintf("personID:%s", seed.personID))
}

// newAppAccessLoginSvc 密码登录路径的 svc 夹具：单租户、已验密、密码状态正常。
func newAppAccessLoginSvc(env *appAccessGateEnv, seed appAccessSeed) *oidcAuthSvc {
	return &oidcAuthSvc{
		provider: env.provider,
		authSvc: &fakePasswordAuthenticator{authenticate: func(ctx context.Context, identifier, password string) (*model.PersonEntity, *model.UserEntity, []objauth.TenantOption, error) {
			return &model.PersonEntity{
					BaseEntity:     gormdao.BaseEntity{StringID: gormdao.StringID{ID: seed.personID}},
					PasswordStatus: model.PasswordStatusNormal,
				},
				&model.UserEntity{
					BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: seed.memberID}},
					TenantID:   seed.tenantID,
					PersonID:   seed.personID,
				},
				[]objauth.TenantOption{{TenantID: seed.tenantID, Name: "t", UserID: seed.memberID}},
				nil
		}},
		ssoSessionStore: &fakeSSOSessionStore{validatedPersonID: seed.personID},
	}
}

// newAppAccessSilentLoginSvc 静默登录/租户选择路径的 svc 夹具（租户选项由用例给定）。
func newAppAccessSilentLoginSvc(env *appAccessGateEnv, personID string, tenants []objauth.TenantOption) *oidcAuthSvc {
	return &oidcAuthSvc{
		provider: env.provider,
		authSvc: &fakePasswordAuthenticator{tenantsForPerson: func(ctx context.Context, got string) ([]objauth.TenantOption, error) {
			if got != personID {
				return nil, fmt.Errorf("unexpected personID %s, want %s", got, personID)
			}
			return tenants, nil
		}},
		ssoSessionStore: &fakeSSOSessionStore{validatedPersonID: personID},
	}
}
