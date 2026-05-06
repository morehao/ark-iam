package objauth

type ConnectorBaseInfo struct {
	TenantID           uint   `json:"tenantID" form:"tenantID"`                     // 租户ID
	SyncProfile        int8   `json:"syncProfile" form:"syncProfile"`               // 是否同步资料
	EnableTokenStorage int8   `json:"enableTokenStorage" form:"enableTokenStorage"` // 是否启用令牌存储
	ConnectorID        string `json:"connectorID" form:"connectorID"`               // 连接器ID
	Config             any    `json:"config" form:"config"`                         // 连接器配置
	Metadata           any    `json:"metadata" form:"metadata"`                     // 元数据
}