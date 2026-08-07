package dtopermission

import (
	"github.com/morehao/ark-iam/pkg/iam/object/objpermission"
	"github.com/morehao/golib/biz/gobject"
)

type MenuCreateReq struct {
	objpermission.MenuBaseInfo
}

type MenuUpdateReq struct {
	MenuID uint `json:"menuID" binding:"required"` // 菜单ID
	objpermission.MenuBaseInfo
}

type MenuDetailReq struct {
	MenuID uint `json:"menuID" form:"menuID" binding:"required"` // 菜单ID
}

type MenuPageListReq struct {
	gobject.PageQuery
	AppID uint   `json:"appId"` // 应用ID
	ParentID      uint   `json:"parentID"`      // 父菜单ID
	Name     string `json:"name"`     // 菜单名称
	Code     string `json:"code"`     // 菜单编码
	Type     string `json:"type"`     // 菜单类型
	Status   string `json:"status"`   // 状态
}

type MenuDeleteReq struct {
	MenuID uint `json:"menuID" binding:"required"` // 菜单ID
}

type MenuTreeReq struct {
	AppID uint `json:"appId" form:"appId"` // 应用ID
}

type RoleCreateReq struct {
	TenantID    uint   `json:"tenantID" form:"tenantID"`       // 租户ID
	Name        string `json:"name" form:"name"`               // 角色名称
	Code        string `json:"code" form:"code"`               // 角色编码
	Description string `json:"description" form:"description"` // 角色描述
	Type        string `json:"type" form:"type"`               // 角色类型
	IsDefault   int8   `json:"isDefault" form:"isDefault"`     // 是否默认角色
}

type RoleUpdateReq struct {
	RoleID      uint   `json:"roleID" form:"roleID"`         // 角色ID
	TenantID    uint   `json:"tenantID" form:"tenantID"`     // 租户ID
	Name        string `json:"name" form:"name"`             // 角色名称
	Code        string `json:"code" form:"code"`             // 角色编码
	Description string `json:"description" form:"description"` // 角色描述
	Type        string `json:"type" form:"type"`             // 角色类型
	IsDefault   int8   `json:"isDefault" form:"isDefault"`     // 是否默认角色
}

type RoleDeleteReq struct {
	RoleID uint `json:"roleID" form:"roleID"` // 角色ID
}

type RoleDetailReq struct {
	RoleID uint `json:"roleID" form:"roleID"` // 角色ID
}

type RolePageListReq struct {
	Page     int    `json:"page" form:"page"`         // 页码
	PageSize int    `json:"pageSize" form:"pageSize"` // 每页数量
	TenantID uint   `json:"tenantID" form:"tenantID"` // 租户ID
	Name     string `json:"name" form:"name"`         // 角色名称
	Code     string `json:"code" form:"code"`         // 角色编码
	Type     string `json:"type" form:"type"`         // 角色类型
}

type ResourceCreateReq struct {
	TenantID       uint   `json:"tenantID" form:"tenantID"`             // 租户ID
	Name           string `json:"name" form:"name"`                     // 资源名称
	Indicator      string `json:"indicator" form:"indicator"`           // 资源标识符
	IsDefault      int8   `json:"isDefault" form:"isDefault"`           // 是否默认
	AccessTokenTtl int64  `json:"accessTokenTtl" form:"accessTokenTtl"` // 访问令牌TTL
}

type ResourceUpdateReq struct {
	ResourceID    uint   `json:"resourceID" form:"resourceID"`     // 资源ID
	TenantID       uint   `json:"tenantID" form:"tenantID"`       // 租户ID
	Name           string `json:"name" form:"name"`               // 资源名称
	Indicator      string `json:"indicator" form:"indicator"`     // 资源标识符
	IsDefault      int8   `json:"isDefault" form:"isDefault"`     // 是否默认
	AccessTokenTtl int64  `json:"accessTokenTtl" form:"accessTokenTtl"` // 访问令牌TTL
}

type ResourceDeleteReq struct {
	ResourceID uint `json:"resourceID" form:"resourceID"` // 资源ID
}

type ResourceDetailReq struct {
	ResourceID uint `json:"resourceID" form:"resourceID"` // 资源ID
}

type ResourcePageListReq struct {
	Page      int    `json:"page" form:"page"`          // 页码
	PageSize  int    `json:"pageSize" form:"pageSize"`  // 每页数量
	TenantID  uint   `json:"tenantID" form:"tenantID"`  // 租户ID
	Name      string `json:"name" form:"name"`          // 资源名称
	Indicator string `json:"indicator" form:"indicator"` // 资源标识符
}

