package svcauth

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/morehao/ark-iam/auth/internal/dto/dtoauth"
	"github.com/morehao/ark-iam/auth/testutil"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/iam/model"
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
	db := testutil.SetupSQLite(t, &model.TenantEntity{}, &model.PersonEntity{}, &model.UserEntity{}, &model.OrganizationEntity{})
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
		Profile:    json.RawMessage(`{}`),
		CustomData: json.RawMessage(`{}`),
	}
	if err := db.Create(seedPerson).Error; err != nil {
		t.Fatalf("seed person: %v", err)
	}
	seedUser := &model.UserEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "1"}},
		TenantID:   "1",
		PersonID:   "1",
		Name:       "seed-user",
		Profile:    json.RawMessage(`{}`),
		CustomData: json.RawMessage(`{}`),
		JoinedAt:   &now,
	}
	if err := db.Create(seedUser).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
}

func TestRegisterCreatesPersonAndUser(t *testing.T) {
	setupIntegrationDB(t)
	enableSelfRegister()

	ctx := testsetup.NewCtx(testutil.WithIamContext("1"))

	username := testsetup.UniqueName("register")
	email := testsetup.UniqueName("reg") + "@example.com"
	phone := testsetup.UniqueName("regphone")
	svc := NewAuthSvc()
	resp, err := svc.Register(ctx, &dtoauth.RegisterReq{
		TenantName:   testsetup.UniqueName("reg-tenant"),
		Username:     username,
		PrimaryEmail: email,
		PrimaryPhone: phone,
		Password:     "Password1",
		Name:         "RegisterTest",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotZero(t, resp.UserID)
	require.NotZero(t, resp.TenantID)

	db := dbclient.IamDB(ctx)
	var person model.PersonEntity
	err = db.Where("primary_email = ?", email).First(&person).Error
	require.NoError(t, err)
	assert.Equal(t, username, model.DerefStr(person.Username))
	assert.True(t, testsetup.PasswordMatches(person.PasswordEncrypted, "Password1"))

	var user model.UserEntity
	err = db.Where("id = ?", resp.UserID).First(&user).Error
	require.NoError(t, err)
	assert.Equal(t, resp.TenantID, user.TenantID)
	// 通道 A：自助开通租户的用户即 owner
	assert.True(t, user.IsOwner)

	defer func() {
		_ = testsetup.CleanupTestData(ctx, testsetup.TestDataIDs{
			PersonIDs: []string{person.ID},
			UserIDs:   []string{resp.UserID},
			TenantIDs: []string{resp.TenantID},
		})
	}()
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
