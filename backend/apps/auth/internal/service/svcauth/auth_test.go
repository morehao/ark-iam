package svcauth

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/internal/dto/dtoauth"
	"github.com/morehao/ark-iam/auth/testutil"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/gerror"
	"gorm.io/gorm"
)

type fakeAuthRefreshTokenStore struct {
	getByCondFunc        func(ctx context.Context, cond *dao.RefreshTokenCond) (*model.RefreshTokenEntity, error)
	insertFunc           func(ctx context.Context, entity *model.RefreshTokenEntity) error
	deleteFunc           func(ctx context.Context, id, userID string) error
	revokeByPersonIDFunc func(ctx context.Context, personID string) error
}

func (f *fakeAuthRefreshTokenStore) GetByCond(ctx context.Context, cond gormdao.Cond) (*model.RefreshTokenEntity, error) {
	if f.getByCondFunc == nil {
		return nil, nil
	}
	refreshTokenCond, _ := cond.(*dao.RefreshTokenCond)
	return f.getByCondFunc(ctx, refreshTokenCond)
}

func (f *fakeAuthRefreshTokenStore) Insert(ctx context.Context, entity *model.RefreshTokenEntity) error {
	if f.insertFunc == nil {
		return nil
	}
	return f.insertFunc(ctx, entity)
}

func (f *fakeAuthRefreshTokenStore) Delete(ctx context.Context, id, userID string) error {
	if f.deleteFunc == nil {
		return nil
	}
	return f.deleteFunc(ctx, id, userID)
}

func (f *fakeAuthRefreshTokenStore) RevokeByPersonID(ctx context.Context, personID string) error {
	if f.revokeByPersonIDFunc == nil {
		return nil
	}
	return f.revokeByPersonIDFunc(ctx, personID)
}

type fakeAuthUserStore struct {
	getByIDFunc       func(ctx context.Context, id string) (*model.UserEntity, error)
	getByCondFunc     func(ctx context.Context, cond *dao.UserCond) (*model.UserEntity, error)
	getListByCondFunc func(ctx context.Context, cond *dao.UserCond) (model.UserEntityList, error)
	insertFunc        func(ctx context.Context, entity *model.UserEntity) error
}

func (f *fakeAuthUserStore) GetByID(ctx context.Context, id string) (*model.UserEntity, error) {
	if f.getByIDFunc == nil {
		return nil, nil
	}
	return f.getByIDFunc(ctx, id)
}

func (f *fakeAuthUserStore) GetByCond(ctx context.Context, cond gormdao.Cond) (*model.UserEntity, error) {
	if f.getByCondFunc == nil {
		return nil, nil
	}
	userCond, _ := cond.(*dao.UserCond)
	return f.getByCondFunc(ctx, userCond)
}

func (f *fakeAuthUserStore) GetListByCond(ctx context.Context, cond gormdao.Cond) (model.UserEntityList, error) {
	if f.getListByCondFunc == nil {
		return nil, nil
	}
	userCond, _ := cond.(*dao.UserCond)
	return f.getListByCondFunc(ctx, userCond)
}

func (f *fakeAuthUserStore) Insert(ctx context.Context, entity *model.UserEntity) error {
	if f.insertFunc == nil {
		return nil
	}
	return f.insertFunc(ctx, entity)
}

func (f *fakeAuthUserStore) UpdateMap(ctx context.Context, id string, updates map[string]any) error {
	return nil
}

func TestLogoutAllRevokesAllRefreshTokensForPerson(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)
	ginCtx.Set(gcontext.KeyPersonID, "42")
	ginCtx.Request.Header.Set("Authorization", "")

	var revokedPersonID string
	restoreRefreshTokenStore := swapRefreshTokenStoreFactory(func() authRefreshTokenStore {
		return &fakeAuthRefreshTokenStore{
			revokeByPersonIDFunc: func(ctx context.Context, personID string) error {
				revokedPersonID = personID
				return nil
			},
		}
	})
	defer restoreRefreshTokenStore()

	svc := &authSvc{}
	err := svc.LogoutAll(ginCtx, &dtoauth.LogoutAllReq{})
	if err != nil {
		t.Fatalf("LogoutAll returned error: %v", err)
	}
	if revokedPersonID != "42" {
		t.Fatalf("expected LogoutAll to revoke refresh tokens for person 42, got %s", revokedPersonID)
	}
}

