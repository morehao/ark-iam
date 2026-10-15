package oidcop

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/morehao/ark-iam/auth/testutil"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/object/objauth"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// appAccessGateFixture 令牌签发门禁用例的夹具：一个「租户 active + 应用启用 + 客户端绑定该应用 +
// 订阅启用」的可用基线，准入行按用例需要播种/停用/删除。
type appAccessGateFixture struct {
	storage  *OIDCStorage
	db       *gorm.DB
	tenantID string
	appID    string
	clientID string
	personID string
	memberID string
}

// newAppAccessGateFixture 构造夹具。enforce=false 用于锁定「开关关闭等价现状」（AC-13）。
func newAppAccessGateFixture(t *testing.T, enforce bool) *appAccessGateFixture {
	t.Helper()
	db := testutil.SetupSQLite(t,
		&model.TenantEntity{},
		&model.ApplicationEntity{},
		&model.ApplicationClientEntity{},
		&model.TenantApplicationEntity{},
		&model.UserApplicationEntity{},
		&model.UserEntity{},
	)

	joinedAt := time.Now()
	f := &appAccessGateFixture{
		storage:  &OIDCStorage{persistentStore: NewPersistentStore(WithAppAccessEnforcement(enforce))},
		db:       db,
		tenantID: "t-app-access",
		appID:    "app-app-access",
		clientID: "app_access_client",
		personID: "p-app-access",
		memberID: "u-app-access",
	}
	require.NoError(t, db.Create(&model.TenantEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: f.tenantID}},
		Code:       "t_app_access",
		Name:       "准入用例租户",
		Status:     model.TenantStatusActive,
	}).Error)
	require.NoError(t, db.Create(&model.ApplicationEntity{
		BaseEntity:   gormdao.BaseEntity{StringID: gormdao.StringID{ID: f.appID}},
		Code:         "app_app_access",
		Name:         "准入用例应用",
		Status:       model.AppStatusEnable,
		RoleTemplate: model.RoleTemplateItemList{},
	}).Error)
	require.NoError(t, db.Create(&model.ApplicationClientEntity{
		BaseEntity:             gormdao.BaseEntity{StringID: gormdao.StringID{ID: "client-app-access"}},
		AppID:                  f.appID,
		Code:                   f.clientID,
		Name:                   "准入用例客户端",
		RedirectURIs:           model.RedirectURIList{},
		PostLogoutRedirectURIs: model.PostLogoutRedirectURIList{},
		GrantTypes:             model.GrantTypeList{model.GrantTypeAuthorizationCode},
		ResponseTypes:          model.ResponseTypeList{model.ResponseTypeCode},
		AllowedOrigins:         model.AllowedOriginList{},
		DefaultScopes:          model.DefaultScopeList{model.ScopeOpenID},
		Status:                 model.ApplicationClientStatusEnable,
	}).Error)
	require.NoError(t, db.Create(&model.TenantApplicationEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "ta-app-access"}},
		TenantID:   f.tenantID,
		AppID:      f.appID,
		Status:     model.TenantApplicationStatusEnable,
	}).Error)
	require.NoError(t, db.Create(&model.UserEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: f.memberID}},
		TenantID:   f.tenantID,
		PersonID:   f.personID,
		UserType:   model.UserTypeMember,
		Name:       "准入用例成员",
		JoinedAt:   &joinedAt,
	}).Error)
	return f
}

// setGrant 写入/覆盖该成员的准入行（status 为空表示删除该行，模拟"从未授权"）。
func (f *appAccessGateFixture) setGrant(t *testing.T, status model.UserApplicationStatus) {
	t.Helper()
	scoped := dbclient.CrossTenantContext(context.Background())
	require.NoError(t, f.db.WithContext(scoped).Where("tenant_id = ? AND user_id = ? AND app_id = ?", f.tenantID, f.memberID, f.appID).
		Delete(&model.UserApplicationEntity{}).Error)
	if status == "" {
		return
	}
	require.NoError(t, f.db.Create(&model.UserApplicationEntity{
		TenantID: f.tenantID,
		UserID:   f.memberID,
		AppID:    f.appID,
		Status:   status,
		Source:   model.UserApplicationSourceManual,
	}).Error)
}

// setSubscription 切换订阅状态（用于验证「订阅是准入的上界」）。
func (f *appAccessGateFixture) setSubscription(t *testing.T, status model.TenantApplicationStatus) {
	t.Helper()
	require.NoError(t, f.db.WithContext(dbclient.CrossTenantContext(context.Background())).
		Model(&model.TenantApplicationEntity{}).
		Where("id = ?", "ta-app-access").Update("status", status).Error)
}

// authRequest 构造授权码流的 token 请求（携带 client_id / 租户 / 自然人主体）。
func (f *appAccessGateFixture) authRequest() *AuthRequest {
	return &AuthRequest{
		ID:        "auth-req-app-access",
		ClientID:  f.clientID,
		Subject:   BuildSubject(f.personID),
		TenantID:  f.tenantID,
		SessionID: "session-app-access",
	}
}

