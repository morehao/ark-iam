package objresource

type ResourceBaseInfo struct {
	TenantID       string `json:"tenantID" form:"tenantID"`             // 租户ID
	Name           string `json:"name" form:"name"`                     // 资源名称
	Indicator      string `json:"indicator" form:"indicator"`           // 资源标识符
	IsDefault      int8   `json:"isDefault" form:"isDefault"`           // 是否默认
	AccessTokenTtl int64  `json:"accessTokenTtl" form:"accessTokenTtl"` // 访问令牌TTL
}
