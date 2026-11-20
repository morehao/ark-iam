package connector

import (
	"net/url"
	"testing"

	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/object/objauth"
)

func TestDefaultConnectorFactories(t *testing.T) {
	factories := Factories()
	if len(factories) < 3 {
		t.Fatalf("expected at least 3 default connector factories, got %d", len(factories))
	}

	var foundGoogleOIDC bool
	var foundGithubOAuth2 bool
	for _, factory := range factories {
		if factory.FactoryID == "" {
			t.Fatalf("factoryId should not be empty")
		}
		if factory.ConfigSchema == nil {
			t.Fatalf("configSchema should not be nil for %s", factory.FactoryID)
		}
		if factory.Protocol == DriverTypeOIDC && factory.Provider == ProviderGoogle {
			foundGoogleOIDC = true
		}
		if factory.Protocol == DriverTypeOAuth2 && factory.Provider == ProviderGithub {
			foundGithubOAuth2 = true
		}
	}
	if !foundGoogleOIDC {
		t.Fatalf("expected google oidc factory in defaults")
	}
	if !foundGithubOAuth2 {
		t.Fatalf("expected github oauth2 factory in defaults")
	}
}

func TestDriverRegistryReturnsOAuth2Driver(t *testing.T) {
	driver, ok := DefaultDriverRegistry().Get(DriverTypeOAuth2)
	if !ok {
		t.Fatalf("expected oauth2 driver in registry")
	}
	if driver == nil {
		t.Fatalf("expected non-nil oauth2 driver")
	}
	if driver.DriverType() != DriverTypeOAuth2 {
		t.Fatalf("expected oauth2 driver type, got %q", driver.DriverType())
	}
}

func TestSelectDriverForConnector(t *testing.T) {
	registry := DefaultDriverRegistry()

	driver, config, err := SelectDriverForConnector(registry, &model.ConnectorEntity{
		Protocol: DriverTypeOIDC,
		Provider: ProviderGoogle,
		Config: model.ConnectorConfig{
			Issuer:       "https://accounts.google.com",
			ClientID:     "oidc-client",
			ClientSecret: "secret",
			RedirectURI:  "https://console.example.com/callback",
		},
	})
	if err != nil {
		t.Fatalf("SelectDriverForConnector returned error for oidc connector: %v", err)
	}
	if driver.DriverType() != DriverTypeOIDC {
		t.Fatalf("expected oidc driver, got %q", driver.DriverType())
	}
	if config.Protocol != DriverTypeOIDC || config.Provider != ProviderGoogle {
		t.Fatalf("expected oidc connector config, got protocol=%q provider=%q", config.Protocol, config.Provider)
	}

	driver, config, err = SelectDriverForConnector(registry, &model.ConnectorEntity{
		Protocol: DriverTypeOAuth2,
		Provider: ProviderGithub,
		Config: model.ConnectorConfig{
			AuthURL:      "https://github.com/login/oauth/authorize",
			TokenURL:     "https://github.com/login/oauth/access_token",
			UserInfoURL:  "https://api.github.com/user",
			ClientID:     "oauth-client",
			ClientSecret: "secret",
			RedirectURI:  "https://console.example.com/callback",
		},
	})
	if err != nil {
		t.Fatalf("SelectDriverForConnector returned error for oauth2 connector: %v", err)
	}
	if driver.DriverType() != DriverTypeOAuth2 {
		t.Fatalf("expected oauth2 driver, got %q", driver.DriverType())
	}
	if config.Protocol != DriverTypeOAuth2 || config.Provider != ProviderGithub {
		t.Fatalf("expected oauth2 connector config, got protocol=%q provider=%q", config.Protocol, config.Provider)
	}

	_, _, err = SelectDriverForConnector(registry, nil)
	if err == nil || err.Error() != code.GetError(code.ConnectorNotExistError).Error() {
		t.Fatalf("expected connector not exist error for nil connector, got: %#v", err)
	}

	_, _, err = SelectDriverForConnector(registry, &model.ConnectorEntity{
		Protocol: "saml",
		Provider: "okta",
		Config:   model.ConnectorConfig{RedirectURI: "https://console.example.com/callback"},
	})
	if err == nil || err.Error() != code.GetError(code.ConnectorGetDetailError).Error() {
		t.Fatalf("expected connector detail error for unsupported protocol, got: %#v", err)
	}
}

func TestSelectDriverForConnectorRejectsInvalidConfig(t *testing.T) {
	registry := DefaultDriverRegistry()

	_, _, err := SelectDriverForConnector(registry, &model.ConnectorEntity{
		Protocol: DriverTypeOIDC,
		Provider: ProviderGoogle,
		Config:   model.ConnectorConfig{},
	})
	if err == nil || err.Error() != code.GetError(code.ConnectorGetDetailError).Error() {
		t.Fatalf("expected stable error for empty oidc config, got: %#v", err)
	}

	_, _, err = SelectDriverForConnector(registry, &model.ConnectorEntity{
		Protocol: DriverTypeOAuth2,
		Provider: ProviderGithub,
		Config:   model.ConnectorConfig{},
	})
	if err == nil || err.Error() != code.GetError(code.ConnectorGetDetailError).Error() {
		t.Fatalf("expected stable error for empty oauth2 config, got: %#v", err)
	}
}

