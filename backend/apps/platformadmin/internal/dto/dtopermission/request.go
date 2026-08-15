package dtopermission

import (
	"github.com/morehao/ark-iam/pkg/iam/object/objpermission"
	"github.com/morehao/golib/biz/gobject"
)

type MenuCreateReq struct {
	objpermission.MenuBaseInfo
}

type MenuUpdateReq struct {
	MenuID string `json:"-" uri:"menuID" binding:"required"` // 菜单ID
	objpermission.MenuBaseInfo
}

type MenuDetailReq struct {
	MenuID string `json:"-" uri:"menuID" binding:"required"` // 菜单ID
}

type MenuPageListReq struct {
	gobject.PageQuery
	AppID    string `json:"appID" form:"appID"`       // 应用ID
	ParentID string `json:"parentID" form:"parentID"` // 父菜单ID
	Name     string `json:"name" form:"name"`         // 菜单名称
	Code     string `json:"code" form:"code"`         // 菜单编码
	Type     string `json:"type" form:"type"`         // 菜单类型
	Status   string `json:"status" form:"status"`     // 状态
}

type MenuDeleteReq struct {
	MenuID string `json:"-" uri:"menuID" binding:"required"` // 菜单ID
}

type MenuTreeReq struct {
	AppID string `json:"appID" form:"appID" binding:"required"` // 应用ID（菜单树按应用维度查询）
}

type RoleCreateReq struct {
	TenantID    string `json:"tenantID" form:"tenantID"`       // 租户ID
	Name        string `json:"name" form:"name"`               // 角色名称
	Code        string `json:"code" form:"code"`               // 角色编码
	Description string `json:"description" form:"description"` // 角色描述
	Type        string `json:"type" form:"type"`               // 角色类型
	IsDefault   int8   `json:"isDefault" form:"isDefault"`     // 是否默认角色
}

type RoleUpdateReq struct {
	RoleID      string `json:"-" uri:"roleID"`                 // 角色ID
	TenantID    string `json:"tenantID" form:"tenantID"`       // 租户ID
	Name        string `json:"name" form:"name"`               // 角色名称
	Code        string `json:"code" form:"code"`               // 角色编码
	Description string `json:"description" form:"description"` // 角色描述
	Type        string `json:"type" form:"type"`               // 角色类型
	IsDefault   int8   `json:"isDefault" form:"isDefault"`     // 是否默认角色
}

type RoleDeleteReq struct {
	RoleID string `json:"-" uri:"roleID"` // 角色ID
}

type RoleDetailReq struct {
	RoleID string `json:"-" uri:"roleID"` // 角色ID
}

type RolePageListReq struct {
	Page     int    `json:"page" form:"page"`         // 页码
	PageSize int    `json:"pageSize" form:"pageSize"` // 每页数量
	TenantID string `json:"tenantID" form:"tenantID"` // 租户ID
	Name     string `json:"name" form:"name"`         // 角色名称
	Code     string `json:"code" form:"code"`         // 角色编码
	Type     string `json:"type" form:"type"`         // 角色类型
}

type ResourceCreateReq struct {
	TenantID       string `json:"tenantID" form:"tenantID"`             // 租户ID
	Name           string `json:"name" form:"name"`                     // 资源名称
	Indicator      string `json:"indicator" form:"indicator"`           // 资源标识符
	IsDefault      int8   `json:"isDefault" form:"isDefault"`           // 是否默认
	AccessTokenTtl int64  `json:"accessTokenTtl" form:"accessTokenTtl"` // 访问令牌TTL
}

type ResourceUpdateReq struct {
	ResourceID     string `json:"-" uri:"resourceID"`                   // 资源ID
	TenantID       string `json:"tenantID" form:"tenantID"`             // 租户ID
	Name           string `json:"name" form:"name"`                     // 资源名称
	Indicator      string `json:"indicator" form:"indicator"`           // 资源标识符
	IsDefault      int8   `json:"isDefault" form:"isDefault"`           // 是否默认
	AccessTokenTtl int64  `json:"accessTokenTtl" form:"accessTokenTtl"` // 访问令牌TTL
}

type ResourceDeleteReq struct {
	ResourceID string `json:"-" uri:"resourceID"` // 资源ID
}

type ResourceDetailReq struct {
	ResourceID string `json:"-" uri:"resourceID"` // 资源ID
}

