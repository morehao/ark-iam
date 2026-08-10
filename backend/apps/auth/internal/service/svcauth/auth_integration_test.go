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
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	ctx := testsetup.NewCtx(testutil.WithIamContext(1))

	tenant, err := testsetup.PrepareTestTenant(ctx, testsetup.UniqueName("tenant"), "test_tag")
	require.NoError(t, err)
	defer func() { _ = testsetup.CleanupTestData(ctx, testsetup.TestDataIDs{TenantIDs: []uint{tenant.ID}}) }()

	svc := NewAuthSvc()
	resp, err := svc.Register(ctx, &dtoauth.RegisterReq{
		TenantID:     tenant.ID,
		Username:     "register_test_user",
		PrimaryEmail: "register_test@example.com",
		Password:     "Password1",
		Name:         "RegisterTest",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotZero(t, resp.UserID)

	defer func() {
		_ = testsetup.CleanupTestData(ctx, testsetup.TestDataIDs{
			PersonIDs: []uint{},
			UserIDs:   []uint{resp.UserID},
		})
	}()

	db := dbclient.IamDB(ctx)
	var person model.PersonEntity
	err = db.Where("primary_email = ?", "register_test@example.com").First(&person).Error
	require.NoError(t, err)
	assert.Equal(t, "register_test_user", person.Username)
	assert.True(t, testsetup.PasswordMatches(person.PasswordEncrypted, "Password1"))
}

func TestMyTenants(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	ctx := testsetup.NewCtx(testutil.WithIamContext(1))
	ctx.Set(gcontext.KeyPersonID, uint(1))

	svc := NewAuthSvc()
	resp, err := svc.MyTenants(ctx, &dtoauth.MyTenantsReq{})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Len(t, resp.List, 1)
}

func TestUserinfo(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

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
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	ctx := testsetup.NewCtx(testutil.WithIamContext(1))
	ctx.Set(gcontext.KeyPersonID, uint(1))

	svc := NewAuthSvc()
	err := svc.Logout(ctx, &dtoauth.LogoutReq{RefreshToken: "test-refresh-token"})
	require.NoError(t, err)
}