func TestBuildConfigKeepsProviderSpecificFields(t *testing.T) {
	config, err := BuildConfig(&model.ConnectorEntity{
		Protocol: DriverTypeOIDC,
		Provider: ProviderMicrosoft,
		Config: model.ConnectorConfig{
			Issuer:       "https://login.microsoftonline.com/common/v2.0",
			ClientID:     "entra-client",
			ClientSecret: "secret",
			RedirectURI:  "https://console.example.com/callback",
			Tenant:       "common",
		},
	})
	if err != nil {
		t.Fatalf("BuildConfig returned error: %v", err)
	}
	if config.Protocol != DriverTypeOIDC || config.Provider != ProviderMicrosoft {
		t.Fatalf("expected connector protocol/provider to be preserved, got protocol=%q provider=%q", config.Protocol, config.Provider)
	}
	if config.Tenant != "common" {
		t.Fatalf("expected tenant field to equal common, got %#v", config.Tenant)
	}
}

func TestSelectDriverForConnectorRequiresMicrosoftTenant(t *testing.T) {
	registry := DefaultDriverRegistry()

	_, _, err := SelectDriverForConnector(registry, &model.ConnectorEntity{
		Protocol: DriverTypeOIDC,
		Provider: ProviderMicrosoft,
		Config: model.ConnectorConfig{
			Issuer:       "https://login.microsoftonline.com/common/v2.0",
			ClientID:     "entra-client",
			ClientSecret: "secret",
			RedirectURI:  "https://console.example.com/callback",
		},
	})
	if err == nil || err.Error() != code.GetError(code.ConnectorGetDetailError).Error() {
		t.Fatalf("expected stable error for microsoft oidc config without tenant, got: %#v", err)
	}

	driver, config, err := SelectDriverForConnector(registry, &model.ConnectorEntity{
		Protocol: DriverTypeOIDC,
		Provider: ProviderMicrosoft,
		Config: model.ConnectorConfig{
			Issuer:       "https://login.microsoftonline.com/common/v2.0",
			ClientID:     "entra-client",
			ClientSecret: "secret",
			RedirectURI:  "https://console.example.com/callback",
			Tenant:       "common",
		},
	})
	if err != nil {
		t.Fatalf("expected microsoft oidc config with tenant to pass, got: %v", err)
	}
	if driver.DriverType() != DriverTypeOIDC {
		t.Fatalf("expected oidc driver, got %q", driver.DriverType())
	}
	if config.Tenant != "common" {
		t.Fatalf("expected tenant to remain readable, got %#v", config.Tenant)
	}
}

func TestFactoryListIncludesOIDCAndOAuth2Templates(t *testing.T) {
	factories := SelectFactories("", "")
	if len(factories) < 3 {
		t.Fatalf("expected at least 3 connector factory templates, got %d", len(factories))
	}

	var hasOIDC bool
	var hasOAuth2 bool
	for _, item := range factories {
		if item.FactoryID == "" {
			t.Fatalf("factoryId should not be empty")
		}
		if item.DisplayName == "" {
			t.Fatalf("displayName should not be empty")
		}
		if len(item.DefaultScopes) == 0 {
			t.Fatalf("defaultScopes should not be empty for %s", item.FactoryID)
		}
		if len(item.Capabilities) == 0 {
			t.Fatalf("capabilities should not be empty for %s", item.FactoryID)
		}
		if item.ConfigSchema == nil {
			t.Fatalf("configSchema should not be nil for %s", item.FactoryID)
		}
		if item.Protocol == DriverTypeOIDC {
			hasOIDC = true
		}
		if item.Protocol == DriverTypeOAuth2 {
			hasOAuth2 = true
		}
	}
	if !hasOIDC {
		t.Fatalf("expected oidc template in factory list")
	}
	if !hasOAuth2 {
		t.Fatalf("expected oauth2 template in factory list")
	}
}

func TestFactoryListSupportsProtocolAndProviderFilters(t *testing.T) {
	list := SelectFactories(DriverTypeOIDC, ProviderGoogle)
	if len(list) != 1 {
		t.Fatalf("expected exactly one filtered factory, got %d", len(list))
	}
	if list[0].Protocol != DriverTypeOIDC || list[0].Provider != ProviderGoogle {
		t.Fatalf("filtered factory should match requested protocol/provider, got %s/%s", list[0].Protocol, list[0].Provider)
	}

	list = SelectFactories("saml", "")
	if len(list) != 0 {
		t.Fatalf("expected empty factory list for unsupported protocol filter, got %d", len(list))
	}
}

