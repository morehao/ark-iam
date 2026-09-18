package svcauth

import (
	"testing"
	"time"

	"github.com/morehao/ark-iam/auth/internal/dto/dtoauth"
	"github.com/morehao/ark-iam/auth/testutil"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/testsetup"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupIntegrationDB 以内存 SQLite 注册为全局 iam 库，并播种操作人 user/person/tenant（ID 均为 "1"），
// 替代原先依赖真实数据库与种子数据的集成测试。
func setupIntegrationDB(t *testing.T) {
	t.Helper()
	db := testutil.SetupSQLite(t, &model.TenantEntity{}, &model.PersonEntity{}, &model.UserEntity{}, &model.DepartmentEntity{},
		&model.UserLoginLogEntity{}, &model.AuditLogEntity{})
	now := time.Now()
	seedTenant := &model.TenantEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "1"}},
		Code:       "seed-tenant",
		Name:       "seed-tenant",
	}
	if err := db.Create(seedTenant).Error; err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	seedPerson := &model.PersonEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "1"}},
		Name:       "seed-person",
	}
	if err := db.Create(seedPerson).Error; err != nil {
		t.Fatalf("seed person: %v", err)
	}
	seedUser := &model.UserEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "1"}},
		TenantID:   "1",
		PersonID:   "1",
		Name:       "seed-user",
		JoinedAt:   &now,
	}
	if err := db.Create(seedUser).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
}

func TestMyTenants(t *testing.T) {
	setupIntegrationDB(t)

	ctx := testsetup.NewCtx(testutil.WithIamContext("1"))
	ctx.Set(gcontext.KeyPersonID, "1")

	svc := NewAuthSvc()
	resp, err := svc.MyTenants(ctx, &dtoauth.MyTenantsReq{})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Len(t, resp.List, 1)
}

func TestUserinfo(t *testing.T) {
	setupIntegrationDB(t)

	ctx := testsetup.NewCtx(testutil.WithIamContext("1"))

	ctx.Set(gcontext.KeyUserID, "1")
	ctx.Set(gcontext.KeyTenantID, "1")
	ctx.Set(gcontext.KeyPersonID, "1")

	svc := NewAuthSvc()
	resp, err := svc.Userinfo(ctx, &dtoauth.UserinfoReq{})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "1", resp.PersonInfo.PersonID)
	assert.Equal(t, "1", resp.UserInfo.UserID)
	assert.Equal(t, "1", resp.UserInfo.TenantID)
}

func TestLogout(t *testing.T) {
	setupIntegrationDB(t)

	ctx := testsetup.NewCtx(testutil.WithIamContext("1"))
	ctx.Set(gcontext.KeyPersonID, "1")

	svc := NewAuthSvc()
	err := svc.Logout(ctx, &dtoauth.LogoutReq{})
	require.NoError(t, err)
}

// TestDefaultRecordLoginLogUpdatesLastSignInAt 回归（2026-09-18）：登录成功必须把 last_sign_in_at
// 落到 tenant_user。该表带 tenant_id，而 /oidc/login 链路没有租户中间件——不在方法内显式声明
// 作用域时，fail-closed 插件会拦下这次 UPDATE，但登录本身照常成功、只留一行 error 日志，
// 因此必须断言"值真的写进去了"，而不是断言登录返回成功。
func TestDefaultRecordLoginLogUpdatesLastSignInAt(t *testing.T) {
	setupIntegrationDB(t)

	// 刻意不声明租户作用域：与 /oidc/login 链路的 ctx 一致
	ctx := testsetup.NewCtx()

	defaultRecordLoginLog(ctx, "1", "1", true)

	user, err := dao.NewUserDao().GetByID(dbclient.CrossTenantContext(ctx), "1")
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.NotNil(t, user.LastSignInAt, "登录成功后 tenant_user.last_sign_in_at 必须落库")
}
