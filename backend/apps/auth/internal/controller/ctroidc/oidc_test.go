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
	"github.com/morehao/ark-iam/auth/internal/service/svcoidc"
	pkgconfig "github.com/morehao/ark-iam/pkg/config"
	"github.com/morehao/ark-iam/pkg/sso"
)

type fakeOIDCAuthSvc struct {
	completeLogin          func(ctx *gin.Context, req *dtooidc.OIDCLoginReq) (*dtooidc.OIDCLoginResp, error)
	selectTenant           func(ctx *gin.Context, authRequestID string, tenantID string) (*dtooidc.OIDCLoginResp, error)
	completeLoginBySession func(ctx *gin.Context, authRequestID string, sessionID string) (string, error)
	registerPerson         func(ctx *gin.Context, req *dtooidc.RegisterPersonReq) (*dtooidc.RegisterPersonResp, error)
	createTenant           func(ctx *gin.Context, req *dtooidc.CreateTenantReq) (*dtooidc.CreateTenantResp, error)
	changePassword         func(ctx *gin.Context, req *dtooidc.OIDCChangePasswordReq) error
	loginConfig            func(ctx *gin.Context, authRequestID string) (*dtooidc.OIDCLoginConfigResp, error)
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

func (f *fakeOIDCAuthSvc) RegisterPerson(ctx *gin.Context, req *dtooidc.RegisterPersonReq) (*dtooidc.RegisterPersonResp, error) {
	if f.registerPerson != nil {
		return f.registerPerson(ctx, req)
	}
	return nil, errors.New("registerPerson not implemented in fake")
}

func (f *fakeOIDCAuthSvc) CreateTenant(ctx *gin.Context, req *dtooidc.CreateTenantReq) (*dtooidc.CreateTenantResp, error) {
	if f.createTenant != nil {
		return f.createTenant(ctx, req)
	}
	return nil, errors.New("createTenant not implemented in fake")
}

func (f *fakeOIDCAuthSvc) ChangePassword(ctx *gin.Context, req *dtooidc.OIDCChangePasswordReq) error {
	if f.changePassword != nil {
		return f.changePassword(ctx, req)
	}
	return errors.New("changePassword not implemented in fake")
}

func (f *fakeOIDCAuthSvc) LoginConfig(ctx *gin.Context, authRequestID string) (*dtooidc.OIDCLoginConfigResp, error) {
	if f.loginConfig != nil {
		return f.loginConfig(ctx, authRequestID)
	}
	return nil, errors.New("loginConfig not implemented in fake")
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

func TestLoginConfigReturnsAllowPersonCreateTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	ctr := &OIDCCtr{oidcAuthSvc: &fakeOIDCAuthSvc{loginConfig: func(ctx *gin.Context, authRequestID string) (*dtooidc.OIDCLoginConfigResp, error) {
		return &dtooidc.OIDCLoginConfigResp{AllowPersonCreateTenant: true}, nil
	}}}
	engine.POST("/oidc/login-config", ctr.LoginConfig)

	reqBody := strings.NewReader(`{"authRequestID":"ar-1"}`)
	req := httptest.NewRequest(http.MethodPost, "/oidc/login-config", reqBody)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	engine.ServeHTTP(resp, req)

	var body struct {
		Code int `json:"code"`
		Data struct {
			AllowPersonCreateTenant bool `json:"allowPersonCreateTenant"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body.Code != 0 {
		t.Fatalf("expected code 0, got %d", body.Code)
	}
	if !body.Data.AllowPersonCreateTenant {
		t.Fatal("expected allowPersonCreateTenant=true in response")
	}
}

func TestLoginConfigMissingAuthRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	ctr := &OIDCCtr{oidcAuthSvc: &fakeOIDCAuthSvc{}}
	engine.POST("/oidc/login-config", ctr.LoginConfig)

	reqBody := strings.NewReader(`{}`)
	req := httptest.NewRequest(http.MethodPost, "/oidc/login-config", reqBody)
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
		t.Fatal("expected non-zero code for missing authRequestID")
	}
}

// TestSSOLoginRendersNoAppAccessPage 无应用准入（AC-5）：
// 静默登录因「本客户端应用下没有任何有准入的租户」被拒时，必须直接渲染无权限页（403），
// **不得**清 cookie 跳回登录页——重新登录仍是同一结论，会形成登录死循环。
func TestSSOLoginRendersNoAppAccessPage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	ctr := &OIDCCtr{oidcAuthSvc: &fakeOIDCAuthSvc{completeLoginBySession: func(ctx *gin.Context, authRequestID string, sessionID string) (string, error) {
		return "", svcoidc.ErrSilentLoginNoAppAccess
	}}}
	engine.GET("/oidc/sso", ctr.SSOLogin)

	req := httptest.NewRequest(http.MethodGet, "/oidc/sso?authRequestID=ar-1", nil)
	req.AddCookie(&http.Cookie{Name: sso.SessionCookieName, Value: "session-1"})
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	if resp.Code != http.StatusForbidden {
		t.Fatalf("expected 403 no-permission page, got %d body=%s", resp.Code, resp.Body.String())
	}
	if ct := resp.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Fatalf("expected html content type, got %q", ct)
	}
	if !strings.Contains(resp.Body.String(), "你暂无该应用的访问权限") {
		t.Fatalf("expected no-permission copy, got %s", resp.Body.String())
	}
	if loc := resp.Header().Get("Location"); loc != "" {
		t.Fatalf("must not redirect to login page (login loop), got Location=%q", loc)
	}
	if sc := resp.Header().Get("Set-Cookie"); strings.Contains(sc, sso.SessionCookieName) {
		t.Fatalf("must keep SSO session cookie untouched, got Set-Cookie=%q", sc)
	}
}

// TestSSOLoginRedirectsToLoginOnOtherSilentLoginFailures 对照用例：
// 其它静默登录失败（prompt=login / max_age 超期 / 会话失效）仍走「清 cookie 回登录页」——
// 那是正确处置（重新认证可能通过），不能被无权限页的分支吞掉。
func TestSSOLoginRedirectsToLoginOnOtherSilentLoginFailures(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config.Conf = &pkgconfig.Config{
		OIDC: pkgconfig.OIDC{FrontendLoginURL: "http://localhost:4000/oidc/login"},
	}
	engine := gin.New()
	ctr := &OIDCCtr{oidcAuthSvc: &fakeOIDCAuthSvc{completeLoginBySession: func(ctx *gin.Context, authRequestID string, sessionID string) (string, error) {
		return "", errors.New("silent login not allowed")
	}}}
	engine.GET("/oidc/sso", ctr.SSOLogin)

	req := httptest.NewRequest(http.MethodGet, "/oidc/sso?authRequestID=ar-1", nil)
	req.AddCookie(&http.Cookie{Name: sso.SessionCookieName, Value: "session-1"})
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	if resp.Code != http.StatusFound {
		t.Fatalf("expected 302 back to login page, got %d", resp.Code)
	}
	if loc := resp.Header().Get("Location"); !strings.Contains(loc, "http://localhost:4000/oidc/login") {
		t.Fatalf("expected redirect to login page, got %q", loc)
	}
	if sc := resp.Header().Get("Set-Cookie"); !strings.Contains(sc, sso.SessionCookieName) {
		t.Fatalf("expected SSO session cookie to be cleared, got Set-Cookie=%q", sc)
	}
}
