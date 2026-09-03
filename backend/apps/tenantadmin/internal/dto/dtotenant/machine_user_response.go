package dtotenant

// 服务账号（租户内机器主体，user_type=machine）响应 DTO。

type MachineUserPageListItem struct {
	MachineUserID  string `json:"machineUserID"`  // 服务账号ID
	TenantID       string `json:"tenantID"`       // 租户ID
	Name           string `json:"name"`           // 名称
	Description    string `json:"description"`    // 描述
	PrimaryOrgID   string `json:"primaryOrgID"`   // 主部门ID
	PrimaryOrgName string `json:"primaryOrgName"` // 主部门名称
	IsSuspended    bool   `json:"isSuspended"`    // 是否挂起
	CreatedAt      int64  `json:"createdAt"`      // 创建时间
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
	Organizations []UserOrganizationItem `json:"organizations"` // 组织归属(primary/secondary)
	Roles         []UserRoleItem         `json:"roles"`         // 已分配角色
}
