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
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/gconstant"
	"github.com/morehao/golib/gcrypto"
	"github.com/morehao/golib/gerror"
	"gorm.io/gorm"
)

type fakeAuthRefreshTokenStore struct {
	getByCondFunc        func(ctx context.Context, cond *dao.RefreshTokenCond) (*model.RefreshTokenEntity, error)
	insertFunc           func(ctx context.Context, entity *model.RefreshTokenEntity) error
	deleteFunc           func(ctx context.Context, id, userID uint) error
	revokeByPersonIDFunc func(ctx context.Context, personID uint) error
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

func (f *fakeAuthRefreshTokenStore) Delete(ctx context.Context, id, userID uint) error {
	if f.deleteFunc == nil {
		return nil
	}
	return f.deleteFunc(ctx, id, userID)
}

func (f *fakeAuthRefreshTokenStore) RevokeByPersonID(ctx context.Context, personID uint) error {
	if f.revokeByPersonIDFunc == nil {
		return nil
	}
	return f.revokeByPersonIDFunc(ctx, personID)
}

type fakeAuthUserStore struct {
	getByIDFunc       func(ctx context.Context, id uint) (*model.UserEntity, error)
	getByCondFunc     func(ctx context.Context, cond *dao.UserCond) (*model.UserEntity, error)
	getListByCondFunc func(ctx context.Context, cond *dao.UserCond) (model.UserEntityList, error)
	insertFunc        func(ctx context.Context, entity *model.UserEntity) error
}

func (f *fakeAuthUserStore) GetByID(ctx context.Context, id uint) (*model.UserEntity, error) {
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

func (f *fakeAuthUserStore) UpdateMap(ctx context.Context, id uint, updates map[string]interface{}) error {
	return nil
}

func TestGenerateTokenStoresRefreshTokenExpiredAt(t *testing.T) {
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
	_, err := svc.generateToken(ginCtx, &model.UserEntity{Model: gorm.Model{ID: 11}, TenantID: 22, Name: "tester"})
	after := time.Now()
	if err != nil {
		t.Fatalf("generateToken returned error: %v", err)
	}
	if inserted == nil {
		t.Fatal("expected refresh token to be inserted")
	}
	if inserted.ExpiredAt == nil {
		t.Fatal("expected refresh token expired_at to be set")
	}
	minExpire := before.Add(RefreshTokenExpireDuration - time.Second)
	maxExpire := after.Add(RefreshTokenExpireDuration + time.Second)
	if inserted.ExpiredAt.Before(minExpire) || inserted.ExpiredAt.After(maxExpire) {
		t.Fatalf("expected expired_at within [%v, %v], got %v", minExpire, maxExpire, *inserted.ExpiredAt)
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
	_, err := svc.generateToken(ginCtx, &model.UserEntity{Model: gorm.Model{ID: 11}, TenantID: 22, Name: "tester"})
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
					Model:     gorm.Model{ID: 1},
					UserID:    7,
					TenantID:  101,
					Token:     token.HashToken(refreshToken),
					ExpiredAt: timePointer(time.Now().Add(time.Hour)),
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
					Model:     gorm.Model{ID: 2},
					UserID:    7,
					TenantID:  100,
					Token:     token.HashToken(refreshToken),
					ExpiredAt: timePointer(time.Now().Add(time.Hour)),
					RevokedAt: timePointer(time.Now().Add(-time.Minute)),
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
					Model:     gorm.Model{ID: 3},
					UserID:    7,
					TenantID:  100,
					Token:     token.HashToken(refreshToken),
					ExpiredAt: timePointer(time.Now().Add(-time.Minute)),
				}, nil
			},
		}
	})
	defer restoreRefreshTokenStore()

	_, err := svc.RefreshToken(ginCtx, &dtoauth.RefreshTokenReq{RefreshToken: refreshToken})
	assertCode(t, err, code.RefreshTokenInvalidError)
}