// refreshRequest 构造刷新流的 token 请求（与授权码流同口径带 client_id）。
func (f *appAccessGateFixture) refreshRequest() *refreshTokenRequest {
	return &refreshTokenRequest{
		subject:   BuildSubject(f.personID),
		clientID:  f.clientID,
		tenantID:  f.tenantID,
		sessionID: "session-app-access",
	}
}

// TestAppAccessTokenGateAuthorizationCodeFlow 授权码流兜底门禁（AC-1）：
// 无准入行 / 准入停用 → access_denied；准入启用 → 放行并产出正确声明；
// 订阅停用 → 同样拒绝（订阅是准入的上界）。
func TestAppAccessTokenGateAuthorizationCodeFlow(t *testing.T) {
	f := newAppAccessGateFixture(t, true)
	ctx := context.Background()

	// 从未授权（无准入行）
	f.setGrant(t, "")
	_, err := f.storage.GetPrivateClaimsFromRequest(ctx, f.authRequest(), nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "access_denied")
	require.Contains(t, err.Error(), "not granted access to the application")

	// 准入行存在但已停用：同样拒绝（撤销准入用 disable，不删行）
	f.setGrant(t, model.UserApplicationStatusDisable)
	_, err = f.storage.GetPrivateClaimsFromRequest(ctx, f.authRequest(), nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "access_denied")

	// 准入启用：放行，且声明口径正确（tenant_id/user_id=tenant_user.id/person_id/sid）
	f.setGrant(t, model.UserApplicationStatusEnable)
	claims, err := f.storage.GetPrivateClaimsFromRequest(ctx, f.authRequest(), nil)
	require.NoError(t, err)
	require.Equal(t, f.tenantID, claims["tenant_id"])
	require.Equal(t, f.memberID, claims["user_id"])
	require.Equal(t, f.personID, claims["person_id"])
	require.Equal(t, "session-app-access", claims["sid"])

	// 订阅停用：准入行仍在，但订阅是准入的上界 → 拒绝
	f.setSubscription(t, model.TenantApplicationStatusDisable)
	_, err = f.storage.GetPrivateClaimsFromRequest(ctx, f.authRequest(), nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "access_denied")
}

