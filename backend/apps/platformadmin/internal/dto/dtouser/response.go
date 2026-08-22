package dtouser

import (
	"github.com/morehao/ark-iam/pkg/iam/object/objuser"
	"github.com/morehao/golib/biz/gobject"
)

type UserDetailResp struct {
	UserID string `json:"userID"` // 用户ID
	objuser.UserBaseInfo
	gobject.OperatorBaseInfo
}

type UserPageListItem struct {
	UserID string `json:"userID"` // 用户ID
	objuser.UserBaseInfo
	gobject.OperatorBaseInfo
}

type UserPageListResp struct {
	List  []UserPageListItem `json:"list"`  // 数据列表
	Total int64              `json:"total"` // 数据总条数
}

type UserIdentityCreateResp struct {
	UserIdentityID string `json:"userIdentityID"` // 用户身份ID
}

type UserIdentityDetailResp struct {
	UserIdentityID string `json:"userIdentityID"` // 用户身份ID
	TenantID       string `json:"tenantID"`       // 租户ID
	UserID         string `json:"userID"`         // 用户ID
	Issuer         string `json:"issuer"`         // 身份提供商
	IdentityID     string `json:"identityID"`     // 第三方用户ID
	Detail         any    `json:"detail"`         // 详细信息
	gobject.OperatorBaseInfo
}

type UserIdentityPageListItem struct {
	UserIdentityID string `json:"userIdentityID"` // 用户身份ID
	TenantID       string `json:"tenantID"`       // 租户ID
	UserID         string `json:"userID"`         // 用户ID
	Issuer         string `json:"issuer"`         // 身份提供商
	IdentityID     string `json:"identityID"`     // 第三方用户ID
	Detail         any    `json:"detail"`         // 详细信息
	gobject.OperatorBaseInfo
}

type UserIdentityPageListResp struct {
	List  []UserIdentityPageListItem `json:"list"`  // 数据列表
	Total int64                      `json:"total"` // 数据总条数
}

type UserLoginLogPageListItem struct {
	UserLoginLogID string `json:"userLoginLogID"` // 登录日志ID
	TenantID       string `json:"tenantID"`       // 租户ID
	UserID         string `json:"userID"`         // 用户ID
	LoginIP        string `json:"loginIP"`        // 登录IP地址
	UserAgent      string `json:"userAgent"`      // 用户代理信息
	LoginTime      int64  `json:"loginTime"`      // 登录时间
	gobject.OperatorBaseInfo
}

type UserLoginLogPageListResp struct {
	List  []UserLoginLogPageListItem `json:"list"`  // 数据列表
	Total int64                      `json:"total"` // 数据总条数
}

type RoleUserResp struct {
	UserID    string `json:"userID"`
	Username  string `json:"username"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	RoleID    string `json:"roleID"`
	CreatedAt int64  `json:"createdAt"` // 创建时间(unix 秒)
}

type RoleUserListResp struct {
	Total int64          `json:"total"`
	Users []RoleUserResp `json:"users"`
}
