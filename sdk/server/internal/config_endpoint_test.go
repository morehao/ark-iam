package internal

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	pkconfig "github.com/morehao/ark-iam/pkg/config"
)

// mountConfigOnly 构造一个仅挂载 MountConfig 的最小 App，避免拉起 DB/auth 域。
// MountConfig 只读 a.cfg，不触碰 DB，故可直接构造。
func mountConfigOnly(t *testing.T, raw *pkconfig.Config, base string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	app := &App{cfg: Config{Raw: raw}}
	e := gin.New()
	if err := app.MountConfig(e, base); err != nil {
		t.Fatalf("MountConfig: %v", err)
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

// TestConfigJSON_DerivesFromRawAndHost 验证 /config.json 的 issuer/clientId 从 Raw 与请求 Host 派生。
func TestConfigJSON_DerivesFromRawAndHost(t *testing.T) {
	raw := &pkconfig.Config{OIDC: pkconfig.OIDC{Issuer: "http://localhost:8100/oidc"}}
	e := mountConfigOnly(t, raw, "")

	out := getConfigJSON(t, e, "http://iam.example.com/config.json")
	if got := out["issuer"]; got != "http://localhost:8100/oidc" {
		t.Errorf("issuer = %v, want http://localhost:8100/oidc", got)
	}
	cid, _ := out["clientId"].(string)
	if cid == "" {
		t.Errorf("clientId must not be empty (derived from console app)")
	}
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

// TestConfigJSON_IssuerFallsBackToRequestHost 验证 Raw 无 issuer 时，issuer 由请求 Host 派生。
func TestConfigJSON_IssuerFallsBackToRequestHost(t *testing.T) {
	e := mountConfigOnly(t, nil, "")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://console.svc.local/config.json", nil)
	req.Host = "console.svc.local"
	e.ServeHTTP(w, req)

	var out map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v body=%s", err, w.Body.String())
	}
	if got := out["issuer"]; got != "http://console.svc.local/oidc" {
		t.Errorf("issuer = %v, want http://console.svc.local/oidc", got)
	}
}

// TestConfigJSON_CORSHeader 验证跨源场景下 /config.json 带 CORS 头。
func TestConfigJSON_CORSHeader(t *testing.T) {
	e := mountConfigOnly(t, nil, "")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://iam.example.com/config.json", nil)
	req.Host = "iam.example.com"
	req.Header.Set("Origin", "http://app.example.com")
	e.ServeHTTP(w, req)
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "http://app.example.com" {
		t.Errorf("Access-Control-Allow-Origin = %q, want http://app.example.com", got)
	}
}

// TestConfigJSON_PrefixedBases 验证非空前缀挂载（内嵌形态，如 /iam）下各端点 base 均带前缀。
func TestConfigJSON_PrefixedBases(t *testing.T) {
	raw := &pkconfig.Config{OIDC: pkconfig.OIDC{Issuer: "http://localhost:8099/iam/oidc"}}
	e := mountConfigOnly(t, raw, "/iam")
	out := getConfigJSON(t, e, "http://localhost:8099/iam/config.json")

	for k, want := range map[string]string{
		"apiBase":     "/iam/v1",
		"oidcBase":    "/iam/oidc",
		"installBase": "/iam/install",
		"loginPath":   "/iam/login",
	} {
		if got := out[k]; got != want {
			t.Errorf("%s = %v, want %s", k, got, want)
		}
	}
	if out["callbackPath"] != "/auth/callback" {
		t.Errorf("callbackPath = %v", out["callbackPath"])
	}
}

// TestConfigJSON_RootBases 验证根挂载（"" 前缀）时 base 仍为无前缀形态。
func TestConfigJSON_RootBases(t *testing.T) {
	e := mountConfigOnly(t, nil, "")
	out := getConfigJSON(t, e, "http://x/config.json")
	for k, want := range map[string]string{
		"apiBase": "/v1", "oidcBase": "/oidc", "installBase": "/install", "loginPath": "/login",
	} {
		if got := out[k]; got != want {
			t.Errorf("%s = %v, want %s", k, got, want)
		}
	}
	if got := out["issuer"]; got != "http://x/oidc" {
		t.Errorf("issuer = %v, want http://x/oidc", got)
	}
}

// TestConfigJSON_EnabledDomains 验证 G4：/config.json 下发 enabledDomains 域白名单。
// 空 Domains 归一为全量域（与 Mount 的解析口径一致）。
func TestConfigJSON_EnabledDomains(t *testing.T) {
	// 空 Domains：全量域集合（非空，且含 core-auth / application）。
	e := mountConfigOnly(t, nil, "")
	out := getConfigJSON(t, e, "http://x/config.json")
	raw, ok := out["enabledDomains"]
	if !ok {
		t.Fatalf("enabledDomains missing from /config.json: %v", out)
	}
	list, ok := raw.([]any)
	if !ok {
		t.Fatalf("enabledDomains is not an array: %T", raw)
	}
	if len(list) == 0 {
		t.Errorf("enabledDomains must not be empty for full-domain instance")
	}
	has := func(d string) bool {
		for _, v := range list {
			if v == d {
				return true
			}
		}
		return false
	}
	if !has(string(DomainCoreAuth)) {
		t.Errorf("enabledDomains missing %q: %v", DomainCoreAuth, list)
	}
	if !has(string(DomainApplication)) {
		t.Errorf("enabledDomains missing %q: %v", DomainApplication, list)
	}
}

// TestConfigJSON_EnabledDomainsSubset 验证显式选定域时只下发所选域的集合。
func TestConfigJSON_EnabledDomainsSubset(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := &App{cfg: Config{Domains: []Domain{DomainCoreAuth, DomainUser, DomainRole, DomainMenu}}}
	e := gin.New()
	if err := app.MountConfig(e, ""); err != nil {
		t.Fatalf("MountConfig: %v", err)
	}
	out := getConfigJSON(t, e, "http://x/config.json")
	list, ok := out["enabledDomains"].([]any)
	if !ok {
		t.Fatalf("enabledDomains is not an array: %T", out["enabledDomains"])
	}
	if len(list) != 4 {
		t.Errorf("enabledDomains length = %d, want 4: %v", len(list), list)
	}
	for _, v := range list {
		s := v.(string)
		if s == string(DomainApplication) || s == string(DomainTenant) {
			t.Errorf("enabledDomains must not contain unselected domain %q: %v", s, list)
		}
	}
}
