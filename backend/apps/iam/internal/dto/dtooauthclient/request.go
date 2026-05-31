package dtooauthclient

type CreateReq struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	LogoURL     string   `json:"logoURL"`
	HomepageURL string   `json:"homepageURL"`
	Type        string   `json:"type"`
	IsThirdParty int8    `json:"isThirdParty"`

	RedirectURIs            []string `json:"redirectURIs"`
	PostLogoutRedirectURIs  []string `json:"postLogoutRedirectURIs"`
	GrantTypes              []string `json:"grantTypes"`
	ResponseTypes           []string `json:"responseTypes"`
	TokenEndpointAuthMethod string   `json:"tokenEndpointAuthMethod"`
	AllowedOrigins          []string `json:"allowedOrigins"`
	RequirePKCE             int8     `json:"requirePKCE"`
	RequireAuthTime         int8     `json:"requireAuthTime"`
	DefaultScopes           []string `json:"defaultScopes"`
	AccessTokenTTL          int64    `json:"accessTokenTTL"`
	RefreshTokenTTL         int64    `json:"refreshTokenTTL"`
}

type UpdateReq struct {
	OAuthClientID uint   `json:"oauthClientId"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	LogoURL       string `json:"logoURL"`
	HomepageURL   string `json:"homepageURL"`
	Type          string `json:"type"`
	Status        string `json:"status"`
	IsThirdParty  int8   `json:"isThirdParty"`

	RedirectURIs            []string `json:"redirectURIs"`
	PostLogoutRedirectURIs  []string `json:"postLogoutRedirectURIs"`
	GrantTypes              []string `json:"grantTypes"`
	ResponseTypes           []string `json:"responseTypes"`
	TokenEndpointAuthMethod string   `json:"tokenEndpointAuthMethod"`
	AllowedOrigins          []string `json:"allowedOrigins"`
	RequirePKCE             int8     `json:"requirePKCE"`
	RequireAuthTime         int8     `json:"requireAuthTime"`
	DefaultScopes           []string `json:"defaultScopes"`
	AccessTokenTTL          int64    `json:"accessTokenTTL"`
	RefreshTokenTTL         int64    `json:"refreshTokenTTL"`
}

type DeleteReq struct {
	OAuthClientID uint `json:"oauthClientId"`
}

type DetailReq struct {
	OAuthClientID uint `json:"oauthClientId"`
}

type PageListReq struct {
	Page        int    `json:"page"`
	PageSize    int    `json:"pageSize"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Status      string `json:"status"`
}

type SecretListReq struct {
	OAuthClientID uint `json:"oauthClientId" form:"oauthClientId" binding:"required"`
}

type CreateSecretReq struct {
	OAuthClientID uint   `json:"oauthClientId" binding:"required"`
	Name          string `json:"name" binding:"required"`
	ExpiredAt     string `json:"expiresAt"`
}

type DeleteSecretReq struct {
	SecretID uint64 `json:"secretId" uri:"secretId" binding:"required"`
}
