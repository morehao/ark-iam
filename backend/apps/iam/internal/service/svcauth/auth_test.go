package svcauth

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/morehao/ark-iam/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/dto/dtoauth"
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/token"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/genericdao"
	"github.com/morehao/golib/gerror"
	"gorm.io/gorm"
)

type fakeAuthRefreshTokenStore struct {
	getByCondFunc func(ctx context.Context, cond *dao.RefreshTokenCond) (*model.RefreshTokenEntity, error)
	insertFunc    func(ctx context.Context, entity *model.RefreshTokenEntity) error
	deleteFunc    func(ctx context.Context, id, userID uint) error
}

func (f *fakeAuthRefreshTokenStore) GetByCond(ctx context.Context, cond genericdao.Cond) (*model.RefreshTokenEntity, error) {
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

func (f *fakeAuthRefreshTokenStore) Delete(ctx context.Context, id, userID uint) error {
	if f.deleteFunc == nil {
		return nil
	}
	return f.deleteFunc(ctx, id, userID)
}

type fakeAuthUserStore struct {
	getByIDFunc   func(ctx context.Context, id uint) (*model.UserEntity, error)
	getByCondFunc func(ctx context.Context, cond *dao.UserCond) (*model.UserEntity, error)
	insertFunc    func(ctx context.Context, entity *model.UserEntity) error
}

func (f *fakeAuthUserStore) GetByID(ctx context.Context, id uint) (*model.UserEntity, error) {
	if f.getByIDFunc == nil {
		return nil, nil
	}
	return f.getByIDFunc(ctx, id)
}

func (f *fakeAuthUserStore) GetByCond(ctx context.Context, cond genericdao.Cond) (*model.UserEntity, error) {
	if f.getByCondFunc == nil {
		return nil, nil
	}
	userCond, _ := cond.(*dao.UserCond)
	return f.getByCondFunc(ctx, userCond)
}

func (f *fakeAuthUserStore) Insert(ctx context.Context, entity *model.UserEntity) error {
	if f.insertFunc == nil {
		return nil
	}
	return f.insertFunc(ctx, entity)
}

func (f *fakeAuthUserStore) UpdateMap(ctx context.Context, id uint, updates map[string]interface{}) error {
	return nil
}

func TestGenerateTokenStoresRefreshTokenExpiresAt(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)

	var inserted *model.RefreshTokenEntity
	restoreRefreshTokenStore := swapRefreshTokenStoreFactory(func() authRefreshTokenStore {
		return &fakeAuthRefreshTokenStore{
			insertFunc: func(ctx context.Context, entity *model.RefreshTokenEntity) error {
				copied := *entity
				inserted = &copied
				return nil
			},
		}
	})
	defer restoreRefreshTokenStore()

	svc := &authSvc{jwtSecret: "test-secret"}
	before := time.Now()
	_, err := svc.generateToken(ginCtx, &model.UserEntity{Model: gorm.Model{ID: 11}, TenantID: 22, Username: "tester"})
	after := time.Now()
	if err != nil {
		t.Fatalf("generateToken returned error: %v", err)
	}
	if inserted == nil {
		t.Fatal("expected refresh token to be inserted")
	}
	if inserted.ExpiresAt == nil || !inserted.ExpiresAt.Valid {
		t.Fatal("expected refresh token expires_at to be set")
	}
	minExpire := before.Add(RefreshTokenExpireDuration - time.Second)
	maxExpire := after.Add(RefreshTokenExpireDuration + time.Second)
	if inserted.ExpiresAt.Time.Before(minExpire) || inserted.ExpiresAt.Time.After(maxExpire) {
		t.Fatalf("expected expires_at within [%v, %v], got %v", minExpire, maxExpire, inserted.ExpiresAt.Time)
	}
}

