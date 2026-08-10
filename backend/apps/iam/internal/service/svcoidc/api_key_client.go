package svcoidc

import (
	"time"

	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/oidc/v3/pkg/op"

	"github.com/morehao/ark-iam/iam/model"
)

type apiKeyOpClient struct {
	entity *model.ApiKeyEntity
}

var _ op.Client = (*apiKeyOpClient)(nil)

func NewApiKeyOpClient(entity *model.ApiKeyEntity) *apiKeyOpClient {
	return &apiKeyOpClient{entity: entity}
}

func (c *apiKeyOpClient) GetID() string    { return c.entity.KeyPrefix }
func (c *apiKeyOpClient) RedirectURIs() []string {
	return nil
}
func (c *apiKeyOpClient) PostLogoutRedirectURIs() []string { return nil }
func (c *apiKeyOpClient) ApplicationType() op.ApplicationType {
	return op.ApplicationTypeWeb
}
func (c *apiKeyOpClient) AuthMethod() oidc.AuthMethod { return oidc.AuthMethodBasic }
func (c *apiKeyOpClient) ResponseTypes() []oidc.ResponseType {
	return nil
}
func (c *apiKeyOpClient) GrantTypes() []oidc.GrantType {
	return []oidc.GrantType{oidc.GrantTypeClientCredentials}
}
func (c *apiKeyOpClient) LoginURL(id string) string { return "" }
func (c *apiKeyOpClient) AccessTokenType() op.AccessTokenType {
	return op.AccessTokenTypeJWT
}
func (c *apiKeyOpClient) IDTokenLifetime() time.Duration { return time.Hour }
func (c *apiKeyOpClient) DevMode() bool                   { return false }
func (c *apiKeyOpClient) RestrictAdditionalIdTokenScopes() func(scopes []string) []string {
	return func(scopes []string) []string { return scopes }
}
func (c *apiKeyOpClient) RestrictAdditionalAccessTokenScopes() func(scopes []string) []string {
	return func(scopes []string) []string { return scopes }
}
func (c *apiKeyOpClient) IsScopeAllowed(scope string) bool { return true }
func (c *apiKeyOpClient) IDTokenUserinfoClaimsAssertion() bool {
	return false
}
func (c *apiKeyOpClient) ClockSkew() time.Duration { return 0 }
