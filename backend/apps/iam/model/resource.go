package model

import (
	"gorm.io/gorm"
)

const TableNameResource = "resource"

type ResourceEntity struct {
	gorm.Model
	TenantID       uint   `gorm:"column:tenant_id;type:bigint unsigned;not null;default 0;comment:租户id" json:"tenantID"`
	Name           string `gorm:"column:name;type:varchar(256);not null;default '';comment:资源名称" json:"name"`
	Indicator      string `gorm:"column:indicator;type:varchar(512);not null;default '';comment:资源标识符" json:"indicator"`
	IsDefault      int8   `gorm:"column:is_default;type:tinyint(1);not null;default 0;comment:是否默认" json:"isDefault"`
	AccessTokenTtl int64  `gorm:"column:access_token_ttl;type:bigint;not null;default 3600;comment:访问令牌TTL" json:"accessTokenTtl"`
	CreatedBy      uint   `gorm:"column:created_by;type:bigint unsigned;not null;default 0;comment:创建人id" json:"createdBy"`
	UpdatedBy      uint   `gorm:"column:updated_by;type:bigint unsigned;not null;default 0;comment:更新人id" json:"updatedBy"`
	DeletedBy      uint   `gorm:"column:deleted_by;type:bigint unsigned;not null;default 0;comment:删除人id" json:"deletedBy"`
}

func (ResourceEntity) TableName() string {
	return TableNameResource
}

type ResourceEntityList []ResourceEntity

func (l ResourceEntityList) ToMap() map[uint]ResourceEntity {
	m := make(map[uint]ResourceEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}