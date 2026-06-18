package svcsession

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/dto/dtouser"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/biz/genericdao"
	"gorm.io/gorm"
)

func TestSessionListReturnsPersonAwareTenantSessions(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	ginCtx.Request = req

	stored := &stubSessionStore{
		list: []model.RefreshTokenEntity{
			{
				Model:         gorm.Model{ID: 1, CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
				PersonID:      101,
				TenantID:      11,
				UserID:        21,
				OAuthClientID: 1,
				SessionID:     "session-1",
				ClientType:    "web",
				ClientIP:      "10.0.0.1",
				UserAgent:     "Mozilla/5.0",
				ExpiredAt:     timePointer(time.Now().Add(time.Hour)),
			},
			{
				Model:         gorm.Model{ID: 2, CreatedAt: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)},
				PersonID:      102,
				TenantID:      12,
				UserID:        22,
				OAuthClientID: 2,
				SessionID:     "session-2",
				ClientType:    "mobile",
				ClientIP:      "10.0.0.2",
				UserAgent:     "App/1.0",
				ExpiredAt:     timePointer(time.Now().Add(time.Hour)),
			},
		},
		total: 2,
	}
	installSessionStore(t, stored)

	svc := &sessionSvc{}
	resp, err := svc.List(ginCtx, &dtouser.SessionListReq{}, 101, 21, 11)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if resp.Total != 2 {
		t.Fatalf("expected total 2, got %d", resp.Total)
	}
	if len(resp.List) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(resp.List))
	}
	if resp.List[0].SessionID != "session-1" || resp.List[0].ClientType != "web" {
		t.Fatalf("unexpected session list item: %+v", resp.List[0])
	}
	if stored.lastCond == nil || stored.lastCond.PersonID != 101 || stored.lastCond.UserID != 21 || stored.lastCond.TenantID != 11 {
		t.Fatalf("expected cond with personID=101 userID=21 tenantID=11, got %+v", stored.lastCond)
	}
}

func timePointer(t time.Time) *time.Time {
	return &t
}

type stubSessionStore struct {
	list     model.RefreshTokenEntityList
	total    int64
	lastCond *dao.SessionCond
}

func (s *stubSessionStore) GetPageListByCond(ctx context.Context, cond genericdao.Cond) (model.RefreshTokenEntityList, int64, error) {
	if sc, ok := cond.(*dao.SessionCond); ok {
		clone := *sc
		s.lastCond = &clone
	}
	return s.list, s.total, nil
}

func (s *stubSessionStore) UpdateMap(ctx context.Context, id uint, updates map[string]any) error {
	return nil
}

func installSessionStore(t *testing.T, store sessionStore) {
	t.Helper()
	prev := newSessionStore
	newSessionStore = func() sessionStore {
		return store
	}
	t.Cleanup(func() {
		newSessionStore = prev
	})
}