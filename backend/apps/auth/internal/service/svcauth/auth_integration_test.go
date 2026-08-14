package svcauth

import (
	"testing"

	"github.com/morehao/ark-iam/auth/internal/dto/dtoauth"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/auth/testutil"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/testsetup"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterCreatesPersonAndUser(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)

	ctx := testsetup.NewCtx(testutil.WithIamContext(1))

	tenant, err := testsetup.PrepareTestTenant(ctx, testsetup.UniqueName("tenant"), "test_tag")
	require.NoError(t, err)
	defer func() { _ = testsetup.CleanupTestData(ctx, testsetup.TestDataIDs{TenantIDs: []uint{tenant.ID}}) }()

	username := testsetup.UniqueName("register")
	email := testsetup.UniqueName("reg") + "@example.com"
	phone := testsetup.UniqueName("regphone")
	svc := NewAuthSvc()
	resp, err := svc.Register(ctx, &dtoauth.RegisterReq{
		TenantID:     tenant.ID,
		Username:     username,
		PrimaryEmail: email,
		PrimaryPhone: phone,
		Password:     "Password1",
		Name:         "RegisterTest",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotZero(t, resp.UserID)

	db := dbclient.IamDB(ctx)
	var person model.PersonEntity
	err = db.Where("primary_email = ?", email).First(&person).Error
	require.NoError(t, err)
	assert.Equal(t, username, model.DerefStr(person.Username))
	assert.True(t, testsetup.PasswordMatches(person.PasswordEncrypted, "Password1"))

	defer func() {
		_ = testsetup.CleanupTestData(ctx, testsetup.TestDataIDs{
			PersonIDs: []uint{person.ID},
			UserIDs:   []uint{resp.UserID},
		})
	}()
}

func TestMyTenants(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)

	ctx := testsetup.NewCtx(testutil.WithIamContext(1))
	ctx.Set(gcontext.KeyPersonID, uint(1))

	svc := NewAuthSvc()
	resp, err := svc.MyTenants(ctx, &dtoauth.MyTenantsReq{})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Len(t, resp.List, 1)
}

func TestUserinfo(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)

	ctx := testsetup.NewCtx(testutil.WithIamContext(1))

	ctx.Set(gcontext.KeyUserID, uint(1))
	ctx.Set(gcontext.KeyTenantID, uint(1))
	ctx.Set(gcontext.KeyPersonID, uint(1))

	svc := NewAuthSvc()
	resp, err := svc.Userinfo(ctx, &dtoauth.UserinfoReq{})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, uint(1), resp.PersonInfo.PersonID)
	assert.Equal(t, uint(1), resp.UserInfo.UserID)
	assert.Equal(t, uint(1), resp.UserInfo.TenantID)
}

func TestLogout(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)

	ctx := testsetup.NewCtx(testutil.WithIamContext(1))
	ctx.Set(gcontext.KeyPersonID, uint(1))

	svc := NewAuthSvc()
	err := svc.Logout(ctx, &dtoauth.LogoutReq{RefreshToken: "test-refresh-token"})
	require.NoError(t, err)
}
