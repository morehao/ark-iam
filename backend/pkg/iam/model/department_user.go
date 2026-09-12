package model

import (
	"github.com/morehao/golib/dbaccess/gormdao"
)

const TableNameDepartmentUser = "department_user"

// DeptUserRelationType 用户↔部门节点关系类型（强类型字符串枚举）。
// 枚举值互斥纯净：primary（行政主部门）与 secondary（跨部门协作参与）、leader（负责人）是不同的关系种类；
// admin 等其他关系种类由业务按需扩展常量，无需改表。
//
// 基数约束：primary 为行政主部门，每用户至多 1 行；secondary/leader 可多条。
// 全链路规范：实体/DAO Cond/DTO 字段一律用本类型，禁止 string(...) 强转与裸字面量。
type DeptUserRelationType string

const (
	DeptUserRelationPrimary   DeptUserRelationType = "primary"   // 归属：行政主部门，每人全局唯一
	DeptUserRelationSecondary DeptUserRelationType = "secondary" // 参与：跨部门协作，可多条
	DeptUserRelationLeader    DeptUserRelationType = "leader"    // 负责：部门负责人身份，可多条
)

// DepartmentUserEntity 部门关系表：用户与部门节点之间的多态关系。
type DepartmentUserEntity struct {
	gormdao.BaseEntity
	TenantID     string               `gorm:"column:tenant_id;type:varchar(36);not null;default:'';comment:租户id" json:"tenantID"`
	DepartmentID string               `gorm:"column:department_id;type:varchar(36);not null;default:'';comment:部门节点ID" json:"departmentID"`
	UserID       string               `gorm:"column:user_id;type:varchar(36);not null;default:'';comment:用户ID" json:"userID"`
	RelationType DeptUserRelationType `gorm:"column:relation_type;type:varchar(32);not null;default:'primary';comment:关系类型(字符串枚举)" json:"relationType"`
	CreatedBy    string               `gorm:"column:created_by;type:varchar(36);not null;default:'';comment:创建人id" json:"createdBy"`
	UpdatedBy    string               `gorm:"column:updated_by;type:varchar(36);not null;default:'';comment:更新人id" json:"updatedBy"`
	DeletedBy    string               `gorm:"column:deleted_by;type:varchar(36);not null;default:'';comment:删除人id" json:"deletedBy"`
}

func (DepartmentUserEntity) TableName() string {
	return TableNameDepartmentUser
}

type DepartmentUserEntityList []DepartmentUserEntity

func (l DepartmentUserEntityList) ToMap() map[string]DepartmentUserEntity {
	m := make(map[string]DepartmentUserEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
