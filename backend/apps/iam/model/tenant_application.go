package model

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const TableNameTenantApplication = "tenant_application"

type TenantApplicationEntity struct {
	gorm.Model
	TenantID      uint           `gorm:"column:tenant_id;type:bigint unsigned;not null;default 0;comment:租户id" json:"tenantID"`
	AppID uint           `gorm:"column:app_id;type:bigint unsigned;not null;default 0;comment:应用id" json:"appId"`
	Status        string         `gorm:"column:status;type:varchar(32);not null;default 'enable';comment:状态" json:"status"`
	Config        datatypes.JSON `gorm:"column:config;type:json;not null;default ('{}');comment:租户级应用配置" json:"config"`
	GrantedScope  datatypes.JSON `gorm:"column:granted_scope;type:json;not null;default ('[]');comment:租户级scope授权" json:"grantedScope"`
	CreatedBy     uint           `gorm:"column:created_by;type:bigint unsigned;not null;default 0;comment:创建人id" json:"createdBy"`
	UpdatedBy     uint           `gorm:"column:updated_by;type:bigint unsigned;not null;default 0;comment:更新人id" json:"updatedBy"`
	DeletedBy     uint           `gorm:"column:deleted_by;type:bigint unsigned;not null;default 0;comment:删除人id" json:"deletedBy"`
}

func (TenantApplicationEntity) TableName() string { return TableNameTenantApplication }

type TenantApplicationEntityList []TenantApplicationEntity

func (l TenantApplicationEntityList) ToMap() map[uint]TenantApplicationEntity {
	m := make(map[uint]TenantApplicationEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
