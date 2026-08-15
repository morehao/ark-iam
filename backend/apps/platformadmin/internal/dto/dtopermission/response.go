package dtopermission

import (
	"github.com/morehao/ark-iam/pkg/iam/object/objpermission"
	"github.com/morehao/ark-iam/pkg/iam/object/objresource"
	"github.com/morehao/golib/biz/gobject"
)

type MenuCreateResp struct {
	MenuID string `json:"menuID"` // 菜单ID
}

type MenuDetailResp struct {
	MenuID string `json:"menuID"` // 菜单ID
	objpermission.MenuBaseInfo
	gobject.OperatorBaseInfo
}

type MenuPageListItem struct {
	MenuID string `json:"menuID"` // 菜单ID
	objpermission.MenuBaseInfo
	gobject.OperatorBaseInfo
}

type MenuPageListResp struct {
	List  []MenuPageListItem `json:"list"`  // 数据列表
	Total int64              `json:"total"` // 数据总条数
}

type MenuTreeItem struct {
	MenuID string `json:"menuID"` // 菜单ID
	objpermission.MenuBaseInfo
	gobject.OperatorBaseInfo
	Children []MenuTreeItem `json:"children"` // 子菜单
}

type MenuTreeResp struct {
	List []MenuTreeItem `json:"list"` // 菜单树
}

type RoleCreateResp struct {
	RoleID string `json:"roleID"` // 角色ID
}

type RoleUpdateResp struct {
}

type RoleDetailResp struct {
	RoleID string `json:"roleID"` // 角色ID
	objpermission.RoleBaseInfo
	OperatorBaseInfo gobject.OperatorBaseInfo `json:"operatorBaseInfo"` // 操作人信息
}

type RolePageListItem struct {
	RoleID string `json:"roleID"` // 角色ID
	objpermission.RoleBaseInfo
	OperatorBaseInfo gobject.OperatorBaseInfo `json:"operatorBaseInfo"` // 操作人信息
}

type RolePageListResp struct {
	List  []RolePageListItem `json:"list"`  // 列表
	Total int64              `json:"total"` // 总数
}

type ResourceCreateResp struct {
	ResourceID string `json:"resourceID"` // 资源ID
}

type ResourceUpdateResp struct {
}

type ResourceDetailResp struct {
	ResourceID string `json:"resourceID"` // 资源ID
	objresource.ResourceBaseInfo
	OperatorBaseInfo gobject.OperatorBaseInfo `json:"operatorBaseInfo"` // 操作人信息
}

type ResourcePageListItem struct {
	ResourceID string `json:"resourceID"` // 资源ID
	objresource.ResourceBaseInfo
	OperatorBaseInfo gobject.OperatorBaseInfo `json:"operatorBaseInfo"` // 操作人信息
}

type ResourcePageListResp struct {
	List  []ResourcePageListItem `json:"list"`  // 列表
	Total int64                  `json:"total"` // 总数
}

type ScopeCreateResp struct {
	ScopeID string `json:"scopeID"` // 权限ID
}

type ScopeUpdateResp struct {
}

type ScopeDetailResp struct {
	ScopeID string `json:"scopeID"` // 权限ID
	objresource.ScopeBaseInfo
	OperatorBaseInfo gobject.OperatorBaseInfo `json:"operatorBaseInfo"` // 操作人信息
}

type ScopePageListItem struct {
	ScopeID string `json:"scopeID"` // 权限ID
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
	RoleID   string `json:"roleID"`   // 角色ID
	ScopeID  string `json:"scopeID"`  // 权限ID
	TenantID string `json:"tenantID"` // 租户ID
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
	UserID   string `json:"userID"`   // 用户ID
	RoleID   string `json:"roleID"`   // 角色ID
	TenantID string `json:"tenantID"` // 租户ID
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
	RoleID   string `json:"roleID"`   // 角色ID
	MenuID   string `json:"menuID"`   // 菜单ID
	TenantID string `json:"tenantID"` // 租户ID
}

type RoleMenuPageListResp struct {
	List  []RoleMenuPageListItem `json:"list"`  // 列表
	Total int64                  `json:"total"` // 总数
}
