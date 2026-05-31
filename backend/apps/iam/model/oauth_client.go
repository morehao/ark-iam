package model

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const TableNameOAuthClient = "oauth_client"

type OAuthClientEntity struct {
	gorm.Model
	TenantID        uint   `gorm:"column:tenant_id;type:bigint unsigned;not null;default 0;comment:租户id" json:"tenantID"`
	ApplicationID   uint   `gorm:"column:application_id;type:bigint unsigned;not null;default 0;comment:所属应用id" json:"applicationID"`
	ClientID        string `gorm:"column:client_id;type:varchar(64);not null;default '';uniqueIndex;comment:OIDC客户端ID" json:"clientID"`
	Name            string `gorm:"column:name;type:varchar(256);not null;default '';comment:客户端名称" json:"name"`

	RedirectURIs            datatypes.JSON `gorm:"column:redirect_uris;type:json;not null;default ('[]');comment:授权回调地址" json:"redirectURIs"`
	PostLogoutRedirectURIs  datatypes.JSON `gorm:"column:post_logout_redirect_uris;type:json;not null;default ('[]');comment:登出回调地址" json:"postLogoutRedirectURIs"`
	GrantTypes              datatypes.JSON `gorm:"column:grant_types;type:json;not null;default ('[\"authorization_code\"]');comment:授权类型" json:"grantTypes"`
	ResponseTypes           datatypes.JSON `gorm:"column:response_types;type:json;not null;default ('[\"code\"]');comment:响应类型" json:"responseTypes"`
	TokenEndpointAuthMethod string         `gorm:"column:token_endpoint_auth_method;type:varchar(32);not null;default 'client_secret_basic';comment:令牌端点认证方式" json:"tokenEndpointAuthMethod"`
	AllowedOrigins          datatypes.JSON `gorm:"column:allowed_origins;type:json;not null;default ('[]');comment:CORS白名单" json:"allowedOrigins"`
	RequirePKCE             int8           `gorm:"column:require_pkce;type:tinyint(1);not null;default 0;comment:是否强制PKCE" json:"requirePKCE"`
	RequireAuthTime         int8           `gorm:"column:require_auth_time;type:tinyint(1);not null;default 0;comment:是否需要auth_time声明" json:"requireAuthTime"`
	DefaultScopes           datatypes.JSON `gorm:"column:default_scopes;type:json;not null;default ('[\"openid\",\"profile\"]');comment:默认权限范围" json:"defaultScopes"`
	AccessTokenTTL          int64          `gorm:"column:access_token_ttl;type:bigint;not null;default 3600;comment:访问令牌有效期(秒)" json:"accessTokenTTL"`
	RefreshTokenTTL         int64          `gorm:"column:refresh_token_ttl;type:bigint;not null;default 2592000;comment:刷新令牌有效期(秒)" json:"refreshTokenTTL"`
	Type                    string         `gorm:"column:type;type:varchar(32);not null;default 'first_party';comment:客户端类型" json:"type"`
	IsThirdParty            int8           `gorm:"column:is_third_party;type:tinyint(1);not null;default 0;comment:是否第三方应用" json:"isThirdParty"`
	Status                  string         `gorm:"column:status;type:varchar(32);not null;default 'enable';comment:状态" json:"status"`

	CreatedBy uint `gorm:"column:created_by;type:bigint unsigned;not null;default 0;comment:创建人id" json:"createdBy"`
	UpdatedBy uint `gorm:"column:updated_by;type:bigint unsigned;not null;default 0;comment:更新人id" json:"updatedBy"`
	DeletedBy uint `gorm:"column:deleted_by;type:bigint unsigned;not null;default 0;comment:删除人id" json:"deletedBy"`
}

func (OAuthClientEntity) TableName() string { return TableNameOAuthClient }

type OAuthClientEntityList []OAuthClientEntity

func (l OAuthClientEntityList) ToMap() map[uint]OAuthClientEntity {
	m := make(map[uint]OAuthClientEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