func TestGenerateTokenReturnsErrorWhenRefreshTokenStoreFails(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)
	restoreRefreshTokenStore := swapRefreshTokenStoreFactory(func() authRefreshTokenStore {
		return &fakeAuthRefreshTokenStore{
			insertFunc: func(ctx context.Context, entity *model.RefreshTokenEntity) error {
				return errors.New("insert failed")
			},
		}
	})
	defer restoreRefreshTokenStore()

	svc := &authSvc{jwtSecret: "test-secret"}
	_, err := svc.generateToken(ginCtx, &model.UserEntity{Model: gorm.Model{ID: 11}, TenantID: 22, Username: "tester"})
	if err == nil {
		t.Fatal("expected generateToken to return error")
	}
}

func TestRefreshTokenRejectsStoredTokenWithTenantMismatch(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)
	svc := &authSvc{jwtSecret: "test-secret"}
	refreshToken := signedRefreshToken(t, svc.jwtSecret, 7, 100)

	restoreRefreshTokenStore := swapRefreshTokenStoreFactory(func() authRefreshTokenStore {
		return &fakeAuthRefreshTokenStore{
			getByCondFunc: func(ctx context.Context, cond *dao.RefreshTokenCond) (*model.RefreshTokenEntity, error) {
				return &model.RefreshTokenEntity{
					Model:    gorm.Model{ID: 1},
					UserID:   7,
					TenantID: 101,
					Token:    token.HashToken(refreshToken),
					ExpiresAt: &gorm.DeletedAt{
						Time:  time.Now().Add(time.Hour),
						Valid: true,
					},
				}, nil
			},
		}
	})
	defer restoreRefreshTokenStore()

	_, err := svc.RefreshToken(ginCtx, &dtoauth.RefreshTokenReq{RefreshToken: refreshToken})
	assertCode(t, err, code.RefreshTokenInvalidError)
}

func TestRefreshTokenRejectsRevokedToken(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)
	svc := &authSvc{jwtSecret: "test-secret"}
	refreshToken := signedRefreshToken(t, svc.jwtSecret, 7, 100)

	restoreRefreshTokenStore := swapRefreshTokenStoreFactory(func() authRefreshTokenStore {
		return &fakeAuthRefreshTokenStore{
			getByCondFunc: func(ctx context.Context, cond *dao.RefreshTokenCond) (*model.RefreshTokenEntity, error) {
				return &model.RefreshTokenEntity{
					Model:    gorm.Model{ID: 2},
					UserID:   7,
					TenantID: 100,
					Token:    token.HashToken(refreshToken),
					ExpiresAt: &gorm.DeletedAt{
						Time:  time.Now().Add(time.Hour),
						Valid: true,
					},
					RevokedAt: &gorm.DeletedAt{
						Time:  time.Now().Add(-time.Minute),
						Valid: true,
					},
				}, nil
			},
		}
	})
	defer restoreRefreshTokenStore()

	_, err := svc.RefreshToken(ginCtx, &dtoauth.RefreshTokenReq{RefreshToken: refreshToken})
	assertCode(t, err, code.RefreshTokenInvalidError)
}

func TestRefreshTokenRejectsExpiredStoredToken(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)
	svc := &authSvc{jwtSecret: "test-secret"}
	refreshToken := signedRefreshToken(t, svc.jwtSecret, 7, 100)

	restoreRefreshTokenStore := swapRefreshTokenStoreFactory(func() authRefreshTokenStore {
		return &fakeAuthRefreshTokenStore{
			getByCondFunc: func(ctx context.Context, cond *dao.RefreshTokenCond) (*model.RefreshTokenEntity, error) {
				return &model.RefreshTokenEntity{
					Model:    gorm.Model{ID: 3},
					UserID:   7,
					TenantID: 100,
					Token:    token.HashToken(refreshToken),
					ExpiresAt: &gorm.DeletedAt{
						Time:  time.Now().Add(-time.Minute),
						Valid: true,
					},
				}, nil
			},
		}
	})
	defer restoreRefreshTokenStore()

	_, err := svc.RefreshToken(ginCtx, &dtoauth.RefreshTokenReq{RefreshToken: refreshToken})
	assertCode(t, err, code.RefreshTokenInvalidError)
}