type ScopeCreateReq struct {
	TenantID    uint   `json:"tenantID" form:"tenantID"`     // 租户ID
	ResourceID  uint   `json:"resourceID" form:"resourceID"` // 资源ID
	Name        string `json:"name" form:"name"`             // 权限名称
	Description string `json:"description" form:"description"` // 权限描述
}

type ScopeUpdateReq struct {
	ScopeID     uint   `json:"scopeID" form:"scopeID"`       // 权限ID
	TenantID    uint   `json:"tenantID" form:"tenantID"`   // 租户ID
	ResourceID  uint   `json:"resourceID" form:"resourceID"` // 资源ID
	Name        string `json:"name" form:"name"`           // 权限名称
	Description string `json:"description" form:"description"` // 权限描述
}

type ScopeDeleteReq struct {
	ScopeID uint `json:"scopeID" form:"scopeID"` // 权限ID
}

type ScopeDetailReq struct {
	ScopeID uint `json:"scopeID" form:"scopeID"` // 权限ID
}

type ScopePageListReq struct {
	Page       int    `json:"page" form:"page"`           // 页码
	PageSize   int    `json:"pageSize" form:"pageSize"`   // 每页数量
	TenantID   uint   `json:"tenantID" form:"tenantID"`  // 租户ID
	ResourceID uint   `json:"resourceID" form:"resourceID"` // 资源ID
	Name       string `json:"name" form:"name"`           // 权限名称
}

type RoleMenuCreateReq struct {
	TenantID uint `json:"tenantID" form:"tenantID"` // 租户ID
	RoleID   uint `json:"roleID" form:"roleID"`     // 角色ID
	MenuID   uint `json:"menuID" form:"menuID"`     // 菜单ID
}

type RoleMenuDeleteReq struct {
	RoleID uint `json:"roleID" form:"roleID"`   // 角色ID
	MenuID uint `json:"menuID" form:"menuID"`   // 菜单ID
	TenantID uint `json:"tenantID" form:"tenantID"` // 租户ID
}

type RoleMenuPageListReq struct {
	Page     int    `json:"page" form:"page"`         // 页码
	PageSize int    `json:"pageSize" form:"pageSize"` // 每页数量
	TenantID uint   `json:"tenantID" form:"tenantID"` // 租户ID
	RoleID   uint   `json:"roleID" form:"roleID"`     // 角色ID
	MenuID   uint   `json:"menuID" form:"menuID"`     // 菜单ID
}

type RoleScopeCreateReq struct {
	TenantID uint `json:"tenantID" form:"tenantID"` // 租户ID
	RoleID   uint `json:"roleID" form:"roleID"`     // 角色ID
	ScopeID  uint `json:"scopeID" form:"scopeID"`   // 权限ID
}

type RoleScopeDeleteReq struct {
	RoleID uint `json:"roleID" form:"roleID"`   // 角色ID
	ScopeID uint `json:"scopeID" form:"scopeID"` // 权限ID
	TenantID uint `json:"tenantID" form:"tenantID"` // 租户ID
}

type RoleScopePageListReq struct {
	Page     int    `json:"page" form:"page"`         // 页码
	PageSize int    `json:"pageSize" form:"pageSize"` // 每页数量
	TenantID uint   `json:"tenantID" form:"tenantID"` // 租户ID
	RoleID   uint   `json:"roleID" form:"roleID"`     // 角色ID
	ScopeID  uint   `json:"scopeID" form:"scopeID"`   // 权限ID
}

type UserRoleCreateReq struct {
	TenantID uint `json:"tenantID" form:"tenantID"` // 租户ID
	UserID   uint `json:"userID" form:"userID"`     // 用户ID
	RoleID   uint `json:"roleID" form:"roleID"`     // 角色ID
}

type UserRoleDeleteReq struct {
	TenantID uint `json:"tenantID" form:"tenantID"` // 租户ID
	UserID   uint `json:"userID" form:"userID"`     // 用户ID
	RoleID   uint `json:"roleID" form:"roleID"`     // 角色ID
}

type UserRolePageListReq struct {
	Page     int    `json:"page" form:"page"`         // 页码
	PageSize int    `json:"pageSize" form:"pageSize"` // 每页数量
	TenantID uint   `json:"tenantID" form:"tenantID"` // 租户ID
	UserID   uint   `json:"userID" form:"userID"`     // 用户ID
	RoleID   uint   `json:"roleID" form:"roleID"`     // 角色ID
}
