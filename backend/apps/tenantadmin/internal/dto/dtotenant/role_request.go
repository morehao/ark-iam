package dtotenant

// RoleCreateReq 创建租户自建角色。
// **不含角色编码**：编码是跨系统授权契约值（OIDC groups 取值、下游策略名），策略在下游是全局
// 命名实体、全租户共用一条，故只能由应用方在应用角色模板里定义一次（见 docs/design/system-design.md §5.4）。
// 自建角色只承载本系统内的菜单权限，不参与跨系统契约，其 code 恒为空串。
type RoleCreateReq struct {
	AppID       string `json:"appID" binding:"required"` // 所属应用ID（租户订阅的启用应用，含系统内置应用）
	Name        string `json:"name" binding:"required"`  // 角色名称(应用内唯一)
	Description string `json:"description"`              // 角色描述
}

type RoleUpdateReq struct {
	RoleID      string `json:"-" uri:"roleID" binding:"required"` // 角色ID
	Name        string `json:"name" binding:"required"`           // 角色名称(应用内唯一)
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
	// Unassigned 仅查询未归属应用的角色（app_id 为空串，即平台内置的系统角色）；
	// false/缺省表示不按归属过滤。空串 appID 在查询里语义是"不过滤"，故需要这个显式开关。
	Unassigned bool `json:"unassigned" form:"unassigned"`
}

type RoleDeleteReq struct {
	RoleID string `json:"-" uri:"roleID" binding:"required"` // 角色ID
}

type RoleMenusUpdateReq struct {
	RoleID  string   `json:"-" uri:"roleID" binding:"required"` // 角色ID
	MenuIDs []string `json:"menuIDs" binding:"required"`        // 菜单ID列表(全量替换)
}
