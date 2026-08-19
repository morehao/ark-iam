package objpermission

type RoleBaseInfo struct {
	TenantID    string `json:"tenantID" form:"tenantID"`       // 租户ID
	Name        string `json:"name" form:"name"`               // 角色名称
	Code        string `json:"code" form:"code"`               // 角色编码
	Description string `json:"description" form:"description"` // 角色描述
	Source      string `json:"source" form:"source"`           // 角色来源(builtin/custom)
	AdminLevel  string `json:"adminLevel" form:"adminLevel"`   // 系统管理等级(none/basic/super)
}
