package dtoapplication

type ApplicationCreateReq struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	LogoURL     string `json:"logoURL"`
	HomepageURL string `json:"homepageURL"`
	Type        string `json:"type"`
	IsThirdParty int8  `json:"isThirdParty"`

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

type ApplicationUpdateReq struct {
	ApplicationID uint   `json:"applicationID"`
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

type ApplicationDeleteReq struct {
	ApplicationID uint `json:"applicationID"`
}

type ApplicationDetailReq struct {
	ApplicationID uint `json:"applicationID"`
}

type ApplicationPageListReq struct {
	Page        int    `json:"page"`
	PageSize    int    `json:"pageSize"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Status      string `json:"status"`
}

type ApplicationRoleListReq struct {
	ApplicationID uint `json:"applicationId" form:"applicationId" binding:"required"`
}

type AssignApplicationRolesReq struct {
	ApplicationID uint64   `json:"applicationId" binding:"required"`
	RoleIDs       []uint64 `json:"roleIds" binding:"required,min=1"`
}

type RemoveApplicationRoleReq struct {
	ApplicationID uint64 `json:"applicationId" form:"applicationId" binding:"required"`
	RoleID        uint64 `json:"roleId" uri:"roleId" binding:"required"`
}

type ApplicationSecretListReq struct {
	ApplicationID uint `json:"applicationId" form:"applicationId" binding:"required"`
}

type CreateApplicationSecretReq struct {
	ApplicationID uint   `json:"applicationId" binding:"required"`
	Name          string `json:"name" binding:"required"`
	ExpiredAt     string `json:"expiresAt"`
}

type DeleteApplicationSecretReq struct {
	SecretID uint64 `json:"secretId" uri:"secretId" binding:"required"`
}
