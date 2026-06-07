package ctroidc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/dto/dtooidc"
)

type fakeOIDCAuthSvc struct {
	completeLogin        func(ctx *gin.Context, req *dtooidc.OIDCLoginReq) (*dtooidc.OIDCLoginResp, error)
	completeLoginBySession func(ctx context.Context, authRequestID string, sessionID string) (string, error)
}

func (f *fakeOIDCAuthSvc) CompleteLogin(ctx *gin.Context, req *dtooidc.OIDCLoginReq) (*dtooidc.OIDCLoginResp, error) {
	return f.completeLogin(ctx, req)
}

func (f *fakeOIDCAuthSvc) CompleteLoginBySession(ctx context.Context, authRequestID string, sessionID string) (string, error) {
	if f.completeLoginBySession != nil {
		return f.completeLoginBySession(ctx, authRequestID, sessionID)
	}
	return "", nil
}

func TestLoginReturnsContinueURLOnSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	ctr := &OIDCCtr{oidcAuthSvc: &fakeOIDCAuthSvc{completeLogin: func(ctx *gin.Context, req *dtooidc.OIDCLoginReq) (*dtooidc.OIDCLoginResp, error) {
		return &dtooidc.OIDCLoginResp{ContinueURL: "http://localhost:8099/v1/iam/oidc/authorize/callback?id=ar-1"}, nil
	}}}
	engine.POST("/oidc/login", ctr.Login)

	req := httptest.NewRequest(http.MethodPost, "/oidc/login", bytes.NewReader([]byte(`{"authRequestID":"ar-1","identifier":"person@example.com","password":"Password1"}`)))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	var body struct {
		Code int `json:"code"`
		Data struct {
			ContinueURL string `json:"continueURL"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body.Code != 0 {
		t.Fatalf("expected success, got code %d body=%s", body.Code, resp.Body.String())
	}
	if body.Data.ContinueURL != "http://localhost:8099/v1/iam/oidc/authorize/callback?id=ar-1" {
		t.Fatalf("unexpected continueURL: %q", body.Data.ContinueURL)
	}
}

func TestLoginReturnsErrorOnServiceFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	ctr := &OIDCCtr{oidcAuthSvc: &fakeOIDCAuthSvc{completeLogin: func(ctx *gin.Context, req *dtooidc.OIDCLoginReq) (*dtooidc.OIDCLoginResp, error) {
		return nil, errors.New("login failed")
	}}}
	engine.POST("/oidc/login", ctr.Login)

	req := httptest.NewRequest(http.MethodPost, "/oidc/login", bytes.NewReader([]byte(`{"authRequestID":"ar-1","identifier":"person@example.com","password":"Password1"}`)))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	var body struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body.Code != -1 {
		t.Fatalf("expected fail code -1, got %d body=%s", body.Code, resp.Body.String())
	}
}

func TestLoginSetsSSOCookieWhenSessionCreated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	ctr := &OIDCCtr{oidcAuthSvc: &fakeOIDCAuthSvc{completeLogin: func(ctx *gin.Context, req *dtooidc.OIDCLoginReq) (*dtooidc.OIDCLoginResp, error) {
		return &dtooidc.OIDCLoginResp{ContinueURL: "http://localhost:8099/v1/iam/oidc/authorize/callback?id=ar-1", SessionID: "session-1"}, nil
	}}}
	engine.POST("/oidc/login", ctr.Login)

	req := httptest.NewRequest(http.MethodPost, "/oidc/login", bytes.NewReader([]byte(`{"authRequestID":"ar-1","identifier":"person@example.com","password":"Password1"}`)))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	setCookie := resp.Header().Get("Set-Cookie")
	if !strings.Contains(setCookie, "iam_sso_session=session-1") {
		t.Fatalf("expected iam_sso_session cookie, got %q", setCookie)
	}
}
