package svcoidc

import (
	"encoding/json"
	"net/url"
	"time"

	"github.com/morehao/ark-iam/iam/config"
	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/oidc/v3/pkg/op"

	"github.com/morehao/ark-iam/pkg/iam/model"
)

type OIDCClient struct {
	clientEntity *model.ApplicationClientEntity
}

var _ op.Client = (*OIDCClient)(nil)

func NewOIDCClient(clientEntity *model.ApplicationClientEntity) *OIDCClient {
	return &OIDCClient{clientEntity: clientEntity}
}

func (c *OIDCClient) GetID() string {
	return c.clientEntity.ClientID
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
	default:
		return oidc.AuthMethodBasic
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
		case "urn:ietf:params:oauth:grant-type:token-exchange":
			types = append(types, oidc.GrantTypeTokenExchange)
		case "urn:ietf:params:oauth:grant-type:jwt-bearer":
			types = append(types, oidc.GrantTypeBearer)
		}
	}
	return types
}

func (c *OIDCClient) LoginURL(id string) string {
	return config.Conf.OIDC.Issuer + "/sso-login?authRequestID=" + url.QueryEscape(id)
}

func (c *OIDCClient) AccessTokenType() op.AccessTokenType {
	return op.AccessTokenTypeJWT
}

func (c *OIDCClient) IDTokenLifetime() time.Duration {
	if c.clientEntity.AccessTokenTTL > 0 {
		return time.Duration(c.clientEntity.AccessTokenTTL) * time.Second
	}
	return time.Hour
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
