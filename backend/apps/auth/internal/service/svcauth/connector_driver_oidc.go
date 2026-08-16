package svcauth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"

	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/golib/glog"
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
	tokenExchanger  func(ctx context.Context, config oauth2.Config, code string, codeVerifier string) (*oauth2.Token, error)
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
	// H10：discovery issuer 出站目标做 SSRF 防护
	if err := validateOutboundURL(input.Config.Issuer); err != nil {
		glog.Warnf(ctx, "[OIDCDriver.BuildAuthorizationURL] invalid issuer, err:%v", err)
		return nil, code.GetError(code.ConnectorGetDetailError)
	}

	provider, err := d.getProvider(ctx, input.Config.Issuer)
	if err != nil {
		return nil, code.GetError(code.ConnectorGetDetailError)
	}
	nonce, err := d.getNonce()
	if err != nil {
		return nil, code.GetError(code.ConnectorGetDetailError)
	}
	// H10：授权码模式启用 PKCE（S256）
	verifier, err := generatePKCEVerifier()
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
	authURL := oauthConfig.AuthCodeURL(input.State,
		oidc.Nonce(nonce),
		oauth2.SetAuthURLParam("code_challenge", pkceChallengeS256(verifier)),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)
	return &ConnectorAuthorizeOutput{
		AuthorizationURL: authURL,
		Nonce:            nonce,
		CodeVerifier:     verifier,
	}, nil
}

func (d *OIDCDriver) ExchangeCallback(ctx *gin.Context, input *ConnectorCallbackInput) (*ConnectorCallbackOutput, error) {
	if input == nil {
		return nil, code.GetError(code.ConnectorGetDetailError)
	}
	if err := d.ValidateConfig(input.Config); err != nil {
		return nil, err
	}
	// H10：discovery issuer 出站目标做 SSRF 防护
	if err := validateOutboundURL(input.Config.Issuer); err != nil {
		glog.Warnf(ctx, "[OIDCDriver.ExchangeCallback] invalid issuer, err:%v", err)
		return nil, code.GetError(code.ConnectorGetDetailError)
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
	token, err := d.exchangeToken(runtimeContext(ctx), oauthConfig, input.Code, input.CodeVerifier)
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
	// H10：nonce 强制校验（不再条件跳过）——nonce 由本系统生成并持久化在 state，
	// 缺失或失配均视为登录请求重放/会话固定攻击。
	if input.Nonce == "" || verifiedToken.Nonce() != input.Nonce {
		glog.Warnf(ctx, "[OIDCDriver.ExchangeCallback] nonce mismatch, connectorID:%s", input.ConnectorID)
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

func (d *OIDCDriver) exchangeToken(ctx context.Context, config oauth2.Config, code string, codeVerifier string) (*oauth2.Token, error) {
	if d.tokenExchanger != nil {
		return d.tokenExchanger(ctx, config, code, codeVerifier)
	}
	authOpts := []oauth2.AuthCodeOption{}
	if codeVerifier != "" {
		authOpts = append(authOpts, oauth2.SetAuthURLParam("code_verifier", codeVerifier))
	}
	return config.Exchange(ctx, code, authOpts...)
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
	// H10：discovery 出站请求带超时，防 IdP 不响应挂起登录流程
	discoveryCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	provider, err := oidc.NewProvider(discoveryCtx, issuer)
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
