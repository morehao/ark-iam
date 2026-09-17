package dtotenant

import "github.com/morehao/ark-iam/pkg/model"

type RoleCreateReq struct {
	AppID       string         `json:"appID" binding:"required"` // 所属应用ID（租户订阅的启用应用，含系统内置应用）
	Code        model.RoleCode `json:"code" binding:"required"`  // 角色编码(租户内唯一)：跨系统授权契约值，即 OIDC groups 取值
	Name        string         `json:"name" binding:"required"`  // 角色名称(应用内唯一)
	Description string         `json:"description"`              // 角色描述
}

type RoleUpdateReq struct {
	RoleID      string         `json:"-" uri:"roleID" binding:"required"` // 角色ID
	Code        model.RoleCode `json:"code" binding:"required"`           // 角色编码(租户内唯一)：改动会改变下游系统按编码授予的权限
	Name        string         `json:"name" binding:"required"`           // 角色名称(应用内唯一)
	Description string         `json:"description"`                       // 角色描述
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
