package dtotenant

import (
	"github.com/morehao/ark-iam/pkg/object/objtenant"
	"github.com/morehao/golib/biz/gobject"
)

type TenantCreateResp struct {
	TenantID    string `json:"tenantID"`    // 租户ID
	AdminUserID string `json:"adminUserID"` // 租户管理员用户ID
	// AdminInitialPassword 租户管理员的初始临时密码，仅在此响应中返回一次，不落库、不可再查；
	// 若管理员的邮箱/手机命中已存在自然人（其密码不被改动），该字段为空串。
	// 待办：邮件/短信通道接入后本字段下线，见 pkg/credential 的 TODO(delivery)。
	AdminInitialPassword string `json:"adminInitialPassword"` // 初始临时密码(仅此一次返回)
}

// TenantAdminResetPasswordResp 重置内置管理员密码的响应。
type TenantAdminResetPasswordResp struct {
	UserID string `json:"userID"` // 被重置的内置管理员用户ID
	// InitialPassword 新临时密码，仅在此响应中返回一次，不落库、不可再查。
	// 待办：邮件/短信通道接入后本字段下线，见 pkg/credential 的 TODO(delivery)。
	InitialPassword string `json:"initialPassword"` // 新临时密码(仅此一次返回)
}

type TenantDetailResp struct {
	TenantID string `json:"tenantID"` // 租户ID
	objtenant.TenantBaseInfo
	gobject.OperatorBaseInfo
}

type TenantPageListItem struct {
	TenantID string `json:"tenantID"` // 租户ID
	objtenant.TenantBaseInfo
	gobject.OperatorBaseInfo
}

type TenantPageListResp struct {
	List  []TenantPageListItem `json:"list"`  // 数据列表
	Total int64                `json:"total"` // 数据总条数
}
