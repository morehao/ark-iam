package model

import (
	"github.com/morehao/golib/dbaccess/gormdao"
)

const TableNameConnector = "connector"

// ConnectorStatus 连接器启停状态。
// 取值属全局启停语义，与其它领域统一为 enable/disable
// （见 docs/design/glossary.md「启停状态」）。
type ConnectorStatus string

// 连接器状态取值（禁止硬编码）。
const (
	ConnectorStatusEnable  ConnectorStatus = "enable"  // 启用：可用，参与授权与回调
	ConnectorStatusDisable ConnectorStatus = "disable" // 停用：保留配置但不参与授权
)

// ConnectorProtocol 连接器协议类型（字典值，落 connector.protocol，同时作为驱动注册键）。
type ConnectorProtocol string

// 连接器协议取值（禁止硬编码）。
const (
	ConnectorProtocolOIDC   ConnectorProtocol = "oidc"   // OIDC 协议
	ConnectorProtocolOAuth2 ConnectorProtocol = "oauth2" // OAuth2 协议
)

// ConnectorProvider 连接器身份提供商（字典值，落 connector.provider 与 user_identity.provider）。
// 取值集合为内置工厂已知的提供商；自定义连接器仍可落在本具名类型上（底层是 string）。
type ConnectorProvider string

// 连接器提供商取值（禁止硬编码）。
const (
	ConnectorProviderGoogle    ConnectorProvider = "google"    // Google
	ConnectorProviderGithub    ConnectorProvider = "github"    // GitHub
	ConnectorProviderMicrosoft ConnectorProvider = "microsoft" // Microsoft Entra ID
	ConnectorProviderWechat    ConnectorProvider = "wechat"    // 微信
)

// ConnectorCapability 连接器能力标识（工厂描述随列表下发，供前端展示可用能力）。
type ConnectorCapability string

// 连接器能力取值（禁止硬编码）。
const (
	ConnectorCapabilityAuthorize    ConnectorCapability = "authorize"     // 发起授权
	ConnectorCapabilityCallback     ConnectorCapability = "callback"      // 处理回调
	ConnectorCapabilityClaimMapping ConnectorCapability = "claim_mapping" // 声明映射
	ConnectorCapabilityDomainPolicy ConnectorCapability = "domain_policy" // 域策略
	ConnectorCapabilityProfileSync  ConnectorCapability = "profile_sync"  // 资料同步
)

// ConnectorAutoCreateUserFlag connector.allow_auto_create_user。
type ConnectorAutoCreateUserFlag string

// 自动建用户开关取值（禁止硬编码）。
const (
	ConnectorAutoCreateUserFlagEnable  ConnectorAutoCreateUserFlag = "enable"
	ConnectorAutoCreateUserFlagDisable ConnectorAutoCreateUserFlag = "disable"
)

// ConnectorAccountLinkFlag connector.allow_account_link。
type ConnectorAccountLinkFlag string

// 账号关联开关取值（禁止硬编码）。
const (
	ConnectorAccountLinkFlagEnable  ConnectorAccountLinkFlag = "enable"
	ConnectorAccountLinkFlagDisable ConnectorAccountLinkFlag = "disable"
)

// ConnectorSyncProfileFlag connector.sync_profile。
type ConnectorSyncProfileFlag string

// 资料同步开关取值（禁止硬编码）。
const (
	ConnectorSyncProfileFlagEnable  ConnectorSyncProfileFlag = "enable"
	ConnectorSyncProfileFlagDisable ConnectorSyncProfileFlag = "disable"
)

// ConnectorTokenStorageFlag connector.enable_token_storage。
type ConnectorTokenStorageFlag string

// 令牌存储开关取值（禁止硬编码）。
const (
	ConnectorTokenStorageFlagEnable  ConnectorTokenStorageFlag = "enable"
	ConnectorTokenStorageFlagDisable ConnectorTokenStorageFlag = "disable"
)

type ConnectorEntity struct {
	gormdao.BaseEntity
	TenantID            string                      `gorm:"column:tenant_id;type:varchar(36);not null;default:'';comment:租户id"`
	Name                string                      `gorm:"column:name;type:varchar(128);not null;default:'';comment:连接器名称"`
	DisplayName         string                      `gorm:"column:display_name;type:varchar(128);not null;default:'';comment:显示名称"`
	Protocol            ConnectorProtocol           `gorm:"column:protocol;type:varchar(64);not null;default:'';comment:协议类型"`
	Provider            ConnectorProvider           `gorm:"column:provider;type:varchar(128);not null;default:'';comment:提供商"`
	Status              ConnectorStatus             `gorm:"column:status;type:varchar(32);not null;default:'enable';comment:状态(enable启用/disable停用)"`
	AllowAutoCreateUser ConnectorAutoCreateUserFlag `gorm:"column:allow_auto_create_user;type:varchar(16);not null;default:'disable';comment:是否允许自动创建用户(enable/disable)"`
	AllowAccountLink    ConnectorAccountLinkFlag    `gorm:"column:allow_account_link;type:varchar(16);not null;default:'disable';comment:是否允许账号关联(enable/disable)"`
	SyncProfile         ConnectorSyncProfileFlag    `gorm:"column:sync_profile;type:varchar(16);not null;default:'disable';comment:是否同步资料(enable/disable)"`
	EnableTokenStorage  ConnectorTokenStorageFlag   `gorm:"column:enable_token_storage;type:varchar(16);not null;default:'disable';comment:是否启用令牌存储(enable/disable)"`
	Config              ConnectorConfig             `gorm:"column:config;type:json;serializer:json;not null;default:'{}';comment:连接器配置"`
	ClaimMapping        ConnectorClaimMapping       `gorm:"column:claim_mapping;type:json;serializer:json;not null;default:'{}';comment:声明映射"`
	DomainPolicy        ConnectorDomainPolicy       `gorm:"column:domain_policy;type:json;serializer:json;not null;default:'{}';comment:域策略"`
	CreatedBy           string                      `gorm:"column:created_by;type:varchar(36);not null;default:'';comment:创建人ID"`
	UpdatedBy           string                      `gorm:"column:updated_by;type:varchar(36);not null;default:'';comment:更新人ID"`
	DeletedBy           string                      `gorm:"column:deleted_by;type:varchar(36);not null;default:'';comment:删除人ID"`
}

func (ConnectorEntity) TableName() string {
	return TableNameConnector
}

type ConnectorEntityList []ConnectorEntity

func (l ConnectorEntityList) ToMap() map[string]ConnectorEntity {
	m := make(map[string]ConnectorEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
