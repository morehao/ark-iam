package svcoidc

import (
	"context"
	"errors"
	"github.com/morehao/ark-iam/pkg/object/objidentity"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	appconfig "github.com/morehao/ark-iam/auth/config"
	"github.com/morehao/ark-iam/auth/internal/dto/dtooidc"
	"github.com/morehao/ark-iam/auth/testutil"
	pkgconfig "github.com/morehao/ark-iam/pkg/config"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/object/objauth"
	"github.com/morehao/ark-iam/pkg/testsetup"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/stretchr/testify/require"
)

// 本文件覆盖 T4「登录事件枚举补齐」：OIDC/SSO 登录成功路径必须落 user_login_log
// 且 login_type='oidc'；写库失败（含 DB 未初始化导致 nil *gorm.DB panic）一律 best-effort。

// newOIDCLoginLogEnv 构造登录日志用例夹具：Redis（授权票据，testsetup 初始化）+
// 内存库（user_login_log / audit_log）。
//
// 只迁这两张表：CompleteLogin 的成功路径在准入门禁默认关闭时不读任何业务表
// （filterTenantsByClientApp / resolveAllowPersonCreateTenant 均在租户数 > 0 时短路），
// 因此库里除登录日志与审计外不会有别的行，断言完全确定。
func newOIDCLoginLogEnv(t *testing.T) (*OIDCProvider, *gin.Context) {
	t.Helper()
	testsetup.Initialize(testsetup.AppNameAuth)
	t.Cleanup(func() { testsetup.Done(testsetup.AppNameAuth) })
	appconfig.Conf = &pkgconfig.Config{
		JWT: pkgconfig.JWT{SignKey: "test-sign-key"},
		OIDC: pkgconfig.OIDC{
			Issuer:           "http://localhost:8099/oidc",
			FrontendLoginURL: "http://localhost:4000/oidc/login",
			AllowInsecure:    true,
		},
	}
	provider, err := SetupOIDCProvider()
	require.NoError(t, err)
	testutil.SetupSQLite(t, &model.UserLoginLogEntity{}, &model.AuditLogEntity{})
	return provider, testsetup.NewCtx()
}

// newOIDCLoginSuccessSvc 返回单租户成功登录的服务实例（person 88 / user 66 / tenant 1）。
func newOIDCLoginSuccessSvc(provider *OIDCProvider) *oidcAuthSvc {
	return &oidcAuthSvc{
		provider: provider,
		authSvc: &fakePasswordAuthenticator{authenticate: func(ctx context.Context, identifier, password string) (*model.PersonEntity, *model.UserEntity, []objauth.TenantOption, error) {
			return &model.PersonEntity{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "88"}}},
				&model.UserEntity{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "66"}}, TenantID: "1", PersonID: "88"},
				[]objauth.TenantOption{{TenantID: "1", Name: "tenant-1", UserID: "66"}}, nil
		}},
	}
}

// loadLoginLogs 以显式跨租户作用域读取登录日志（user_login_log 含 tenant_id 列，
// SELECT 侧受 fail-closed 插件约束，必须声明作用域）。
func loadLoginLogs(t *testing.T) []model.UserLoginLogEntity {
	t.Helper()
	readCtx := dbclient.CrossTenantContext(context.Background())
	db := dbclient.IamDB(readCtx)
	require.NotNil(t, db, "测试库必须已注册")
	scoped := db.WithContext(readCtx)
	var rows []model.UserLoginLogEntity
	require.NoError(t, scoped.Order("created_at asc").Find(&rows).Error)
	return rows
}

// swapOIDCLoginLogInserter 在用例内替换落库点，结束后恢复。
// 同包用例串行执行（无 t.Parallel），无并发改写风险。
func swapOIDCLoginLogInserter(t *testing.T, fn func(ctx context.Context, entity *model.UserLoginLogEntity) error) {
	t.Helper()
	original := insertOIDCLoginLog
	insertOIDCLoginLog = fn
	t.Cleanup(func() { insertOIDCLoginLog = original })
}

