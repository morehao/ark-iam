package dtotenant

import (
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/object/objidentity"
)

// AppMenuFields 租户侧菜单的可写字段（不含 AppID：AppID 只能来自路径，不可由 body 伪造）。
// 与 objpermission.MenuBaseInfo 字段一致，但刻意不含 AppID，避免 body 覆盖 path 造成参数污染。
type AppMenuFields struct {
	ParentID     string                     `json:"parentID"`                            // 父菜单ID
	Name         string                     `json:"name"`                                // 菜单名称
	Code         string                     `json:"code"`                                // 菜单编码
	Path         string                     `json:"path"`                                // 菜单路径
	Icon         string                     `json:"icon"`                                // 菜单图标
	Sort         int                        `json:"sort"`                                // 排序
	Type         model.MenuType             `json:"type"`                                // 菜单类型
	Visibility   model.MenuVisibility       `json:"visibility"`                          // 可见性门槛(public/member/admin)
	Component    string                     `json:"component"`                           // 组件路径
	Redirect     string                     `json:"redirect"`                            // 重定向路径
	Hidden       model.MenuHiddenFlag       `json:"hidden"`                              // 是否隐藏(enable/disable)
	ExternalLink model.MenuExternalLinkFlag `json:"externalLink"`                        // 是否外链(enable/disable)
	KeepAlive    model.MenuKeepAliveFlag    `json:"keepAlive"`                           // 是否缓存(enable/disable)
	Status       model.MenuStatus           `json:"status"`                              // 状态
}

// AppMenuCreateReq 租户侧创建应用菜单请求。AppID 只来自 path。
type AppMenuCreateReq struct {
	AppID string `json:"-" uri:"appID" binding:"required"`
	AppMenuFields
	Operator objidentity.Operator
}

// AppMenuCreateResp 创建菜单响应。
type AppMenuCreateResp struct {
	MenuID string `json:"menuID"`
}

// AppMenuUpdateReq 租户侧更新应用菜单请求。AppID/MenuID 只来自 path，不允许改 app_id。
type AppMenuUpdateReq struct {
	AppID  string `json:"-" uri:"appID" binding:"required"`
	MenuID string `json:"-" uri:"menuID" binding:"required"`
	AppMenuFields
	Operator objidentity.Operator
}

// AppMenuDeleteReq 租户侧删除应用菜单请求。
type AppMenuDeleteReq struct {
	AppID  string `json:"-" uri:"appID" binding:"required"`
	MenuID string `json:"-" uri:"menuID" binding:"required"`
	Operator objidentity.Operator
}

// AppMenuTreeReq 租户侧应用菜单树（管理视图）请求。
type AppMenuTreeReq struct {
	AppID    string `json:"-" uri:"appID" binding:"required"`
	Operator objidentity.Operator
}

// AppMenuDetailReq 租户侧菜单详情请求。
type AppMenuDetailReq struct {
	AppID  string `json:"-" uri:"appID" binding:"required"`
	MenuID string `json:"-" uri:"menuID" binding:"required"`
	Operator objidentity.Operator
}
