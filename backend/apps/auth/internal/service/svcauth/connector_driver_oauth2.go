package svcauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"

	"github.com/morehao/ark-iam/pkg/code"
)

const connectorProviderWechat = "wechat"

type oauth2IdentityNormalizer func(config ConnectorConfig, claims map[string]any) (StandardIdentity, error)

type OAuth2Driver struct {
	normalizers     map[string]oauth2IdentityNormalizer
	tokenExchanger  func(ctx context.Context, config oauth2.Config, code string) (*oauth2.Token, error)
	userInfoFetcher func(ctx context.Context, token *oauth2.Token, config ConnectorConfig) (map[string]any, error)
}

var _ ConnectorDriver = (*OAuth2Driver)(nil)

func NewOAuth2Driver() ConnectorDriver {
	return &OAuth2Driver{
		normalizers: map[string]oauth2IdentityNormalizer{
			connectorProviderGithub: normalizeGitHubIdentity,
			connectorProviderWechat: normalizeOAuth2IdentityPassthrough,
		},
	}
}

func (d *OAuth2Driver) DriverType() string {
	return connectorDriverTypeOAuth2
}

func (d *OAuth2Driver) ValidateConfig(config ConnectorConfig) error {
	return validateOAuth2ConnectorConfig(config)
}

func (d *OAuth2Driver) BuildAuthorizationURL(ctx *gin.Context, input *ConnectorAuthorizeInput) (*ConnectorAuthorizeOutput, error) {
	_ = ctx
	if input == nil {
		return nil, code.GetError(code.ConnectorGetDetailError)
	}
	if err := d.ValidateConfig(input.Config); err != nil {
		return nil, err
	}

	oauthConfig := oauth2.Config{
		ClientID:     input.Config.ClientID,
		ClientSecret: input.Config.ClientSecret,
		RedirectURL:  resolveConnectorRedirectURI(input),
		Scopes:       input.Config.Scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  input.Config.AuthURL,
			TokenURL: input.Config.TokenURL,
		},
	}

	return &ConnectorAuthorizeOutput{
		AuthorizationURL: oauthConfig.AuthCodeURL(input.State),
		Nonce:            "",
	}, nil
}

func (d *OAuth2Driver) ExchangeCallback(ctx *gin.Context, input *ConnectorCallbackInput) (*ConnectorCallbackOutput, error) {
	if input == nil {
		return nil, code.GetError(code.ConnectorGetDetailError)
	}
	if err := d.ValidateConfig(input.Config); err != nil {
		return nil, err
	}
	oauthConfig := oauth2.Config{
		ClientID:     input.Config.ClientID,
		ClientSecret: input.Config.ClientSecret,
		RedirectURL:  resolveConnectorCallbackRedirectURI(input),
		Scopes:       input.Config.Scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  input.Config.AuthURL,
			TokenURL: input.Config.TokenURL,
		},
	}
	token, err := d.exchangeToken(runtimeContext(ctx), oauthConfig, input.Code)
	if err != nil {
		return nil, code.GetError(code.AuthLoginFailedError)
	}
	claims, err := d.fetchUserInfo(runtimeContext(ctx), token, input.Config)
	if err != nil {
		return nil, code.GetError(code.AuthLoginFailedError)
	}
	identity, err := d.normalizeIdentity(input.Config, claims)
	if err != nil {
		return nil, err
	}
	return &ConnectorCallbackOutput{
		Identity:     identity,
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
	}, nil
}

func (d *OAuth2Driver) TestConnection(ctx *gin.Context, input *ConnectorTestInput) (*ConnectorTestOutput, error) {
	_ = ctx
	_ = input
	return &ConnectorTestOutput{
		Success: true,
		Message: "连接成功",
	}, nil
}

func (d *OAuth2Driver) normalizeIdentity(config ConnectorConfig, claims map[string]any) (StandardIdentity, error) {
	if claims == nil {
		claims = map[string]any{}
	}
	normalizer, ok := d.getNormalizers()[config.Provider]
	if !ok {
		return normalizeOAuth2IdentityPassthrough(config, claims)
	}
	return normalizer(config, claims)
}

func (d *OAuth2Driver) getNormalizers() map[string]oauth2IdentityNormalizer {
	if d.normalizers != nil {
		return d.normalizers
	}
	return NewOAuth2Driver().(*OAuth2Driver).normalizers
}

func (d *OAuth2Driver) exchangeToken(ctx context.Context, config oauth2.Config, code string) (*oauth2.Token, error) {
	if d.tokenExchanger != nil {
		return d.tokenExchanger(ctx, config, code)
	}
	return config.Exchange(ctx, code)
}

func (d *OAuth2Driver) fetchUserInfo(ctx context.Context, token *oauth2.Token, config ConnectorConfig) (map[string]any, error) {
	if d.userInfoFetcher != nil {
		return d.userInfoFetcher(ctx, token, config)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, config.UserInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	claims := map[string]any{}
	if err := json.NewDecoder(resp.Body).Decode(&claims); err != nil {
		return nil, err
	}
	return claims, nil
}

func normalizeGitHubIdentity(config ConnectorConfig, claims map[string]any) (StandardIdentity, error) {
	subject, err := oauth2ClaimString(claims, "id")
	if err != nil {
		return StandardIdentity{}, err
	}
	return StandardIdentity{
		Issuer:      config.Provider,
		Subject:     subject,
		Username:    oauth2OptionalClaimString(claims, "login"),
		DisplayName: oauth2OptionalClaimString(claims, "name"),
		AvatarURL:   oauth2OptionalClaimString(claims, "avatar_url"),
		Claims:      claims,
	}, nil
}

func normalizeOAuth2IdentityPassthrough(config ConnectorConfig, claims map[string]any) (StandardIdentity, error) {
	return StandardIdentity{
		Issuer: config.Provider,
		Claims: claims,
	}, nil
}

func oauth2ClaimString(claims map[string]any, key string) (string, error) {
	value, ok := claims[key]
	if !ok {
		return "", code.GetError(code.ConnectorGetDetailError)
	}
	str, ok := oauth2StringValue(value)
	if !ok || str == "" {
		return "", code.GetError(code.ConnectorGetDetailError)
	}
	return str, nil
}

func oauth2OptionalClaimString(claims map[string]any, key string) string {
	value, ok := claims[key]
	if !ok {
		return ""
	}
	str, ok := oauth2StringValue(value)
	if !ok {
		return ""
	}
	return str
}

func oauth2StringValue(value any) (string, bool) {
	switch v := value.(type) {
	case string:
		return v, true
	case fmt.Stringer:
		return v.String(), true
	case int:
		return strconv.Itoa(v), true
	case int8:
		return strconv.FormatInt(int64(v), 10), true
	case int16:
		return strconv.FormatInt(int64(v), 10), true
	case int32:
		return strconv.FormatInt(int64(v), 10), true
	case int64:
		return strconv.FormatInt(v, 10), true
	case uint:
		return strconv.FormatUint(uint64(v), 10), true
	case uint8:
		return strconv.FormatUint(uint64(v), 10), true
	case uint16:
		return strconv.FormatUint(uint64(v), 10), true
	case uint32:
		return strconv.FormatUint(uint64(v), 10), true
	case uint64:
		return strconv.FormatUint(v, 10), true
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32), true
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), true
	default:
		return "", false
	}
}
