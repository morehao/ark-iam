package ctroidc

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/config"
	"github.com/morehao/ark-iam/auth/internal/dto/dtooidc"
	pkgconfig "github.com/morehao/ark-iam/pkg/config"
)

type fakeOIDCAuthSvc struct {
	completeLogin          func(ctx *gin.Context, req *dtooidc.OIDCLoginReq) (*dtooidc.OIDCLoginResp, error)
	selectTenant           func(ctx *gin.Context, authRequestID string, tenantID string) (*dtooidc.OIDCLoginResp, error)
	completeLoginBySession func(ctx *gin.Context, authRequestID string, sessionID string) (string, error)
}

func (f *fakeOIDCAuthSvc) CompleteLogin(ctx *gin.Context, req *dtooidc.OIDCLoginReq) (*dtooidc.OIDCLoginResp, error) {
	return f.completeLogin(ctx, req)
}

func (f *fakeOIDCAuthSvc) SelectTenant(ctx *gin.Context, authRequestID string, tenantID string) (*dtooidc.OIDCLoginResp, error) {
	if f.selectTenant != nil {
		return f.selectTenant(ctx, authRequestID, tenantID)
	}
	return nil, errors.New("selectTenant not implemented in fake")
}

func (f *fakeOIDCAuthSvc) CompleteLoginBySession(ctx *gin.Context, authRequestID string, sessionID string) (string, error) {
	if f.completeLoginBySession != nil {
		return f.completeLoginBySession(ctx, authRequestID, sessionID)
	}
	return "", nil
}

func TestLoginReturnsContinueURLOnSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	ctr := &OIDCCtr{oidcAuthSvc: &fakeOIDCAuthSvc{completeLogin: func(ctx *gin.Context, req *dtooidc.OIDCLoginReq) (*dtooidc.OIDCLoginResp, error) {
		return &dtooidc.OIDCLoginResp{ContinueURL: "http://localhost:8099/oidc/authorize/callback?id=ar-1"}, nil
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
	if body.Data.ContinueURL != "http://localhost:8099/oidc/authorize/callback?id=ar-1" {
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

func TestSelectTenantReturnsContinueURLOnSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	ctr := &OIDCCtr{oidcAuthSvc: &fakeOIDCAuthSvc{selectTenant: func(ctx *gin.Context, authRequestID string, tenantID string) (*dtooidc.OIDCLoginResp, error) {
		if authRequestID != "ar-1" || tenantID != "7" {
			t.Fatalf("unexpected args authRequestID=%q tenantID=%s", authRequestID, tenantID)
		}
		return &dtooidc.OIDCLoginResp{ContinueURL: "http://localhost:8099/oidc/authorize/callback?id=ar-1", TenantID: "7"}, nil
	}}}
	engine.POST("/oidc/login/selectTenant", ctr.SelectTenant)

	req := httptest.NewRequest(http.MethodPost, "/oidc/login/selectTenant", bytes.NewReader([]byte(`{"authRequestID":"ar-1","tenantID":"7"}`)))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	var body struct {
		Code int `json:"code"`
		Data struct {
			ContinueURL string `json:"continueURL"`
			TenantID    string `json:"tenantID"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body.Code != 0 {
		t.Fatalf("expected success, got code %d body=%s", body.Code, resp.Body.String())
	}
	if body.Data.ContinueURL != "http://localhost:8099/oidc/authorize/callback?id=ar-1" {
		t.Fatalf("unexpected continueURL: %q", body.Data.ContinueURL)
	}
	if body.Data.TenantID != "7" {
		t.Fatalf("expected tenantID 7, got %s", body.Data.TenantID)
	}
}

func TestSelectTenantSetsSSOCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config.Conf = &pkgconfig.Config{}
	engine := gin.New()
	ctr := &OIDCCtr{oidcAuthSvc: &fakeOIDCAuthSvc{selectTenant: func(ctx *gin.Context, authRequestID string, tenantID string) (*dtooidc.OIDCLoginResp, error) {
		return &dtooidc.OIDCLoginResp{ContinueURL: "http://localhost:8099/oidc/authorize/callback?id=ar-1", TenantID: "7", SessionID: "session-1"}, nil
	}}}
	engine.POST("/oidc/login/selectTenant", ctr.SelectTenant)

	req := httptest.NewRequest(http.MethodPost, "/oidc/login/selectTenant", bytes.NewReader([]byte(`{"authRequestID":"ar-1","tenantID":"7"}`)))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	setCookie := resp.Header().Get("Set-Cookie")
	if !strings.Contains(setCookie, "iam_sso_session=session-1") {
		t.Fatalf("expected iam_sso_session cookie on SelectTenant, got %q", setCookie)
	}
}

func TestSelectTenantReturnsErrorOnInvalidTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	ctr := &OIDCCtr{oidcAuthSvc: &fakeOIDCAuthSvc{selectTenant: func(ctx *gin.Context, authRequestID string, tenantID string) (*dtooidc.OIDCLoginResp, error) {
		return nil, errors.New("tenant not allowed")
	}}}
	engine.POST("/oidc/login/selectTenant", ctr.SelectTenant)

	req := httptest.NewRequest(http.MethodPost, "/oidc/login/selectTenant", bytes.NewReader([]byte(`{"authRequestID":"ar-1","tenantID":99}`)))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	var body struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body.Code == 0 {
		t.Fatalf("expected fail code, got body=%s", resp.Body.String())
	}
}

func TestLoginSetsSSOCookieWithoutDomainByDefault(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config.Conf = &pkgconfig.Config{}
	engine := gin.New()
	ctr := &OIDCCtr{oidcAuthSvc: &fakeOIDCAuthSvc{completeLogin: func(ctx *gin.Context, req *dtooidc.OIDCLoginReq) (*dtooidc.OIDCLoginResp, error) {
		return &dtooidc.OIDCLoginResp{ContinueURL: "http://localhost:8099/oidc/authorize/callback?id=ar-1", SessionID: "session-1"}, nil
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
	if strings.Contains(setCookie, "Domain=") {
		t.Fatalf("expected host-only cookie without Domain by default, got %q", setCookie)
	}
}

func TestLoginSetsSSOCookieWithConfiguredDomain(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config.Conf = &pkgconfig.Config{OIDC: pkgconfig.OIDC{CookieDomain: "example.com"}}
	engine := gin.New()
	ctr := &OIDCCtr{oidcAuthSvc: &fakeOIDCAuthSvc{completeLogin: func(ctx *gin.Context, req *dtooidc.OIDCLoginReq) (*dtooidc.OIDCLoginResp, error) {
		return &dtooidc.OIDCLoginResp{ContinueURL: "http://localhost:8099/oidc/authorize/callback?id=ar-1", SessionID: "session-1"}, nil
	}}}
	engine.POST("/login", ctr.Login)

	reqBody := strings.NewReader(`{"authRequestID":"ar-1","identifier":"demo","password":"pass"}`)
	req := httptest.NewRequest(http.MethodPost, "/login", reqBody)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	engine.ServeHTTP(resp, req)

	setCookie := resp.Header().Get("Set-Cookie")
	if !strings.Contains(setCookie, "Domain=example.com") {
		t.Fatalf("expected cookie to use configured domain, got %q", setCookie)
	}
}

func TestLoginSetsSSOCookieWhenConfigIsNil(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config.Conf = nil
	engine := gin.New()
	ctr := &OIDCCtr{oidcAuthSvc: &fakeOIDCAuthSvc{completeLogin: func(ctx *gin.Context, req *dtooidc.OIDCLoginReq) (*dtooidc.OIDCLoginResp, error) {
		return &dtooidc.OIDCLoginResp{ContinueURL: "http://localhost:8099/oidc/authorize/callback?id=ar-1", SessionID: "session-1"}, nil
	}}}
	engine.POST("/login", ctr.Login)

	reqBody := strings.NewReader(`{"authRequestID":"ar-1","identifier":"demo","password":"pass"}`)
	req := httptest.NewRequest(http.MethodPost, "/login", reqBody)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	engine.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200 without config panic, got %d", resp.Code)
	}
	setCookie := resp.Header().Get("Set-Cookie")
	if !strings.Contains(setCookie, "iam_sso_session=session-1") {
		t.Fatalf("expected iam_sso_session cookie, got %q", setCookie)
	}
	if strings.Contains(setCookie, "Domain=") {
		t.Fatalf("expected host-only cookie when config is nil, got %q", setCookie)
	}
}
