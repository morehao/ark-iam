package model

import (
	"time"

	"gorm.io/gorm"
)

const TableNameOAuthClientSecret = "oauth_client_secret"

type OAuthClientSecretEntity struct {
	gorm.Model
	OAuthClientID uint       `gorm:"column:oauth_client_id;type:bigint unsigned;not null;default 0;comment:客户端ID" json:"oauthClientID"`
	Name          string     `gorm:"column:name;type:varchar(256);not null;default '';comment:密钥名称" json:"name"`
	ValueHash     string     `gorm:"column:value_hash;type:varchar(256);not null;default '';comment:密钥哈希" json:"-"`
	ValuePrefix   string     `gorm:"column:value_prefix;type:varchar(16);not null;default '';comment:密钥前缀" json:"valuePrefix"`
	ExpiredAt     *time.Time `gorm:"column:expired_at;type:datetime;comment:过期时间" json:"expiresAt"`
	RevokedAt     *time.Time `gorm:"column:revoked_at;type:datetime;comment:撤销时间" json:"-"`
	CreatedBy     uint       `gorm:"column:created_by;type:bigint unsigned;not null;default 0;comment:创建人id" json:"createdBy"`
	UpdatedBy     uint       `gorm:"column:updated_by;type:bigint unsigned;not null;default 0;comment:更新人id" json:"updatedBy"`
	DeletedBy     uint       `gorm:"column:deleted_by;type:bigint unsigned;not null;default 0;comment:删除人id" json:"deletedBy"`
}

func (OAuthClientSecretEntity) TableName() string { return TableNameOAuthClientSecret }

type OAuthClientSecretEntityList []OAuthClientSecretEntity

func (l OAuthClientSecretEntityList) ToMap() map[uint]OAuthClientSecretEntity {
	m := make(map[uint]OAuthClientSecretEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
