package model

import (
	"github.com/morehao/golib/dbaccess/gormdao"
)

const TableNameTenant = "tenant"

type TenantType string

const (
	TenantTypeCustomer TenantType = "customer"
	TenantTypePlatform TenantType = "platform"
)

type TenantEntity struct {
	gormdao.BaseEntity
	Code        string     `gorm:"column:code;type:varchar(64);not null;default:'';uniqueIndex;comment:租户编码"`
	CreatedBy   string     `gorm:"column:created_by;type:varchar(36);not null;default:'';comment:创建人id"`
	DbUser      string     `gorm:"column:db_user;type:varchar(64);not null;default:'';comment:数据库用户"`
	DeletedBy   string     `gorm:"column:deleted_by;type:varchar(36);not null;default:'';comment:删除人id"`
	IsSuspended int8       `gorm:"column:is_suspended;type:smallint;not null;default:0;comment:是否挂起"`
	Name        string     `gorm:"column:name;type:varchar(128);not null;default:'';comment:租户名称"`
	Type        TenantType `gorm:"column:type;type:varchar(32);not null;default:'customer';comment:租户类型: customer-客户租户, platform-平台租户"`
	Tag         string     `gorm:"column:tag;type:varchar(64);not null;default:'';comment:标签"`
	UpdatedBy   string     `gorm:"column:updated_by;type:varchar(36);not null;default:'';comment:更新人id"`
}

func (TenantEntity) TableName() string {
	return TableNameTenant
}

type TenantEntityList []TenantEntity

func (l TenantEntityList) ToMap() map[string]TenantEntity {
	m := make(map[string]TenantEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
