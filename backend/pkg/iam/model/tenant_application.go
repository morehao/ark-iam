package model

import (
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/datatypes"
)

const TableNameTenantApplication = "tenant_application"

type TenantApplicationEntity struct {
	gormdao.BaseEntity
	TenantID     string         `gorm:"column:tenant_id;type:varchar(36);not null;default:'';comment:租户id" json:"tenantID"`
	AppID        string         `gorm:"column:app_id;type:varchar(36);not null;default:'';comment:应用id" json:"appID"`
	Status       string         `gorm:"column:status;type:varchar(32);not null;default:'enable';comment:状态" json:"status"`
	Config       datatypes.JSON `gorm:"column:config;type:json;not null;default:('{}');comment:租户级应用配置" json:"config"`
	GrantedScope datatypes.JSON `gorm:"column:granted_scope;type:json;not null;default:('[]');comment:租户级scope授权" json:"grantedScope"`
	CreatedBy    string         `gorm:"column:created_by;type:varchar(36);not null;default:'';comment:创建人id" json:"createdBy"`
	UpdatedBy    string         `gorm:"column:updated_by;type:varchar(36);not null;default:'';comment:更新人id" json:"updatedBy"`
	DeletedBy    string         `gorm:"column:deleted_by;type:varchar(36);not null;default:'';comment:删除人id" json:"deletedBy"`
}

func (TenantApplicationEntity) TableName() string { return TableNameTenantApplication }

type TenantApplicationEntityList []TenantApplicationEntity

func (l TenantApplicationEntityList) ToMap() map[string]TenantApplicationEntity {
	m := make(map[string]TenantApplicationEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
