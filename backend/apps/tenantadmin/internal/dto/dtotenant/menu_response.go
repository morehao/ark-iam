package dtotenant

import "github.com/morehao/ark-iam/pkg/iam/object/objpermission"

// MenuTreeItem 租户侧菜单树节点（含所属应用）
type MenuTreeItem struct {
	MenuID uint `json:"menuID"` // 菜单ID
	objpermission.MenuBaseInfo
	Children []MenuTreeItem `json:"children"` // 子菜单
}

// MenuTreeResp 租户侧菜单树响应
type MenuTreeResp struct {
	List []MenuTreeItem `json:"list"` // 菜单树
}
