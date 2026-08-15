package svcauth

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/morehao/ark-iam/auth/internal/dto/dtoauth"
	"github.com/morehao/ark-iam/auth/internal/dto/dtoconnector"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/iam/object/objauth"
)

func TestDefaultConnectorFactories(t *testing.T) {
	factories := defaultConnectorFactories()
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
		if factory.Protocol == "oidc" && factory.Provider == "google" {
			foundGoogleOIDC = true
		}
		if factory.Protocol == "oauth2" && factory.Provider == "github" {
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
	driver, ok := defaultConnectorDriverRegistry().Get("oauth2")
	if !ok {
		t.Fatalf("expected oauth2 driver in registry")
	}
	if driver == nil {
		t.Fatalf("expected non-nil oauth2 driver")
	}
	if driver.DriverType() != "oauth2" {
		t.Fatalf("expected oauth2 driver type, got %q", driver.DriverType())
	}
}

func TestSelectDriverForConnector(t *testing.T) {
	registry := defaultConnectorDriverRegistry()

	oidcConfig, err := json.Marshal(map[string]any{
		"issuer":       "https://accounts.google.com",
		"clientId":     "oidc-client",
		"clientSecret": "secret",
		"redirectUri":  "https://console.example.com/callback",
	})
	if err != nil {
		t.Fatalf("marshal oidc config failed: %v", err)
	}

	driver, config, err := selectDriverForConnector(registry, &model.ConnectorEntity{
		Protocol: connectorDriverTypeOIDC,
		Provider: connectorProviderGoogle,
		Config:   oidcConfig,
	})
	if err != nil {
		t.Fatalf("selectDriverForConnector returned error for oidc connector: %v", err)
	}
	if driver.DriverType() != connectorDriverTypeOIDC {
		t.Fatalf("expected oidc driver, got %q", driver.DriverType())
	}
	if config.Protocol != connectorDriverTypeOIDC || config.Provider != connectorProviderGoogle {
		t.Fatalf("expected oidc connector config, got protocol=%q provider=%q", config.Protocol, config.Provider)
	}

	oauth2Config, err := json.Marshal(map[string]any{
		"authUrl":      "https://github.com/login/oauth/authorize",
		"tokenUrl":     "https://github.com/login/oauth/access_token",
		"userInfoUrl":  "https://api.github.com/user",
		"clientId":     "oauth-client",
		"clientSecret": "secret",
		"redirectUri":  "https://console.example.com/callback",
	})
	if err != nil {
		t.Fatalf("marshal oauth2 config failed: %v", err)
	}

	driver, config, err = selectDriverForConnector(registry, &model.ConnectorEntity{
		Protocol: connectorDriverTypeOAuth2,
		Provider: connectorProviderGithub,
		Config:   oauth2Config,
	})
	if err != nil {
		t.Fatalf("selectDriverForConnector returned error for oauth2 connector: %v", err)
	}
	if driver.DriverType() != connectorDriverTypeOAuth2 {
		t.Fatalf("expected oauth2 driver, got %q", driver.DriverType())
	}
	if config.Protocol != connectorDriverTypeOAuth2 || config.Provider != connectorProviderGithub {
		t.Fatalf("expected oauth2 connector config, got protocol=%q provider=%q", config.Protocol, config.Provider)
	}

	_, _, err = selectDriverForConnector(registry, nil)
	if err == nil || err.Error() != code.GetError(code.ConnectorNotExistError).Error() {
		t.Fatalf("expected connector not exist error for nil connector, got: %#v", err)
	}

	unsupportedConfig, err := json.Marshal(map[string]any{"redirectUri": "https://console.example.com/callback"})
	if err != nil {
		t.Fatalf("marshal unsupported config failed: %v", err)
	}
	_, _, err = selectDriverForConnector(registry, &model.ConnectorEntity{
		Protocol: "saml",
		Provider: "okta",
		Config:   unsupportedConfig,
	})
	if err == nil || err.Error() != code.GetError(code.ConnectorGetDetailError).Error() {
		t.Fatalf("expected connector detail error for unsupported protocol, got: %#v", err)
	}
}

func TestSelectDriverForConnectorRejectsInvalidConfig(t *testing.T) {
	registry := defaultConnectorDriverRegistry()

	_, _, err := selectDriverForConnector(registry, &model.ConnectorEntity{
		Protocol: connectorDriverTypeOIDC,
		Provider: connectorProviderGoogle,
		Config:   json.RawMessage(`{}`),
	})
	if err == nil || err.Error() != code.GetError(code.ConnectorGetDetailError).Error() {
		t.Fatalf("expected stable error for empty oidc config, got: %#v", err)
	}

	_, _, err = selectDriverForConnector(registry, &model.ConnectorEntity{
		Protocol: connectorDriverTypeOAuth2,
		Provider: connectorProviderGithub,
		Config:   json.RawMessage(`{}`),
	})
	if err == nil || err.Error() != code.GetError(code.ConnectorGetDetailError).Error() {
		t.Fatalf("expected stable error for empty oauth2 config, got: %#v", err)
	}
}

func TestBuildConnectorConfigKeepsProviderSpecificFields(t *testing.T) {
	configData, err := json.Marshal(map[string]any{
		"issuer":       "https://login.microsoftonline.com/common/v2.0",
		"clientId":     "entra-client",
		"clientSecret": "secret",
		"redirectUri":  "https://console.example.com/callback",
		"tenant":       "common",
	})
	if err != nil {
		t.Fatalf("marshal config failed: %v", err)
	}

	config, err := buildConnectorConfig(&model.ConnectorEntity{
		Protocol: connectorDriverTypeOIDC,
		Provider: connectorProviderMicrosoft,
		Config:   configData,
	})
	if err != nil {
		t.Fatalf("buildConnectorConfig returned error: %v", err)
	}
	if config.Protocol != connectorDriverTypeOIDC || config.Provider != connectorProviderMicrosoft {
		t.Fatalf("expected connector protocol/provider to be preserved, got protocol=%q provider=%q", config.Protocol, config.Provider)
	}
	tenant, ok := config.Raw["tenant"]
	if !ok {
		t.Fatalf("expected provider-specific field tenant to be preserved")
	}
	if tenant != "common" {
		t.Fatalf("expected tenant field to equal common, got %#v", tenant)
	}
}

func TestSelectDriverForConnectorRequiresMicrosoftTenant(t *testing.T) {
	registry := defaultConnectorDriverRegistry()

	missingTenantConfig, err := json.Marshal(map[string]any{
		"issuer":       "https://login.microsoftonline.com/common/v2.0",
		"clientId":     "entra-client",
		"clientSecret": "secret",
		"redirectUri":  "https://console.example.com/callback",
	})
	if err != nil {
		t.Fatalf("marshal missing tenant config failed: %v", err)
	}

	_, _, err = selectDriverForConnector(registry, &model.ConnectorEntity{
		Protocol: connectorDriverTypeOIDC,
		Provider: connectorProviderMicrosoft,
		Config:   missingTenantConfig,
	})
	if err == nil || err.Error() != code.GetError(code.ConnectorGetDetailError).Error() {
		t.Fatalf("expected stable error for microsoft oidc config without tenant, got: %#v", err)
	}

	validConfig, err := json.Marshal(map[string]any{
		"issuer":       "https://login.microsoftonline.com/common/v2.0",
		"clientId":     "entra-client",
		"clientSecret": "secret",
		"redirectUri":  "https://console.example.com/callback",
		"tenant":       "common",
	})
	if err != nil {
		t.Fatalf("marshal valid microsoft config failed: %v", err)
	}

	driver, config, err := selectDriverForConnector(registry, &model.ConnectorEntity{
		Protocol: connectorDriverTypeOIDC,
		Provider: connectorProviderMicrosoft,
		Config:   validConfig,
	})
	if err != nil {
		t.Fatalf("expected microsoft oidc config with tenant to pass, got: %v", err)
	}
	if driver.DriverType() != connectorDriverTypeOIDC {
		t.Fatalf("expected oidc driver, got %q", driver.DriverType())
	}
	if config.Raw["tenant"] != "common" {
		t.Fatalf("expected tenant to remain readable, got %#v", config.Raw["tenant"])
	}
}

func TestConnectorSvcUsesInjectedRegistry(t *testing.T) {
	registry := newConnectorDriverRegistry(newStubConnectorDriver(connectorDriverTypeOIDC))
	svc := &connectorSvc{driverRegistry: registry}
	if svc.getDriverRegistry() != registry {
		t.Fatalf("expected connector service to keep injected registry")
	}
	if _, ok := svc.getDriverRegistry().Get(connectorDriverTypeOAuth2); ok {
		t.Fatalf("expected injected registry to be used as-is")
	}
}

func TestFactoryListIncludesOIDCAndOAuth2Templates(t *testing.T) {
	svc := NewConnectorSvc()

	resp, err := svc.ListFactories(nil, &dtoconnector.ConnectorFactoryListReq{})
	if err != nil {
		t.Fatalf("ListFactories returned error: %v", err)
	}
	if len(resp.List) < 3 {
		t.Fatalf("expected at least 3 connector factory templates, got %d", len(resp.List))
	}

	var hasOIDC bool
	var hasOAuth2 bool
	for _, item := range resp.List {
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
		if item.Protocol == "oidc" {
			hasOIDC = true
		}
		if item.Protocol == "oauth2" {
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
	svc := NewConnectorSvc()

	resp, err := svc.ListFactories(nil, &dtoconnector.ConnectorFactoryListReq{
		Protocol: "oidc",
		Provider: "google",
	})
	if err != nil {
		t.Fatalf("ListFactories returned error: %v", err)
	}
	if len(resp.List) != 1 {
		t.Fatalf("expected exactly one filtered factory, got %d", len(resp.List))
	}
	if resp.List[0].Protocol != "oidc" || resp.List[0].Provider != "google" {
		t.Fatalf("filtered factory should match requested protocol/provider, got %s/%s", resp.List[0].Protocol, resp.List[0].Provider)
	}

	resp, err = svc.ListFactories(nil, &dtoconnector.ConnectorFactoryListReq{Protocol: "saml"})
	if err != nil {
		t.Fatalf("ListFactories returned error: %v", err)
	}
	if len(resp.List) != 0 {
		t.Fatalf("expected empty factory list for unsupported protocol filter, got %d", len(resp.List))
	}
}

func TestConnectorTypesExposeNewContractFields(t *testing.T) {
	authorizeReq := dtoconnector.ConnectorAuthorizeReq{
		ConnectorID:  "12",
		RedirectURI:  "https://console.example.com/callback",
		State:        "state-1",
		LoginHint:    "demo@example.com",
		ResponseMode: "query",
	}
	callbackReq := dtoconnector.ConnectorCallbackReq{
		ConnectorID: "12",
		Code:        "code-1",
		State:       "state-1",
	}
	factoryListReq := dtoconnector.ConnectorFactoryListReq{
		Protocol: "oidc",
		Provider: "google",
	}
	createReq := dtoauth.ConnectorCreateReq{
		ConnectorBaseInfo: objauth.ConnectorBaseInfo{
			TenantID:            "1",
			Name:                "google-workspace",
			DisplayName:         "Google Workspace",
			Protocol:            "oidc",
			Provider:            "google",
			Status:              "enabled",
			AllowAutoCreateUser: 1,
			AllowAccountLink:    1,
			SyncProfile:         1,
			EnableTokenStorage:  1,
			Config: map[string]any{
				"issuer": "https://accounts.google.com",
			},
			ClaimMapping: map[string]any{
				"email": "email",
			},
			DomainPolicy: map[string]any{
				"mode": "allow_all",
			},
		},
	}
	pageReq := dtoauth.ConnectorPageListReq{
		TenantID:    "1",
		Protocol:    "oidc",
		Provider:    "google",
		Status:      "enabled",
		Name:        "google-workspace",
		DisplayName: "Google Workspace",
	}

	if authorizeReq.ConnectorID == "" || callbackReq.ConnectorID == "" {
		t.Fatalf("connector authorize/callback req should expose connector id")
	}
	authorizeReqType := reflect.TypeOf(authorizeReq)
	connectorIDField, ok := authorizeReqType.FieldByName("ConnectorID")
	if !ok {
		t.Fatalf("ConnectorAuthorizeReq should define ConnectorID field")
	}
	if connectorIDField.Tag.Get("uri") != "connectorID" || connectorIDField.Tag.Get("binding") != "required" {
		t.Fatalf("ConnectorAuthorizeReq.ConnectorID should keep required uri binding tag contract")
	}
	redirectURIField, ok := authorizeReqType.FieldByName("RedirectURI")
	if !ok || redirectURIField.Tag.Get("json") != "redirectUri" || redirectURIField.Tag.Get("binding") != "required" {
		t.Fatalf("ConnectorAuthorizeReq.RedirectURI should keep required redirectUri contract")
	}
	callbackReqType := reflect.TypeOf(callbackReq)
	callbackConnectorIDField, ok := callbackReqType.FieldByName("ConnectorID")
	if !ok || callbackConnectorIDField.Tag.Get("json") != "connectorID" {
		t.Fatalf("ConnectorCallbackReq.ConnectorID should keep connectorId contract")
	}
	codeField, ok := callbackReqType.FieldByName("Code")
	if !ok || codeField.Tag.Get("json") != "code" || codeField.Tag.Get("binding") != "required" {
		t.Fatalf("ConnectorCallbackReq.Code should keep required code contract")
	}
	factoryReqType := reflect.TypeOf(factoryListReq)
	protocolField, ok := factoryReqType.FieldByName("Protocol")
	if !ok || protocolField.Tag.Get("form") != "protocol" {
		t.Fatalf("ConnectorFactoryListReq.Protocol should expose form tag")
	}
	providerField, ok := factoryReqType.FieldByName("Provider")
	if !ok || providerField.Tag.Get("form") != "provider" {
		t.Fatalf("ConnectorFactoryListReq.Provider should expose form tag")
	}
	if factoryListReq.Protocol == "" || factoryListReq.Provider == "" {
		t.Fatalf("factory list req should expose protocol and provider filters")
	}
	factoryResp := dtoconnector.ConnectorFactoryResp{
		FactoryID:     "oidc-google",
		Protocol:      "oidc",
		Provider:      "google",
		DisplayName:   "Google",
		IsStandard:    true,
		DefaultScopes: []string{"openid", "profile", "email"},
		Capabilities:  []string{"authorize", "callback"},
		ConfigSchema: map[string]any{
			"type": "object",
		},
	}
	if createReq.Name == "" || createReq.DisplayName == "" {
		t.Fatalf("connector create req should embed new connector base info fields")
	}
	if pageReq.Name == "" || pageReq.DisplayName == "" {
		t.Fatalf("connector page list req should expose name/displayName filters")
	}
	if factoryResp.FactoryID == "" || len(factoryResp.DefaultScopes) == 0 || len(factoryResp.Capabilities) == 0 || factoryResp.ConfigSchema == nil {
		t.Fatalf("connector factory resp should expose new factory contract fields")
	}
}

func TestOAuth2DriverBuildAuthorizationURLUsesOAuthConfig(t *testing.T) {
	driver, ok := defaultConnectorDriverRegistry().Get(connectorDriverTypeOAuth2)
	if !ok {
		t.Fatalf("expected oauth2 driver in registry")
	}

	resp, err := driver.BuildAuthorizationURL(nil, &ConnectorAuthorizeInput{
		Config: ConnectorConfig{
			Protocol:     connectorDriverTypeOAuth2,
			Provider:     connectorProviderGithub,
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
	expected := "https://github.com/login/oauth/authorize?client_id=oauth-client&redirect_uri=https%3A%2F%2Fconsole.example.com%2Fcallback%3Fnext%3D%2Fsettings+profile&response_type=code&state=tenant%3Da+b%26return%3D%2Fhome"
	if resp.AuthorizationURL != expected {
		t.Fatalf("authorizationUrl should be URL-encoded, got %q", resp.AuthorizationURL)
	}
}
