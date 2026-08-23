package svcauth

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/internal/dto/dtoauth"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/gerror"
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
