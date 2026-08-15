package model

import (
	"github.com/morehao/golib/dbaccess/gormdao"
)

const TableNameOrganization = "organization"

// 组织节点状态枚举（字符串，全系统约束：枚举一律字符串）
type OrgNodeStatus string

const (
	OrgNodeStatusActive   OrgNodeStatus = "active"   // 启用
	OrgNodeStatusInactive OrgNodeStatus = "inactive" // 停用
)

// MaxOrgDepth 组织树深度上限，防病态树。
const MaxOrgDepth = 10

// OrganizationEntity 组织节点树：租户下用户的容器。
// IAM 只维护结构机制（树形层级 + 物化路径），节点是部门/项目组/班级由业务侧解释。
type OrganizationEntity struct {
	gormdao.BaseEntity
	TenantID string `gorm:"column:tenant_id;type:varchar(36);not null;default:'';comment:租户id" json:"tenantID"`
	ParentID string `gorm:"column:parent_id;type:varchar(36);not null;default:'';comment:父节点ID,空为根节点" json:"parentID"`
	OrgPath  string `gorm:"column:org_path;type:varchar(1024);not null;default:'';comment:祖先链路径,含自身,如 /rootID/midID/nodeID" json:"orgPath"`
	OrgDepth int    `gorm:"column:org_depth;type:int;not null;default:1;comment:节点深度,根=1" json:"orgDepth"`
	Name     string `gorm:"column:name;type:varchar(128);not null;default:'';comment:组织名称" json:"name"`
	Code     string `gorm:"column:code;type:varchar(64);not null;default:'';comment:组织编码(租户内唯一,可空,外部系统同步用)" json:"code"`
	Sort     int    `gorm:"column:sort;type:int;not null;default:0;comment:同级排序" json:"sort"`
	Status   string `gorm:"column:status;type:varchar(32);not null;default:'active';comment:状态(字符串枚举)" json:"status"`
	CreatedBy string `gorm:"column:created_by;type:varchar(36);not null;default:'';comment:创建人id" json:"createdBy"`
	UpdatedBy string `gorm:"column:updated_by;type:varchar(36);not null;default:'';comment:更新人id" json:"updatedBy"`
	DeletedBy string `gorm:"column:deleted_by;type:varchar(36);not null;default:'';comment:删除人id" json:"deletedBy"`
}

func (OrganizationEntity) TableName() string {
	return TableNameOrganization
}

type OrganizationEntityList []OrganizationEntity

func (l OrganizationEntityList) ToMap() map[string]OrganizationEntity {
	m := make(map[string]OrganizationEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
