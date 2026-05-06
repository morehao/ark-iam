package dtomenu

import (
	"github.com/morehao/ark-iam/iam/object/objmenu"
	"github.com/morehao/golib/biz/gobject"
)

type MenuCreateReq struct {
	objmenu.MenuBaseInfo
}

type MenuUpdateReq struct {
	MenuID uint `json:"menuID" binding:"required"` // 菜单ID
	objmenu.MenuBaseInfo
}

type MenuDetailReq struct {
	MenuID uint `json:"menuID" binding:"required"` // 菜单ID
}

type MenuPageListReq struct {
	gobject.PageQuery
	TenantID uint   `json:"tenantID"` // 租户ID
	ParentID uint   `json:"parentID"` // 父菜单ID
	Name     string `json:"name"`     // 菜单名称
	Code     string `json:"code"`     // 菜单编码
	Type     string `json:"type"`     // 菜单类型
	Status   string `json:"status"`   // 状态
}

type MenuDeleteReq struct {
	MenuID uint `json:"menuID" binding:"required"` // 菜单ID
}

type MenuTreeReq struct {
	TenantID uint `json:"tenantID"` // 租户ID
}