package svcauth

import (
	"context"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/dto/dtoauth"
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/iam/testutil"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/testsetup"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/biz/testkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testJWTSecret = "your-jwt-secret-key"

func TestLoginByEmail(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	ctx := testsetup.NewContext(testkit.WithContext(testutil.BuildIamContext(1)))

	tenant, err := testsetup.PrepareTestTenant(ctx, testsetup.UniqueName("tenant"), "test_tag")
	require.NoError(t, err)
	defer testsetup.CleanupTestData(ctx, testsetup.TestDataIDs{TenantIDs: []uint{tenant.ID}})

	person, err := testsetup.PrepareTestPerson(ctx, "", "login_email_test@example.com", "", "Password1", "TestUser")
	require.NoError(t, err)
	defer testsetup.CleanupTestData(ctx, testsetup.TestDataIDs{PersonIDs: []uint{person.ID}})

	_, err = testsetup.PrepareTestUser(ctx, tenant.ID, person.ID, "TestUser", 1)
	require.NoError(t, err)

	ginCtx := createTestGinContext(ctx)
	svc := NewAuthSvc(testJWTSecret)
	resp, err := svc.Login(ginCtx, &dtoauth.LoginReq{
		Identifier: "login_email_test@example.com",
		Password:   "Password1",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotEmpty(t, resp.PersonToken.AccessToken)
	require.NotEmpty(t, resp.Tenants)
	assert.Equal(t, tenant.ID, resp.Tenants[0].TenantID)
}

func TestLoginByPhone(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	ctx := testsetup.NewContext(testkit.WithContext(testutil.BuildIamContext(1)))

	tenant, err := testsetup.PrepareTestTenant(ctx, testsetup.UniqueName("tenant"), "test_tag")
	require.NoError(t, err)
	defer testsetup.CleanupTestData(ctx, testsetup.TestDataIDs{TenantIDs: []uint{tenant.ID}})

	person, err := testsetup.PrepareTestPerson(ctx, "phone_test_user", "", "13800138000", "Password1", "TestUser")
	require.NoError(t, err)
	defer testsetup.CleanupTestData(ctx, testsetup.TestDataIDs{PersonIDs: []uint{person.ID}})

	_, err = testsetup.PrepareTestUser(ctx, tenant.ID, person.ID, "TestUser", 1)
	require.NoError(t, err)

	ginCtx := createTestGinContext(ctx)
	svc := NewAuthSvc(testJWTSecret)
	resp, err := svc.Login(ginCtx, &dtoauth.LoginReq{
		Identifier: "13800138000",
		Password:   "Password1",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotEmpty(t, resp.PersonToken.AccessToken)
}

func TestLoginByUsername(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	ctx := testsetup.NewContext(testkit.WithContext(testutil.BuildIamContext(1)))

	tenant, err := testsetup.PrepareTestTenant(ctx, testsetup.UniqueName("tenant"), "test_tag")
	require.NoError(t, err)
	defer testsetup.CleanupTestData(ctx, testsetup.TestDataIDs{TenantIDs: []uint{tenant.ID}})

	person, err := testsetup.PrepareTestPerson(ctx, "username_test_user", "", "", "Password1", "TestUser")
	require.NoError(t, err)
	defer testsetup.CleanupTestData(ctx, testsetup.TestDataIDs{PersonIDs: []uint{person.ID}})

	_, err = testsetup.PrepareTestUser(ctx, tenant.ID, person.ID, "TestUser", 1)
	require.NoError(t, err)

	ginCtx := createTestGinContext(ctx)
	svc := NewAuthSvc(testJWTSecret)
	resp, err := svc.Login(ginCtx, &dtoauth.LoginReq{
		Identifier: "username_test_user",
		Password:   "Password1",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotEmpty(t, resp.PersonToken.AccessToken)
}

func TestLoginWithWrongPassword(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	ctx := testsetup.NewContext(testkit.WithContext(testutil.BuildIamContext(1)))

	tenant, err := testsetup.PrepareTestTenant(ctx, testsetup.UniqueName("tenant"), "test_tag")
	require.NoError(t, err)
	defer testsetup.CleanupTestData(ctx, testsetup.TestDataIDs{TenantIDs: []uint{tenant.ID}})

	person, err := testsetup.PrepareTestPerson(ctx, "wrong_pwd_user", "", "", "Password1", "TestUser")
	require.NoError(t, err)
	defer testsetup.CleanupTestData(ctx, testsetup.TestDataIDs{PersonIDs: []uint{person.ID}})

	_, err = testsetup.PrepareTestUser(ctx, tenant.ID, person.ID, "TestUser", 1)
	require.NoError(t, err)

	ginCtx := createTestGinContext(ctx)
	svc := NewAuthSvc(testJWTSecret)
	_, err = svc.Login(ginCtx, &dtoauth.LoginReq{
		Identifier: "wrong_pwd_user",
		Password:   "WrongPassword",
	})
	assert.Error(t, err)
}

func TestLoginForSuspendedUser(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	ctx := testsetup.NewContext(testkit.WithContext(testutil.BuildIamContext(1)))

	tenant, err := testsetup.PrepareTestTenant(ctx, testsetup.UniqueName("tenant"), "test_tag")
	require.NoError(t, err)
	defer testsetup.CleanupTestData(ctx, testsetup.TestDataIDs{TenantIDs: []uint{tenant.ID}})

	person, err := testsetup.PrepareTestPerson(ctx, "suspended_user", "", "", "Password1", "TestUser")
	require.NoError(t, err)
	defer testsetup.CleanupTestData(ctx, testsetup.TestDataIDs{PersonIDs: []uint{person.ID}})

	user, err := testsetup.PrepareTestUser(ctx, tenant.ID, person.ID, "TestUser", 1)
	require.NoError(t, err)

	db := dbclient.IamDB(ctx)
	err = db.Model(user).Update("is_suspended", 1).Error
	require.NoError(t, err)

	ginCtx := createTestGinContext(ctx)
	svc := NewAuthSvc(testJWTSecret)
	_, err = svc.Login(ginCtx, &dtoauth.LoginReq{
		Identifier: "suspended_user",
		Password:   "Password1",
	})
	assert.Error(t, err)

	db.Model(user).Update("is_suspended", 0)
}

func TestRegisterCreatesPersonAndUser(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	ctx := testsetup.NewContext(testkit.WithContext(testutil.BuildIamContext(1)))

	tenant, err := testsetup.PrepareTestTenant(ctx, testsetup.UniqueName("tenant"), "test_tag")
	require.NoError(t, err)
	defer testsetup.CleanupTestData(ctx, testsetup.TestDataIDs{TenantIDs: []uint{tenant.ID}})

	ginCtx := createTestGinContext(ctx)
	svc := NewAuthSvc(testJWTSecret)
	resp, err := svc.Register(ginCtx, &dtoauth.RegisterReq{
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

	ctx := testsetup.NewContext(testkit.WithContext(testutil.BuildIamContext(1)))

	tenant, err := testsetup.PrepareTestTenant(ctx, testsetup.UniqueName("tenant"), "test_tag")
	require.NoError(t, err)
	defer testsetup.CleanupTestData(ctx, testsetup.TestDataIDs{TenantIDs: []uint{tenant.ID}})

	person, err := testsetup.PrepareTestPerson(ctx, "select_tenant_user", "", "", "Password1", "TestUser")
	require.NoError(t, err)
	defer testsetup.CleanupTestData(ctx, testsetup.TestDataIDs{PersonIDs: []uint{person.ID}})

	user, err := testsetup.PrepareTestUser(ctx, tenant.ID, person.ID, "TestUser", 1)
	require.NoError(t, err)
	defer testsetup.CleanupTestData(ctx, testsetup.TestDataIDs{UserIDs: []uint{user.ID}})

	ginCtx := createTestGinContext(ctx)
	ginCtx.Set(gcontext.KeyPersonID, person.ID)

	svc := NewAuthSvc(testJWTSecret)
	resp, err := svc.SelectTenant(ginCtx, &dtoauth.SelectTenantReq{
		TenantID: tenant.ID,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotEmpty(t, resp.TokenInfo.AccessToken)
}

func TestSwitchTenantToNewTenant(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	ctx := testsetup.NewContext(testkit.WithContext(testutil.BuildIamContext(1)))

	tenant1, err := testsetup.PrepareTestTenant(ctx, testsetup.UniqueName("tenant1"), "test_tag1")
	require.NoError(t, err)

	tenant2, err := testsetup.PrepareTestTenant(ctx, testsetup.UniqueName("tenant2"), "test_tag2")
	require.NoError(t, err)
	defer testsetup.CleanupTestData(ctx, testsetup.TestDataIDs{TenantIDs: []uint{tenant1.ID, tenant2.ID}})

	person, err := testsetup.PrepareTestPerson(ctx, "switch_tenant_user", "", "", "Password1", "TestUser")
	require.NoError(t, err)
	defer testsetup.CleanupTestData(ctx, testsetup.TestDataIDs{PersonIDs: []uint{person.ID}})

	user1, err := testsetup.PrepareTestUser(ctx, tenant1.ID, person.ID, "TestUser", 1)
	require.NoError(t, err)
	defer testsetup.CleanupTestData(ctx, testsetup.TestDataIDs{UserIDs: []uint{user1.ID}})

	user2, err := testsetup.PrepareTestUser(ctx, tenant2.ID, person.ID, "TestUser2", 0)
	require.NoError(t, err)
	defer testsetup.CleanupTestData(ctx, testsetup.TestDataIDs{UserIDs: []uint{user2.ID}})

	ginCtx := createTestGinContext(ctx)
	ginCtx.Set(gcontext.KeyPersonID, person.ID)

	svc := NewAuthSvc(testJWTSecret)
	resp, err := svc.SwitchTenant(ginCtx, &dtoauth.SwitchTenantReq{
		TenantID: tenant2.ID,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotEmpty(t, resp.TokenInfo.AccessToken)
}

func TestSwitchTenantRejectsUnjoined(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	ctx := testsetup.NewContext(testkit.WithContext(testutil.BuildIamContext(1)))

	tenant, err := testsetup.PrepareTestTenant(ctx, testsetup.UniqueName("tenant"), "test_tag")
	require.NoError(t, err)
	defer testsetup.CleanupTestData(ctx, testsetup.TestDataIDs{TenantIDs: []uint{tenant.ID}})

	person, err := testsetup.PrepareTestPerson(ctx, "unjoined_user", "", "", "Password1", "TestUser")
	require.NoError(t, err)
	defer testsetup.CleanupTestData(ctx, testsetup.TestDataIDs{PersonIDs: []uint{person.ID}})

	ginCtx := createTestGinContext(ctx)
	ginCtx.Set(gcontext.KeyPersonID, person.ID)

	svc := NewAuthSvc(testJWTSecret)
	_, err = svc.SwitchTenant(ginCtx, &dtoauth.SwitchTenantReq{
		TenantID: tenant.ID,
	})
	assert.Error(t, err)
}

func TestMyTenants(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	ctx := testsetup.NewContext(testkit.WithContext(testutil.BuildIamContext(1)))

	tenant1, err := testsetup.PrepareTestTenant(ctx, testsetup.UniqueName("tenant1"), "test_tag1")
	require.NoError(t, err)

	tenant2, err := testsetup.PrepareTestTenant(ctx, testsetup.UniqueName("tenant2"), "test_tag2")
	require.NoError(t, err)
	defer testsetup.CleanupTestData(ctx, testsetup.TestDataIDs{TenantIDs: []uint{tenant1.ID, tenant2.ID}})

	person, err := testsetup.PrepareTestPerson(ctx, "my_tenants_user", "", "", "Password1", "TestUser")
	require.NoError(t, err)
	defer testsetup.CleanupTestData(ctx, testsetup.TestDataIDs{PersonIDs: []uint{person.ID}})

	_, err = testsetup.PrepareTestUser(ctx, tenant1.ID, person.ID, "TestUser1", 1)
	require.NoError(t, err)

	_, err = testsetup.PrepareTestUser(ctx, tenant2.ID, person.ID, "TestUser2", 0)
	require.NoError(t, err)

	ginCtx := createTestGinContext(ctx)
	ginCtx.Set(gcontext.KeyPersonID, person.ID)

	svc := NewAuthSvc(testJWTSecret)
	resp, err := svc.MyTenants(ginCtx, &dtoauth.MyTenantsReq{})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Len(t, resp.List, 2)
}

func TestJoinTenant(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	ctx := testsetup.NewContext(testkit.WithContext(testutil.BuildIamContext(1)))

	tenant, err := testsetup.PrepareTestTenant(ctx, testsetup.UniqueName("tenant"), "test_tag")
	require.NoError(t, err)
	defer testsetup.CleanupTestData(ctx, testsetup.TestDataIDs{TenantIDs: []uint{tenant.ID}})

	person, err := testsetup.PrepareTestPerson(ctx, "join_tenant_user", "", "", "Password1", "TestUser")
	require.NoError(t, err)
	defer testsetup.CleanupTestData(ctx, testsetup.TestDataIDs{PersonIDs: []uint{person.ID}})

	ginCtx := createTestGinContext(ctx)
	ginCtx.Set(gcontext.KeyPersonID, person.ID)

	svc := NewAuthSvc(testJWTSecret)
	resp, err := svc.JoinTenant(ginCtx, &dtoauth.JoinTenantReq{
		TenantID: tenant.ID,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotZero(t, resp.UserID)

	defer testsetup.CleanupTestData(ctx, testsetup.TestDataIDs{UserIDs: []uint{resp.UserID}})

	db := dbclient.IamDB(ctx)
	var user model.UserEntity
	err = db.First(&user, resp.UserID).Error
	require.NoError(t, err)
	assert.Equal(t, tenant.ID, user.TenantID)
	assert.Equal(t, person.ID, user.PersonID)
	assert.Equal(t, int8(0), user.IsOwner)
}

func TestUserinfo(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	ctx := testsetup.NewContext(testkit.WithContext(testutil.BuildIamContext(1)))

	tenant, err := testsetup.PrepareTestTenant(ctx, testsetup.UniqueName("tenant"), "test_tag")
	require.NoError(t, err)
	defer testsetup.CleanupTestData(ctx, testsetup.TestDataIDs{TenantIDs: []uint{tenant.ID}})

	person, err := testsetup.PrepareTestPerson(ctx, "userinfo_user", "", "", "Password1", "TestUser")
	require.NoError(t, err)
	defer testsetup.CleanupTestData(ctx, testsetup.TestDataIDs{PersonIDs: []uint{person.ID}})

	user, err := testsetup.PrepareTestUser(ctx, tenant.ID, person.ID, "UserInfoTest", 1)
	require.NoError(t, err)
	defer testsetup.CleanupTestData(ctx, testsetup.TestDataIDs{UserIDs: []uint{user.ID}})

	ginCtx := createTestGinContext(ctx)
	ginCtx.Set(gcontext.KeyUserID, user.ID)
	ginCtx.Set(gcontext.KeyTenantID, tenant.ID)
	ginCtx.Set(gcontext.KeyPersonID, person.ID)

	svc := NewAuthSvc(testJWTSecret)
	resp, err := svc.Userinfo(ginCtx, &dtoauth.UserinfoReq{})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, person.ID, resp.PersonInfo.PersonID)
	assert.Equal(t, user.ID, resp.UserInfo.UserID)
	assert.Equal(t, tenant.ID, resp.UserInfo.TenantID)
	assert.Equal(t, "UserInfoTest", resp.UserInfo.Name)
}

func TestRefreshTokenValid(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	ctx := testsetup.NewContext(testkit.WithContext(testutil.BuildIamContext(1)))

	tenant, err := testsetup.PrepareTestTenant(ctx, testsetup.UniqueName("tenant"), "test_tag")
	require.NoError(t, err)
	defer testsetup.CleanupTestData(ctx, testsetup.TestDataIDs{TenantIDs: []uint{tenant.ID}})

	person, err := testsetup.PrepareTestPerson(ctx, "refresh_token_user", "", "", "Password1", "TestUser")
	require.NoError(t, err)
	defer testsetup.CleanupTestData(ctx, testsetup.TestDataIDs{PersonIDs: []uint{person.ID}})

	user, err := testsetup.PrepareTestUser(ctx, tenant.ID, person.ID, "TestUser", 1)
	require.NoError(t, err)
	defer testsetup.CleanupTestData(ctx, testsetup.TestDataIDs{UserIDs: []uint{user.ID}})

	ginCtx := createTestGinContext(ctx)
	ginCtx.Set(gcontext.KeyPersonID, person.ID)

	svc := NewAuthSvc(testJWTSecret)
	tokenResp, err := svc.SelectTenant(ginCtx, &dtoauth.SelectTenantReq{
		TenantID: tenant.ID,
	})
	require.NoError(t, err)

	resp, err := svc.RefreshToken(ginCtx, &dtoauth.RefreshTokenReq{
		RefreshToken: tokenResp.TokenInfo.RefreshToken,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotEmpty(t, resp.TokenInfo.AccessToken)
}

func TestLogout(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	ctx := testsetup.NewContext(testkit.WithContext(testutil.BuildIamContext(1)))

	tenant, err := testsetup.PrepareTestTenant(ctx, testsetup.UniqueName("tenant"), "test_tag")
	require.NoError(t, err)
	defer testsetup.CleanupTestData(ctx, testsetup.TestDataIDs{TenantIDs: []uint{tenant.ID}})

	person, err := testsetup.PrepareTestPerson(ctx, "logout_user", "", "", "Password1", "TestUser")
	require.NoError(t, err)
	defer testsetup.CleanupTestData(ctx, testsetup.TestDataIDs{PersonIDs: []uint{person.ID}})

	user, err := testsetup.PrepareTestUser(ctx, tenant.ID, person.ID, "TestUser", 1)
	require.NoError(t, err)
	defer testsetup.CleanupTestData(ctx, testsetup.TestDataIDs{UserIDs: []uint{user.ID}})

	ginCtx := createTestGinContext(ctx)
	ginCtx.Set(gcontext.KeyPersonID, person.ID)

	svc := NewAuthSvc(testJWTSecret)
	tokenResp, err := svc.SelectTenant(ginCtx, &dtoauth.SelectTenantReq{
		TenantID: tenant.ID,
	})
	require.NoError(t, err)

	ginCtx.Request.Header.Set("Authorization", tokenResp.TokenInfo.AccessToken)

	err = svc.Logout(ginCtx, &dtoauth.LogoutReq{
		RefreshToken: tokenResp.TokenInfo.RefreshToken,
	})
	require.NoError(t, err)
}

func createTestGinContext(ctx context.Context) *gin.Context {
	ginCtx, _ := gin.CreateTestContext(nil)
	req, _ := http.NewRequest(http.MethodPost, "/", nil)
	ginCtx.Request = req
	ginCtx.Request = ginCtx.Request.WithContext(ctx)
	return ginCtx
}
