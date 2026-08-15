package model

import (
	"github.com/morehao/golib/dbaccess/gormdao"
)

const TableNameOrganizationUser = "organization_user"

type OrganizationUserEntity struct {
	gormdao.BaseEntity
	TenantID       string `gorm:"column:tenant_id;type:varchar(36);not null;default:'';comment:租户id" json:"tenantID"`
	OrganizationID string `gorm:"column:organization_id;type:varchar(36);not null;default:'';comment:组织ID" json:"organizationID"`
	UserID         string `gorm:"column:user_id;type:varchar(36);not null;default:'';comment:用户ID" json:"userID"`
	CreatedBy      string `gorm:"column:created_by;type:varchar(36);not null;default:'';comment:创建人id" json:"createdBy"`
	UpdatedBy      string `gorm:"column:updated_by;type:varchar(36);not null;default:'';comment:更新人id" json:"updatedBy"`
	DeletedBy      string `gorm:"column:deleted_by;type:varchar(36);not null;default:'';comment:删除人id" json:"deletedBy"`
}

func (OrganizationUserEntity) TableName() string {
	return TableNameOrganizationUser
}

type OrganizationUserEntityList []OrganizationUserEntity

func (l OrganizationUserEntityList) ToMap() map[string]OrganizationUserEntity {
	m := make(map[string]OrganizationUserEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
