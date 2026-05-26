package model

import (
	"time"

	"gorm.io/gorm"
)

const TableNameApplicationSecret = "application_secret"

type ApplicationSecretEntity struct {
	gorm.Model
	TenantID      uint  `gorm:"column:tenant_id;type:bigint unsigned;not null;default 0;comment:租户id" json:"tenantID"`
	ApplicationID uint  `gorm:"column:application_id;type:bigint unsigned;not null;default 0;comment:应用ID" json:"applicationID"`
	Name          string `gorm:"column:name;type:varchar(256);not null;default '';comment:密钥名称" json:"name"`
	Value         string `gorm:"column:value;type:varchar(64);not null;default '';comment:密钥值" json:"value"`
	ExpiredAt     *time.Time `gorm:"column:expired_at;type:datetime;comment:过期时间" json:"expiresAt"`
	CreatedBy     uint  `gorm:"column:created_by;type:bigint unsigned;not null;default 0;comment:创建人id" json:"createdBy"`
	UpdatedBy     uint  `gorm:"column:updated_by;type:bigint unsigned;not null;default 0;comment:更新人id" json:"updatedBy"`
	DeletedBy     uint  `gorm:"column:deleted_by;type:bigint unsigned;not null;default 0;comment:删除人id" json:"deletedBy"`
}

func (ApplicationSecretEntity) TableName() string {
	return TableNameApplicationSecret
}

type ApplicationSecretEntityList []ApplicationSecretEntity

func (l ApplicationSecretEntityList) ToMap() map[uint]ApplicationSecretEntity {
	m := make(map[uint]ApplicationSecretEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}