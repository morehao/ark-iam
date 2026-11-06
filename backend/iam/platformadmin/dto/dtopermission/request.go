package dtopermission

import (
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/object/objidentity"
	"github.com/morehao/ark-iam/pkg/object/objpermission"
	"github.com/morehao/golib/biz/gobject"
)

type MenuCreateReq struct {
	objpermission.MenuBaseInfo
	// Operator 由 controller 从鉴权结果填入（json:"-"，客户端不可伪造）。
	Operator objidentity.Operator
}

type MenuUpdateReq struct {
	MenuID string `json:"-" uri:"menuID" binding:"required"` // 菜单ID
	objpermission.MenuBaseInfo
	// Operator 由 controller 从鉴权结果填入（json:"-"，客户端不可伪造）。
	Operator objidentity.Operator
}

type MenuDetailReq struct {
	MenuID string `json:"-" uri:"menuID" binding:"required"` // 菜单ID
}

type MenuPageListReq struct {
	gobject.PageQuery
	AppID      string               `json:"appID" form:"appID"`           // 应用ID
	ParentID   string               `json:"parentID" form:"parentID"`     // 父菜单ID
	Name       string               `json:"name" form:"name"`             // 菜单名称
	Code       string               `json:"code" form:"code"`             // 菜单编码
	Type       model.MenuType       `json:"type" form:"type"`             // 菜单类型
	Status     model.MenuStatus     `json:"status" form:"status"`         // 状态
	Visibility model.MenuVisibility `json:"visibility" form:"visibility"` // 可见性门槛(public/member/admin)
}

type MenuDeleteReq struct {
	MenuID string `json:"-" uri:"menuID" binding:"required"` // 菜单ID
	// Operator 由 controller 从鉴权结果填入（json:"-"，客户端不可伪造）。
	Operator objidentity.Operator
}

type MenuTreeReq struct {
	AppID string `json:"appID" form:"appID" binding:"required"` // 应用ID（菜单树按应用维度查询）
}

// MenuMyTreeReq 平台菜单树请求（无业务参数，仅携带操作人：
// 可见菜单取决于该用户持有的角色与管理员身份）。
type MenuMyTreeReq struct {
	// Operator 由 controller 从鉴权结果填入（json:"-"，客户端不可伪造）。
	Operator objidentity.Operator
}
