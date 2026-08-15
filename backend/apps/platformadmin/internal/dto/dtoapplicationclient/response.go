package dtoapplicationclient

type ApplicationClientCreateResp struct {
	ApplicationClientID string `json:"applicationClientID"` // OAuth客户端ID
	Code                string `json:"code"`                // 客户端编码(OIDC client_id)
}

type ApplicationClientDetailResp struct {
	ApplicationClientID     string   `json:"applicationClientID"`     // OAuth客户端ID
	TenantID                string   `json:"tenantID"`                // 租户ID
	AppID                   string   `json:"appID"`                   // 所属应用ID
	Code                    string   `json:"code"`                    // 客户端编码(OIDC client_id)
	Name                    string   `json:"name"`                    // 客户端名称
	RedirectURIs            []string `json:"redirectURIs"`            // 授权回调地址
	PostLogoutRedirectURIs  []string `json:"postLogoutRedirectURIs"`  // 登出回调地址
	BackChannelLogoutURI    string   `json:"backChannelLogoutURI"`    // OIDC背信道登出通知地址
	GrantTypes              []string `json:"grantTypes"`              // 授权类型
	ResponseTypes           []string `json:"responseTypes"`           // 响应类型
	TokenEndpointAuthMethod string   `json:"tokenEndpointAuthMethod"` // 令牌端点认证方式
	AllowedOrigins          []string `json:"allowedOrigins"`          // CORS白名单
	RequirePKCE bool     `json:"requirePKCE"`             // 是否强制PKCE
	RequireAuthTime bool     `json:"requireAuthTime"`         // 是否需要auth_time声明
	DefaultScopes           []string `json:"defaultScopes"`           // 默认权限范围
	AccessTokenTTL          int64    `json:"accessTokenTTL"`          // 访问令牌有效期(秒)
	RefreshTokenTTL         int64    `json:"refreshTokenTTL"`         // 刷新令牌有效期(秒)
	Type                    string   `json:"type"`                    // 客户端类型
	IsThirdParty bool     `json:"isThirdParty"`            // 是否第三方应用
	Status                  string   `json:"status"`                  // 状态
	CreatedAt               string   `json:"createdAt"`               // 创建时间
}

type ApplicationClientPageListResp struct {
	List  []PageListItem `json:"list"`  // 列表数据
	Total int64          `json:"total"` // 总数
}

type PageListItem struct {
	ApplicationClientID     string   `json:"applicationClientID"`     // OAuth客户端ID
	AppID                   string   `json:"appID"`                   // 所属应用ID
	Code                    string   `json:"code"`                    // 客户端编码(OIDC client_id)
	Name                    string   `json:"name"`                    // 客户端名称
	Type                    string   `json:"type"`                    // 客户端类型
	Status                  string   `json:"status"`                  // 状态
	IsThirdParty bool     `json:"isThirdParty"`            // 是否第三方应用
	GrantTypes              []string `json:"grantTypes"`              // 授权类型
	TokenEndpointAuthMethod string   `json:"tokenEndpointAuthMethod"` // 令牌端点认证方式
	CreatedAt               string   `json:"createdAt"`               // 创建时间
}

type SecretResp struct {
	ID                  string  `json:"id"`                  // 密钥ID
	ApplicationClientID string  `json:"applicationClientID"` // OAuth客户端ID
	Name                string  `json:"name"`                // 密钥名称
	ValuePrefix         string  `json:"valuePrefix"`         // 密钥前缀
	ExpiredAt           *string `json:"expiresAt"`           // 过期时间
	CreatedAt           string  `json:"createdAt"`           // 创建时间
}

type SecretListResp struct {
	Total   int64        `json:"total"`   // 总数
	Secrets []SecretResp `json:"secrets"` // 密钥列表
}

type SecretCreateResp struct {
	ID          string `json:"id"`          // 密钥ID
	Name        string `json:"name"`        // 密钥名称
	ValuePrefix string `json:"valuePrefix"` // 密钥前缀
	Secret      string `json:"secret"`      // 密钥明文（仅创建时返回）
}
