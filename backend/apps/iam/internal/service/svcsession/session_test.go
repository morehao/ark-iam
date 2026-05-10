package svcsession

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/dto/dtouser"
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/golib/biz/gobject"
	"gorm.io/gorm"
)

type fakeSessionStore struct {
	getPageListByCondFunc func(ctx context.Context, cond *dao.SessionCond, page, pageSize int) ([]model.RefreshTokenEntity, int64, error)
	revokeByIDFunc        func(ctx context.Context, id, userID uint) error
	revokeAllByUserIDFunc func(ctx context.Context, userID uint) error
}

func (f *fakeSessionStore) GetPageListByCond(ctx context.Context, cond *dao.SessionCond, page, pageSize int) ([]model.RefreshTokenEntity, int64, error) {
	if f.getPageListByCondFunc == nil {
		return nil, 0, nil
	}
	return f.getPageListByCondFunc(ctx, cond, page, pageSize)
}

func (f *fakeSessionStore) RevokeByID(ctx context.Context, id, userID uint) error {
	if f.revokeByIDFunc == nil {
		return nil
	}
	return f.revokeByIDFunc(ctx, id, userID)
}

func (f *fakeSessionStore) RevokeAllByUserID(ctx context.Context, userID uint) error {
	if f.revokeAllByUserIDFunc == nil {
		return nil
	}
	return f.revokeAllByUserIDFunc(ctx, userID)
}

func TestSessionListMarksActivityByExpiresAt(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = newSessionTestRequest(t)
	now := time.Now()

	restoreSessionStore := swapSessionStoreFactory(func() sessionStore {
		return &fakeSessionStore{
			getPageListByCondFunc: func(ctx context.Context, cond *dao.SessionCond, page, pageSize int) ([]model.RefreshTokenEntity, int64, error) {
				return []model.RefreshTokenEntity{
					{
						Model: gorm.Model{ID: 1, CreatedAt: now},
					},
					{
						Model: gorm.Model{ID: 2, CreatedAt: now},
						ExpiresAt: &gorm.DeletedAt{
							Time:  now.Add(-time.Minute),
							Valid: true,
						},
					},
					{
						Model: gorm.Model{ID: 3, CreatedAt: now},
						ExpiresAt: &gorm.DeletedAt{
							Time:  now.Add(time.Minute),
							Valid: true,
						},
					},
				}, 3, nil
			},
		}
	})
	defer restoreSessionStore()

	resp, err := NewSessionSvc().List(ginCtx, &dtouser.SessionListReq{PageQuery: gobject.PageQuery{Page: 1, PageSize: 10}}, 99)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(resp.List) != 3 {
		t.Fatalf("expected 3 sessions, got %d", len(resp.List))
	}
	if resp.List[0].IsActive {
		t.Fatal("expected missing expires_at session to be inactive")
	}
	if resp.List[1].IsActive {
		t.Fatal("expected expired session to be inactive")
	}
	if !resp.List[2].IsActive {
		t.Fatal("expected unexpired session to be active")
	}
}

func newSessionTestRequest(t *testing.T) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, "/", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	return req
}

func swapSessionStoreFactory(factory func() sessionStore) func() {
	prev := newSessionStore
	newSessionStore = factory
	return func() {
		newSessionStore = prev
	}
}