func TestRefreshTokenRejectsStoredTokenWithoutExpiresAt(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)
	svc := &authSvc{jwtSecret: "test-secret"}
	refreshToken := signedRefreshToken(t, svc.jwtSecret, 7, 100)

	restoreRefreshTokenStore := swapRefreshTokenStoreFactory(func() authRefreshTokenStore {
		return &fakeAuthRefreshTokenStore{
			getByCondFunc: func(ctx context.Context, cond *dao.RefreshTokenCond) (*model.RefreshTokenEntity, error) {
				return &model.RefreshTokenEntity{
					Model:    gorm.Model{ID: 30},
					UserID:   7,
					TenantID: 100,
					Token:    token.HashToken(refreshToken),
				}, nil
			},
		}
	})
	defer restoreRefreshTokenStore()

	_, err := svc.RefreshToken(ginCtx, &dtoauth.RefreshTokenReq{RefreshToken: refreshToken})
	assertCode(t, err, code.RefreshTokenInvalidError)
}

func TestRefreshTokenRejectsStoredTokenWithInvalidExpiresAt(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)
	svc := &authSvc{jwtSecret: "test-secret"}
	refreshToken := signedRefreshToken(t, svc.jwtSecret, 7, 100)

	restoreRefreshTokenStore := swapRefreshTokenStoreFactory(func() authRefreshTokenStore {
		return &fakeAuthRefreshTokenStore{
			getByCondFunc: func(ctx context.Context, cond *dao.RefreshTokenCond) (*model.RefreshTokenEntity, error) {
				return &model.RefreshTokenEntity{
					Model:    gorm.Model{ID: 31},
					UserID:   7,
					TenantID: 100,
					Token:    token.HashToken(refreshToken),
					ExpiresAt: &gorm.DeletedAt{
						Valid: false,
					},
				}, nil
			},
		}
	})
	defer restoreRefreshTokenStore()

	_, err := svc.RefreshToken(ginCtx, &dtoauth.RefreshTokenReq{RefreshToken: refreshToken})
	assertCode(t, err, code.RefreshTokenInvalidError)
}

func TestRefreshTokenUsesStoredUserIDWhenDeletingOldToken(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)
	svc := &authSvc{jwtSecret: "test-secret"}
	refreshToken := signedRefreshToken(t, svc.jwtSecret, 7, 100)

	var deletedID uint
	var deletedUserID uint
	restoreRefreshTokenStore := swapRefreshTokenStoreFactory(func() authRefreshTokenStore {
		return &fakeAuthRefreshTokenStore{
			getByCondFunc: func(ctx context.Context, cond *dao.RefreshTokenCond) (*model.RefreshTokenEntity, error) {
				return &model.RefreshTokenEntity{
					Model:    gorm.Model{ID: 4},
					UserID:   7,
					TenantID: 100,
					Token:    token.HashToken(refreshToken),
					ExpiresAt: &gorm.DeletedAt{
						Time:  time.Now().Add(time.Hour),
						Valid: true,
					},
				}, nil
			},
			deleteFunc: func(ctx context.Context, id, userID uint) error {
				deletedID = id
				deletedUserID = userID
				return nil
			},
			insertFunc: func(ctx context.Context, entity *model.RefreshTokenEntity) error {
				return nil
			},
		}
	})
	defer restoreRefreshTokenStore()

	restoreUserStore := swapUserStoreFactory(func() authUserStore {
		return &fakeAuthUserStore{
			getByIDFunc: func(ctx context.Context, id uint) (*model.UserEntity, error) {
				return &model.UserEntity{Model: gorm.Model{ID: id}, TenantID: 100, Username: "tester"}, nil
			},
		}
	})
	defer restoreUserStore()

	resp, err := svc.RefreshToken(ginCtx, &dtoauth.RefreshTokenReq{RefreshToken: refreshToken})
	if err != nil {
		t.Fatalf("RefreshToken returned error: %v", err)
	}
	if resp == nil || resp.TokenInfo.AccessToken == "" {
		t.Fatal("expected refreshed token response")
	}
	if deletedID != 4 {
		t.Fatalf("expected deleted token id 4, got %d", deletedID)
	}
	if deletedUserID != 7 {
		t.Fatalf("expected delete to use stored user id 7, got %d", deletedUserID)
	}
}

