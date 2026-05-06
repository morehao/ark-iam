package objsso_connector

type SsoConnectorBaseInfo struct {
	TenantID           uint   `json:"tenantID" form:"tenantID"`                       // 租户ID
	ProviderName       string `json:"providerName" form:"providerName"`               // 提供商名称
	ConnectorName      string `json:"connectorName" form:"connectorName"`             // 连接器名称
	Config             any    `json:"config" form:"config"`                           // 配置
	Domains            any    `json:"domains" form:"domains"`                         // 域名列表
	Branding           any    `json:"branding" form:"branding"`                       // 品牌配置
	SyncProfile        int8   `json:"syncProfile" form:"syncProfile"`                 // 是否同步资料
	EnableTokenStorage int8   `json:"enableTokenStorage" form:"enableTokenStorage"`   // 是否启用令牌存储
}