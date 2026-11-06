package iam

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	pkconfig "github.com/morehao/ark-iam/pkg/config"
)

// mountFrontendOnly 构造一个仅挂载 MountFrontend 的最小 App，避免拉起 DB/auth 域。
// MountFrontend 只读 a.cfg，不触碰 DB，故可直接构造。
func mountFrontendOnly(t *testing.T, raw *pkconfig.Config, base string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	app := &App{cfg: Config{Raw: raw}}
	e := gin.New()
	if err := app.MountFrontend(e, base); err != nil {
		t.Fatalf("MountFrontend: %v", err)
	}
	return e
}

func getConfigJSON(t *testing.T, e *gin.Engine, url string) map[string]any {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, url, nil)
	e.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return out
}

// TestConfigJSON_DerivesFromRawAndHost 验证 /config.json 的 issuer/clientId 从 Raw 与请求 Host 派生，
// 不再留空串。
func TestConfigJSON_DerivesFromRawAndHost(t *testing.T) {
	raw := &pkconfig.Config{
		OIDC: pkconfig.OIDC{
			Issuer: "http://localhost:8100/oidc",
		},
	}
	e := mountFrontendOnly(t, raw, "")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://iam.example.com/config.json", nil)
	req.Host = "iam.example.com"
	e.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var out map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}

	// issuer 取自 Raw.OIDC.Issuer。
	if got := out["issuer"]; got != "http://localhost:8100/oidc" {
		t.Errorf("issuer = %v, want http://localhost:8100/oidc", got)
	}
	// clientId 不能为空，且应由控制台应用的 OIDC client_id 派生。
	cid, _ := out["clientId"].(string)
	if cid == "" {
		t.Errorf("clientId must not be empty (derived from console app)")
	}
	// apiBase/oidcBase/callbackPath 保持既有契约。
	if out["apiBase"] != "/v1" {
		t.Errorf("apiBase = %v", out["apiBase"])
	}
	if out["oidcBase"] != "/oidc" {
		t.Errorf("oidcBase = %v", out["oidcBase"])
	}
	if out["callbackPath"] != "/auth/callback" {
		t.Errorf("callbackPath = %v", out["callbackPath"])
	}
}

// TestConfigJSON_IssuerFallsBackToRequestHost 验证 Raw 无 issuer 时，issuer 由请求 Host 派生为绝对地址，
// 而不是空串。
func TestConfigJSON_IssuerFallsBackToRequestHost(t *testing.T) {
	e := mountFrontendOnly(t, nil, "")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://console.svc.local/config.json", nil)
	req.Host = "console.svc.local"
	e.ServeHTTP(w, req)
	_ = w

	var out map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v body=%s", err, w.Body.String())
	}
	if got := out["issuer"]; got != "http://console.svc.local/oidc" {
		t.Errorf("issuer = %v, want http://console.svc.local/oidc", got)
	}
}

// TestConfigJSON_CORSHeader 验证跨源场景（业务应用独立 origin）下 /config.json 带 CORS 头。
func TestConfigJSON_CORSHeader(t *testing.T) {
	e := mountFrontendOnly(t, nil, "")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://iam.example.com/config.json", nil)
	req.Host = "iam.example.com"
	req.Header.Set("Origin", "http://app.example.com")
	e.ServeHTTP(w, req)
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "http://app.example.com" {
		t.Errorf("Access-Control-Allow-Origin = %q, want http://app.example.com", got)
	}
}
