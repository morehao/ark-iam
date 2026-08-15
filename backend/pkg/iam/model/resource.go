package model

import (
	"github.com/morehao/golib/dbaccess/gormdao"
)

const TableNameResource = "resource"

type ResourceEntity struct {
	gormdao.BaseEntity
	TenantID       string `gorm:"column:tenant_id;type:varchar(36);not null;default:'';comment:租户id" json:"tenantID"`
	Name           string `gorm:"column:name;type:varchar(256);not null;default:'';comment:资源名称" json:"name"`
	Indicator      string `gorm:"column:indicator;type:varchar(512);not null;default:'';comment:资源标识符" json:"indicator"`
	IsDefault      int8   `gorm:"column:is_default;type:smallint;not null;default:0;comment:是否默认" json:"isDefault"`
	AccessTokenTtl int64  `gorm:"column:access_token_ttl;type:bigint;not null;default:3600;comment:访问令牌TTL" json:"accessTokenTtl"`
	CreatedBy      string `gorm:"column:created_by;type:varchar(36);not null;default:'';comment:创建人id" json:"createdBy"`
	UpdatedBy      string `gorm:"column:updated_by;type:varchar(36);not null;default:'';comment:更新人id" json:"updatedBy"`
	DeletedBy      string `gorm:"column:deleted_by;type:varchar(36);not null;default:'';comment:删除人id" json:"deletedBy"`
}

func (ResourceEntity) TableName() string {
	return TableNameResource
}

type ResourceEntityList []ResourceEntity

func (l ResourceEntityList) ToMap() map[string]ResourceEntity {
	m := make(map[string]ResourceEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