func TestOAuth2DriverBuildAuthorizationURLUsesOAuthConfig(t *testing.T) {
	driver, ok := DefaultDriverRegistry().Get(DriverTypeOAuth2)
	if !ok {
		t.Fatalf("expected oauth2 driver in registry")
	}

	resp, err := driver.BuildAuthorizationURL(nil, &AuthorizeInput{
		Config: Config{
			Protocol:     DriverTypeOAuth2,
			Provider:     ProviderGithub,
			AuthURL:      "https://github.com/login/oauth/authorize",
			TokenURL:     "https://github.com/login/oauth/access_token",
			UserInfoURL:  "https://api.github.com/user",
			ClientID:     "oauth-client",
			ClientSecret: "secret",
			RedirectURI:  "https://console.example.com/callback?next=/settings profile",
		},
		ConnectorID: "12",
		State:       "tenant=a b&return=/home",
	})
	if err != nil {
		t.Fatalf("BuildAuthorizationURL returned error: %v", err)
	}
	if resp.AuthorizationURL == "" {
		t.Fatalf("authorizationUrl should not be empty")
	}
	authorizeURL, err := url.Parse(resp.AuthorizationURL)
	if err != nil {
		t.Fatalf("parse authorization url: %v", err)
	}
	if authorizeURL.Query().Get("client_id") != "oauth-client" {
		t.Fatalf("expected client_id, got %q", authorizeURL.Query().Get("client_id"))
	}
	if authorizeURL.Query().Get("state") != "tenant=a b&return=/home" {
		t.Fatalf("expected URL-encoded state, got %q", authorizeURL.Query().Get("state"))
	}
	if authorizeURL.Query().Get("redirect_uri") != "https://console.example.com/callback?next=/settings profile" {
		t.Fatalf("expected URL-encoded redirect_uri, got %q", authorizeURL.Query().Get("redirect_uri"))
	}
	// H10：OAuth2 授权码模式启用 PKCE S256
	if authorizeURL.Query().Get("code_challenge_method") != "S256" {
		t.Fatalf("expected PKCE S256 challenge method, got %q", authorizeURL.Query().Get("code_challenge_method"))
	}
	if authorizeURL.Query().Get("code_challenge") == "" {
		t.Fatal("expected PKCE code_challenge parameter")
	}
	if resp.CodeVerifier == "" {
		t.Fatal("expected code verifier to be returned for state persistence")
	}
}

// --- status / switch validation domain tests (moved from svcauth) ---

// TestIsValidStatus 白名单本身：只认 enable/disable 与空值（空值语义为「不指定/不修改」）。
func TestIsValidStatus(t *testing.T) {
	for _, s := range []model.ConnectorStatus{"", model.ConnectorStatusEnable, model.ConnectorStatusDisable} {
		if !IsValidStatus(s) {
			t.Fatalf("status=%q 应合法", s)
		}
	}
	// 历史值 enabled、同类拼写与大小写、数字等脏值一律拒绝
	for _, s := range []model.ConnectorStatus{"enabled", "disabled", "active", "inactive", "ENABLE", "Enable", "1", "0", "on", " enable"} {
		if IsValidStatus(s) {
			t.Fatalf("status=%q 应非法", s)
		}
	}
}

// TestNormalizeSwitches 开关枚举白名单本身：空串归一为 disable（与列默认值一致），
// enable/disable 原样通过，布尔字面量/数字/大小写脏值一律拒绝。
func TestNormalizeSwitches(t *testing.T) {
	enable, disable := model.ConnectorAutoCreateUserFlagEnable, model.ConnectorAutoCreateUserFlagDisable
	cases := []struct {
		in     model.ConnectorAutoCreateUserFlag
		want   model.ConnectorAutoCreateUserFlag
		wantOK bool
	}{
		{in: "", want: disable, wantOK: true},
		{in: enable, want: enable, wantOK: true},
		{in: disable, want: disable, wantOK: true},
		{in: "true", want: "", wantOK: false},
		{in: "false", want: "", wantOK: false},
		{in: "1", want: "", wantOK: false},
		{in: "0", want: "", wantOK: false},
		{in: "ENABLE", want: "", wantOK: false},
		{in: " enable", want: "", wantOK: false},
	}
	for _, tc := range cases {
		info := &objauth.ConnectorBaseInfo{AllowAutoCreateUser: tc.in}
		ok := NormalizeSwitches(info)
		if ok != tc.wantOK {
			t.Fatalf("NormalizeSwitches(%q) ok = %v, want %v", tc.in, ok, tc.wantOK)
		}
		if tc.wantOK && info.AllowAutoCreateUser != tc.want {
			t.Fatalf("NormalizeSwitches(%q) = (%q), want %q", tc.in, info.AllowAutoCreateUser, tc.want)
		}
	}
	// 单个非法开关导致整体失败
	info := &objauth.ConnectorBaseInfo{AllowAutoCreateUser: model.ConnectorAutoCreateUserFlagEnable, SyncProfile: "yes"}
	if NormalizeSwitches(info) {
		t.Fatalf("NormalizeSwitches 应拒绝非法开关值")
	}
}
