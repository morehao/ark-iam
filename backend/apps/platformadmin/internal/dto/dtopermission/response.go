package dtopermission

import (
	"github.com/morehao/ark-iam/pkg/iam/object/objpermission"
	"github.com/morehao/ark-iam/pkg/iam/object/objresource"
	"github.com/morehao/golib/biz/gobject"
)

type MenuCreateResp struct {
	MenuID uint `json:"menuID"` // 菜单ID
}

type MenuDetailResp struct {
	MenuID uint `json:"menuID"` // 菜单ID
	objpermission.MenuBaseInfo
	gobject.OperatorBaseInfo
}

type MenuPageListItem struct {
	MenuID uint `json:"menuID"` // 菜单ID
	objpermission.MenuBaseInfo
	gobject.OperatorBaseInfo
}

type MenuPageListResp struct {
	List  []MenuPageListItem `json:"list"`  // 数据列表
	Total int64              `json:"total"` // 数据总条数
}

type MenuTreeItem struct {
	MenuID uint `json:"menuID"` // 菜单ID
	objpermission.MenuBaseInfo
	gobject.OperatorBaseInfo
	Children []MenuTreeItem `json:"children"` // 子菜单
}

type MenuTreeResp struct {
	List []MenuTreeItem `json:"list"` // 菜单树
}

type RoleCreateResp struct {
	RoleID uint `json:"roleID"` // 角色ID
}

type RoleUpdateResp struct {
}

type RoleDetailResp struct {
	RoleID uint `json:"roleID"` // 角色ID
	objpermission.RoleBaseInfo
	OperatorBaseInfo gobject.OperatorBaseInfo `json:"operatorBaseInfo"` // 操作人信息
}

type RolePageListItem struct {
	RoleID uint `json:"roleID"` // 角色ID
	objpermission.RoleBaseInfo
	OperatorBaseInfo gobject.OperatorBaseInfo `json:"operatorBaseInfo"` // 操作人信息
}

type RolePageListResp struct {
	List  []RolePageListItem `json:"list"`  // 列表
	Total int64              `json:"total"` // 总数
}

type ResourceCreateResp struct {
	ResourceID uint `json:"resourceID"` // 资源ID
}

type ResourceUpdateResp struct {
}

type ResourceDetailResp struct {
	ResourceID uint `json:"resourceID"` // 资源ID
	objresource.ResourceBaseInfo
	OperatorBaseInfo gobject.OperatorBaseInfo `json:"operatorBaseInfo"` // 操作人信息
}

type ResourcePageListItem struct {
	ResourceID uint `json:"resourceID"` // 资源ID
	objresource.ResourceBaseInfo
	OperatorBaseInfo gobject.OperatorBaseInfo `json:"operatorBaseInfo"` // 操作人信息
}

type ResourcePageListResp struct {
	List  []ResourcePageListItem `json:"list"`  // 列表
	Total int64                  `json:"total"` // 总数
}

type ScopeCreateResp struct {
	ScopeID uint `json:"scopeID"` // 权限ID
}

type ScopeUpdateResp struct {
}

type ScopeDetailResp struct {
	ScopeID uint `json:"scopeID"` // 权限ID
	objresource.ScopeBaseInfo
	OperatorBaseInfo gobject.OperatorBaseInfo `json:"operatorBaseInfo"` // 操作人信息
}

type ScopePageListItem struct {
	ScopeID uint `json:"scopeID"` // 权限ID
	objresource.ScopeBaseInfo
	OperatorBaseInfo gobject.OperatorBaseInfo `json:"operatorBaseInfo"` // 操作人信息
}

type ScopePageListResp struct {
	List  []ScopePageListItem `json:"list"`  // 列表
	Total int64               `json:"total"` // 总数
}

type RoleScopeCreateResp struct {
}

type RoleScopeDeleteResp struct {
}

type RoleScopePageListItem struct {
	RoleID   uint `json:"roleID"`   // 角色ID
	ScopeID  uint `json:"scopeID"`  // 权限ID
	TenantID uint `json:"tenantID"` // 租户ID
}

type RoleScopePageListResp struct {
	List  []RoleScopePageListItem `json:"list"`  // 列表
	Total int64                   `json:"total"` // 总数
}

type UserRoleCreateResp struct {
}

type UserRoleDeleteResp struct {
}

type UserRolePageListItem struct {
	UserID   uint `json:"userID"`   // 用户ID
	RoleID   uint `json:"roleID"`   // 角色ID
	TenantID uint `json:"tenantID"` // 租户ID
}

type UserRolePageListResp struct {
	List  []UserRolePageListItem `json:"list"`  // 列表
	Total int64                  `json:"total"` // 总数
}

type RoleMenuCreateResp struct {
}

type RoleMenuDeleteResp struct {
}

type RoleMenuPageListItem struct {
	RoleID   uint `json:"roleID"`   // 角色ID
	MenuID   uint `json:"menuID"`   // 菜单ID
	TenantID uint `json:"tenantID"` // 租户ID
}

type RoleMenuPageListResp struct {
	List  []RoleMenuPageListItem `json:"list"`  // 列表
	Total int64                  `json:"total"` // 总数
}
