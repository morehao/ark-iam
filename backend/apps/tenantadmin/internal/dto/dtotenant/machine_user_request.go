package dtotenant

// 服务账号（租户内机器主体，user_type=machine）请求 DTO。

type MachineUserPageListReq struct {
	Page        int    `json:"page" form:"page"`               // 页码
	PageSize    int    `json:"pageSize" form:"pageSize"`       // 每页数量
	Name        string `json:"name" form:"name"`               // 名称(模糊)
	IsSuspended *bool  `json:"isSuspended" form:"isSuspended"` // 状态过滤(挂起)
}

type MachineUserCreateReq struct {
	Name        string `json:"name" binding:"required"` // 服务账号名称
	Description string `json:"description"`             // 描述
}

type MachineUserUpdateReq struct {
	MachineUserID string `json:"-" uri:"machineUserID" binding:"required"` // 服务账号ID
	Name          string `json:"name" binding:"required"`                  // 服务账号名称
	Description   string `json:"description"`                              // 描述
}

type MachineUserStatusReq struct {
	MachineUserID string `json:"-" uri:"machineUserID" binding:"required"` // 服务账号ID
	IsSuspended   bool   `json:"isSuspended"`                              // true=挂起 false=启用
}

type MachineUserDetailReq struct {
	MachineUserID string `json:"-" uri:"machineUserID" binding:"required"` // 服务账号ID
}

type MachineUserDeleteReq struct {
	MachineUserID string `json:"-" uri:"machineUserID" binding:"required"` // 服务账号ID
}

type MachineUserRolesListReq struct {
	MachineUserID string `json:"-" uri:"machineUserID" binding:"required"` // 服务账号ID
}

type MachineUserRolesUpdateReq struct {
	MachineUserID string   `json:"-" uri:"machineUserID" binding:"required"` // 服务账号ID
	RoleIDs       []string `json:"roleIDs" binding:"required"`               // 角色ID列表(全量替换)
}
