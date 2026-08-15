package svcsession

import (
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/internal/dto/dtouser"
	"github.com/morehao/ark-iam/auth/testutil"
	"github.com/morehao/ark-iam/pkg/iam/model"
)

func TestSessionListReturnsPersonAwareTenantSessions(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	ginCtx.Request = req

	db := testutil.SetupSQLite(t, &model.RefreshTokenEntity{})
	expiresAt := time.Now().Add(time.Hour)
	// 仅 person 101/tenant 11/user 21 的会话应被返回
	if err := db.Create(&model.RefreshTokenEntity{
		PersonID:            "101",
		TenantID:            "11",
		UserID:              "21",
		ApplicationClientID: "1",
		SessionID:           "session-1",
		ClientType:          "web",
		ClientIP:            "10.0.0.1",
		UserAgent:           "Mozilla/5.0",
		ExpiredAt:           &expiresAt,
	}).Error; err != nil {
		t.Fatalf("seed session-1: %v", err)
	}
	if err := db.Create(&model.RefreshTokenEntity{
		PersonID:            "102",
		TenantID:            "12",
		UserID:              "22",
		ApplicationClientID: "2",
		SessionID:           "session-2",
		ClientType:          "mobile",
		ClientIP:            "10.0.0.2",
		UserAgent:           "App/1.0",
		ExpiredAt:           &expiresAt,
	}).Error; err != nil {
		t.Fatalf("seed session-2: %v", err)
	}

	svc := &sessionSvc{}
	resp, err := svc.List(ginCtx, &dtouser.SessionListReq{}, "101", "21", "11")
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if resp.Total != 1 {
		t.Fatalf("expected total 1 (person/tenant/user scoped), got %d", resp.Total)
	}
	if len(resp.List) != 1 {
		t.Fatalf("expected 1 session, got %d", len(resp.List))
	}
	if resp.List[0].SessionID != "session-1" || resp.List[0].ClientType != "web" {
		t.Fatalf("unexpected session list item: %+v", resp.List[0])
	}
	if !resp.List[0].IsActive {
		t.Fatalf("expected session-1 active, got %+v", resp.List[0])
	}
}
