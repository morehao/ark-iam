package ctrsession

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/internal/dto/dtouser"
)

type stubSessionSvc struct {
	revokeReq *dtouser.SessionRevokeReq
}

func (s *stubSessionSvc) List(ctx context.Context, req *dtouser.SessionListReq) (*dtouser.SessionListResp, error) {
	panic("unexpected call")
}

func (s *stubSessionSvc) Revoke(ctx context.Context, req *dtouser.SessionRevokeReq) error {
	s.revokeReq = req
	return nil
}

func (s *stubSessionSvc) RevokeAll(ctx context.Context, req *dtouser.SessionRevokeAllReq) error {
	panic("unexpected call")
}

func TestSessionControllerRevokeBindsURI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &stubSessionSvc{}
	ctr := &sessionCtr{sessionSvc: svc}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodDelete, "/v1/auth/user/sessions/42", nil)
	ctx.Request = req
	ctx.Params = gin.Params{{Key: "sessionID", Value: "42"}}

	ctr.Revoke(ctx)

	if svc.revokeReq == nil {
		t.Fatal("expected Revoke service to receive request")
	}
	if svc.revokeReq.SessionID != "42" {
		t.Fatalf("expected URI binding to populate sessionID=42, got %s", svc.revokeReq.SessionID)
	}
}
