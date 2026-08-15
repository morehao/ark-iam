package model

import (
	"github.com/morehao/golib/dbaccess/gormdao"
)

const TableNameScope = "scope"

type ScopeEntity struct {
	gormdao.BaseEntity
	TenantID    string `gorm:"column:tenant_id;type:varchar(36);not null;default:'';comment:租户id" json:"tenantID"`
	ResourceID  string `gorm:"column:resource_id;type:varchar(36);not null;default:'';comment:资源ID" json:"resourceID"`
	Name        string `gorm:"column:name;type:varchar(256);not null;default:'';comment:权限名称" json:"name"`
	Description string `gorm:"column:description;type:text;comment:权限描述" json:"description"`
	CreatedBy   string `gorm:"column:created_by;type:varchar(36);not null;default:'';comment:创建人id" json:"createdBy"`
	UpdatedBy   string `gorm:"column:updated_by;type:varchar(36);not null;default:'';comment:更新人id" json:"updatedBy"`
	DeletedBy   string `gorm:"column:deleted_by;type:varchar(36);not null;default:'';comment:删除人id" json:"deletedBy"`
}

func (ScopeEntity) TableName() string {
	return TableNameScope
}

type ScopeEntityList []ScopeEntity

func (l ScopeEntityList) ToMap() map[string]ScopeEntity {
	m := make(map[string]ScopeEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
