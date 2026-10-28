package platformadmin

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/morehao/ark-iam/pkg/code"
	pkgconfig "github.com/morehao/ark-iam/pkg/config"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/platformadmin/testutil"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// 本文件是 T1.4 的集成验收：走**真实的 app.Init 装配链**（OIDCAuth → RequireAppAccess → 控制器 → 服务 → DAO），
// 用真实签名的 access token 打真实路由，证明 P0-0 的越权链路已被关闭、且被拒时目标租户零副作用。
//
// 四个用例（对应计划 T1.4 表）：
//  1. 他租户成员调 POST /v1/platform/tenants/{id}/builtin-admin/reset-password → 110016，
//     且目标租户管理员口令未变、refresh token 未被吊销、无审计行；
//  2. 持有平台控制台准入者调同一端点 → 成功（口令确实被改、会话被吊销）；
//  3. 把该持有者准入行置 disable 后重调 → 立即 110016（证明判定走准入行，不是角色）；
//  4. 无准入者调 GET /v1/platform/menus/my → 110016（确认整组受保护）。

const (
	itIssuer = "http://localhost:8081/oidc"

	// 调用方：某业务租户的成员（他租户成员，平台控制台无准入）。
	itCallerTenant = "tenant-caller"
	itCallerUser   = "user-caller"
	itCallerPerson = "person-caller"

	// 目标租户：平台管理端点要改的是它的内置管理员口令。
	itTargetTenant       = "tenant-target"
	itTargetAdminUser    = "user-target-admin"
	itTargetAdminPerson  = "person-target-admin"
	itTargetPasswordHash = "HASH-BEFORE-ATTACK"

	itAppID       = "app-platform"
	itAppClientID = model.SeedBuiltinClientPlatformAdminWeb
)

// itKeySource 是只含一把公钥的进程内 KeySource：不依赖 JWKS 网络，也不引入 sdk 依赖。
// 实现 rp.KeySource（单一方法接口），因此与生产验签链完全同源。
type itKeySource struct {
	kid string
	pub *rsa.PublicKey
}

func (s itKeySource) PublicKey(_ context.Context, kid string) (*rsa.PublicKey, error) {
	if kid != s.kid {
		return nil, assert.AnError
	}
	return s.pub, nil
}

// itFixture 是集成验收的夹具：真实引擎 + 已播种的库。
type itFixture struct {
	db     *gorm.DB
	engine *gin.Engine
	key    *rsa.PrivateKey
	kid    string
}

func newAccessIT(t *testing.T) *itFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db := testutil.SetupSQLite(t,
		&model.TenantEntity{},
		&model.UserEntity{},
		&model.PersonEntity{},
		&model.RefreshTokenEntity{},
		&model.AuditLogEntity{},
		&model.ApplicationEntity{},
		&model.ApplicationClientEntity{},
		&model.TenantApplicationEntity{},
		&model.UserApplicationEntity{},
	)

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	const kid = "platformadmin-it-kid"

	f := &itFixture{db: db, key: key, kid: kid}

	engine := gin.New()
	engine.ContextWithFallback = true
	engine.Use(gin.Recovery())
	// 与生产同一装配路径：版本组中间件 = [OIDCAuth, RequireAppAccess]（见 app.go）。
	Init(engine, &pkgconfig.Config{
		Server: pkgconfig.Server{Env: "test"},
		OIDC:   pkgconfig.OIDC{Issuer: itIssuer},
	}, itKeySource{kid: kid, pub: &key.PublicKey})
	f.engine = engine

	f.seed(t)
	return f
}

