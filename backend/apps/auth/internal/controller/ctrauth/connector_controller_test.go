package ctrauth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/internal/dto/dtoauth"
	"github.com/morehao/ark-iam/auth/internal/dto/dtoconnector"
)

type stubConnectorSvc struct {
	getFactoryListReq  *dtoconnector.ConnectorFactoryListReq
	getFactoryListResp *dtoconnector.ConnectorFactoryListResp
	getFactoryListErr  error

	testConnectorReq  *dtoconnector.ConnectorIDReq
	testConnectorResp *dtoconnector.TestConnectorResp
	testConnectorErr  error

	authorizeReq         *dtoconnector.ConnectorAuthorizeReq
	authorizeConnectorID uint
	authorizeResp        *dtoconnector.ConnectorAuthorizeResp
	authorizeErr         error

	callbackReq  *dtoconnector.ConnectorCallbackReq
	callbackResp *dtoauth.LoginResp
	callbackErr  error
}

func (s *stubConnectorSvc) Create(ctx *gin.Context, req *dtoauth.ConnectorCreateReq) (*dtoauth.ConnectorCreateResp, error) {
	panic("unexpected call")
}

func (s *stubConnectorSvc) Delete(ctx *gin.Context, req *dtoauth.ConnectorDeleteReq) error {
	panic("unexpected call")
}

func (s *stubConnectorSvc) Update(ctx *gin.Context, req *dtoauth.ConnectorUpdateReq) error {
	panic("unexpected call")
}

func (s *stubConnectorSvc) Detail(ctx *gin.Context, req *dtoauth.ConnectorDetailReq) (*dtoauth.ConnectorDetailResp, error) {
	panic("unexpected call")
}

func (s *stubConnectorSvc) PageList(ctx *gin.Context, req *dtoauth.ConnectorPageListReq) (*dtoauth.ConnectorPageListResp, error) {
	panic("unexpected call")
}

func (s *stubConnectorSvc) GetFactoryList(ctx *gin.Context, req *dtoconnector.ConnectorFactoryListReq) (*dtoconnector.ConnectorFactoryListResp, error) {
	s.getFactoryListReq = req
	return s.getFactoryListResp, s.getFactoryListErr
}

func (s *stubConnectorSvc) ListFactories(ctx *gin.Context, req *dtoconnector.ConnectorFactoryListReq) (*dtoconnector.ConnectorFactoryListResp, error) {
	panic("unexpected call")
}

func (s *stubConnectorSvc) TestConnector(ctx *gin.Context, req *dtoconnector.ConnectorIDReq) (*dtoconnector.TestConnectorResp, error) {
	s.testConnectorReq = req
	return s.testConnectorResp, s.testConnectorErr
}

func (s *stubConnectorSvc) Authorize(ctx *gin.Context, req *dtoconnector.ConnectorAuthorizeReq, connectorID uint) (*dtoconnector.ConnectorAuthorizeResp, error) {
	s.authorizeReq = req
	s.authorizeConnectorID = connectorID
	return s.authorizeResp, s.authorizeErr
}

func (s *stubConnectorSvc) GetAuthorizationURL(ctx *gin.Context, req *dtoconnector.ConnectorAuthorizeReq) (*dtoconnector.ConnectorAuthorizeResp, error) {
	panic("unexpected call")
}

func (s *stubConnectorSvc) Callback(ctx *gin.Context, req *dtoconnector.ConnectorCallbackReq) (*dtoauth.LoginResp, error) {
	s.callbackReq = req
	return s.callbackResp, s.callbackErr
}

func TestConnectorControllerGetFactoryList(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &stubConnectorSvc{
		getFactoryListResp: &dtoconnector.ConnectorFactoryListResp{List: []dtoconnector.ConnectorFactoryResp{{FactoryID: "oidc-google"}}},
	}
	ctr := &connectorCtr{connectorSvc: svc}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodPost, "/v1/connector/getFactoryList", strings.NewReader(`{"protocol":"oidc","provider":"google"}`))
	req.Header.Set("Content-Type", "application/json")
	ctx.Request = req

	ctr.GetFactoryList(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if svc.getFactoryListReq == nil {
		t.Fatal("expected GetFactoryList to receive request")
	}
	if svc.getFactoryListReq.Protocol != "oidc" || svc.getFactoryListReq.Provider != "google" {
		t.Fatalf("expected JSON body to bind into request, got %+v", *svc.getFactoryListReq)
	}
	assertJSONContainsData(t, recorder.Body.Bytes(), "oidc-google")
}

func TestConnectorControllerAuthorize(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &stubConnectorSvc{
		authorizeResp: &dtoconnector.ConnectorAuthorizeResp{AuthorizationURL: "https://idp.example.com/auth"},
	}
	ctr := &connectorCtr{connectorSvc: svc}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodPost, "/v1/connector/42/authorize", strings.NewReader(`{"redirectUri":"https://app.example.com/callback","state":"state-1","loginHint":"user@example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	ctx.Request = req
	ctx.Params = gin.Params{{Key: "connectorId", Value: "42"}}

	ctr.Authorize(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if svc.authorizeReq == nil {
		t.Fatal("expected Authorize to receive request")
	}
	if svc.authorizeConnectorID != 42 {
		t.Fatalf("expected connectorID 42, got %d", svc.authorizeConnectorID)
	}
	if svc.authorizeReq.ConnectorID != 42 {
		t.Fatalf("expected URI binding to populate request connectorID, got %d", svc.authorizeReq.ConnectorID)
	}
	if svc.authorizeReq.RedirectURI != "https://app.example.com/callback" || svc.authorizeReq.State != "state-1" || svc.authorizeReq.LoginHint != "user@example.com" {
		t.Fatalf("expected JSON body to bind into authorize request, got %+v", *svc.authorizeReq)
	}
	assertJSONContainsData(t, recorder.Body.Bytes(), "https://idp.example.com/auth")
}

func TestConnectorControllerTestConnector(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &stubConnectorSvc{
		testConnectorResp: &dtoconnector.TestConnectorResp{Success: true, Message: "ok"},
	}
	ctr := &connectorCtr{connectorSvc: svc}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodPost, "/v1/connector/9/test", nil)
	ctx.Request = req
	ctx.Params = gin.Params{{Key: "connectorId", Value: "9"}}

	ctr.TestConnector(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if svc.testConnectorReq == nil {
		t.Fatal("expected TestConnector to receive request")
	}
	if svc.testConnectorReq.ConnectorID != 9 {
		t.Fatalf("expected URI binding to populate connectorID=9, got %d", svc.testConnectorReq.ConnectorID)
	}
	assertJSONContainsData(t, recorder.Body.Bytes(), "ok")
}

func TestConnectorControllerCallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &stubConnectorSvc{
		callbackResp: &dtoauth.LoginResp{SSOSessionID: "sso-session-7"},
	}
	ctr := &connectorCtr{connectorSvc: svc}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodGet, "/v1/connector/callback?connectorId=7&code=callback-code&state=state-2", nil)
	ctx.Request = req

	ctr.Callback(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if svc.callbackReq == nil {
		t.Fatal("expected Callback to receive request")
	}
	if svc.callbackReq.ConnectorID != 7 || svc.callbackReq.Code != "callback-code" || svc.callbackReq.State != "state-2" {
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
	req := httptest.NewRequest(http.MethodGet, "/v1/connector/callback?connectorId=7&code=callback-code", nil)
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