func TestRegisterReqBindingAllowsEmailOnlyIdentifier(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	body := []byte(`{"tenantID":"1","primaryEmail":"mail@example.com","password":"Password1","name":"tester"}`)
	req, err := http.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	ginCtx.Request = req

	var registerReq dtoauth.RegisterReq
	if err := ginCtx.ShouldBindJSON(&registerReq); err != nil {
		t.Fatalf("expected email-only registration request to bind, got error: %v", err)
	}
	if registerReq.Username != "" {
		t.Fatalf("expected empty username after bind, got %q", registerReq.Username)
	}
	if registerReq.PrimaryEmail != "mail@example.com" {
		t.Fatalf("expected bound primary email, got %q", registerReq.PrimaryEmail)
	}
}

func TestRegisterReqBindingAllowsPhoneOnlyIdentifier(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	body := []byte(`{"tenantID":"1","primaryPhone":"13800138000","password":"Password1","name":"tester"}`)
	req, err := http.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	ginCtx.Request = req

	var registerReq dtoauth.RegisterReq
	if err := ginCtx.ShouldBindJSON(&registerReq); err != nil {
		t.Fatalf("expected phone-only registration request to bind, got error: %v", err)
	}
	if registerReq.Username != "" {
		t.Fatalf("expected empty username after bind, got %q", registerReq.Username)
	}
	if registerReq.PrimaryPhone != "13800138000" {
		t.Fatalf("expected bound primary phone, got %q", registerReq.PrimaryPhone)
	}
}

func TestRegisterAllowsEmailOnlyIdentifier(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)

	db := testutil.SetupSQLite(t, &model.TenantEntity{}, &model.PersonEntity{}, &model.UserEntity{})
	seedTestTenant(t, db, "1", "租户A")

	svc := &authSvc{}
	resp, err := svc.Register(ginCtx, &dtoauth.RegisterReq{
		TenantID:     "1",
		PrimaryEmail: "mail@example.com",
		Password:     "Password1",
		Name:         "tester",
	})
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	if resp == nil || resp.UserID == "" {
		t.Fatalf("expected created user id, got %#v", resp)
	}
	var user model.UserEntity
	if err := db.Where("id = ?", resp.UserID).First(&user).Error; err != nil {
		t.Fatalf("expected user persisted: %v", err)
	}
	if user.TenantID != "1" || user.Name != "tester" || user.PersonID == "" {
		t.Fatalf("unexpected persisted user: %+v", user)
	}
	var person model.PersonEntity
	if err := db.Where("id = ?", user.PersonID).First(&person).Error; err != nil {
		t.Fatalf("expected person persisted: %v", err)
	}
	if model.DerefStr(person.PrimaryEmail) != "mail@example.com" {
		t.Fatalf("expected email persisted, got %q", model.DerefStr(person.PrimaryEmail))
	}
}

func TestRegisterAllowsPhoneOnlyIdentifier(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)

	db := testutil.SetupSQLite(t, &model.TenantEntity{}, &model.PersonEntity{}, &model.UserEntity{})
	seedTestTenant(t, db, "1", "租户A")

	svc := &authSvc{}
	resp, err := svc.Register(ginCtx, &dtoauth.RegisterReq{
		TenantID:     "1",
		PrimaryPhone: "13800138000",
		Password:     "Password1",
		Name:         "tester",
	})
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	if resp == nil || resp.UserID == "" {
		t.Fatalf("expected created user id, got %#v", resp)
	}
	var user model.UserEntity
	if err := db.Where("id = ?", resp.UserID).First(&user).Error; err != nil {
		t.Fatalf("expected user persisted: %v", err)
	}
	if user.TenantID != "1" || user.Name != "tester" {
		t.Fatalf("unexpected persisted user: %+v", user)
	}
}

func TestRegisterRejectsNonExistentTenant(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)

	testutil.SetupSQLite(t, &model.TenantEntity{}, &model.PersonEntity{}, &model.UserEntity{})

	svc := &authSvc{}
	_, err := svc.Register(ginCtx, &dtoauth.RegisterReq{
		TenantID: "999",
		Username: "new-user",
		Password: "Password1",
		Name:     "tester",
	})
	assertCode(t, err, code.TenantNotExistError)
}