// TestAppAccessTokenGateRefreshFlow 刷新流兜底门禁（AC-2）：
// 撤销准入后，刷新轮换必须按 invalid_grant 拒绝（促使 RP 丢弃 refresh token），
// 而不是继续签发——这是"撤销准入在下一个刷新周期内生效"的依据。
func TestAppAccessTokenGateRefreshFlow(t *testing.T) {
	f := newAppAccessGateFixture(t, true)
	ctx := context.Background()

	f.setGrant(t, model.UserApplicationStatusEnable)
	claims, err := f.storage.GetPrivateClaimsFromRequest(ctx, f.refreshRequest(), nil)
	require.NoError(t, err)
	require.Equal(t, f.tenantID, claims["tenant_id"])

	f.setGrant(t, model.UserApplicationStatusDisable)
	_, err = f.storage.GetPrivateClaimsFromRequest(ctx, f.refreshRequest(), nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid_grant")
	require.NotContains(t, err.Error(), "access_denied")
}

// TestAppAccessTokenGateDisabledMatchesLegacyBehavior 开关关闭等价现状（AC-13）：
// 无准入行、客户端未绑定应用都不再拦截，与门禁上线前的签发行为逐字节一致。
func TestAppAccessTokenGateDisabledMatchesLegacyBehavior(t *testing.T) {
	f := newAppAccessGateFixture(t, false)
	ctx := context.Background()

	// 无任何准入行
	f.setGrant(t, "")
	claims, err := f.storage.GetPrivateClaimsFromRequest(ctx, f.authRequest(), nil)
	require.NoError(t, err)
	require.Equal(t, f.memberID, claims["user_id"])

	// 客户端不存在、甚至未绑定应用：开关关闭时也不拦（历史行为）
	req := f.authRequest()
	req.ClientID = "no_such_client"
	claims, err = f.storage.GetPrivateClaimsFromRequest(ctx, req, nil)
	require.NoError(t, err)
	require.Equal(t, f.memberID, claims["user_id"])
}

// TestAppAccessTokenGateSkipsMachineChannels 不变式 I4：
// client_credentials / API Key 通道没有自然人主体，不查 user_application，保持 rpapi 目录调用不变。
func TestAppAccessTokenGateSkipsMachineChannels(t *testing.T) {
	f := newAppAccessGateFixture(t, true)
	ctx := context.Background()
	f.setGrant(t, "")

	ccReq := &clientCredentialsTokenRequest{
		subject:       "machine:svc-app-access",
		clientID:      f.clientID,
		isApiKey:      true,
		ownerTenantID: f.tenantID,
		ownerUserID:   f.memberID,
	}
	claims, err := f.storage.GetPrivateClaimsFromRequest(ctx, ccReq, nil)
	require.NoError(t, err)
	require.Equal(t, f.tenantID, claims["tenant_id"])
	require.Equal(t, f.memberID, claims["user_id"])
	require.Equal(t, objauth.TokenUsageMachine, claims["token_usage"])
}

// TestAppAccessTokenGateFailClosedOnUnresolvableClient fail-closed 边界（P5-7）：
// client_id 为空、客户端未绑定应用 → 拒绝；客户端不存在 → 如实报错（不伪装成"无权限"）。
func TestAppAccessTokenGateFailClosedOnUnresolvableClient(t *testing.T) {
	f := newAppAccessGateFixture(t, true)
	ctx := context.Background()
	f.setGrant(t, model.UserApplicationStatusEnable)

	// client_id 为空：应用作用域不明，拒签
	emptyClient := f.authRequest()
	emptyClient.ClientID = ""
	_, err := f.storage.GetPrivateClaimsFromRequest(ctx, emptyClient, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not bound to an application")

	// 客户端存在但未绑定应用（app_id=''）
	require.NoError(t, f.db.Create(&model.ApplicationClientEntity{
		BaseEntity:             gormdao.BaseEntity{StringID: gormdao.StringID{ID: "client-unbound"}},
		AppID:                  "",
		Code:                   "app_access_unbound",
		Name:                   "未绑定应用的客户端",
		RedirectURIs:           model.RedirectURIList{},
		PostLogoutRedirectURIs: model.PostLogoutRedirectURIList{},
		GrantTypes:             model.GrantTypeList{model.GrantTypeAuthorizationCode},
		ResponseTypes:          model.ResponseTypeList{model.ResponseTypeCode},
		AllowedOrigins:         model.AllowedOriginList{},
		DefaultScopes:          model.DefaultScopeList{model.ScopeOpenID},
		Status:                 model.ApplicationClientStatusEnable,
	}).Error)
	unbound := f.authRequest()
	unbound.ClientID = "app_access_unbound"
	_, err = f.storage.GetPrivateClaimsFromRequest(ctx, unbound, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not bound to an application")

	// 客户端不存在：解析失败属数据不一致，如实报错（不静默放行、也不伪装成"无权限"）
	missing := f.authRequest()
	missing.ClientID = "no_such_client"
	_, err = f.storage.GetPrivateClaimsFromRequest(ctx, missing, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "application client not found")
	require.False(t, strings.Contains(err.Error(), "access_denied"))
}

// TestAppAccessTokenGateKeepsTenantGatePriority 门禁的判定顺序：
// 租户挂起优先报租户错误（便于区分"租户挂了"与"没权限"），不因准入通过而放行。
func TestAppAccessTokenGateKeepsTenantGatePriority(t *testing.T) {
	f := newAppAccessGateFixture(t, true)
	ctx := context.Background()
	f.setGrant(t, model.UserApplicationStatusEnable)

	require.NoError(t, f.db.WithContext(dbclient.CrossTenantContext(context.Background())).
		Model(&model.TenantEntity{}).
		Where("id = ?", f.tenantID).Update("status", model.TenantStatusSuspended).Error)

	_, err := f.storage.GetPrivateClaimsFromRequest(ctx, f.authRequest(), nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "tenant is suspended")
}

// TestResolveClientAppIDUsesExplicitCrossTenantScope 门禁取应用的口径：
// client_id 全局唯一，解析必须显式声明「全租户」作用域（测试库挂着同一份 fail-closed 插件，
// 漏声明会以 ErrTenantScopeMissing 失败）。
func TestResolveClientAppIDUsesExplicitCrossTenantScope(t *testing.T) {
	f := newAppAccessGateFixture(t, true)
	appID, err := f.storage.persistentStore.resolveClientAppID(context.Background(), f.clientID)
	require.NoError(t, err)
	require.Equal(t, f.appID, appID)
}

// TestAppAccessTokenGateUsesUserApplicationDao 锁定门禁读取的准入来源是 user_application
// （而不是靠角色/订阅推断）：无准入行但存在其它表数据时仍必须拒绝。
func TestAppAccessTokenGateUsesUserApplicationDao(t *testing.T) {
	f := newAppAccessGateFixture(t, true)
	ctx := context.Background()
	f.setGrant(t, "")

	grant, err := dao.NewUserApplicationDao().GetByCond(
		dbclient.ExplicitTenantContext(context.Background(), f.tenantID),
		&dao.UserApplicationCond{TenantID: f.tenantID, UserID: f.memberID, AppID: f.appID},
	)
	require.NoError(t, err)
	require.Nil(t, grant)

	_, err = f.storage.GetPrivateClaimsFromRequest(ctx, f.authRequest(), nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "access_denied")
}
