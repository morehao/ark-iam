package objauth

import "github.com/morehao/ark-iam/pkg/model"

type ConnectorBaseInfo struct {
	TenantID            string                            `json:"tenantID" form:"tenantID"`                       // 租户ID
	Name                string                            `json:"name" form:"name"`                               // 连接器名称
	DisplayName         string                            `json:"displayName" form:"displayName"`                 // 显示名称
	Protocol            model.ConnectorProtocol           `json:"protocol" form:"protocol"`                       // 协议类型
	Provider            model.ConnectorProvider           `json:"provider" form:"provider"`                       // 提供商
	Status              model.ConnectorStatus             `json:"status" form:"status"`                           // 状态
	AllowAutoCreateUser model.ConnectorAutoCreateUserFlag `json:"allowAutoCreateUser" form:"allowAutoCreateUser"` // 是否允许自动创建用户(enable/disable)
	AllowAccountLink    model.ConnectorAccountLinkFlag    `json:"allowAccountLink" form:"allowAccountLink"`       // 是否允许账号关联(enable/disable)
	SyncProfile         model.ConnectorSyncProfileFlag    `json:"syncProfile" form:"syncProfile"`                 // 是否同步资料(enable/disable)
	EnableTokenStorage  model.ConnectorTokenStorageFlag   `json:"enableTokenStorage" form:"enableTokenStorage"`   // 是否启用令牌存储(enable/disable)
	Config              model.ConnectorConfig             `json:"config" form:"config"`                           // 连接器配置
	ClaimMapping        model.ConnectorClaimMapping       `json:"claimMapping" form:"claimMapping"`               // 声明映射
	DomainPolicy        model.ConnectorDomainPolicy       `json:"domainPolicy" form:"domainPolicy"`               // 域策略
}