func TestRegisterSetsUserJoinedAt(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)

	db := testutil.SetupSQLite(t, &model.TenantEntity{}, &model.PersonEntity{}, &model.UserEntity{})
	seedTestTenant(t, db, "1", "租户A")

	svc := &authSvc{}
	resp, err := svc.Register(ginCtx, &dtoauth.RegisterReq{
		TenantID: "1",
		Username: "new-user",
		Password: "Password1",
		Name:     "tester",
	})
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	if resp == nil || resp.UserID == "" {
		t.Fatalf("expected created user id, got %#v", resp)
	}
	var user model.UserEntity
	if err := db.Where("id = ?", resp.UserID).First(&user).Error; err != nil {
		t.Fatalf("expected user persisted: %v", err)
	}
	// H2：开放注册加入已有租户不授予 owner（owner 由租户创建/管理员授予）
	if user.IsOwner {
		t.Fatalf("expected register user NOT to be owner (H2), got isOwner=true")
	}
	if user.JoinedAt == nil {
		t.Fatal("expected register user to have joined_at set")
	}
}

func TestRegisterRequiresAtLeastOneIdentifier(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)

	restoreUserStore := swapUserStoreFactory(func() authUserStore {
		return &fakeAuthUserStore{}
	})
	defer restoreUserStore()

	svc := &authSvc{}
	_, err := svc.Register(ginCtx, &dtoauth.RegisterReq{
		TenantID: "1",
		Password: "Password1",
		Name:     "tester",
	})
	assertCode(t, err, code.AuthIdentifierRequiredError)
}

func TestRegisterCreatesPersonAccount(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)

	db := testutil.SetupSQLite(t, &model.TenantEntity{}, &model.PersonEntity{}, &model.UserEntity{})
	seedTestTenant(t, db, "1", "租户A")

	svc := &authSvc{}
	resp, err := svc.Register(ginCtx, &dtoauth.RegisterReq{
		TenantID:     "1",
		Username:     "person-user",
		PrimaryEmail: "mail@example.com",
		Password:     "Password1",
		Name:         "tester",
	})
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	if resp == nil || resp.UserID == "" {
		t.Fatalf("expected created tenant user id, got %#v", resp)
	}
	var user model.UserEntity
	if err := db.Where("id = ?", resp.UserID).First(&user).Error; err != nil {
		t.Fatalf("expected user persisted: %v", err)
	}
	var person model.PersonEntity
	if err := db.Where("id = ?", user.PersonID).First(&person).Error; err != nil {
		t.Fatalf("expected person account persisted: %v", err)
	}
	if model.DerefStr(person.Username) != "person-user" || model.DerefStr(person.PrimaryEmail) != "mail@example.com" {
		t.Fatalf("expected person identifiers to persist, got %#v", person)
	}
	if person.PasswordEncrypted == "" {
		t.Fatal("expected person password hash to be persisted")
	}
}

func assertCode(t *testing.T, err error, want int) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error code %d, got nil", want)
	}
	var gerr gerror.Error
	if !errors.As(err, &gerr) {
		t.Fatalf("expected gerror.Error, got %T", err)
	}
	if gerr.Code != want {
		t.Fatalf("expected error code %d, got %d", want, gerr.Code)
	}
}

func httptestRequest(t *testing.T) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, "/", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	return req
}

func swapRefreshTokenStoreFactory(factory func() authRefreshTokenStore) func() {
	prev := newAuthRefreshTokenStore
	newAuthRefreshTokenStore = factory
	return func() {
		newAuthRefreshTokenStore = prev
	}
}

func swapUserStoreFactory(factory func() authUserStore) func() {
	prev := newAuthUserStore
	newAuthUserStore = factory
	return func() {
		newAuthUserStore = prev
	}
}

func swapPersonStoreFactory(factory func() authPersonStore) func() {
	prev := newAuthPersonStore
	newAuthPersonStore = factory
	return func() {
		newAuthPersonStore = prev
	}
}

func swapTenantStoreFactory(factory func() authTenantStore) func() {
	prev := newAuthTenantStore
	newAuthTenantStore = factory
	return func() {
		newAuthTenantStore = prev
	}
}

// seedTestTenant 向测试库播种一个租户。
func seedTestTenant(t *testing.T, db *gorm.DB, id, name string) {
	t.Helper()
	tenant := &model.TenantEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: id}},
		Name:       name,
		Type:       model.TenantTypeCustomer,
	}
	if err := db.Create(tenant).Error; err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
}
