package oidcop

import (
	"net/url"
	"time"

	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/oidc/v3/pkg/op"

	"github.com/morehao/ark-iam/pkg/model"
)

// idTokenLifetime 是 ID token 的默认有效期（10 分钟）。
const idTokenLifetime = 10 * time.Minute

type OIDCClient struct {
	clientEntity *model.ApplicationClientEntity
	// issuer 为 OP 的 issuer，用于构造 LoginURL（由组装方注入）。
	issuer string
}

var _ op.Client = (*OIDCClient)(nil)

func NewOIDCClient(clientEntity *model.ApplicationClientEntity, issuer string) *OIDCClient {
	return &OIDCClient{clientEntity: clientEntity, issuer: issuer}
}

func (c *OIDCClient) GetID() string {
	// 实体业务编码 code 即 OIDC 协议中的 client_id（唯一映射点，§5.5）
	return c.clientEntity.Code
}

func (c *OIDCClient) RedirectURIs() []string {
	return c.clientEntity.RedirectURIs.Strings()
}

func (c *OIDCClient) PostLogoutRedirectURIs() []string {
	return c.clientEntity.PostLogoutRedirectURIs.Strings()
}

func (c *OIDCClient) ApplicationType() op.ApplicationType {
	return op.ApplicationTypeWeb
}

func (c *OIDCClient) AuthMethod() oidc.AuthMethod {
	switch c.clientEntity.TokenEndpointAuthMethod {
	case model.TokenEndpointAuthMethodPost:
		return oidc.AuthMethodPost
	case model.TokenEndpointAuthMethodNone:
		return oidc.AuthMethodNone
	case model.TokenEndpointAuthMethodBasic:
		return oidc.AuthMethodBasic
	default:
		// M5：private_key_jwt / client_secret_jwt 等未实现的认证方式显式失败（fail-closed），
		// 不再静默退化为 Basic——zitadel 会因 AuthMethodPrivateKeyJWT 不受支持而干净地拒绝该客户端。
		return oidc.AuthMethodPrivateKeyJWT
	}
}

func (c *OIDCClient) ResponseTypes() []oidc.ResponseType {
	types := make([]oidc.ResponseType, 0, len(c.clientEntity.ResponseTypes))
	for _, rt := range c.clientEntity.ResponseTypes {
		switch rt {
		case model.ResponseTypeCode:
			types = append(types, oidc.ResponseTypeCode)
		case model.ResponseTypeIDToken:
			types = append(types, oidc.ResponseTypeIDTokenOnly)
		case model.ResponseTypeIDTokenToken:
			types = append(types, oidc.ResponseTypeIDToken)
		}
	}
	return types
}

func (c *OIDCClient) GrantTypes() []oidc.GrantType {
	types := make([]oidc.GrantType, 0, len(c.clientEntity.GrantTypes))
	for _, gt := range c.clientEntity.GrantTypes {
		switch gt {
		case model.GrantTypeAuthorizationCode:
			types = append(types, oidc.GrantTypeCode)
		case model.GrantTypeClientCredentials:
			types = append(types, oidc.GrantTypeClientCredentials)
		case model.GrantTypeRefreshToken:
			types = append(types, oidc.GrantTypeRefreshToken)
		}
		// M5：token-exchange / jwt-bearer 尚未实现（无 TokenExchangeStorage / JWT 公钥注册），
		// 一律不映射，避免向客户端宣称实际不支持的能力。
	}
	return types
}

func (c *OIDCClient) LoginURL(id string) string {
	return c.issuer + "/sso-login?authRequestID=" + url.QueryEscape(id)
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
	for _, s := range c.clientEntity.DefaultScopes {
		if s == scope {
			return true
		}
	}
	switch scope {
	case model.ScopeOpenID, model.ScopeProfile, model.ScopeEmail, model.ScopePhone:
		return true
	}
	return false
}

// IDTokenUserinfoClaimsAssertion 声明「把 userinfo 声明合并进 ID token」。
//
// 必须为 true：zitadel 在 false 时会从 ID token 的 scope 里裁掉 profile/email/phone
// （见 pkg/op/token.go 的 removeUserinfoScopes），ID token 只剩 sub；而本系统依赖 profile
// scope 承载 groups（角色编码，见 PersistentStore.appendRoleGroupClaims）这类跨系统授权声明
// ——RP 通常只校验 ID token 而不回查 userinfo，被裁掉就等于授权信息静默丢失。
func (c *OIDCClient) IDTokenUserinfoClaimsAssertion() bool {
	return true
}

func (c *OIDCClient) ClockSkew() time.Duration {
	return 0
}
