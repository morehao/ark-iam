package model

import (
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/datatypes"
)

const TableNameApplicationClient = "application_client"

const (
	ApplicationClientTypeFirstParty = "first_party"
	ApplicationClientTypeThirdParty = "third_party"
)

const (
	ApplicationClientStatusEnable  = "enable"
	ApplicationClientStatusDisable = "disable"
)

const (
	GrantTypeAuthorizationCode = "authorization_code"
	GrantTypeClientCredentials = "client_credentials"
	GrantTypeRefreshToken      = "refresh_token"
)

const (
	TokenEndpointAuthMethodBasic = "client_secret_basic"
	TokenEndpointAuthMethodPost  = "client_secret_post"
	TokenEndpointAuthMethodNone  = "none"
)

type ApplicationClientEntity struct {
	gormdao.BaseEntity
	TenantID string `gorm:"column:tenant_id;type:varchar(36);not null;default:'';comment:租户id" json:"tenantID"`
	AppID    string `gorm:"column:app_id;type:varchar(36);not null;default:'';comment:所属应用id" json:"appID"`
	ClientID string `gorm:"column:client_id;type:varchar(64);not null;default:'';uniqueIndex;comment:OIDC客户端ID" json:"clientID"`
	Name     string `gorm:"column:name;type:varchar(256);not null;default:'';comment:客户端名称" json:"name"`

	RedirectURIs            datatypes.JSON `gorm:"column:redirect_uris;type:json;not null;default:('[]');comment:授权回调地址" json:"redirectURIs"`
	PostLogoutRedirectURIs  datatypes.JSON `gorm:"column:post_logout_redirect_uris;type:json;not null;default:('[]');comment:登出回调地址" json:"postLogoutRedirectURIs"`
	BackChannelLogoutURI    string         `gorm:"column:back_channel_logout_uri;type:varchar(512);not null;default:'';comment:OIDC背信道登出通知地址" json:"backChannelLogoutURI"`
	GrantTypes              datatypes.JSON `gorm:"column:grant_types;type:json;not null;default:('[\"authorization_code\"]');comment:授权类型" json:"grantTypes"`
	ResponseTypes           datatypes.JSON `gorm:"column:response_types;type:json;not null;default:('[\"code\"]');comment:响应类型" json:"responseTypes"`
	TokenEndpointAuthMethod string         `gorm:"column:token_endpoint_auth_method;type:varchar(32);not null;default:'client_secret_basic';comment:令牌端点认证方式" json:"tokenEndpointAuthMethod"`
	AllowedOrigins          datatypes.JSON `gorm:"column:allowed_origins;type:json;not null;default:('[]');comment:CORS白名单" json:"allowedOrigins"`
	RequirePKCE             int8           `gorm:"column:require_pkce;type:smallint;not null;default:0;comment:是否强制PKCE" json:"requirePKCE"`
	RequireAuthTime         int8           `gorm:"column:require_auth_time;type:smallint;not null;default:0;comment:是否需要auth_time声明" json:"requireAuthTime"`
	DefaultScopes           datatypes.JSON `gorm:"column:default_scopes;type:json;not null;default:('[\"openid\",\"profile\"]');comment:默认权限范围" json:"defaultScopes"`
	AccessTokenTTL          int64          `gorm:"column:access_token_ttl;type:bigint;not null;default:900;comment:访问令牌有效期(秒)" json:"accessTokenTTL"`
	RefreshTokenTTL         int64          `gorm:"column:refresh_token_ttl;type:bigint;not null;default:2592000;comment:刷新令牌有效期(秒)" json:"refreshTokenTTL"`
	Type                    string         `gorm:"column:type;type:varchar(32);not null;default:'first_party';comment:客户端类型" json:"type"`
	IsThirdParty            int8           `gorm:"column:is_third_party;type:smallint;not null;default:0;comment:是否第三方应用" json:"isThirdParty"`
	Status                  string         `gorm:"column:status;type:varchar(32);not null;default:'enable';comment:状态" json:"status"`
	IsSystem                int8           `gorm:"column:is_system;type:smallint;not null;default:0;comment:是否系统内置" json:"isSystem"`

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
