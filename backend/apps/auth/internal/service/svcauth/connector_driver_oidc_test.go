package svcauth

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"testing"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/morehao/ark-iam/pkg/code"
)

type fakeOIDCProvider struct {
	endpoint oauth2.Endpoint
	userInfo OIDCUserInfo
}

type fakeOIDCVerifier struct {
	verifiedToken OIDCVerifiedToken
	err           error
}

type fakeOIDCVerifiedToken struct {
	claims map[string]any
	nonce  string
	sub    string
	iss    string
}

type fakeOIDCUserInfo struct {
	claims map[string]any
}

func (p *fakeOIDCProvider) Endpoint() oauth2.Endpoint {
	return p.endpoint
}

func (p *fakeOIDCProvider) Verifier(_ *oidc.Config) OIDCVerifier {
	return fakeOIDCVerifier{}
}

func (p *fakeOIDCProvider) UserInfo(_ context.Context, _ oauth2.TokenSource) (OIDCUserInfo, error) {
	return p.userInfo, nil
}

func (v fakeOIDCVerifier) Verify(_ context.Context, _ string) (OIDCVerifiedToken, error) {
	if v.err != nil {
		return nil, v.err
	}
	if v.verifiedToken == nil {
		return nil, errors.New("missing verified token")
	}
	return v.verifiedToken, nil
}

