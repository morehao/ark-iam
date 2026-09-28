package dtotenant

import "github.com/morehao/ark-iam/pkg/object/objpermission"

// MenuTreeItem 租户侧菜单树节点（含所属应用）
type MenuTreeItem struct {
	MenuID string `json:"menuID"` // 菜单ID
	objpermission.MenuBaseInfo
	Children []MenuTreeItem `json:"children"` // 子菜单
}

// MenuTreeResp 租户侧菜单树响应
type MenuTreeResp struct {
	List []MenuTreeItem `json:"list"` // 菜单树
}

// TenantApplicationItem 租户订阅的应用（角色归属 / 菜单授权 / 应用准入的应用选项）
type TenantApplicationItem struct {
	AppID string `json:"appID"` // 应用ID
	Code  string `json:"code"`  // 应用编码
	Name  string `json:"name"`  // 应用名称
}

// TenantApplicationsResp 租户订阅应用列表响应（GET /v1/tenant/applications）
type TenantApplicationsResp struct {
	List []TenantApplicationItem `json:"list"` // 应用列表
}
