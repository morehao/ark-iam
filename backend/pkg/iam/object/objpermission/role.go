package objpermission

type RoleBaseInfo struct {
	TenantID    string `json:"tenantID" form:"tenantID"`       // 租户ID
	Name        string `json:"name" form:"name"`               // 角色名称
	Code        string `json:"code" form:"code"`               // 角色编码
	Description string `json:"description" form:"description"` // 角色描述
	Type        string `json:"type" form:"type"`               // 角色类型
	IsDefault   int8   `json:"isDefault" form:"isDefault"`     // 是否默认角色
}
