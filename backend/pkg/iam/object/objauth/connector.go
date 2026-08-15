package objauth

type ConnectorBaseInfo struct {
	TenantID            string `json:"tenantID" form:"tenantID"`                       // 租户ID
	Name                string `json:"name" form:"name"`                               // 连接器名称
	DisplayName         string `json:"displayName" form:"displayName"`                 // 显示名称
	Protocol            string `json:"protocol" form:"protocol"`                       // 协议类型
	Provider            string `json:"provider" form:"provider"`                       // 提供商
	Status              string `json:"status" form:"status"`                           // 状态
	AllowAutoCreateUser int8   `json:"allowAutoCreateUser" form:"allowAutoCreateUser"` // 是否允许自动创建用户
	AllowAccountLink    int8   `json:"allowAccountLink" form:"allowAccountLink"`       // 是否允许账号关联
	SyncProfile         int8   `json:"syncProfile" form:"syncProfile"`                 // 是否同步资料
	EnableTokenStorage  int8   `json:"enableTokenStorage" form:"enableTokenStorage"`   // 是否启用令牌存储
	Config              any    `json:"config" form:"config"`                           // 连接器配置
	ClaimMapping        any    `json:"claimMapping" form:"claimMapping"`               // 声明映射
	DomainPolicy        any    `json:"domainPolicy" form:"domainPolicy"`               // 域策略
}
