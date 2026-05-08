package svcauth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"

	"github.com/morehao/ark-iam/pkg/code"
)

type oidcProvider interface {
	Endpoint() oauth2.Endpoint
	Verifier(config *oidc.Config) OIDCVerifier
	UserInfo(ctx context.Context, tokenSource oauth2.TokenSource) (OIDCUserInfo, error)
}

type OIDCVerifier interface {
	Verify(ctx context.Context, rawIDToken string) (OIDCVerifiedToken, error)
}

type OIDCVerifiedToken interface {
	Claims(target any) error
	Nonce() string
	Subject() string
	Issuer() string
}

type OIDCUserInfo interface {
	Claims(target any) error
}

type OIDCDriver struct {
	providerFactory func(ctx context.Context, issuer string) (oidcProvider, error)
	nonceGenerator  func() (string, error)
	tokenExchanger  func(ctx context.Context, config oauth2.Config, code string) (*oauth2.Token, error)
	verifierFactory func(provider oidcProvider, config *oidc.Config) OIDCVerifier
}

type coreOIDCProvider struct {
	provider *oidc.Provider
}

var _ ConnectorDriver = (*OIDCDriver)(nil)

func NewOIDCDriver() ConnectorDriver {
	return &OIDCDriver{
		providerFactory: defaultOIDCProviderFactory,
		nonceGenerator:  defaultOIDCNonceGenerator,
	}
}

func (d *OIDCDriver) DriverType() string {
	return connectorDriverTypeOIDC
}

func (d *OIDCDriver) ValidateConfig(config ConnectorConfig) error {
	return validateOIDCConnectorConfig(config)
}

func (d *OIDCDriver) BuildAuthorizationURL(ctx *gin.Context, input *ConnectorAuthorizeInput) (*ConnectorAuthorizeOutput, error) {
	if input == nil {
		return nil, code.GetError(code.ConnectorGetDetailError)
	}
	if err := d.ValidateConfig(input.Config); err != nil {
		return nil, err
	}

	provider, err := d.getProvider(ctx, input.Config.Issuer)
	if err != nil {
		return nil, code.GetError(code.ConnectorGetDetailError)
	}
	nonce, err := d.getNonce()
	if err != nil {
		return nil, code.GetError(code.ConnectorGetDetailError)
	}

	oauthConfig := oauth2.Config{
		ClientID:     input.Config.ClientID,
		ClientSecret: input.Config.ClientSecret,
		RedirectURL:  resolveConnectorRedirectURI(input),
		Scopes:       input.Config.Scopes,
		Endpoint:     provider.Endpoint(),
	}
	return &ConnectorAuthorizeOutput{
		AuthorizationURL: oauthConfig.AuthCodeURL(input.State, oidc.Nonce(nonce)),
		Nonce:            nonce,
	}, nil
}

func (d *OIDCDriver) ExchangeCallback(ctx *gin.Context, input *ConnectorCallbackInput) (*ConnectorCallbackOutput, error) {
	if input == nil {
		return nil, code.GetError(code.ConnectorGetDetailError)
	}
	if err := d.ValidateConfig(input.Config); err != nil {
		return nil, err
	}
	provider, err := d.getProvider(runtimeContext(ctx), input.Config.Issuer)
	if err != nil {
		return nil, code.GetError(code.ConnectorGetDetailError)
	}
	oauthConfig := oauth2.Config{
		ClientID:     input.Config.ClientID,
		ClientSecret: input.Config.ClientSecret,
		RedirectURL:  resolveConnectorCallbackRedirectURI(input),
		Scopes:       input.Config.Scopes,
		Endpoint:     provider.Endpoint(),
	}
	token, err := d.exchangeToken(runtimeContext(ctx), oauthConfig, input.Code)
	if err != nil {
		return nil, code.GetError(code.AuthLoginFailedError)
	}
	rawIDToken, _ := token.Extra("id_token").(string)
	if rawIDToken == "" {
		return nil, code.GetError(code.AuthLoginFailedError)
	}
	verifiedToken, err := d.getVerifier(provider, &oidc.Config{ClientID: input.Config.ClientID}).Verify(runtimeContext(ctx), rawIDToken)
	if err != nil {
		return nil, code.GetError(code.AuthLoginFailedError)
	}
	if input.Nonce != "" && verifiedToken.Nonce() != input.Nonce {
		return nil, code.GetError(code.AuthLoginFailedError)
	}
	claims := map[string]any{}
	if err := verifiedToken.Claims(&claims); err != nil {
		return nil, code.GetError(code.AuthLoginFailedError)
	}
	userInfoClaims, err := d.fetchUserInfoClaims(runtimeContext(ctx), provider, token)
	if err != nil {
		return nil, code.GetError(code.AuthLoginFailedError)
	}
	for key, value := range userInfoClaims {
		if _, exists := claims[key]; !exists {
			claims[key] = value
		}
	}
	identity, err := buildOIDCStandardIdentity(verifiedToken, claims)
	if err != nil {
		return nil, err
	}
	return &ConnectorCallbackOutput{
		Identity:     identity,
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
	}, nil
}

