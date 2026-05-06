package dtomenu

import (
	"github.com/morehao/ark-iam/iam/object/objmenu"
	"github.com/morehao/golib/biz/gobject"
)

type MenuCreateResp struct {
	MenuID uint `json:"menuID"` // 菜单ID
}

type MenuDetailResp struct {
	MenuID uint `json:"menuID"` // 菜单ID
	objmenu.MenuBaseInfo
	gobject.OperatorBaseInfo
}

type MenuPageListItem struct {
	MenuID uint `json:"menuID"` // 菜单ID
	objmenu.MenuBaseInfo
	gobject.OperatorBaseInfo
}

type MenuPageListResp struct {
	List  []MenuPageListItem `json:"list"`  // 数据列表
	Total int64             `json:"total"` // 数据总条数
}

type MenuTreeItem struct {
	MenuID uint `json:"menuID"` // 菜单ID
	objmenu.MenuBaseInfo
	gobject.OperatorBaseInfo
	Children []MenuTreeItem `json:"children"` // 子菜单
}

type MenuTreeResp struct {
	List []MenuTreeItem `json:"list"` // 菜单树
}