func (f *itFixture) seed(t *testing.T) {
	t.Helper()
	// 库为跨租户播种/断言，统一显式声明「全部租户」作用域（测试库挂着同一份 fail-closed 插件）。
	xdb := f.db.WithContext(dbclient.CrossTenantContext(context.Background()))

	require.NoError(t, xdb.Create(&[]model.TenantEntity{
		{BaseEntity: be("tenant-caller"), Code: "t_caller", Name: "调用方租户", Status: model.TenantStatusActive},
		{BaseEntity: be("tenant-target"), Code: "t_target", Name: "目标租户", Status: model.TenantStatusActive},
	}).Error)

	require.NoError(t, xdb.Create(&[]model.PersonEntity{
		{BaseEntity: be(itCallerPerson), PasswordEncrypted: "CALLER-HASH"},
		{BaseEntity: be(itTargetAdminPerson), PasswordEncrypted: itTargetPasswordHash},
	}).Error)

	require.NoError(t, xdb.Create(&[]model.UserEntity{
		{
			BaseEntity: be(itCallerUser), TenantID: itCallerTenant, PersonID: itCallerPerson,
			UserType: model.UserTypeMember, Source: model.UserSourceManual, Name: "调用方成员", Status: model.UserStatusActive,
		},
		{
			BaseEntity: be(itTargetAdminUser), TenantID: itTargetTenant, PersonID: itTargetAdminPerson,
			UserType: model.UserTypeMember, Source: model.UserSourceBuiltin, Name: "目标租户管理员", Status: model.UserStatusActive,
		},
	}).Error)

	require.NoError(t, xdb.Create(&model.ApplicationEntity{
		BaseEntity:   be(itAppID),
		Code:         "platform_admin",
		SeedKey:      "platform_admin",
		Name:         "平台管理后台",
		Status:       model.AppStatusEnable,
		RoleTemplate: model.RoleTemplateItemList{},
	}).Error)

	require.NoError(t, xdb.Create(&model.ApplicationClientEntity{
		BaseEntity:             be("client-platform"),
		AppID:                  itAppID,
		Code:                   itAppClientID,
		Name:                   "平台管理后台",
		RedirectURIs:           model.RedirectURIList{},
		PostLogoutRedirectURIs: model.PostLogoutRedirectURIList{},
		GrantTypes:             model.GrantTypeList{model.GrantTypeAuthorizationCode},
		ResponseTypes:          model.ResponseTypeList{"code"},
		AllowedOrigins:         model.AllowedOriginList{},
		DefaultScopes:          model.DefaultScopeList{"openid"},
		Status:                 model.ApplicationClientStatusEnable,
	}).Error)

	// 调用方租户**订阅了**平台控制台应用、但调用方成员**没有准入行**：
	// 这样"被拒"只可能是准入行这一层造成的（订阅是准入的上界，缺订阅会让判定退化成必然拒绝）。
	require.NoError(t, xdb.Create(&model.TenantApplicationEntity{
		BaseEntity: be("sub-caller"),
		TenantID:   itCallerTenant,
		AppID:      itAppID,
		Status:     model.TenantApplicationStatusEnable,
	}).Error)

	// 目标租户管理员的活跃会话（used to证明被拒时未被吊销）。
	require.NoError(t, xdb.Create(&model.RefreshTokenEntity{
		BaseEntity: be("rt-target"),
		PersonID:   itTargetAdminPerson,
		TenantID:   itTargetTenant,
		UserID:     itTargetAdminUser,
		Token:      "token-hash",
		Scopes:     model.ScopeList{"openid"},
		AMR:        model.AuthMethodList{},
		ExpiredAt:  ptrTime(time.Now().Add(time.Hour)),
	}).Error)
}

// be 构造带主键的 BaseEntity。
func be(id string) gormdao.BaseEntity {
	return gormdao.BaseEntity{StringID: gormdao.StringID{ID: id}}
}

// ptrTime 取时间指针（可空时间列用）。
func ptrTime(v time.Time) *time.Time { return &v }

// grantCaller 给调用方成员写入平台控制台应用准入行（enable）。
func (f *itFixture) grantCaller(t *testing.T, status model.UserApplicationStatus) {
	t.Helper()
	xdb := f.db.WithContext(dbclient.CrossTenantContext(context.Background()))
	require.NoError(t, xdb.Create(&model.UserApplicationEntity{
		BaseEntity: be("grant-caller"),
		TenantID:   itCallerTenant,
		UserID:     itCallerUser,
		AppID:      itAppID,
		Status:     status,
		Source:     model.UserApplicationSourceManual,
	}).Error)
}

func (f *itFixture) setGrantStatus(t *testing.T, status model.UserApplicationStatus) {
	t.Helper()
	xdb := f.db.WithContext(dbclient.CrossTenantContext(context.Background()))
	require.NoError(t, xdb.Model(&model.UserApplicationEntity{}).Where("id = ?", "grant-caller").Update("status", status).Error)
}

// token 签发调用方的 access token：aud/client_id 都是平台控制台客户端（这正是被利用的那条链路：
// 攻击者能通过 platform_admin_web 完成授权码流程），但 tenant_id/user_id 指向**他自己**的租户与成员。
func (f *itFixture) callerToken(t *testing.T) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub":         "person:" + itCallerPerson,
		"aud":         itAppClientID,
		"iss":         itIssuer,
		"exp":         time.Now().Add(time.Hour).Unix(),
		"iat":         time.Now().Unix(),
		"tenant_id":   itCallerTenant,
		"user_id":     itCallerUser,
		"client_id":   itAppClientID,
		"token_usage": "person",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = f.kid
	signed, err := token.SignedString(f.key)
	require.NoError(t, err)
	return signed
}

// resetPassword 调平台管理端点的真实 HTTP 路径。
func (f *itFixture) resetPassword(t *testing.T, token string) (int, bool) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost,
		"/v1/platform/tenants/"+itTargetTenant+"/builtin-admin/reset-password", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	f.engine.ServeHTTP(rec, req)

	var body struct {
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body), "响应体：%s", rec.Body.String())
	return body.Code, body.Code == 0
}

func (f *itFixture) myMenus(t *testing.T, token string) int {
	t.Helper()
	return f.getCode(t, token, "/v1/platform/menus/my")
}

// getCode 打一条 GET 路由并返回响应体里的业务码。
func (f *itFixture) getCode(t *testing.T, token, path string) int {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	f.engine.ServeHTTP(rec, req)

	var body struct {
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body), "响应体：%s", rec.Body.String())
	return body.Code
}

