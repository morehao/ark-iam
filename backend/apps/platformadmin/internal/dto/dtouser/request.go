package dtouser

import (
	"github.com/morehao/ark-iam/pkg/iam/object/objuser"
	"github.com/morehao/golib/biz/gobject"
)

type UserDetailReq struct {
	UserID string `json:"-" uri:"userID" binding:"required"` // 用户ID
}

type UserPageListReq struct {
	gobject.PageQuery
	TenantID     string `json:"tenantID" form:"tenantID"`         // 租户ID
	Username     string `json:"username" form:"username"`         // 用户名
	PrimaryEmail string `json:"primaryEmail" form:"primaryEmail"` // 主要邮箱
	PrimaryPhone string `json:"primaryPhone" form:"primaryPhone"` // 主要手机号
	Name         string `json:"name" form:"name"`                 // 姓名
	IsSuspended  *bool  `json:"isSuspended" form:"isSuspended"`   // 是否挂起
}

type UserPasswordUpdateReq struct {
	UserID string `json:"-" uri:"userID" binding:"required"` // 用户ID
	// Password 新密码明文（必填，服务端哈希后写入自然人）
	Password string `json:"password" binding:"required"`
	objuser.UserPasswordInfo
}

type UserStatusUpdateReq struct {
	UserID      string `json:"-" uri:"userID" binding:"required"` // 用户ID
	IsSuspended bool   `json:"isSuspended"`                       // 是否挂起
}

// UserOwnerUpdateReq 平台管理员显式指派/取消某租户用户为租户拥有者。
// owner 只由平台管理员（或自助开通租户时的注册人）产生，普通自助加入永不 owner。
type UserOwnerUpdateReq struct {
	UserID  string `json:"-" uri:"userID" binding:"required"` // 用户ID
	IsOwner bool   `json:"isOwner"`                           // 是否租户拥有者
}

type UserIdentityCreateReq struct {
	TenantID   string `json:"tenantID" binding:"required"`   // 租户ID
	UserID     string `json:"-" uri:"userID"`                // 用户ID（path）
	Issuer     string `json:"issuer" binding:"required"`     // 身份提供商
	IdentityID string `json:"identityID" binding:"required"` // 第三方用户ID
	Detail     any    `json:"detail"`                        // 详细信息
}

type UserIdentityUpdateReq struct {
	UserIdentityID string `json:"-" uri:"identityID" binding:"required"` // 用户身份ID（path）
	TenantID       string `json:"tenantID"`                              // 租户ID
	UserID         string `json:"-" uri:"userID"`                        // 用户ID（path）
	Issuer         string `json:"issuer"`                                // 身份提供商
	IdentityID     string `json:"identityID"`                            // 第三方用户ID
	Detail         any    `json:"detail"`                                // 详细信息
}

type UserIdentityDetailReq struct {
	UserIdentityID string `json:"-" uri:"identityID" binding:"required"` // 用户身份ID（path）
	UserID         string `json:"-" uri:"userID"`                        // 用户ID（path）
}

type UserIdentityPageListReq struct {
	gobject.PageQuery
	TenantID   string `json:"tenantID" form:"tenantID"`     // 租户ID
	UserID     string `json:"userID" form:"userID"`         // 用户ID
	Issuer     string `json:"issuer" form:"issuer"`         // 身份提供商
	IdentityID string `json:"identityID" form:"identityID"` // 第三方用户ID
}

type UserIdentityDeleteReq struct {
	UserIdentityID string `json:"-" uri:"identityID" binding:"required"` // 用户身份ID（path）
	UserID         string `json:"-" uri:"userID"`                        // 用户ID（path）
}

type UserIdentityByUserReq struct {
	UserID string `json:"-" uri:"userID" binding:"required"` // 用户ID
}

type UserLoginLogByUserReq struct {
	UserID string `json:"-" uri:"userID" binding:"required"` // 用户ID
}

type RoleUserListReq struct {
	RoleID string `json:"-" uri:"roleID" binding:"required"`
}
