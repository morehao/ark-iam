package model

import (
	"time"

	"github.com/morehao/golib/dbaccess/gormdao"
)

const TableNameApplicationClientSecret = "application_client_secret"

type ApplicationClientSecretEntity struct {
	gormdao.BaseEntity
	ApplicationClientID string     `gorm:"column:application_client_id;type:varchar(36);not null;default:'';comment:客户端ID" json:"applicationClientID"`
	Name                string     `gorm:"column:name;type:varchar(256);not null;default:'';comment:密钥名称" json:"name"`
	ValueHash           string     `gorm:"column:value_hash;type:varchar(256);not null;default:'';comment:密钥哈希" json:"-"`
	ValuePrefix         string     `gorm:"column:value_prefix;type:varchar(16);not null;default:'';comment:密钥前缀" json:"valuePrefix"`
	ExpiredAt           *time.Time `gorm:"column:expired_at;comment:过期时间" json:"expiresAt"`
	RevokedAt           *time.Time `gorm:"column:revoked_at;comment:撤销时间" json:"-"`
	CreatedBy           string     `gorm:"column:created_by;type:varchar(36);not null;default:'';comment:创建人id" json:"createdBy"`
	UpdatedBy           string     `gorm:"column:updated_by;type:varchar(36);not null;default:'';comment:更新人id" json:"updatedBy"`
	DeletedBy           string     `gorm:"column:deleted_by;type:varchar(36);not null;default:'';comment:删除人id" json:"deletedBy"`
}

func (ApplicationClientSecretEntity) TableName() string { return TableNameApplicationClientSecret }

type ApplicationClientSecretEntityList []ApplicationClientSecretEntity

func (l ApplicationClientSecretEntityList) ToMap() map[string]ApplicationClientSecretEntity {
	m := make(map[string]ApplicationClientSecretEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
