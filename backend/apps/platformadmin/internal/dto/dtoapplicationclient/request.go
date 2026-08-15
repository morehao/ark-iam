package dtoapplicationclient

type ApplicationClientCreateReq struct {
	AppID        uint   `json:"appID" binding:"required"` // 所属应用ID
	Name         string `json:"name" binding:"required"`  // 客户端名称
	Type         string `json:"type"`                     // 客户端类型: first_party-第一方, third_party-第三方
	IsThirdParty int8   `json:"isThirdParty"`             // 是否第三方应用

	RedirectURIs            []string `json:"redirectURIs"`            // 授权回调地址
	PostLogoutRedirectURIs  []string `json:"postLogoutRedirectURIs"`  // 登出回调地址
	BackChannelLogoutURI    string   `json:"backChannelLogoutURI"`    // OIDC背信道登出通知地址
	GrantTypes              []string `json:"grantTypes"`              // 授权类型
	ResponseTypes           []string `json:"responseTypes"`           // 响应类型
	TokenEndpointAuthMethod string   `json:"tokenEndpointAuthMethod"` // 令牌端点认证方式
	AllowedOrigins          []string `json:"allowedOrigins"`          // CORS白名单
	RequirePKCE             int8     `json:"requirePKCE"`             // 是否强制PKCE
	RequireAuthTime         int8     `json:"requireAuthTime"`         // 是否需要auth_time声明
	DefaultScopes           []string `json:"defaultScopes"`           // 默认权限范围
	AccessTokenTTL          int64    `json:"accessTokenTTL"`          // 访问令牌有效期(秒)
	RefreshTokenTTL         int64    `json:"refreshTokenTTL"`         // 刷新令牌有效期(秒)
}

type ApplicationClientUpdateReq struct {
	ApplicationClientID uint   `json:"applicationClientID" binding:"required"` // OAuth客户端ID
	Name                string `json:"name"`                                   // 客户端名称
	Type                string `json:"type"`                                   // 客户端类型
	Status              string `json:"status"`                                 // 状态: enable-启用, disable-停用
	IsThirdParty        int8   `json:"isThirdParty"`                           // 是否第三方应用

	RedirectURIs            []string `json:"redirectURIs"`            // 授权回调地址
	PostLogoutRedirectURIs  []string `json:"postLogoutRedirectURIs"`  // 登出回调地址
	BackChannelLogoutURI    string   `json:"backChannelLogoutURI"`    // OIDC背信道登出通知地址
	GrantTypes              []string `json:"grantTypes"`              // 授权类型
	ResponseTypes           []string `json:"responseTypes"`           // 响应类型
	TokenEndpointAuthMethod string   `json:"tokenEndpointAuthMethod"` // 令牌端点认证方式
	AllowedOrigins          []string `json:"allowedOrigins"`          // CORS白名单
	RequirePKCE             int8     `json:"requirePKCE"`             // 是否强制PKCE
	RequireAuthTime         int8     `json:"requireAuthTime"`         // 是否需要auth_time声明
	DefaultScopes           []string `json:"defaultScopes"`           // 默认权限范围
	AccessTokenTTL          int64    `json:"accessTokenTTL"`          // 访问令牌有效期(秒)
	RefreshTokenTTL         int64    `json:"refreshTokenTTL"`         // 刷新令牌有效期(秒)
}

type ApplicationClientDeleteReq struct {
	ApplicationClientID uint `json:"applicationClientID" binding:"required"` // OAuth客户端ID
}

type ApplicationClientDetailReq struct {
	ApplicationClientID uint `json:"applicationClientID" form:"applicationClientID" binding:"required"` // OAuth客户端ID
}

type ApplicationClientPageListReq struct {
	Page     int    `json:"page"`     // 页码
	PageSize int    `json:"pageSize"` // 每页条数
	Name     string `json:"name"`     // 客户端名称（模糊搜索）
	Type     string `json:"type"`     // 客户端类型
	Status   string `json:"status"`   // 状态
}

type SecretListReq struct {
	ApplicationClientID uint `json:"applicationClientID" form:"applicationClientID" binding:"required"` // OAuth客户端ID
}

type SecretCreateReq struct {
	ApplicationClientID uint   `json:"applicationClientID" binding:"required"` // OAuth客户端ID
	Name                string `json:"name" binding:"required"`                // 密钥名称
	ExpiredAt           string `json:"expiresAt"`                              // 过期时间
}

type SecretDeleteReq struct {
	SecretID uint `json:"secretID" uri:"secretID" binding:"required"` // 密钥ID
}