func TestRefreshTokenRejectsInvalidTenantClaim(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)
	svc := &authSvc{jwtSecret: "test-secret"}
	refreshToken := signedRefreshTokenWithoutTenant(t, svc.jwtSecret, 7)

	_, err := svc.RefreshToken(ginCtx, &dtoauth.RefreshTokenReq{RefreshToken: refreshToken})
	assertCode(t, err, gconstant.TokenInvalidErr)
}

func TestRefreshTokenRejectsNegativeTenantClaim(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)
	svc := &authSvc{jwtSecret: "test-secret"}
	refreshToken := signedRefreshTokenWithTenantClaim(t, svc.jwtSecret, 7, -1)

	_, err := svc.RefreshToken(ginCtx, &dtoauth.RefreshTokenReq{RefreshToken: refreshToken})
	assertCode(t, err, gconstant.TokenInvalidErr)
}

func TestRefreshTokenRejectsFractionalTenantClaim(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)
	svc := &authSvc{jwtSecret: "test-secret"}
	refreshToken := signedRefreshTokenWithTenantClaim(t, svc.jwtSecret, 7, 1.5)

	_, err := svc.RefreshToken(ginCtx, &dtoauth.RefreshTokenReq{RefreshToken: refreshToken})
	assertCode(t, err, gconstant.TokenInvalidErr)
}

func TestRefreshTokenRejectsNegativeUserClaim(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)
	svc := &authSvc{jwtSecret: "test-secret"}
	refreshToken := signedRefreshTokenWithClaims(t, svc.jwtSecret, -1, 100)

	_, err := svc.RefreshToken(ginCtx, &dtoauth.RefreshTokenReq{RefreshToken: refreshToken})
	assertCode(t, err, gconstant.TokenInvalidErr)
}

func TestRefreshTokenRejectsFractionalUserClaim(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)
	svc := &authSvc{jwtSecret: "test-secret"}
	refreshToken := signedRefreshTokenWithClaims(t, svc.jwtSecret, 1.5, 100)

	_, err := svc.RefreshToken(ginCtx, &dtoauth.RefreshTokenReq{RefreshToken: refreshToken})
	assertCode(t, err, gconstant.TokenInvalidErr)
}

func TestRegisterReqBindingAllowsEmailOnlyIdentifier(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	body := []byte(`{"tenantID":1,"primaryEmail":"mail@example.com","password":"Password1","name":"tester"}`)
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
	body := []byte(`{"tenantID":1,"primaryPhone":"13800138000","password":"Password1","name":"tester"}`)
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

	var inserted *model.UserEntity
	restoreUserStore := swapUserStoreFactory(func() authUserStore {
		return &fakeAuthUserStore{
			getByCondFunc: func(ctx context.Context, cond *dao.UserCond) (*model.UserEntity, error) {
				return nil, nil
			},
			insertFunc: func(ctx context.Context, entity *model.UserEntity) error {
				entity.ID = 101
				copied := *entity
				inserted = &copied
				return nil
			},
		}
	})
	defer restoreUserStore()

	svc := &authSvc{jwtSecret: "test-secret"}
	resp, err := svc.Register(ginCtx, &dtoauth.RegisterReq{
		TenantID:     1,
		PrimaryEmail: "mail@example.com",
		Password:     "Password1",
		Name:         "tester",
	})
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	if resp == nil || resp.UserID != 101 {
		t.Fatalf("expected user id 101, got %#v", resp)
	}
	if inserted == nil {
		t.Fatal("expected user to be inserted")
	}
	if inserted.Username != "" {
		t.Fatalf("expected empty username, got %q", inserted.Username)
	}
	if inserted.PrimaryEmail != "mail@example.com" {
		t.Fatalf("expected primary email to be stored, got %q", inserted.PrimaryEmail)
	}
	if inserted.PrimaryPhone != "" {
		t.Fatalf("expected empty primary phone, got %q", inserted.PrimaryPhone)
	}
}

