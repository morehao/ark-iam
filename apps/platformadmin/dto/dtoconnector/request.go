package dtoconnector

import (
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/object/objidentity"
)

// ConnectorCreateReq 平台侧创建连接器入参。
// 连接器是平台级全局对象：DefaultTenantID 是「自动创建用户默认落租户」，
// 与协议级 config.tenant（Microsoft IdP tenant）是两个独立概念，禁止混淆。
type ConnectorCreateReq struct {
	DefaultTenantID     string                            `json:"defaultTenantID"`                       // 自动创建用户默认落租户ID
	Name                string                            `json:"name"`                                  // 连接器名称
	DisplayName         string                            `json:"displayName"`                           // 显示名称
	Protocol            model.ConnectorProtocol           `json:"protocol"`                              // 协议类型
	Provider            model.ConnectorProvider           `json:"provider"`                              // 提供商
	Status              model.ConnectorStatus             `json:"status"`                                // 状态: enable-启用, disable-停用
	AllowAutoCreateUser model.ConnectorAutoCreateUserFlag `json:"allowAutoCreateUser"`                   // 是否允许自动创建用户(enable/disable)
	AllowAccountLink    model.ConnectorAccountLinkFlag    `json:"allowAccountLink"`                      // 是否允许账号关联(enable/disable)
	SyncProfile         model.ConnectorSyncProfileFlag    `json:"syncProfile"`                           // 是否同步资料(enable/disable)
	EnableTokenStorage  model.ConnectorTokenStorageFlag   `json:"enableTokenStorage"`                    // 是否启用令牌存储(enable/disable)
	Config              model.ConnectorConfig             `json:"config"`                                // 连接器配置
	ClaimMapping        model.ConnectorClaimMapping       `json:"claimMapping"`                          // 声明映射
	DomainPolicy        model.ConnectorDomainPolicy       `json:"domainPolicy"`                          // 域策略
	// Operator 由 controller 从鉴权结果填入（json:"-"，客户端不可伪造）。
	Operator objidentity.Operator
}

// ConnectorUpdateReq 平台侧修改连接器入参（字段语义同 Create，全量更新）。
type ConnectorUpdateReq struct {
	ConnectorID         string                            `json:"-" uri:"connectorID" binding:"required"` // 连接器ID（path 唯一来源）
	DefaultTenantID     string                            `json:"defaultTenantID"`                       // 自动创建用户默认落租户ID
	Name                string                            `json:"name"`                                  // 连接器名称
	DisplayName         string                            `json:"displayName"`                           // 显示名称
	Protocol            model.ConnectorProtocol           `json:"protocol"`                              // 协议类型
	Provider            model.ConnectorProvider           `json:"provider"`                              // 提供商
	Status              model.ConnectorStatus             `json:"status"`                                // 状态
	AllowAutoCreateUser model.ConnectorAutoCreateUserFlag `json:"allowAutoCreateUser"`                   // 是否允许自动创建用户(enable/disable)
	AllowAccountLink    model.ConnectorAccountLinkFlag    `json:"allowAccountLink"`                      // 是否允许账号关联(enable/disable)
	SyncProfile         model.ConnectorSyncProfileFlag    `json:"syncProfile"`                           // 是否同步资料(enable/disable)
	EnableTokenStorage  model.ConnectorTokenStorageFlag   `json:"enableTokenStorage"`                    // 是否启用令牌存储(enable/disable)
	Config              model.ConnectorConfig             `json:"config"`                                // 连接器配置
	ClaimMapping        model.ConnectorClaimMapping       `json:"claimMapping"`                          // 声明映射
	DomainPolicy        model.ConnectorDomainPolicy       `json:"domainPolicy"`                          // 域策略
	// Operator 由 controller 从鉴权结果填入（json:"-"，客户端不可伪造）。
	Operator objidentity.Operator
}

// ConnectorDeleteReq 删除连接器入参。
type ConnectorDeleteReq struct {
	ConnectorID string `json:"-" uri:"connectorID" binding:"required"` // 连接器ID（path 唯一来源）
	// Operator 由 controller 从鉴权结果填入（json:"-"，客户端不可伪造）。
	Operator objidentity.Operator
}

// ConnectorDetailReq 连接器详情入参。
type ConnectorDetailReq struct {
	ConnectorID string `json:"-" uri:"connectorID" binding:"required"` // 连接器ID（path 唯一来源）
	// Operator 由 controller 从鉴权结果填入（json:"-"，客户端不可伪造）。
	Operator objidentity.Operator
}

// ConnectorPageListReq 连接器列表入参（跨租户视角，defaultTenantID 为筛选条件，留空即全部）。
type ConnectorPageListReq struct {
	Page            int                      `json:"page" form:"page"`                       // 页码
	PageSize        int                      `json:"pageSize" form:"pageSize"`               // 每页条数
	DefaultTenantID string                   `json:"defaultTenantID" form:"defaultTenantID"` // 默认落租户ID（筛选）
	Protocol        model.ConnectorProtocol  `json:"protocol" form:"protocol"`               // 协议类型
	Provider        model.ConnectorProvider  `json:"provider" form:"provider"`               // 提供商
	Status          model.ConnectorStatus    `json:"status" form:"status"`                   // 状态
	Name            string                   `json:"name" form:"name"`                       // 名称
	DisplayName     string                   `json:"displayName" form:"displayName"`         // 显示名称
}

// ConnectorFactoryListReq 连接器工厂列表入参（按 protocol/provider 过滤，空值不过滤）。
type ConnectorFactoryListReq struct {
	Protocol model.ConnectorProtocol `json:"protocol" form:"protocol"` // 协议类型
	Provider model.ConnectorProvider `json:"provider" form:"provider"` // 提供商
}

// ConnectorIDReq 仅含连接器ID的动作入参（如测试连接）。
type ConnectorIDReq struct {
	ConnectorID string `json:"-" uri:"connectorID" binding:"required"` // 连接器ID（path 唯一来源）
	// Operator 由 controller 从鉴权结果填入（json:"-"，客户端不可伪造）。
	Operator objidentity.Operator
}
