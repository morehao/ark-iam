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
				ApplicationID: 1,
				SessionID:     "session-1",
				ClientType:    "web",
				ClientIP:      "10.0.0.1",
				UserAgent:     "Mozilla/5.0",
				ExpiresAt:     timePointer(time.Now().Add(time.Hour)),
			},
			{
				Model:         gorm.Model{ID: 2, CreatedAt: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)},
				PersonID:      102,
				TenantID:      12,
				UserID:        22,
				ApplicationID: 2,
				SessionID:     "session-2",
				ClientType:    "mobile",
				ClientIP:      "10.0.0.2",
				UserAgent:     "App/1.0",
				ExpiresAt:     timePointer(time.Now().Add(time.Hour)),
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

func TestSessionRevokeUsesPersonIDAndTenantIDAndUserID(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	req, _ := http.NewRequest(http.MethodPost, "/", nil)
	ginCtx.Request = req

	stored := &stubSessionStore{}
	installSessionStore(t, stored)

	svc := &sessionSvc{}
	err := svc.Revoke(ginCtx, &dtouser.SessionRevokeReq{SessionID: 3}, 21, 11, 101)
	if err != nil {
		t.Fatalf("Revoke returned error: %v", err)
	}
	if stored.revokedID != 3 || stored.revokedUserID != 21 || stored.revokedPersonID != 101 || stored.revokedTenantID != 11 {
		t.Fatalf("expected revoke with id=3 userID=21 personID=101 tenantID=11, got id=%d userID=%d personID=%d tenantID=%d",
			stored.revokedID, stored.revokedUserID, stored.revokedPersonID, stored.revokedTenantID)
	}
}

func TestSessionRevokeAllUsesPersonIDAndTenantID(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	req, _ := http.NewRequest(http.MethodPost, "/", nil)
	ginCtx.Request = req

	stored := &stubSessionStore{}
	installSessionStore(t, stored)

	svc := &sessionSvc{}
	err := svc.RevokeAll(ginCtx, 21, 11, 101)
	if err != nil {
		t.Fatalf("RevokeAll returned error: %v", err)
	}
	if stored.revokeAllUserID != 21 || stored.revokeAllPersonID != 101 || stored.revokeAllTenantID != 11 {
		t.Fatalf("expected revokeAll with userID=21 personID=101 tenantID=11, got userID=%d personID=%d tenantID=%d",
			stored.revokeAllUserID, stored.revokeAllPersonID, stored.revokeAllTenantID)
	}
}

func timePointer(t time.Time) *time.Time {
	return &t
}

type stubSessionStore struct {
	list            []model.RefreshTokenEntity
	total           int64
	lastCond        *dao.SessionCond
	revokedID       uint
	revokedPersonID uint
	revokedTenantID uint
	revokedUserID   uint
	revokeAllPersonID uint
	revokeAllTenantID uint
	revokeAllUserID uint
}

func (s *stubSessionStore) GetPageListByCond(ctx context.Context, cond *dao.SessionCond, page, pageSize int) ([]model.RefreshTokenEntity, int64, error) {
	clone := *cond
	s.lastCond = &clone
	return s.list, s.total, nil
}

func (s *stubSessionStore) RevokeByID(ctx context.Context, id, personID, tenantID, userID uint) error {
	s.revokedID = id
	s.revokedPersonID = personID
	s.revokedTenantID = tenantID
	s.revokedUserID = userID
	return nil
}

func (s *stubSessionStore) RevokeAllByUserID(ctx context.Context, personID, tenantID, userID uint) error {
	s.revokeAllPersonID = personID
	s.revokeAllTenantID = tenantID
	s.revokeAllUserID = userID
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