func TestRegisterAllowsPhoneOnlyIdentifier(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)

	var inserted *model.UserEntity
	restoreUserStore := swapUserStoreFactory(func() authUserStore {
		return &fakeAuthUserStore{
			getByCondFunc: func(ctx context.Context, cond *dao.UserCond) (*model.UserEntity, error) {
				return nil, nil
			},
			insertFunc: func(ctx context.Context, entity *model.UserEntity) error {
				entity.ID = 102
				copied := *entity
				inserted = &copied
				return nil
			},
		}
	})
	defer restoreUserStore()

	svc := &authSvc{jwtSecret: "test-secret"}
	resp, err := svc.Register(ginCtx, &dtoauth.RegisterReq{
		TenantID:     1,
		PrimaryPhone: "13800138000",
		Password:     "Password1",
		Name:         "tester",
	})
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	if resp == nil || resp.UserID != 102 {
		t.Fatalf("expected user id 102, got %#v", resp)
	}
	if inserted == nil {
		t.Fatal("expected user to be inserted")
	}
	if inserted.Username != "" {
		t.Fatalf("expected empty username, got %q", inserted.Username)
	}
	if inserted.PrimaryEmail != "" {
		t.Fatalf("expected empty primary email, got %q", inserted.PrimaryEmail)
	}
	if inserted.PrimaryPhone != "13800138000" {
		t.Fatalf("expected primary phone to be stored, got %q", inserted.PrimaryPhone)
	}
}

func TestRegisterRequiresAtLeastOneIdentifier(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)

	restoreUserStore := swapUserStoreFactory(func() authUserStore {
		return &fakeAuthUserStore{}
	})
	defer restoreUserStore()

	svc := &authSvc{jwtSecret: "test-secret"}
	_, err := svc.Register(ginCtx, &dtoauth.RegisterReq{
		TenantID: 1,
		Password: "Password1",
		Name:     "tester",
	})
	assertCode(t, err, code.AuthIdentifierRequiredError)
}

func signedRefreshToken(t *testing.T, secret string, userID, tenantID uint) string {
	t.Helper()
	claims := jwt.MapClaims{
		"user_id":   userID,
		"tenant_id": tenantID,
		"exp":       time.Now().Add(time.Hour).Unix(),
		"iat":       time.Now().Unix(),
		"type":      "refresh",
	}
	tokenString, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign refresh token: %v", err)
	}
	return tokenString
}

func signedRefreshTokenWithoutTenant(t *testing.T, secret string, userID uint) string {
	t.Helper()
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour).Unix(),
		"iat":     time.Now().Unix(),
		"type":    "refresh",
	}
	tokenString, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign refresh token: %v", err)
	}
	return tokenString
}

func signedRefreshTokenWithTenantClaim(t *testing.T, secret string, userID uint, tenantID any) string {
	t.Helper()
	return signedRefreshTokenWithClaims(t, secret, userID, tenantID)
}

func signedRefreshTokenWithClaims(t *testing.T, secret string, userID, tenantID any) string {
	t.Helper()
	claims := jwt.MapClaims{
		"user_id":   userID,
		"tenant_id": tenantID,
		"exp":       time.Now().Add(time.Hour).Unix(),
		"iat":       time.Now().Unix(),
		"type":      "refresh",
	}
	tokenString, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign refresh token: %v", err)
	}
	return tokenString
}

func assertCode(t *testing.T, err error, want int) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error code %d, got nil", want)
	}
	gerr, ok := err.(*gerror.Error)
	if !ok {
		t.Fatalf("expected *gerror.Error, got %T", err)
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
