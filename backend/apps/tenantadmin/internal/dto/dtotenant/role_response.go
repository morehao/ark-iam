package dtotenant

type RoleCreateResp struct {
	RoleID string `json:"roleID"` // 角色ID
}

type RolePageListResp struct {
	List  []RolePageListItem `json:"list"`  // 角色列表
	Total int64              `json:"total"` // 总条数
}

type RolePageListItem struct {
	RoleID      string `json:"roleID"`      // 角色ID
	AppID       string `json:"appID"`       // 所属应用ID
	AppName     string `json:"appName"`     // 所属应用名称
	Name        string `json:"name"`        // 角色名称
	Code        string `json:"code"`        // 角色编码
	Description string `json:"description"` // 角色描述
	Type        string `json:"type"`        // 角色类型
	IsDefault   bool   `json:"isDefault"`   // 是否默认角色
	MemberCount int64  `json:"memberCount"` // 成员数
	MenuCount   int64  `json:"menuCount"`   // 授权菜单数
	CreatedAt   int64  `json:"createdAt"`   // 创建时间
}

type RoleDetailResp struct {
	RoleID      string `json:"roleID"`      // 角色ID
	AppID       string `json:"appID"`       // 所属应用ID
	AppName     string `json:"appName"`     // 所属应用名称
	Name        string `json:"name"`        // 角色名称
	Code        string `json:"code"`        // 角色编码
	Description string `json:"description"` // 角色描述
	Type        string `json:"type"`        // 角色类型
	IsDefault   bool   `json:"isDefault"`   // 是否默认角色
	MemberCount int64  `json:"memberCount"` // 成员数
	MenuCount   int64  `json:"menuCount"`   // 授权菜单数
	CreatedAt   int64  `json:"createdAt"`   // 创建时间
	UpdatedAt   int64  `json:"updatedAt"`   // 更新时间
}

// RoleMenuTreeResp 角色菜单授权回显：租户控制台完整菜单树 + 已授权菜单ID。
type RoleMenuTreeResp struct {
	List    []MenuTreeItem `json:"list"`    // 租户控制台菜单树
	MenuIDs []string       `json:"menuIDs"` // 已授权菜单ID
}
