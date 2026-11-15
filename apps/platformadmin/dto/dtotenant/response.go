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
	// 通道启用且投递成功时本字段为空、由 delivered 表达结果；回显仅通道未启用/投递失败降级时才有值。
	AdminInitialPassword string `json:"adminInitialPassword"` // 初始临时密码(通道未启用/投递失败回退时才有值)
	// Delivered 表示临时密码是否已通过邮件/短信投递给账号本人。true=已投递(明文不回显)；
	// false=未投递(通道未启用或投递失败回退)。见 docs/design/system-design.md §5.8。
	Delivered bool `json:"delivered"`
}

// TenantAdminResetPasswordResp 重置内置管理员密码的响应。
type TenantAdminResetPasswordResp struct {
	UserID string `json:"userID"` // 被重置的内置管理员用户ID
	// InitialPassword 新临时密码，仅在此响应中返回一次，不落库、不可再查。
	// 通道启用且投递成功时本字段为空、由 delivered 表达结果；回显仅通道未启用/投递失败降级时才有值。
	InitialPassword string `json:"initialPassword"` // 新临时密码(通道未启用/投递失败回退时才有值)
	// Delivered 表示临时密码是否已通过邮件/短信投递给账号本人。
	Delivered bool `json:"delivered"`
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