func TestRefreshTokenRejectsStoredTokenWithoutExpiredAt(t *testing.T) {
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

func TestRefreshTokenRejectsStoredTokenWithInvalidExpiredAt(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)
	svc := &authSvc{jwtSecret: "test-secret"}
	refreshToken := signedRefreshToken(t, svc.jwtSecret, 7, 100)

	restoreRefreshTokenStore := swapRefreshTokenStoreFactory(func() authRefreshTokenStore {
		return &fakeAuthRefreshTokenStore{
			getByCondFunc: func(ctx context.Context, cond *dao.RefreshTokenCond) (*model.RefreshTokenEntity, error) {
				return &model.RefreshTokenEntity{
					Model:     gorm.Model{ID: 31},
					UserID:    7,
					TenantID:  100,
					Token:     token.HashToken(refreshToken),
					ExpiredAt: nil,
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
					Model:     gorm.Model{ID: 4},
					UserID:    7,
					TenantID:  100,
					Token:     token.HashToken(refreshToken),
					ExpiredAt: timePointer(time.Now().Add(time.Hour)),
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
				return &model.UserEntity{Model: gorm.Model{ID: id}, TenantID: 100, Name: "tester"}, nil
			},
		}
	})
	defer restoreUserStore()

	resp, err := svc.RefreshToken(ginCtx, &dtoauth.RefreshTokenReq{RefreshToken: refreshToken})
	if err != nil {
		t.Fatalf("RefreshToken returned error: %v", err)
	}
	if resp == nil || resp.AccessToken == "" {
		t.Fatal("expected refreshed token response")
	}
	if deletedID != 4 {
		t.Fatalf("expected deleted token id 4, got %d", deletedID)
	}
	if deletedUserID != 7 {
		t.Fatalf("expected delete to use stored user id 7, got %d", deletedUserID)
	}
}

func TestLogoutAllRevokesAllRefreshTokensForPerson(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)
	ginCtx.Set(gcontext.KeyPersonID, uint(42))
	ginCtx.Request.Header.Set("Authorization", "")

	var revokedPersonID uint
	restoreRefreshTokenStore := swapRefreshTokenStoreFactory(func() authRefreshTokenStore {
		return &fakeAuthRefreshTokenStore{
			revokeByPersonIDFunc: func(ctx context.Context, personID uint) error {
				revokedPersonID = personID
				return nil
			},
		}
	})
	defer restoreRefreshTokenStore()

	svc := &authSvc{jwtSecret: "test-secret"}
	err := svc.LogoutAll(ginCtx, &dtoauth.LogoutAllReq{})
	if err != nil {
		t.Fatalf("LogoutAll returned error: %v", err)
	}
	if revokedPersonID != 42 {
		t.Fatalf("expected LogoutAll to revoke refresh tokens for person 42, got %d", revokedPersonID)
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
	restorePersonStore := swapPersonStoreFactory(func() authPersonStore {
		return &fakeAuthPersonStore{
			getByCondFunc: func(ctx context.Context, cond *dao.PersonCond) (*model.PersonEntity, error) {
				return nil, nil
			},
			insertFunc: func(ctx context.Context, entity *model.PersonEntity) error {
				entity.ID = 100
				return nil
			},
		}
	})
	defer restorePersonStore()

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

	restoreTenantStore := swapTenantStoreFactory(func() authTenantStore {
		return &fakeAuthTenantStore{
			getByIDFunc: func(ctx context.Context, id uint) (*model.TenantEntity, error) {
				return &model.TenantEntity{Model: gorm.Model{ID: 1}, Name: "租户A", Tag: "a"}, nil
			},
		}
	})
	defer restoreTenantStore()

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
	if inserted.TenantID != 1 {
		t.Fatalf("expected tenant id 1, got %d", inserted.TenantID)
	}
	if inserted.Name != "tester" {
		t.Fatalf("expected name tester, got %q", inserted.Name)
	}
}

func TestRegisterAllowsPhoneOnlyIdentifier(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)

	var inserted *model.UserEntity
	restorePersonStore := swapPersonStoreFactory(func() authPersonStore {
		return &fakeAuthPersonStore{
			getByCondFunc: func(ctx context.Context, cond *dao.PersonCond) (*model.PersonEntity, error) {
				return nil, nil
			},
			insertFunc: func(ctx context.Context, entity *model.PersonEntity) error {
				entity.ID = 100
				return nil
			},
		}
	})
	defer restorePersonStore()

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

	restoreTenantStore := swapTenantStoreFactory(func() authTenantStore {
		return &fakeAuthTenantStore{
			getByIDFunc: func(ctx context.Context, id uint) (*model.TenantEntity, error) {
				return &model.TenantEntity{Model: gorm.Model{ID: 1}, Name: "租户A", Tag: "a"}, nil
			},
		}
	})
	defer restoreTenantStore()

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
	if inserted.TenantID != 1 {
		t.Fatalf("expected tenant id 1, got %d", inserted.TenantID)
	}
	if inserted.Name != "tester" {
		t.Fatalf("expected name tester, got %q", inserted.Name)
	}
}

