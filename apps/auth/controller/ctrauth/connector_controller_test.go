package ctrauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/apps/auth/dto/dtoauth"
	"github.com/morehao/ark-iam/apps/auth/dto/dtoconnector"
)

type stubConnectorSvc struct {
	authorizeReq         *dtoconnector.ConnectorAuthorizeReq
	authorizeConnectorID string
	authorizeResp        *dtoconnector.ConnectorAuthorizeResp
	authorizeErr         error

	callbackReq  *dtoconnector.ConnectorCallbackReq
	callbackResp *dtoauth.LoginResp
	callbackErr  error
}

func (s *stubConnectorSvc) Authorize(ctx context.Context, req *dtoconnector.ConnectorAuthorizeReq, connectorID string) (*dtoconnector.ConnectorAuthorizeResp, error) {
	s.authorizeReq = req
	s.authorizeConnectorID = connectorID
	return s.authorizeResp, s.authorizeErr
}

func (s *stubConnectorSvc) Callback(ctx context.Context, req *dtoconnector.ConnectorCallbackReq) (*dtoauth.LoginResp, error) {
	s.callbackReq = req
	return s.callbackResp, s.callbackErr
}

func TestConnectorControllerAuthorize(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &stubConnectorSvc{
		authorizeResp: &dtoconnector.ConnectorAuthorizeResp{AuthorizationURL: "https://idp.example.com/auth"},
	}
	ctr := &connectorCtr{connectorSvc: svc}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/connector/42/authorize", strings.NewReader(`{"redirectUri":"https://app.example.com/callback","state":"state-1","loginHint":"user@example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	ctx.Request = req
	ctx.Params = gin.Params{{Key: "connectorID", Value: "42"}}

	ctr.Authorize(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if svc.authorizeReq == nil {
		t.Fatal("expected Authorize to receive request")
	}
	if svc.authorizeConnectorID != "42" {
		t.Fatalf("expected connectorID 42, got %s", svc.authorizeConnectorID)
	}
	if svc.authorizeReq.ConnectorID != "42" {
		t.Fatalf("expected URI binding to populate request connectorID, got %s", svc.authorizeReq.ConnectorID)
	}
	if svc.authorizeReq.RedirectURI != "https://app.example.com/callback" || svc.authorizeReq.State != "state-1" || svc.authorizeReq.LoginHint != "user@example.com" {
		t.Fatalf("expected JSON body to bind into authorize request, got %+v", *svc.authorizeReq)
	}
	assertJSONContainsData(t, recorder.Body.Bytes(), "https://idp.example.com/auth")
}

func TestConnectorControllerCallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &stubConnectorSvc{
		callbackResp: &dtoauth.LoginResp{SSOSessionID: "sso-session-7"},
	}
	ctr := &connectorCtr{connectorSvc: svc}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodGet, "/v1/auth/connector/callback?connectorID=7&code=callback-code&state=state-2", nil)
	ctx.Request = req

	ctr.Callback(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if svc.callbackReq == nil {
		t.Fatal("expected Callback to receive request")
	}
	if svc.callbackReq.ConnectorID != "7" || svc.callbackReq.Code != "callback-code" || svc.callbackReq.State != "state-2" {
		t.Fatalf("expected query binding to populate callback request, got %+v", *svc.callbackReq)
	}
	assertJSONContainsData(t, recorder.Body.Bytes(), "sso-session-7")
}

func TestConnectorControllerCallbackRequiresState(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &stubConnectorSvc{}
	ctr := &connectorCtr{connectorSvc: svc}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodGet, "/v1/auth/connector/callback?connectorID=7&code=callback-code", nil)
	ctx.Request = req

	ctr.Callback(ctx)

	if svc.callbackReq != nil {
		t.Fatal("expected Callback service not to be called when state is missing")
	}
}

func assertJSONContainsData(t *testing.T, body []byte, expected string) {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	dataBytes, err := json.Marshal(payload["data"])
	if err != nil {
		t.Fatalf("failed to encode response data: %v", err)
	}
	if !strings.Contains(string(dataBytes), expected) {
		t.Fatalf("expected response data to contain %q, got %s", expected, string(dataBytes))
	}
}