// TestCompleteLoginRecordsOIDCLoginLog 主路径：授权码登录成功后新增恰好 1 行登录日志，
// login_type='oidc'，并带上租户/用户/自然人/UA（T4.2 + 出参零变更）。
func TestCompleteLoginRecordsOIDCLoginLog(t *testing.T) {
	provider, ctx := newOIDCLoginLogEnv(t)
	ctx.Request.Header.Set("User-Agent", "oidc-login-test/1.0")
	authReq := newAuthReq(t, provider, "client-1")

	res, err := newOIDCLoginSuccessSvc(provider).CompleteLogin(ctx, &dtooidc.OIDCLoginReq{
		AuthRequestID: authReq.GetID(),
		Identifier:    "person@example.com",
		Password:      "Password1",

		Operator: operatorOf(ctx)})
	require.NoError(t, err)
	require.NotEmpty(t, res.ContinueURL, "登录成功必须继续签发授权码")

	rows := loadLoginLogs(t)
	require.Len(t, rows, 1, "成功登录必须恰好落 1 行登录日志")
	require.Equal(t, model.LoginTypeOIDC, rows[0].LoginType)
	require.Equal(t, "1", rows[0].TenantID)
	require.Equal(t, "66", rows[0].UserID)
	require.Equal(t, "88", rows[0].PersonID)
	require.Equal(t, operatorOf(ctx).UserAgent, rows[0].UserAgent)
	require.False(t, rows[0].LoginTime.IsZero(), "登录时间必须落库")
}

// TestCompleteLoginNoOIDCLoginLogBeforeTenantSelection 反例：多租户待选租户时登录尚未完成
// （done=false、不发 code），不得记成功日志——否则会被误读成"该租户已登录成功"。
func TestCompleteLoginNoOIDCLoginLogBeforeTenantSelection(t *testing.T) {
	provider, ctx := newOIDCLoginLogEnv(t)
	authReq := newAuthReq(t, provider, "client-1")

	svc := &oidcAuthSvc{
		provider: provider,
		authSvc: &fakePasswordAuthenticator{authenticate: func(ctx context.Context, identifier, password string) (*model.PersonEntity, *model.UserEntity, []objauth.TenantOption, error) {
			return &model.PersonEntity{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "88"}}},
				&model.UserEntity{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "66"}}, TenantID: "1", PersonID: "88"},
				[]objauth.TenantOption{
					{TenantID: "1", Name: "tenant-1", UserID: "66"},
					{TenantID: "7", Name: "tenant-7", UserID: "67"},
				}, nil
		}},
	}

	res, err := svc.CompleteLogin(requestCtx(ctx), &dtooidc.OIDCLoginReq{
		AuthRequestID: authReq.GetID(),
		Identifier:    "person@example.com",
		Password:      "Password1",

		Operator: operatorOf(ctx)})
	require.NoError(t, err)
	require.True(t, res.RequiresTenantSelection)
	require.Empty(t, res.ContinueURL)
	require.Empty(t, loadLoginLogs(t), "待选租户不是登录成功，不得写登录日志")
}

// TestSelectTenantRecordsOIDCLoginLog 多租户完成态：成功落点补在 SelectTenant，
// 保证"多租户登录"最终也有且仅有 1 行 login_type='oidc' 的记录。
func TestSelectTenantRecordsOIDCLoginLog(t *testing.T) {
	provider, ctx := newOIDCLoginLogEnv(t)
	authReq := newAuthReq(t, provider, "client-1")
	require.NoError(t, provider.Storage.CompleteAuthRequest(ctx, authReq.GetID(), "person:88",
		time.Now(), []string{"pwd"}, "", "", false))

	svc := &oidcAuthSvc{
		provider: provider,
		authSvc: &fakePasswordAuthenticator{tenantsForPerson: func(ctx context.Context, personID string) ([]objauth.TenantOption, error) {
			return []objauth.TenantOption{
				{TenantID: "3", Name: "tenant-3", UserID: "63"},
				{TenantID: "7", Name: "tenant-7", UserID: "67"},
			}, nil
		}},
	}

	res, err := svc.SelectTenant(requestCtx(ctx), operatorOf(ctx), authReq.GetID(), "7")
	require.NoError(t, err)
	require.NotEmpty(t, res.ContinueURL)

	rows := loadLoginLogs(t)
	require.Len(t, rows, 1)
	require.Equal(t, model.LoginTypeOIDC, rows[0].LoginType)
	require.Equal(t, "7", rows[0].TenantID)
	require.Equal(t, "67", rows[0].UserID)
	require.Equal(t, "88", rows[0].PersonID)
}

// TestCompleteLoginBySessionRecordsOIDCLoginLog 静默 SSO（prompt=none）成功：
// 这条路径没有密码校验，上游不会写 password 行，必须由本处补记 login_type='oidc'。
func TestCompleteLoginBySessionRecordsOIDCLoginLog(t *testing.T) {
	provider, ctx := newOIDCLoginLogEnv(t)
	authReq := newAuthReq(t, provider, "client-1")

	svc := &oidcAuthSvc{
		provider: provider,
		authSvc: &fakePasswordAuthenticator{tenantsForPerson: func(ctx context.Context, personID string) ([]objauth.TenantOption, error) {
			return []objauth.TenantOption{{TenantID: "7", Name: "tenant-7", UserID: "67"}}, nil
		}},
		ssoSessionStore: &fakeSSOSessionStore{validatedPersonID: "88"},
	}

	continueURL, err := svc.CompleteLoginBySession(requestCtx(ctx), operatorOf(ctx), authReq.GetID(), "session-x")
	require.NoError(t, err)
	require.Contains(t, continueURL, "id="+authReq.GetID())

	rows := loadLoginLogs(t)
	require.Len(t, rows, 1)
	require.Equal(t, model.LoginTypeOIDC, rows[0].LoginType)
	require.Equal(t, "7", rows[0].TenantID)
	require.Equal(t, "67", rows[0].UserID)
	require.Equal(t, "88", rows[0].PersonID)
}

