package model

import (
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/datatypes"
)

const TableNameApplicationClient = "application_client"

// ApplicationClientSource 应用客户端来源。取值与语义同 AppSource（见 application.go），
// 独立具名类型以避免跨实体混用常量。
type ApplicationClientSource string

// 应用客户端来源取值（禁止硬编码）。
const (
	ApplicationClientSourceBuiltin    ApplicationClientSource = "builtin"     // 平台内置：种子播种，禁删
	ApplicationClientSourceFirstParty ApplicationClientSource = "first_party" // 平台自建：非内置的第一方客户端
	ApplicationClientSourceThirdParty ApplicationClientSource = "third_party" // 第三方接入：外部/租户接入的客户端
)

// IsBuiltin 判断是否为平台内置客户端（禁删）。
func (s ApplicationClientSource) IsBuiltin() bool { return s == ApplicationClientSourceBuiltin }

// ApplicationClientStatus 应用客户端启停状态（启停语义统一使用 enable/disable）。
type ApplicationClientStatus string

// 客户端状态取值（禁止硬编码）。
const (
	ApplicationClientStatusEnable  ApplicationClientStatus = "enable"  // 启用
	ApplicationClientStatusDisable ApplicationClientStatus = "disable" // 停用
)

// GrantType OAuth 2.0 授权类型（grant_types JSON 数组元素，见 RFC 6749）。
type GrantType string

// 授权类型取值（禁止硬编码），取值为 OAuth2/OIDC 协议标准字面量。
const (
	GrantTypeAuthorizationCode GrantType = "authorization_code"
	GrantTypeClientCredentials GrantType = "client_credentials"
	GrantTypeRefreshToken      GrantType = "refresh_token"
)

// TokenEndpointAuthMethod 令牌端点客户端认证方式（token_endpoint_auth_method，见 RFC 8414）。
type TokenEndpointAuthMethod string

// 令牌端点认证方式取值（禁止硬编码），取值为 OAuth2/OIDC 协议标准字面量。
const (
	TokenEndpointAuthMethodBasic TokenEndpointAuthMethod = "client_secret_basic"
	TokenEndpointAuthMethodPost  TokenEndpointAuthMethod = "client_secret_post"
	TokenEndpointAuthMethodNone  TokenEndpointAuthMethod = "none"
)

type ApplicationClientEntity struct {
	gormdao.BaseEntity
	TenantID string `gorm:"column:tenant_id;type:varchar(36);not null;default:'';comment:租户id" json:"tenantID"`
	AppID    string `gorm:"column:app_id;type:varchar(36);not null;default:'';comment:所属应用id" json:"appID"`
	// Code 是客户端编码，即 OIDC 协议里的 client_id（客户端唯一标识）：
	// - 控制台创建时随机生成（generateClientCode → UUID），内置客户端由种子写入可读值（如 platform-admin-web）；
	// - 「编码」在本表指协议标识符，**不适用** AppCodePattern（那条规则只管 application.code，禁连字符）：
	//   客户端编码允许连字符，取值口径见 docs/design/sso-oidc-concepts.md §3.2；
	// - 与 application_client.id（控制台内部主键，其他表以其为外键）不是一回事。
	Code string `gorm:"column:code;type:varchar(64);not null;default:'';uniqueIndex;comment:客户端编码(= OIDC client_id)" json:"code"`
	Name string `gorm:"column:name;type:varchar(256);not null;default:'';comment:客户端名称" json:"name"`

	RedirectURIs            datatypes.JSON          `gorm:"column:redirect_uris;type:json;not null;default:('[]');comment:授权回调地址" json:"redirectURIs"`
	PostLogoutRedirectURIs  datatypes.JSON          `gorm:"column:post_logout_redirect_uris;type:json;not null;default:('[]');comment:登出回调地址" json:"postLogoutRedirectURIs"`
	BackChannelLogoutURI    string                  `gorm:"column:back_channel_logout_uri;type:varchar(512);not null;default:'';comment:OIDC背信道登出通知地址" json:"backChannelLogoutURI"`
	GrantTypes              datatypes.JSON          `gorm:"column:grant_types;type:json;not null;default:('[\"authorization_code\"]');comment:授权类型" json:"grantTypes"`
	ResponseTypes           datatypes.JSON          `gorm:"column:response_types;type:json;not null;default:('[\"code\"]');comment:响应类型" json:"responseTypes"`
	TokenEndpointAuthMethod TokenEndpointAuthMethod `gorm:"column:token_endpoint_auth_method;type:varchar(32);not null;default:'client_secret_basic';comment:令牌端点认证方式" json:"tokenEndpointAuthMethod"`
	AllowedOrigins          datatypes.JSON          `gorm:"column:allowed_origins;type:json;not null;default:('[]');comment:CORS白名单" json:"allowedOrigins"`
	RequirePKCE             bool                    `gorm:"column:require_pkce;type:boolean;not null;default:false;comment:是否强制PKCE" json:"requirePKCE"`
	RequireAuthTime         bool                    `gorm:"column:require_auth_time;type:boolean;not null;default:false;comment:是否需要auth_time声明" json:"requireAuthTime"`
	DefaultScopes           datatypes.JSON          `gorm:"column:default_scopes;type:json;not null;default:('[\"openid\",\"profile\"]');comment:默认权限范围" json:"defaultScopes"`
	AccessTokenTTL          int64                   `gorm:"column:access_token_ttl;type:bigint;not null;default:900;comment:访问令牌有效期(秒)" json:"accessTokenTTL"`
	RefreshTokenTTL         int64                   `gorm:"column:refresh_token_ttl;type:bigint;not null;default:2592000;comment:刷新令牌有效期(秒)" json:"refreshTokenTTL"`
	Source                  ApplicationClientSource `gorm:"column:source;type:varchar(32);not null;default:'third_party';comment:客户端来源(builtin内置/first_party第一方/third_party第三方)" json:"source"`
	Status                  ApplicationClientStatus `gorm:"column:status;type:varchar(32);not null;default:'enable';comment:状态" json:"status"`

	CreatedBy string `gorm:"column:created_by;type:varchar(36);not null;default:'';comment:创建人id" json:"createdBy"`
	UpdatedBy string `gorm:"column:updated_by;type:varchar(36);not null;default:'';comment:更新人id" json:"updatedBy"`
	DeletedBy string `gorm:"column:deleted_by;type:varchar(36);not null;default:'';comment:删除人id" json:"deletedBy"`
}

func (ApplicationClientEntity) TableName() string { return TableNameApplicationClient }

type ApplicationClientEntityList []ApplicationClientEntity

func (l ApplicationClientEntityList) ToMap() map[string]ApplicationClientEntity {
	m := make(map[string]ApplicationClientEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