func TestRegisterRejectsNonExistentTenant(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)

	restorePersonStore := swapPersonStoreFactory(func() authPersonStore {
		return &fakeAuthPersonStore{
			getByCondFunc: func(ctx context.Context, cond *dao.PersonCond) (*model.PersonEntity, error) {
				return nil, nil
			},
		}
	})
	defer restorePersonStore()
	restoreUserStore := swapUserStoreFactory(func() authUserStore {
		return &fakeAuthUserStore{
			getByCondFunc: func(ctx context.Context, cond *dao.UserCond) (*model.UserEntity, error) {
				return nil, nil
			},
		}
	})
	defer restoreUserStore()

	var tenantLookup uint
	restoreTenantStore := swapTenantStoreFactory(func() authTenantStore {
		return &fakeAuthTenantStore{
			getByIDFunc: func(ctx context.Context, id uint) (*model.TenantEntity, error) {
				tenantLookup = id
				return nil, nil
			},
		}
	})
	defer restoreTenantStore()

	svc := &authSvc{jwtSecret: "test-secret"}
	_, err := svc.Register(ginCtx, &dtoauth.RegisterReq{
		TenantID: 999,
		Username: "new-user",
		Password: "Password1",
		Name:     "tester",
	})
	assertCode(t, err, code.TenantNotExistError)
	if tenantLookup != 999 {
		t.Fatalf("expected tenant lookup with id 999, got %d", tenantLookup)
	}
}

func TestRegisterSetsUserAsOwnerAndJoinedAt(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)

	var insertedUser *model.UserEntity
	restorePersonStore := swapPersonStoreFactory(func() authPersonStore {
		return &fakeAuthPersonStore{
			getByCondFunc: func(ctx context.Context, cond *dao.PersonCond) (*model.PersonEntity, error) {
				return nil, nil
			},
			insertFunc: func(ctx context.Context, entity *model.PersonEntity) error {
				entity.ID = 88
				return nil
			},
		}
	})
	defer restorePersonStore()
	restoreUserStore := swapUserStoreFactory(func() authUserStore {
		return &fakeAuthUserStore{
			getByCondFunc: func(ctx context.Context, cond *dao.UserCond) (*model.UserEntity, error) {
				return nil, nil
			},
			insertFunc: func(ctx context.Context, entity *model.UserEntity) error {
				entity.ID = 101
				copied := *entity
				insertedUser = &copied
				return nil
			},
		}
	})
	defer restoreUserStore()
	restoreTenantStore := swapTenantStoreFactory(func() authTenantStore {
		return &fakeAuthTenantStore{
			getByIDFunc: func(ctx context.Context, id uint) (*model.TenantEntity, error) {
				return &model.TenantEntity{Model: gorm.Model{ID: 1}, Name: "租户A", Tag: "a"}, nil
			},
		}
	})
	defer restoreTenantStore()

	svc := &authSvc{jwtSecret: "test-secret"}
	resp, err := svc.Register(ginCtx, &dtoauth.RegisterReq{
		TenantID: 1,
		Username: "new-user",
		Password: "Password1",
		Name:     "tester",
	})
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	if resp == nil || resp.UserID != 101 {
		t.Fatalf("expected user id 101, got %#v", resp)
	}
	if insertedUser == nil {
		t.Fatal("expected user to be inserted")
	}
	if insertedUser.IsOwner != 1 {
		t.Fatalf("expected register user to be owner (isOwner=1), got %d", insertedUser.IsOwner)
	}
	if insertedUser.JoinedAt == nil {
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

	svc := &authSvc{jwtSecret: "test-secret"}
	_, err := svc.Register(ginCtx, &dtoauth.RegisterReq{
		TenantID: 1,
		Password: "Password1",
		Name:     "tester",
	})
	assertCode(t, err, code.AuthIdentifierRequiredError)
}