type ResourcePageListReq struct {
	Page      int    `json:"page" form:"page"`           // 页码
	PageSize  int    `json:"pageSize" form:"pageSize"`   // 每页数量
	TenantID  string `json:"tenantID" form:"tenantID"`   // 租户ID
	Name      string `json:"name" form:"name"`           // 资源名称
	Indicator string `json:"indicator" form:"indicator"` // 资源标识符
}

type ScopeCreateReq struct {
	TenantID    string `json:"tenantID" form:"tenantID"`       // 租户ID
	ResourceID  string `json:"resourceID" form:"resourceID"`   // 资源ID
	Name        string `json:"name" form:"name"`               // 权限名称
	Description string `json:"description" form:"description"` // 权限描述
}

type ScopeUpdateReq struct {
	ScopeID     string `json:"-" uri:"scopeID"`                // 权限ID
	TenantID    string `json:"tenantID" form:"tenantID"`       // 租户ID
	ResourceID  string `json:"resourceID" form:"resourceID"`   // 资源ID
	Name        string `json:"name" form:"name"`               // 权限名称
	Description string `json:"description" form:"description"` // 权限描述
}

type ScopeDeleteReq struct {
	ScopeID string `json:"-" uri:"scopeID"` // 权限ID
}

type ScopeDetailReq struct {
	ScopeID string `json:"-" uri:"scopeID"` // 权限ID
}

type ScopePageListReq struct {
	Page       int    `json:"page" form:"page"`             // 页码
	PageSize   int    `json:"pageSize" form:"pageSize"`     // 每页数量
	TenantID   string `json:"tenantID" form:"tenantID"`     // 租户ID
	ResourceID string `json:"resourceID" form:"resourceID"` // 资源ID
	Name       string `json:"name" form:"name"`             // 权限名称
}

type RoleMenuCreateReq struct {
	TenantID string `json:"tenantID" form:"tenantID"` // 租户ID
	RoleID   string `json:"-" uri:"roleID"`           // 角色ID
	MenuID   string `json:"menuID" form:"menuID"`     // 菜单ID
}

type RoleMenuDeleteReq struct {
	RoleID   string `json:"-" uri:"roleID"`           // 角色ID
	MenuID   string `json:"-" uri:"menuID"`           // 菜单ID
	TenantID string `json:"tenantID" form:"tenantID"` // 租户ID
}

type RoleMenuPageListReq struct {
	Page     int    `json:"page" form:"page"`         // 页码
	PageSize int    `json:"pageSize" form:"pageSize"` // 每页数量
	TenantID string `json:"tenantID" form:"tenantID"` // 租户ID
	RoleID   string `json:"-" uri:"roleID"`           // 角色ID
	MenuID   string `json:"menuID" form:"menuID"`     // 菜单ID
}

type RoleScopeCreateReq struct {
	TenantID string `json:"tenantID" form:"tenantID"` // 租户ID
	RoleID   string `json:"-" uri:"roleID"`           // 角色ID
	ScopeID  string `json:"scopeID" form:"scopeID"`   // 权限ID
}

type RoleScopeDeleteReq struct {
	RoleID   string `json:"-" uri:"roleID"`           // 角色ID
	ScopeID  string `json:"-" uri:"scopeID"`          // 权限ID
	TenantID string `json:"tenantID" form:"tenantID"` // 租户ID
}

type RoleScopePageListReq struct {
	Page     int    `json:"page" form:"page"`         // 页码
	PageSize int    `json:"pageSize" form:"pageSize"` // 每页数量
	TenantID string `json:"tenantID" form:"tenantID"` // 租户ID
	RoleID   string `json:"-" uri:"roleID"`           // 角色ID
	ScopeID  string `json:"scopeID" form:"scopeID"`   // 权限ID
}

type UserRoleCreateReq struct {
	TenantID string `json:"tenantID" form:"tenantID"` // 租户ID
	UserID   string `json:"-" uri:"userID"`           // 用户ID
	RoleID   string `json:"roleID" form:"roleID"`     // 角色ID
}

type UserRoleDeleteReq struct {
	TenantID string `json:"tenantID" form:"tenantID"` // 租户ID
	UserID   string `json:"-" uri:"userID"`           // 用户ID
	RoleID   string `json:"-" uri:"roleID"`           // 角色ID
}

type UserRolePageListReq struct {
	Page     int    `json:"page" form:"page"`         // 页码
	PageSize int    `json:"pageSize" form:"pageSize"` // 每页数量
	TenantID string `json:"tenantID" form:"tenantID"` // 租户ID
	UserID   string `json:"-" uri:"userID"`           // 用户ID
	RoleID   string `json:"roleID" form:"roleID"`     // 角色ID
}
