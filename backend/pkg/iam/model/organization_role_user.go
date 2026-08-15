package model

import (
	"github.com/morehao/golib/dbaccess/gormdao"
)

const TableNameOrganizationRoleUser = "organization_role_user"

type OrganizationRoleUserEntity struct {
	gormdao.BaseEntity
	TenantID           string `gorm:"column:tenant_id;type:varchar(36);not null;default:'';comment:租户id" json:"tenantID"`
	OrganizationID     string `gorm:"column:organization_id;type:varchar(36);not null;default:'';comment:组织ID" json:"organizationID"`
	OrganizationRoleID string `gorm:"column:organization_role_id;type:varchar(36);not null;default:'';comment:组织角色ID" json:"organizationRoleID"`
	UserID             string `gorm:"column:user_id;type:varchar(36);not null;default:'';comment:用户ID" json:"userID"`
	CreatedBy          string `gorm:"column:created_by;type:varchar(36);not null;default:'';comment:创建人id" json:"createdBy"`
	UpdatedBy          string `gorm:"column:updated_by;type:varchar(36);not null;default:'';comment:更新人id" json:"updatedBy"`
	DeletedBy          string `gorm:"column:deleted_by;type:varchar(36);not null;default:'';comment:删除人id" json:"deletedBy"`
}

func (OrganizationRoleUserEntity) TableName() string {
	return TableNameOrganizationRoleUser
}

type OrganizationRoleUserEntityList []OrganizationRoleUserEntity

func (l OrganizationRoleUserEntityList) ToMap() map[string]OrganizationRoleUserEntity {
	m := make(map[string]OrganizationRoleUserEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
