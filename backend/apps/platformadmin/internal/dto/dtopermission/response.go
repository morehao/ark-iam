package dtopermission

import (
	"github.com/morehao/ark-iam/pkg/iam/object/objpermission"
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

