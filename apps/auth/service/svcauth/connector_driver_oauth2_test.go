package svcauth

import (
	"context"
	"net/url"
	"testing"

	"golang.org/x/oauth2"
)

type oauth2IdentityNormalizerDriver interface {
	normalizeIdentity(config ConnectorConfig, claims map[string]any) (StandardIdentity, error)
}

func TestOAuth2DriverBuildAuthorizationURL(t *testing.T) {
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
			ClientID:     "github-client-id",
			ClientSecret: "github-client-secret",
			RedirectURI:  "https://console.example.com/callback",
			Scopes:       []string{"read:user", "user:email"},
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
	if authorizeURL.Scheme != "https" || authorizeURL.Host != "github.com" || authorizeURL.Path != "/login/oauth/authorize" {
		t.Fatalf("unexpected authorization URL: %q", resp.AuthorizationURL)
	}
	if authorizeURL.Query().Get("client_id") != "github-client-id" {
		t.Fatalf("expected client_id query parameter, got %q", authorizeURL.Query().Get("client_id"))
	}
	if authorizeURL.Query().Get("redirect_uri") != "https://console.example.com/callback" {
		t.Fatalf("expected config redirectUri, got %q", authorizeURL.Query().Get("redirect_uri"))
	}
	if authorizeURL.Query().Get("scope") != "read:user user:email" {
		t.Fatalf("expected joined scope query parameter, got %q", authorizeURL.Query().Get("scope"))
	}
	if authorizeURL.Query().Get("state") != "state-123" {
		t.Fatalf("expected state query parameter, got %q", authorizeURL.Query().Get("state"))
	}
	if resp.Nonce != "" {
		t.Fatalf("expected oauth2 authorize output nonce to stay empty, got %q", resp.Nonce)
	}
	if authorizeURL.Query().Get("response_type") != "code" {
		t.Fatalf("expected response_type=code, got %q", authorizeURL.Query().Get("response_type"))
	}
}

func TestOAuth2DriverUsesRequestRedirectURI(t *testing.T) {
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
			ClientID:     "github-client-id",
			ClientSecret: "github-client-secret",
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

func TestOAuth2DriverNormalizeGitHubIdentity(t *testing.T) {
	driver, ok := defaultConnectorDriverRegistry().Get(connectorDriverTypeOAuth2)
	if !ok {
		t.Fatalf("expected oauth2 driver in registry")
	}

	normalizer, ok := driver.(oauth2IdentityNormalizerDriver)
	if !ok {
		t.Fatalf("expected oauth2 driver to expose normalizeIdentity hook, got %T", driver)
	}

	identity, err := normalizer.normalizeIdentity(ConnectorConfig{
		Protocol: connectorDriverTypeOAuth2,
		Provider: connectorProviderGithub,
	}, map[string]any{
		"id":         float64(12345),
		"login":      "octocat",
		"name":       "The Octocat",
		"avatar_url": "https://avatars.githubusercontent.com/u/583231?v=4",
		"company":    "GitHub",
	})
	if err != nil {
		t.Fatalf("normalizeIdentity returned error: %v", err)
	}
	if identity.Subject != "12345" {
		t.Fatalf("expected github id to map to subject, got %q", identity.Subject)
	}
	if identity.Username != "octocat" {
		t.Fatalf("expected github login to map to username, got %q", identity.Username)
	}
	if identity.DisplayName != "The Octocat" {
		t.Fatalf("expected github name to map to displayName, got %q", identity.DisplayName)
	}
	if identity.AvatarURL != "https://avatars.githubusercontent.com/u/583231?v=4" {
		t.Fatalf("expected github avatar_url to map to avatarUrl, got %q", identity.AvatarURL)
	}
	if identity.Claims["company"] != "GitHub" {
		t.Fatalf("expected raw claims to remain readable, got %#v", identity.Claims["company"])
	}
}

func TestDefaultConnectorDriverRegistryUsesRealOAuth2Driver(t *testing.T) {
	driver, ok := defaultConnectorDriverRegistry().Get(connectorDriverTypeOAuth2)
	if !ok {
		t.Fatalf("expected oauth2 driver in registry")
	}
	if _, ok := driver.(*OAuth2Driver); !ok {
		t.Fatalf("expected real OAuth2Driver, got %T", driver)
	}
}

func TestOAuth2DriverExchangeCallbackReturnsNormalizedIdentity(t *testing.T) {
	driver := &OAuth2Driver{
		normalizers: NewOAuth2Driver().(*OAuth2Driver).normalizers,
		tokenExchanger: func(ctx context.Context, config oauth2.Config, code string, _ string) (*oauth2.Token, error) {
			if code != "oauth-code" {
				t.Fatalf("expected exchange code oauth-code, got %q", code)
			}
			if config.RedirectURL != "https://request.example.com/callback" {
				t.Fatalf("expected request redirect uri, got %q", config.RedirectURL)
			}
			return &oauth2.Token{AccessToken: "provider-access", RefreshToken: "provider-refresh"}, nil
		},
		userInfoFetcher: func(ctx context.Context, token *oauth2.Token, config ConnectorConfig) (map[string]any, error) {
			if token.AccessToken != "provider-access" {
				t.Fatalf("expected provider access token, got %q", token.AccessToken)
			}
			return map[string]any{
				"id":         float64(12345),
				"login":      "octocat",
				"name":       "The Octocat",
				"avatar_url": "https://avatars.githubusercontent.com/u/583231?v=4",
			}, nil
		},
	}

	resp, err := driver.ExchangeCallback(nil, &ConnectorCallbackInput{
		Config: ConnectorConfig{
			Protocol:     connectorDriverTypeOAuth2,
			Provider:     connectorProviderGithub,
			AuthURL:      "https://github.com/login/oauth/authorize",
			TokenURL:     "https://github.com/login/oauth/access_token",
			UserInfoURL:  "https://api.github.com/user",
			ClientID:     "github-client-id",
			ClientSecret: "github-client-secret",
			RedirectURI:  "https://config.example.com/callback",
		},
		Code:        "oauth-code",
		RedirectURI: "https://request.example.com/callback",
	})
	if err != nil {
		t.Fatalf("ExchangeCallback returned error: %v", err)
	}
	if resp.Identity.Subject != "12345" || resp.Identity.Username != "octocat" {
		t.Fatalf("expected normalized github identity, got %+v", resp.Identity)
	}
	if resp.AccessToken != "provider-access" || resp.RefreshToken != "provider-refresh" {
		t.Fatalf("expected provider tokens to be preserved, got %+v", resp)
	}
}
