package dtotenant

import (
	"github.com/morehao/ark-iam/pkg/model"
)

// 服务账号（租户内机器主体，user_type=machine）响应 DTO。

type MachineUserPageListItem struct {
	MachineUserID         string           `json:"machineUserID"`         // 服务账号ID
	TenantID              string           `json:"tenantID"`              // 租户ID
	Name                  string           `json:"name"`                  // 名称
	Description           string           `json:"description"`           // 描述
	PrimaryDepartmentID   string           `json:"primaryDepartmentID"`   // 主部门ID
	PrimaryDepartmentName string           `json:"primaryDepartmentName"` // 主部门名称
	Status                model.UserStatus `json:"status"`                // 状态(active正常/suspended挂起)
	CreatedAt             int64            `json:"createdAt"`             // 创建时间
	UpdatedAt             int64            `json:"updatedAt"`             // 更新时间
}

type MachineUserPageListResp struct {
	List  []MachineUserPageListItem `json:"list"`  // 数据列表
	Total int64                     `json:"total"` // 数据总条数
}

type MachineUserCreateResp struct {
	MachineUserID string `json:"machineUserID"` // 服务账号ID
}

type MachineUserDetailResp struct {
	MachineUserPageListItem
	Departments []UserDepartmentItem `json:"departments"` // 部门归属(primary/secondary)
	Roles       []UserRoleItem       `json:"roles"`       // 已分配角色
}
