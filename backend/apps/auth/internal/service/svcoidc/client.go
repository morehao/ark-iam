package svcoidc

import (
	"encoding/json"
	"net/url"
	"time"

	"github.com/morehao/ark-iam/auth/config"
	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/oidc/v3/pkg/op"

	"github.com/morehao/ark-iam/pkg/iam/model"
)

// idTokenLifetime 是 ID token 的默认有效期（10 分钟）。
const idTokenLifetime = 10 * time.Minute

type OIDCClient struct {
	clientEntity *model.ApplicationClientEntity
}

var _ op.Client = (*OIDCClient)(nil)

func NewOIDCClient(clientEntity *model.ApplicationClientEntity) *OIDCClient {
	return &OIDCClient{clientEntity: clientEntity}
}

func (c *OIDCClient) GetID() string {
	// 实体业务编码 code 即 OIDC 协议中的 client_id（唯一映射点，§5.5）
	return c.clientEntity.Code
}

func (c *OIDCClient) RedirectURIs() []string {
	var uris []string
	if err := json.Unmarshal(c.clientEntity.RedirectURIs, &uris); err != nil {
		return nil
	}
	return uris
}

func (c *OIDCClient) PostLogoutRedirectURIs() []string {
	var uris []string
	if err := json.Unmarshal(c.clientEntity.PostLogoutRedirectURIs, &uris); err != nil {
		return nil
	}
	return uris
}

func (c *OIDCClient) ApplicationType() op.ApplicationType {
	return op.ApplicationTypeWeb
}

func (c *OIDCClient) AuthMethod() oidc.AuthMethod {
	switch c.clientEntity.TokenEndpointAuthMethod {
	case "client_secret_post":
		return oidc.AuthMethodPost
	case "none":
		return oidc.AuthMethodNone
	case "client_secret_basic":
		return oidc.AuthMethodBasic
	default:
		// M5：private_key_jwt / client_secret_jwt 等未实现的认证方式显式失败（fail-closed），
		// 不再静默退化为 Basic——zitadel 会因 AuthMethodPrivateKeyJWT 不受支持而干净地拒绝该客户端。
		return oidc.AuthMethodPrivateKeyJWT
	}
}

func (c *OIDCClient) ResponseTypes() []oidc.ResponseType {
	var rawTypes []string
	if err := json.Unmarshal(c.clientEntity.ResponseTypes, &rawTypes); err != nil {
		return nil
	}
	types := make([]oidc.ResponseType, 0, len(rawTypes))
	for _, rt := range rawTypes {
		switch rt {
		case "code":
			types = append(types, oidc.ResponseTypeCode)
		case "id_token":
			types = append(types, oidc.ResponseTypeIDTokenOnly)
		case "id_token token":
			types = append(types, oidc.ResponseTypeIDToken)
		}
	}
	return types
}

func (c *OIDCClient) GrantTypes() []oidc.GrantType {
	var rawTypes []string
	if err := json.Unmarshal(c.clientEntity.GrantTypes, &rawTypes); err != nil {
		return nil
	}
	types := make([]oidc.GrantType, 0, len(rawTypes))
	for _, gt := range rawTypes {
		switch gt {
		case "authorization_code":
			types = append(types, oidc.GrantTypeCode)
		case "client_credentials":
			types = append(types, oidc.GrantTypeClientCredentials)
		case "refresh_token":
			types = append(types, oidc.GrantTypeRefreshToken)
		}
		// M5：token-exchange / jwt-bearer 尚未实现（无 TokenExchangeStorage / JWT 公钥注册），
		// 一律不映射，避免向客户端宣称实际不支持的能力。
	}
	return types
}

func (c *OIDCClient) LoginURL(id string) string {
	return config.Conf.OIDC.Issuer + "/sso-login?authRequestID=" + url.QueryEscape(id)
}

func (c *OIDCClient) AccessTokenType() op.AccessTokenType {
	return op.AccessTokenTypeJWT
}

// IDTokenLifetime 返回 ID token 的有效期。ID token 是短生命周期凭证
// （主流 IdP 通常 5~10 分钟），与 access token 的 AccessTokenTTL 解耦。
func (c *OIDCClient) IDTokenLifetime() time.Duration {
	return idTokenLifetime
}

func (c *OIDCClient) DevMode() bool {
	return false
}

func (c *OIDCClient) RestrictAdditionalIdTokenScopes() func(scopes []string) []string {
	return func(scopes []string) []string {
		return scopes
	}
}

func (c *OIDCClient) RestrictAdditionalAccessTokenScopes() func(scopes []string) []string {
	return func(scopes []string) []string {
		return scopes
	}
}

func (c *OIDCClient) IsScopeAllowed(scope string) bool {
	var defaultScopes []string
	if err := json.Unmarshal(c.clientEntity.DefaultScopes, &defaultScopes); err != nil {
		return false
	}
	for _, s := range defaultScopes {
		if s == scope {
			return true
		}
	}
	switch scope {
	case "openid", "profile", "email", "phone":
		return true
	}
	return false
}

func (c *OIDCClient) IDTokenUserinfoClaimsAssertion() bool {
	return false
}

func (c *OIDCClient) ClockSkew() time.Duration {
	return 0
}
