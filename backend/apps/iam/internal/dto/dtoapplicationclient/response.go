package dtoapplicationclient

type CreateResp struct {
	ApplicationClientID uint   `json:"applicationClientId"`   // OAuth客户端ID
	ClientID      string `json:"clientID"`         // OIDC客户端ID
}

type DetailResp struct {
	ApplicationClientID           uint     `json:"applicationClientId"`            // OAuth客户端ID
	TenantID                uint     `json:"tenantId"`                 // 租户ID
	AppID                   uint     `json:"appId"`                    // 所属应用ID
	ClientID                string   `json:"clientID"`                 // OIDC客户端ID
	Name                    string   `json:"name"`                     // 客户端名称
	RedirectURIs            []string `json:"redirectURIs"`             // 授权回调地址
	PostLogoutRedirectURIs  []string `json:"postLogoutRedirectURIs"`   // 登出回调地址
	GrantTypes              []string `json:"grantTypes"`               // 授权类型
	ResponseTypes           []string `json:"responseTypes"`            // 响应类型
	TokenEndpointAuthMethod string   `json:"tokenEndpointAuthMethod"`  // 令牌端点认证方式
	AllowedOrigins          []string `json:"allowedOrigins"`           // CORS白名单
	RequirePKCE             int8     `json:"requirePKCE"`              // 是否强制PKCE
	RequireAuthTime         int8     `json:"requireAuthTime"`          // 是否需要auth_time声明
	DefaultScopes           []string `json:"defaultScopes"`            // 默认权限范围
	AccessTokenTTL          int64    `json:"accessTokenTTL"`           // 访问令牌有效期(秒)
	RefreshTokenTTL         int64    `json:"refreshTokenTTL"`          // 刷新令牌有效期(秒)
	Type                    string   `json:"type"`                     // 客户端类型
	IsThirdParty            int8     `json:"isThirdParty"`             // 是否第三方应用
	Status                  string   `json:"status"`                   // 状态
	CreatedAt               string   `json:"createdAt"`                // 创建时间
}

type PageListResp struct {
	List  []PageListItem `json:"list"`              // 列表数据
	Total int64          `json:"total"`             // 总数
}

type PageListItem struct {
	ApplicationClientID           uint     `json:"applicationClientId"`            // OAuth客户端ID
	AppID                   uint     `json:"appId"`                    // 所属应用ID
	ClientID                string   `json:"clientID"`                 // OIDC客户端ID
	Name                    string   `json:"name"`                     // 客户端名称
	Type                    string   `json:"type"`                     // 客户端类型
	Status                  string   `json:"status"`                   // 状态
	IsThirdParty            int8     `json:"isThirdParty"`             // 是否第三方应用
	GrantTypes              []string `json:"grantTypes"`               // 授权类型
	TokenEndpointAuthMethod string   `json:"tokenEndpointAuthMethod"`  // 令牌端点认证方式
	CreatedAt               string   `json:"createdAt"`                // 创建时间
}

type SecretResp struct {
	ID            uint64  `json:"id"`                                    // 密钥ID
	ApplicationClientID uint64  `json:"applicationClientId"`                        // OAuth客户端ID
	Name          string  `json:"name"`                                  // 密钥名称
	ValuePrefix   string  `json:"valuePrefix"`                          // 密钥前缀
	ExpiredAt     *string `json:"expiresAt"`                             // 过期时间
	CreatedAt     string  `json:"createdAt"`                             // 创建时间
}

type SecretListResp struct {
	Total   int64        `json:"total"`                                 // 总数
	Secrets []SecretResp `json:"secrets"`                               // 密钥列表
}

type CreateSecretResp struct {
	ID          uint64 `json:"id"`                                      // 密钥ID
	Name        string `json:"name"`                                    // 密钥名称
	ValuePrefix string `json:"valuePrefix"`                             // 密钥前缀
	Secret      string `json:"secret"`                                  // 密钥明文（仅创建时返回）
}