func (d *OIDCDriver) TestConnection(ctx *gin.Context, input *ConnectorTestInput) (*ConnectorTestOutput, error) {
	_ = ctx
	_ = input
	return &ConnectorTestOutput{
		Success: true,
		Message: "连接成功",
	}, nil
}

func (d *OIDCDriver) getProvider(ctx context.Context, issuer string) (oidcProvider, error) {
	if d.providerFactory != nil {
		return d.providerFactory(ctx, issuer)
	}
	return defaultOIDCProviderFactory(ctx, issuer)
}

func (d *OIDCDriver) getNonce() (string, error) {
	if d.nonceGenerator != nil {
		return d.nonceGenerator()
	}
	return defaultOIDCNonceGenerator()
}

func (d *OIDCDriver) exchangeToken(ctx context.Context, config oauth2.Config, code string) (*oauth2.Token, error) {
	if d.tokenExchanger != nil {
		return d.tokenExchanger(ctx, config, code)
	}
	return config.Exchange(ctx, code)
}

func (d *OIDCDriver) getVerifier(provider oidcProvider, config *oidc.Config) OIDCVerifier {
	if d.verifierFactory != nil {
		return d.verifierFactory(provider, config)
	}
	return provider.Verifier(config)
}

func (d *OIDCDriver) fetchUserInfoClaims(ctx context.Context, provider oidcProvider, token *oauth2.Token) (map[string]any, error) {
	userInfo, err := provider.UserInfo(ctx, oauth2.StaticTokenSource(token))
	if err != nil {
		return nil, err
	}
	if userInfo == nil {
		return map[string]any{}, nil
	}
	claims := map[string]any{}
	if err := userInfo.Claims(&claims); err != nil {
		return nil, err
	}
	return claims, nil
}

func defaultOIDCProviderFactory(ctx context.Context, issuer string) (oidcProvider, error) {
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, err
	}
	return &coreOIDCProvider{provider: provider}, nil
}

func defaultOIDCNonceGenerator() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func (p *coreOIDCProvider) Endpoint() oauth2.Endpoint {
	return p.provider.Endpoint()
}

func (p *coreOIDCProvider) UserInfo(ctx context.Context, tokenSource oauth2.TokenSource) (OIDCUserInfo, error) {
	return p.provider.UserInfo(ctx, tokenSource)
}

type coreOIDCVerifiedToken struct {
	token *oidc.IDToken
}

func (t coreOIDCVerifiedToken) Claims(target any) error {
	return t.token.Claims(target)
}

func (t coreOIDCVerifiedToken) Nonce() string {
	return t.token.Nonce
}

func (t coreOIDCVerifiedToken) Subject() string {
	return t.token.Subject
}

func (t coreOIDCVerifiedToken) Issuer() string {
	return t.token.Issuer
}

func buildOIDCStandardIdentity(token OIDCVerifiedToken, claims map[string]any) (StandardIdentity, error) {
	if token == nil || token.Subject() == "" || token.Issuer() == "" {
		return StandardIdentity{}, code.GetError(code.AuthLoginFailedError)
	}
	identity := StandardIdentity{
		Issuer:        token.Issuer(),
		Subject:       token.Subject(),
		Email:         oauth2OptionalClaimString(claims, "email"),
		Username:      oauth2OptionalClaimString(claims, "preferred_username"),
		DisplayName:   oauth2OptionalClaimString(claims, "name"),
		AvatarURL:     oauth2OptionalClaimString(claims, "picture"),
		EmailVerified: oidcClaimBool(claims, "email_verified"),
		Claims:        claims,
	}
	if identity.Username == "" {
		identity.Username = identity.Email
	}
	return identity, nil
}

func oidcClaimBool(claims map[string]any, key string) bool {
	value, ok := claims[key]
	if !ok {
		return false
	}
	switch v := value.(type) {
	case bool:
		return v
	default:
		return false
	}
}

func resolveConnectorCallbackRedirectURI(input *ConnectorCallbackInput) string {
	if input == nil {
		return ""
	}
	if input.RedirectURI != "" {
		return input.RedirectURI
	}
	return input.Config.RedirectURI
}

var _ OIDCVerifiedToken = (*coreOIDCVerifiedToken)(nil)

func (p *coreOIDCProvider) Verifier(config *oidc.Config) OIDCVerifier {
	return coreOIDCVerifier{verifier: p.provider.Verifier(config)}
}

type coreOIDCVerifier struct {
	verifier *oidc.IDTokenVerifier
}

func (v coreOIDCVerifier) Verify(ctx context.Context, rawIDToken string) (OIDCVerifiedToken, error) {
	if v.verifier == nil {
		return nil, errors.New("oidc verifier is nil")
	}
	idToken, err := v.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, err
	}
	return coreOIDCVerifiedToken{token: idToken}, nil
}
