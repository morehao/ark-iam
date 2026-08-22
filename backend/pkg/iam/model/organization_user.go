package model

import (
	"github.com/morehao/golib/dbaccess/gormdao"
)

const TableNameOrganizationUser = "organization_user"

// OrgUserRelationType 用户↔组织节点关系类型（强类型字符串枚举）。
// 枚举值互斥纯净：primary（行政主部门）与 secondary（跨部门协作参与）、leader（负责人）是不同的关系种类；
// admin 等其他关系种类由业务按需扩展常量，无需改表。
//
// 基数约束：primary 为行政主部门，每用户至多 1 行；secondary/leader 可多条。
// 全链路规范：实体/DAO Cond/DTO 字段一律用本类型，禁止 string(...) 强转与裸字面量。
type OrgUserRelationType string

const (
	OrgUserRelationPrimary   OrgUserRelationType = "primary"   // 归属：行政主部门，每人全局唯一
	OrgUserRelationSecondary OrgUserRelationType = "secondary" // 参与：跨部门协作，可多条
	OrgUserRelationLeader    OrgUserRelationType = "leader"    // 负责：部门负责人身份，可多条
)

// OrganizationUserEntity 组织关系表：用户与组织节点之间的多态关系。
type OrganizationUserEntity struct {
	gormdao.BaseEntity
	TenantID       string              `gorm:"column:tenant_id;type:varchar(36);not null;default:'';comment:租户id" json:"tenantID"`
	OrganizationID string              `gorm:"column:organization_id;type:varchar(36);not null;default:'';comment:组织节点ID" json:"organizationID"`
	UserID         string              `gorm:"column:user_id;type:varchar(36);not null;default:'';comment:用户ID" json:"userID"`
	RelationType   OrgUserRelationType `gorm:"column:relation_type;type:varchar(32);not null;default:'primary';comment:关系类型(字符串枚举)" json:"relationType"`
	CreatedBy      string              `gorm:"column:created_by;type:varchar(36);not null;default:'';comment:创建人id" json:"createdBy"`
	UpdatedBy      string              `gorm:"column:updated_by;type:varchar(36);not null;default:'';comment:更新人id" json:"updatedBy"`
	DeletedBy      string              `gorm:"column:deleted_by;type:varchar(36);not null;default:'';comment:删除人id" json:"deletedBy"`
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
