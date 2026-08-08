package model

import (
	"time"

	"gorm.io/gorm"
)

const TableNameApplicationClientSecret = "application_client_secret"

type ApplicationClientSecretEntity struct {
	gorm.Model
	ApplicationClientID uint       `gorm:"column:application_client_id;type:bigint unsigned;not null;default 0;comment:客户端ID" json:"applicationClientID"`
	Name          string     `gorm:"column:name;type:varchar(256);not null;default '';comment:密钥名称" json:"name"`
	ValueHash     string     `gorm:"column:value_hash;type:varchar(256);not null;default '';comment:密钥哈希" json:"-"`
	ValuePrefix   string     `gorm:"column:value_prefix;type:varchar(16);not null;default '';comment:密钥前缀" json:"valuePrefix"`
	ExpiredAt     *time.Time `gorm:"column:expired_at;type:datetime;comment:过期时间" json:"expiresAt"`
	RevokedAt     *time.Time `gorm:"column:revoked_at;type:datetime;comment:撤销时间" json:"-"`
	CreatedBy     uint       `gorm:"column:created_by;type:bigint unsigned;not null;default 0;comment:创建人id" json:"createdBy"`
	UpdatedBy     uint       `gorm:"column:updated_by;type:bigint unsigned;not null;default 0;comment:更新人id" json:"updatedBy"`
	DeletedBy     uint       `gorm:"column:deleted_by;type:bigint unsigned;not null;default 0;comment:删除人id" json:"deletedBy"`
}

func (ApplicationClientSecretEntity) TableName() string { return TableNameApplicationClientSecret }

type ApplicationClientSecretEntityList []ApplicationClientSecretEntity

func (l ApplicationClientSecretEntityList) ToMap() map[uint]ApplicationClientSecretEntity {
	m := make(map[uint]ApplicationClientSecretEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