func TestRegisterCreatesPersonAccount(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)

	var insertedPerson *model.PersonEntity
	var insertedUser *model.UserEntity
	restorePersonStore := swapPersonStoreFactory(func() authPersonStore {
		return &fakeAuthPersonStore{
			getByCondFunc: func(ctx context.Context, cond *dao.PersonCond) (*model.PersonEntity, error) {
				return nil, nil
			},
			insertFunc: func(ctx context.Context, entity *model.PersonEntity) error {
				entity.ID = 88
				copied := *entity
				insertedPerson = &copied
				return nil
			},
		}
	})
	defer restorePersonStore()
	restoreUserStore := swapUserStoreFactory(func() authUserStore {
		return &fakeAuthUserStore{
			getByCondFunc: func(ctx context.Context, cond *dao.UserCond) (*model.UserEntity, error) {
				return nil, nil
			},
			insertFunc: func(ctx context.Context, entity *model.UserEntity) error {
				entity.ID = 101
				copied := *entity
				insertedUser = &copied
				return nil
			},
		}
	})
	defer restoreUserStore()
	restoreTenantStore := swapTenantStoreFactory(func() authTenantStore {
		return &fakeAuthTenantStore{
			getByIDFunc: func(ctx context.Context, id uint) (*model.TenantEntity, error) {
				return &model.TenantEntity{Model: gorm.Model{ID: 1}, Name: "租户A", Tag: "a"}, nil
			},
		}
	})
	defer restoreTenantStore()

	svc := &authSvc{jwtSecret: "test-secret"}
	resp, err := svc.Register(ginCtx, &dtoauth.RegisterReq{
		TenantID:     1,
		Username:     "person-user",
		PrimaryEmail: "mail@example.com",
		Password:     "Password1",
		Name:         "tester",
	})
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	if resp == nil || resp.UserID != 101 {
		t.Fatalf("expected created tenant user id 101, got %#v", resp)
	}
	if insertedPerson == nil {
		t.Fatal("expected person account to be inserted")
	}
	if insertedPerson.Username != "person-user" || insertedPerson.PrimaryEmail != "mail@example.com" {
		t.Fatalf("expected person identifiers to persist, got %#v", insertedPerson)
	}
	if insertedPerson.PasswordEncrypted == "" {
		t.Fatal("expected person password hash to be persisted")
	}
	if err := gcrypto.ComparePasswordHash(insertedPerson.PasswordEncrypted, "Password1"); err != nil {
		t.Fatalf("expected person password hash to match original password: %v", err)
	}
	if insertedUser == nil {
		t.Fatal("expected tenant user to be inserted")
	}
	if insertedUser.PersonID != 88 || insertedUser.TenantID != 1 {
		t.Fatalf("expected tenant user to reference created person, got %#v", insertedUser)
	}
	if insertedUser.Name != "tester" {
		t.Fatalf("expected tenant user display name tester, got %#v", insertedUser)
	}
}

func TestLoginReqBindingDoesNotRequireTenantID(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	body := []byte(`{"identifier":"person@example.com","password":"Password1"}`)
	req, err := http.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	ginCtx.Request = req

	var loginReq dtoauth.LoginReq
	if err := ginCtx.ShouldBindJSON(&loginReq); err != nil {
		t.Fatalf("expected login request without tenantID to bind, got error: %v", err)
	}
	if loginReq.Identifier != "person@example.com" {
		t.Fatalf("expected identifier to bind, got %q", loginReq.Identifier)
	}
}

func mustGeneratePasswordHash(t *testing.T, password string) string {
	t.Helper()
	hash, err := gcrypto.GeneratePasswordHash(password)
	if err != nil {
		t.Fatalf("GeneratePasswordHash failed: %v", err)
	}
	return hash
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

func swapLoginRecorder(recorder func(ctx *gin.Context, tenantID, userID uint, success bool)) func() {
	prev := authLoginRecorder
	authLoginRecorder = recorder
	return func() {
		authLoginRecorder = prev
	}
}
