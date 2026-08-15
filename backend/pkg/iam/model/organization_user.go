package model

import (
	"github.com/morehao/golib/dbaccess/gormdao"
)

const TableNameOrganizationUser = "organization_user"

// OrgUserRelationType 用户↔组织节点关系类型（字符串枚举，全系统约束：枚举一律字符串）。
// 枚举值互斥纯净：member（归属）与 leader（负责）是不同的关系种类；
// admin 等其他关系种类由业务按需扩展常量，无需改表。
type OrgUserRelationType string

const (
	OrgUserRelationMember OrgUserRelationType = "member" // 归属（成员）
	OrgUserRelationLeader OrgUserRelationType = "leader" // 负责人（独立于归属，不要求同时是成员）
)

// OrganizationUserEntity 组织关系表：用户与组织节点之间的多态关系。
// 主归属为 member 关系的属性（is_primary，仅 member 可置位，租户内每用户至多 1 个）。
type OrganizationUserEntity struct {
	gormdao.BaseEntity
	TenantID       string `gorm:"column:tenant_id;type:varchar(36);not null;default:'';comment:租户id" json:"tenantID"`
	OrganizationID string `gorm:"column:organization_id;type:varchar(36);not null;default:'';comment:组织节点ID" json:"organizationID"`
	UserID         string `gorm:"column:user_id;type:varchar(36);not null;default:'';comment:用户ID" json:"userID"`
	RelationType   string `gorm:"column:relation_type;type:varchar(32);not null;default:'member';comment:关系类型(字符串枚举)" json:"relationType"`
	IsPrimary      bool   `gorm:"column:is_primary;type:boolean;not null;default:false;comment:是否主归属(仅member关系可置位)" json:"isPrimary"`
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