func (t fakeOIDCVerifiedToken) Claims(target any) error {
	data, err := json.Marshal(t.claims)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func (t fakeOIDCVerifiedToken) Nonce() string {
	return t.nonce
}

func (t fakeOIDCVerifiedToken) Subject() string {
	return t.sub
}

func (t fakeOIDCVerifiedToken) Issuer() string {
	return t.iss
}

func (u fakeOIDCUserInfo) Claims(target any) error {
	data, err := json.Marshal(u.claims)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func TestOIDCDriverValidateConfigRequiresIssuer(t *testing.T) {
	driver := &OIDCDriver{}

	err := driver.ValidateConfig(ConnectorConfig{
		Protocol:     connectorDriverTypeOIDC,
		Provider:     connectorProviderGoogle,
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURI:  "https://console.example.com/callback",
	})
	if err == nil || err.Error() != code.GetError(code.ConnectorGetDetailError).Error() {
		t.Fatalf("expected stable error for missing oidc issuer, got: %#v", err)
	}
}

func TestOIDCDriverValidateConfigMatchesSharedValidator(t *testing.T) {
	config := ConnectorConfig{
		Protocol:     connectorDriverTypeOIDC,
		Provider:     connectorProviderMicrosoft,
		Issuer:       "https://issuer.example.com",
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURI:  "https://console.example.com/callback",
		Raw: map[string]any{
			"tenant": "common",
		},
	}

	driver := &OIDCDriver{}
	if err := driver.ValidateConfig(config); err != nil {
		t.Fatalf("OIDCDriver ValidateConfig returned error: %v", err)
	}
	if err := validateOIDCConnectorConfig(config); err != nil {
		t.Fatalf("shared OIDC validator returned error: %v", err)
	}
}

func TestOIDCDriverBuildAuthorizationURLIncludesNonce(t *testing.T) {
	driver := &OIDCDriver{
		providerFactory: func(_ context.Context, issuer string) (oidcProvider, error) {
			if issuer != "https://issuer.example.com" {
				t.Fatalf("unexpected issuer: %q", issuer)
			}
			return &fakeOIDCProvider{endpoint: oauth2.Endpoint{AuthURL: "https://issuer.example.com/oauth2/authorize"}}, nil
		},
		nonceGenerator: func() (string, error) {
			return "nonce-123", nil
		},
	}

	resp, err := driver.BuildAuthorizationURL(nil, &ConnectorAuthorizeInput{
		Config: ConnectorConfig{
			Protocol:     connectorDriverTypeOIDC,
			Provider:     connectorProviderGoogle,
			Issuer:       "https://issuer.example.com",
			ClientID:     "client-id",
			ClientSecret: "client-secret",
			RedirectURI:  "https://console.example.com/callback",
		},
		State: "state-123",
	})
	if err != nil {
		t.Fatalf("BuildAuthorizationURL returned error: %v", err)
	}

	authorizeURL, err := url.Parse(resp.AuthorizationURL)
	if err != nil {
		t.Fatalf("parse authorizationUrl failed: %v", err)
	}
	if authorizeURL.Scheme != "https" || authorizeURL.Host != "issuer.example.com" || authorizeURL.Path != "/oauth2/authorize" {
		t.Fatalf("unexpected authorization URL: %q", resp.AuthorizationURL)
	}
	if authorizeURL.Query().Get("nonce") != "nonce-123" {
		t.Fatalf("expected nonce query parameter, got %q", authorizeURL.Query().Get("nonce"))
	}
	if resp.Nonce != "nonce-123" {
		t.Fatalf("expected nonce output, got %q", resp.Nonce)
	}
	if authorizeURL.Query().Get("state") != "state-123" {
		t.Fatalf("expected state query parameter, got %q", authorizeURL.Query().Get("state"))
	}
}

func TestOIDCDriverBuildAuthorizationURLUsesRequestRedirectURI(t *testing.T) {
	driver := &OIDCDriver{
		providerFactory: func(_ context.Context, _ string) (oidcProvider, error) {
			return &fakeOIDCProvider{endpoint: oauth2.Endpoint{AuthURL: "https://issuer.example.com/oauth2/authorize"}}, nil
		},
		nonceGenerator: func() (string, error) {
			return "nonce-456", nil
		},
	}

	resp, err := driver.BuildAuthorizationURL(nil, &ConnectorAuthorizeInput{
		Config: ConnectorConfig{
			Protocol:     connectorDriverTypeOIDC,
			Provider:     connectorProviderGoogle,
			Issuer:       "https://issuer.example.com",
			ClientID:     "client-id",
			ClientSecret: "client-secret",
			RedirectURI:  "https://config.example.com/callback",
		},
		RedirectURI: "https://request.example.com/callback",
		State:       "state-456",
	})
	if err != nil {
		t.Fatalf("BuildAuthorizationURL returned error: %v", err)
	}

	authorizeURL, err := url.Parse(resp.AuthorizationURL)
	if err != nil {
		t.Fatalf("parse authorizationUrl failed: %v", err)
	}
	if authorizeURL.Query().Get("redirect_uri") != "https://request.example.com/callback" {
		t.Fatalf("expected request redirectUri override, got %q", authorizeURL.Query().Get("redirect_uri"))
	}
}

func TestDefaultConnectorDriverRegistryUsesRealOIDCDriver(t *testing.T) {
	driver, ok := defaultConnectorDriverRegistry().Get(connectorDriverTypeOIDC)
	if !ok {
		t.Fatalf("expected oidc driver in registry")
	}
	if _, ok := driver.(*OIDCDriver); !ok {
		t.Fatalf("expected real OIDCDriver, got %T", driver)
	}
}

func TestOIDCDriverExchangeCallbackReturnsStandardIdentity(t *testing.T) {
	driver := &OIDCDriver{
		providerFactory: func(_ context.Context, issuer string) (oidcProvider, error) {
			if issuer != "https://issuer.example.com" {
				t.Fatalf("unexpected issuer: %q", issuer)
			}
			return &fakeOIDCProvider{
				endpoint: oauth2.Endpoint{AuthURL: "https://issuer.example.com/oauth2/authorize", TokenURL: "https://issuer.example.com/oauth2/token"},
				userInfo: fakeOIDCUserInfo{claims: map[string]any{"email": "oidc@example.com", "name": "OIDC User", "picture": "https://cdn.example.com/oidc.png"}},
			}, nil
		},
		tokenExchanger: func(ctx context.Context, config oauth2.Config, code string) (*oauth2.Token, error) {
			if code != "oidc-code" {
				t.Fatalf("expected exchange code oidc-code, got %q", code)
			}
			token := &oauth2.Token{AccessToken: "oidc-access", RefreshToken: "oidc-refresh"}
			return token.WithExtra(map[string]any{"id_token": "raw-id-token"}), nil
		},
		verifierFactory: func(provider oidcProvider, config *oidc.Config) OIDCVerifier {
			return fakeOIDCVerifier{verifiedToken: fakeOIDCVerifiedToken{
				claims: map[string]any{"sub": "oidc-sub", "iss": "https://issuer.example.com", "email_verified": true},
				nonce:  "nonce-oidc",
				sub:    "oidc-sub",
				iss:    "https://issuer.example.com",
			}}
		},
	}

	resp, err := driver.ExchangeCallback(nil, &ConnectorCallbackInput{
		Config: ConnectorConfig{
			Protocol:     connectorDriverTypeOIDC,
			Provider:     connectorProviderGoogle,
			Issuer:       "https://issuer.example.com",
			ClientID:     "oidc-client-id",
			ClientSecret: "oidc-client-secret",
			RedirectURI:  "https://config.example.com/callback",
		},
		Code:        "oidc-code",
		Nonce:       "nonce-oidc",
		RedirectURI: "https://request.example.com/callback",
	})
	if err != nil {
		t.Fatalf("ExchangeCallback returned error: %v", err)
	}
	if resp.Identity.Subject != "oidc-sub" || resp.Identity.Issuer != "https://issuer.example.com" {
		t.Fatalf("expected verified OIDC identity, got %+v", resp.Identity)
	}
	if resp.Identity.Email != "oidc@example.com" || resp.Identity.DisplayName != "OIDC User" {
		t.Fatalf("expected userinfo fallback fields, got %+v", resp.Identity)
	}
	if !resp.Identity.EmailVerified {
		t.Fatal("expected emailVerified to be true")
	}
	if resp.AccessToken != "oidc-access" || resp.RefreshToken != "oidc-refresh" {
		t.Fatalf("expected provider tokens to be preserved, got %+v", resp)
	}
}
