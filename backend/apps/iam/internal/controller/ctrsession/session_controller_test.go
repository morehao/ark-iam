package ctrsession

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/dto/dtouser"
)

type stubSessionSvc struct {
	revokeReq *dtouser.SessionRevokeReq
}

func (s *stubSessionSvc) List(ctx *gin.Context, req *dtouser.SessionListReq, personID, userID, tenantID uint) (*dtouser.SessionListResp, error) {
	panic("unexpected call")
}

func (s *stubSessionSvc) Revoke(ctx *gin.Context, req *dtouser.SessionRevokeReq, userID, tenantID, personID uint) error {
	s.revokeReq = req
	return nil
}

func (s *stubSessionSvc) RevokeAll(ctx *gin.Context, userID, tenantID, personID uint) error {
	panic("unexpected call")
}

func TestSessionControllerRevokeBindsURI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &stubSessionSvc{}
	ctr := &sessionCtr{sessionSvc: svc}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodDelete, "/v1/iam/user/sessions/42", nil)
	ctx.Request = req
	ctx.Params = gin.Params{{Key: "sessionId", Value: "42"}}

	ctr.Revoke(ctx)

	if svc.revokeReq == nil {
		t.Fatal("expected Revoke service to receive request")
	}
	if svc.revokeReq.SessionID != 42 {
		t.Fatalf("expected URI binding to populate sessionID=42, got %d", svc.revokeReq.SessionID)
	}
}