// TestCompleteLoginSucceedsWhenLoginLogWriteFails best-effort（写库返回 error）：
// 日志写入失败绝不上抛、绝不阻断登录——授权票据照常完成、continueURL 照常返回。
func TestCompleteLoginSucceedsWhenLoginLogWriteFails(t *testing.T) {
	provider, ctx := newOIDCLoginLogEnv(t)
	swapOIDCLoginLogInserter(t, func(ctx context.Context, entity *model.UserLoginLogEntity) error {
		return errors.New("simulated insert failure")
	})
	authReq := newAuthReq(t, provider, "client-1")

	res, err := newOIDCLoginSuccessSvc(provider).CompleteLogin(ctx, &dtooidc.OIDCLoginReq{
		AuthRequestID: authReq.GetID(),
		Identifier:    "person@example.com",
		Password:      "Password1",

		Operator: operatorOf(ctx)})
	require.NoError(t, err, "登录日志写失败不得影响登录结果")
	require.NotEmpty(t, res.ContinueURL)
	require.Empty(t, loadLoginLogs(t))
}

// TestCompleteLoginSucceedsWhenLoginLogWritePanics best-effort（panic）：
// 等价于 DB 未初始化时 dao 拿到 nil *gorm.DB 的 panic——必须被 defer recover 兜住，
// 登录主流程照常完成（panic 若逃逸，本用例会直接失败）。
func TestCompleteLoginSucceedsWhenLoginLogWritePanics(t *testing.T) {
	provider, ctx := newOIDCLoginLogEnv(t)
	swapOIDCLoginLogInserter(t, func(ctx context.Context, entity *model.UserLoginLogEntity) error {
		panic("simulated nil *gorm.DB panic")
	})
	authReq := newAuthReq(t, provider, "client-1")

	res, err := newOIDCLoginSuccessSvc(provider).CompleteLogin(ctx, &dtooidc.OIDCLoginReq{
		AuthRequestID: authReq.GetID(),
		Identifier:    "person@example.com",
		Password:      "Password1",

		Operator: operatorOf(ctx)})
	require.NoError(t, err, "登录日志写入 panic 不得影响登录结果")
	require.NotEmpty(t, res.ContinueURL)
	require.Empty(t, loadLoginLogs(t))
}

// TestRecordOIDCLoginLogSurvivesUnavailableDB 直击 nil *gorm.DB 形态：临时清空 iam 库注册
// （DB 未初始化），recordOIDCLoginLog 不得 panic，写失败只记日志。
func TestRecordOIDCLoginLogSurvivesUnavailableDB(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	// 收尾重建全局 iam 库（Done 会重置 once，Initialize 因此重新 InitMultiDB）：
	// 既回收被本用例清空的注册，又避免污染同包后续用例。
	t.Cleanup(func() {
		testsetup.Done(testsetup.AppNameAuth)
		testsetup.Initialize(testsetup.AppNameAuth)
	})
	ctx := testsetup.NewCtx()
	ctx.Request.Header.Set("User-Agent", "oidc-login-test/1.0")

	dbclient.ClearDBForTest(dbclient.ServiceNameIam)
	require.Nil(t, dbclient.IamDB(ctx), "本用例要求 DB 不可用")

	require.NotPanics(t, func() {
		recordOIDCLoginLog(requestCtx(ctx), operatorOf(ctx), "1", "66", "88")
	})
}

// TestRecordOIDCLoginLogToleratesMissingRequest 边界：ctx 无 Request（gincontext 取 IP 会 panic）
// 时同样不得 panic。当前登录链路不会走到，纯防御性断言。
func TestRecordOIDCLoginLogToleratesMissingRequest(t *testing.T) {
	ctx := &gin.Context{}
	require.NotPanics(t, func() {
		recordOIDCLoginLog(requestCtx(ctx), operatorOf(ctx), "1", "66", "88")
	})
	require.NotPanics(t, func() {
		recordOIDCLoginLog(nil, objidentity.Operator{}, "1", "66", "88")
	})
}