// targetAdminPassword 读目标租户管理员当前的口令摘要。
func (f *itFixture) targetAdminPassword(t *testing.T) string {
	t.Helper()
	var person model.PersonEntity
	require.NoError(t, f.db.WithContext(dbclient.CrossTenantContext(context.Background())).
		Where("id = ?", itTargetAdminPerson).Take(&person).Error)
	return person.PasswordEncrypted
}

// targetRefreshTokenRevoked 读目标租户管理员 refresh token 是否已被吊销。
func (f *itFixture) targetRefreshTokenRevoked(t *testing.T) bool {
	t.Helper()
	var rt model.RefreshTokenEntity
	require.NoError(t, f.db.WithContext(dbclient.CrossTenantContext(context.Background())).
		Where("id = ?", "rt-target").Take(&rt).Error)
	return rt.RevokedAt != nil
}

func (f *itFixture) auditCount(t *testing.T) int64 {
	t.Helper()
	var n int64
	require.NoError(t, f.db.WithContext(dbclient.CrossTenantContext(context.Background())).
		Model(&model.AuditLogEntity{}).Count(&n).Error)
	return n
}

// TestP0_0_ForeignTenantMemberDeniedWithNoSideEffect 用例 1：越权链路被关闭且零副作用。
func TestP0_0_ForeignTenantMemberDeniedWithNoSideEffect(t *testing.T) {
	f := newAccessIT(t)
	// 刻意不给调用方准入行。

	got, ok := f.resetPassword(t, f.callerToken(t))

	assert.False(t, ok, "他租户成员不得成功")
	assert.Equal(t, code.AuthAppAccessDeniedError, got, "必须是 110016 应用准入被拒")
	assert.Equal(t, itTargetPasswordHash, f.targetAdminPassword(t), "被拒时目标管理员口令不得变动")
	assert.False(t, f.targetRefreshTokenRevoked(t), "被拒时目标管理员会话不得被吊销")
	assert.EqualValues(t, 0, f.auditCount(t), "被拒时不得写入审计（handler 未被调用）")
}

// TestP0_0_GrantedMemberAllowed 用例 2：准入持有者调用成功，且副作用真实发生。
func TestP0_0_GrantedMemberAllowed(t *testing.T) {
	f := newAccessIT(t)
	f.grantCaller(t, model.UserApplicationStatusEnable)

	got, ok := f.resetPassword(t, f.callerToken(t))

	require.True(t, ok, "准入持有者应成功，实际 code=%d", got)
	assert.NotEqual(t, itTargetPasswordHash, f.targetAdminPassword(t), "成功后目标管理员口令必须已改")
	assert.True(t, f.targetRefreshTokenRevoked(t), "成功后目标管理员会话必须被吊销")
	assert.Greater(t, f.auditCount(t), int64(0), "成功后必须留审计")
}

// TestP0_0_DisablingGrantImmediatelyDenies 用例 3（本项最关键的自证）：
// 准入行置 disable 后立即被拒 —— 证明判定走的是准入行，而不是角色/租户成员关系。
func TestP0_0_DisablingGrantImmediatelyDenies(t *testing.T) {
	f := newAccessIT(t)
	f.grantCaller(t, model.UserApplicationStatusEnable)

	// 先确认同一身份是被放行的（否则"disable 后被拒"可能只是因为一开始就进不来）。
	first, ok := f.resetPassword(t, f.callerToken(t))
	require.True(t, ok, "前置条件：enable 准入时应放行，实际 code=%d", first)

	f.setGrantStatus(t, model.UserApplicationStatusDisable)

	got, ok := f.resetPassword(t, f.callerToken(t))
	assert.False(t, ok, "准入行 disable 后不得放行")
	assert.Equal(t, code.AuthAppAccessDeniedError, got)
}

// TestP0_0_NoGrantDeniedOnMenusMy 用例 4：整组受保护（含菜单自身端点）。
func TestP0_0_NoGrantDeniedOnMenusMy(t *testing.T) {
	f := newAccessIT(t)

	got := f.myMenus(t, f.callerToken(t))

	assert.Equal(t, code.AuthAppAccessDeniedError, got)
}

// TestP0_0_NoGrantDeniedOnAuditLogs 用例 4 补充（T6.5 新增的审计只读查询同样受 L1 闸口保护）：
// L1 闸口在版本组中间件统一注入（见 app.go），新增路由自动被覆盖，这里从整链路上验一次。
func TestP0_0_NoGrantDeniedOnAuditLogs(t *testing.T) {
	f := newAccessIT(t)

	got := f.getCode(t, f.callerToken(t), "/v1/platform/audit-logs")

	assert.Equal(t, code.AuthAppAccessDeniedError, got)
}

// TestP0_0_MissingTokenUnauthorized 附加：无令牌在 OIDCAuth 就被 401 拦下（不进入准入判定）。
func TestP0_0_MissingTokenUnauthorized(t *testing.T) {
	f := newAccessIT(t)

	req := httptest.NewRequest(http.MethodPost,
		"/v1/platform/tenants/"+itTargetTenant+"/builtin-admin/reset-password", nil)
	rec := httptest.NewRecorder()
	f.engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, itTargetPasswordHash, f.targetAdminPassword(t))
}
