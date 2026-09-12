package model

import (
	"github.com/morehao/golib/dbaccess/gormdao"
)

const TableNameDepartment = "department"

// DeptNodeStatus 部门节点启停状态。
// 取值属全局启停语义，与其它领域统一为 enable/disable
// （见 status-source-consistency-design-20260912.md D1）。
type DeptNodeStatus string

// 部门节点状态取值（禁止硬编码）。
const (
	DeptNodeStatusEnable  DeptNodeStatus = "enable"  // 启用
	DeptNodeStatusDisable DeptNodeStatus = "disable" // 停用
)

// MaxDeptDepth 部门树深度上限，防病态树。
const MaxDeptDepth = 10

// DepartmentEntity 部门树节点：租户下用户归属的容器（可嵌套，物化路径维护祖先链）。
type DepartmentEntity struct {
	gormdao.BaseEntity
	TenantID  string         `gorm:"column:tenant_id;type:varchar(36);not null;default:'';index:idx_department_parent,priority:1;comment:租户id" json:"tenantID"`
	ParentID  string         `gorm:"column:parent_id;type:varchar(36);not null;default:'';index:idx_department_parent,priority:2;comment:父节点ID,空为根节点" json:"parentID"`
	DeptPath  string         `gorm:"column:dept_path;type:varchar(1024);not null;default:'';index:idx_department_path;comment:祖先链路径,含自身,如 /rootID/midID/nodeID" json:"deptPath"`
	DeptDepth int            `gorm:"column:dept_depth;type:int;not null;default:1;comment:节点深度,根=1" json:"deptDepth"`
	Name      string         `gorm:"column:name;type:varchar(128);not null;default:'';comment:部门名称" json:"name"`
	Sort      int            `gorm:"column:sort;type:int;not null;default:0;comment:同级排序" json:"sort"`
	Status    DeptNodeStatus `gorm:"column:status;type:varchar(32);not null;default:'enable';comment:状态(字符串枚举)" json:"status"`
	CreatedBy string         `gorm:"column:created_by;type:varchar(36);not null;default:'';comment:创建人id" json:"createdBy"`
	UpdatedBy string         `gorm:"column:updated_by;type:varchar(36);not null;default:'';comment:更新人id" json:"updatedBy"`
	DeletedBy string         `gorm:"column:deleted_by;type:varchar(36);not null;default:'';comment:删除人id" json:"deletedBy"`
}

func (DepartmentEntity) TableName() string {
	return TableNameDepartment
}

type DepartmentEntityList []DepartmentEntity

func (l DepartmentEntityList) ToMap() map[string]DepartmentEntity {
	m := make(map[string]DepartmentEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
