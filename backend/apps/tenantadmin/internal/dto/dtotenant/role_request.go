package dtotenant

type RoleCreateReq struct {
	AppID       string `json:"appID" binding:"required"` // 所属应用ID（租户订阅的非系统应用）
	Name        string `json:"name" binding:"required"`  // 角色名称
	Code        string `json:"code" binding:"required"`  // 角色编码(应用内唯一)
	Description string `json:"description"`              // 角色描述
}

type RoleUpdateReq struct {
	RoleID      string `json:"-" uri:"roleID" binding:"required"` // 角色ID
	Name        string `json:"name" binding:"required"`           // 角色名称
	Code        string `json:"code" binding:"required"`           // 角色编码
	Description string `json:"description"`                       // 角色描述
}

type RoleDetailReq struct {
	RoleID string `json:"-" uri:"roleID" binding:"required"` // 角色ID
}

type RolePageListReq struct {
	Page     int    `json:"page" form:"page"`         // 页码
	PageSize int    `json:"pageSize" form:"pageSize"` // 每页数量
	AppID    string `json:"appID" form:"appID"`       // 应用过滤
	Keyword  string `json:"keyword" form:"keyword"`   // 关键词(名称/编码 模糊)
}

type RoleDeleteReq struct {
	RoleID string `json:"-" uri:"roleID" binding:"required"` // 角色ID
}

type RoleMenusUpdateReq struct {
	RoleID  string   `json:"-" uri:"roleID" binding:"required"` // 角色ID
	MenuIDs []string `json:"menuIDs" binding:"required"`        // 菜单ID列表(全量替换)
}
