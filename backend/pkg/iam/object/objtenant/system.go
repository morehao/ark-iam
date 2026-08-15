package objtenant

type SystemBaseInfo struct {
	TenantID string `json:"tenantID" form:"tenantID"` // 租户ID
	Key      string `json:"key" form:"key"`           // 配置键
	Value    any    `json:"value" form:"value"`       // 配置值
}
