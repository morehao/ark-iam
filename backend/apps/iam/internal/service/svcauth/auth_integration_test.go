package svcauth

import (
	"testing"

	"github.com/morehao/ark-iam/iam/internal/dto/dtoauth"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/iam/testutil"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/testsetup"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/biz/testkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testJWTSecret = "your-jwt-secret-key"

func TestLoginByUsername(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	ctx := testsetup.NewCtx(testkit.WithContext(testutil.BuildIamContext(1)))

	svc := NewAuthSvc(testJWTSecret)
	resp, err := svc.Login(ctx, &dtoauth.LoginReq{
		Identifier: "admin",
		Password:   "admin123",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotEmpty(t, resp.PersonToken.AccessToken)
}

func TestLoginWithWrongPassword(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	ctx := testsetup.NewCtx(testkit.WithContext(testutil.BuildIamContext(1)))

	svc := NewAuthSvc(testJWTSecret)
	_, err := svc.Login(ctx, &dtoauth.LoginReq{
		Identifier: "admin",
		Password:   "WrongPassword",
	})
	assert.Error(t, err)
}

func TestRegisterCreatesPersonAndUser(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	ctx := testsetup.NewCtx(testkit.WithContext(testutil.BuildIamContext(1)))

	tenant, err := testsetup.PrepareTestTenant(ctx, testsetup.UniqueName("tenant"), "test_tag")
	require.NoError(t, err)
	defer testsetup.CleanupTestData(ctx, testsetup.TestDataIDs{TenantIDs: []uint{tenant.ID}})

	svc := NewAuthSvc(testJWTSecret)
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

	defer testsetup.CleanupTestData(ctx, testsetup.TestDataIDs{
		PersonIDs: []uint{},
		UserIDs:   []uint{resp.UserID},
	})

	db := dbclient.IamDB(ctx)
	var person model.PersonEntity
	err = db.Where("primary_email = ?", "register_test@example.com").First(&person).Error
	require.NoError(t, err)
	assert.Equal(t, "register_test_user", person.Username)
	assert.True(t, testsetup.PasswordMatches(person.PasswordEncrypted, "Password1"))
}

func TestSelectTenant(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	ctx := testsetup.NewCtx(testkit.WithContext(testutil.BuildIamContext(1)))

	svc := NewAuthSvc(testJWTSecret)
	loginResp, err := svc.Login(ctx, &dtoauth.LoginReq{
		Identifier: "admin",
		Password:   "admin123",
	})
	require.NoError(t, err)
	require.NotNil(t, loginResp)

	ctx.Set(gcontext.KeyPersonID, uint(1))

	resp, err := svc.SelectTenant(ctx, &dtoauth.SelectTenantReq{
		TenantID: 1,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotEmpty(t, resp.TokenInfo.AccessToken)
}

func TestSwitchTenantRejectsUnjoined(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	ctx := testsetup.NewCtx(testkit.WithContext(testutil.BuildIamContext(1)))

	ctx.Set(gcontext.KeyPersonID, uint(1))

	svc := NewAuthSvc(testJWTSecret)
	_, err := svc.SwitchTenant(ctx, &dtoauth.SwitchTenantReq{
		TenantID: 99999,
	})
	assert.Error(t, err)
}

func TestMyTenants(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	ctx := testsetup.NewCtx(testkit.WithContext(testutil.BuildIamContext(1)))

	svc := NewAuthSvc(testJWTSecret)
	loginResp, err := svc.Login(ctx, &dtoauth.LoginReq{
		Identifier: "admin",
		Password:   "admin123",
	})
	require.NoError(t, err)
	require.NotNil(t, loginResp)

	ctx.Set(gcontext.KeyPersonID, uint(1))

	resp, err := svc.MyTenants(ctx, &dtoauth.MyTenantsReq{})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Len(t, resp.List, 1)
}

func TestUserinfo(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	ctx := testsetup.NewCtx(testkit.WithContext(testutil.BuildIamContext(1)))

	ctx.Set(gcontext.KeyUserID, uint(1))
	ctx.Set(gcontext.KeyTenantID, uint(1))
	ctx.Set(gcontext.KeyPersonID, uint(1))

	svc := NewAuthSvc(testJWTSecret)
	resp, err := svc.Userinfo(ctx, &dtoauth.UserinfoReq{})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, uint(1), resp.PersonInfo.PersonID)
	assert.Equal(t, uint(1), resp.UserInfo.UserID)
	assert.Equal(t, uint(1), resp.UserInfo.TenantID)
}

func TestRefreshTokenValid(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	ctx := testsetup.NewCtx(testkit.WithContext(testutil.BuildIamContext(1)))

	svc := NewAuthSvc(testJWTSecret)
	loginResp, err := svc.Login(ctx, &dtoauth.LoginReq{
		Identifier: "admin",
		Password:   "admin123",
	})
	require.NoError(t, err)
	require.NotNil(t, loginResp)

	ctx.Set(gcontext.KeyPersonID, uint(1))

	tokenResp, err := svc.SelectTenant(ctx, &dtoauth.SelectTenantReq{
		TenantID: 1,
	})
	require.NoError(t, err)

	resp, err := svc.RefreshToken(ctx, &dtoauth.RefreshTokenReq{
		RefreshToken: tokenResp.TokenInfo.RefreshToken,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotEmpty(t, resp.TokenInfo.AccessToken)
}

func TestLogout(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	ctx := testsetup.NewCtx(testkit.WithContext(testutil.BuildIamContext(1)))

	svc := NewAuthSvc(testJWTSecret)
	loginResp, err := svc.Login(ctx, &dtoauth.LoginReq{
		Identifier: "admin",
		Password:   "admin123",
	})
	require.NoError(t, err)
	require.NotNil(t, loginResp)

	ctx.Set(gcontext.KeyPersonID, uint(1))

	tokenResp, err := svc.SelectTenant(ctx, &dtoauth.SelectTenantReq{
		TenantID: 1,
	})
	require.NoError(t, err)

	ctx.Request.Header.Set("Authorization", tokenResp.TokenInfo.AccessToken)

	err = svc.Logout(ctx, &dtoauth.LogoutReq{
		RefreshToken: tokenResp.TokenInfo.RefreshToken,
	})
	require.NoError(t, err)
}
