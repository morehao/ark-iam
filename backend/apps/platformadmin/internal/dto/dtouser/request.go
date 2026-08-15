package dtouser

import (
	"github.com/morehao/ark-iam/pkg/iam/object/objuser"
	"github.com/morehao/golib/biz/gobject"
)

type UserCreateReq struct {
	// PersonID 自然人ID：>0 时关联已有自然人（不重复创建 person）
	PersonID string `json:"personID"`
	// Password 明文密码：与 username/primaryEmail/primaryPhone 组合使用，
	// 提供时自动创建自然人（person）并建立账号，使该用户可登录。
	Password string `json:"password"`
	objuser.UserBaseInfo
}

type UserUpdateReq struct {
	UserID string `json:"-" uri:"userID" binding:"required"` // 用户ID
	objuser.UserBaseInfo
}

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
	IsSuspended  *int8  `json:"isSuspended" form:"isSuspended"`   // 是否挂起
}

type UserDeleteReq struct {
	UserID string `json:"-" uri:"userID" binding:"required"` // 用户ID
}

type UserPasswordUpdateReq struct {
	UserID string `json:"-" uri:"userID" binding:"required"` // 用户ID
	// Password 新密码明文（必填，服务端哈希后写入自然人）
	Password string `json:"password" binding:"required"`
	objuser.UserPasswordInfo
}

type UserStatusUpdateReq struct {
	UserID      string `json:"-" uri:"userID" binding:"required"` // 用户ID
	IsSuspended int8   `json:"isSuspended"`                       // 是否挂起
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

type UserLoginLogDetailReq struct {
	UserLoginLogID string `json:"-" uri:"loginLogID" binding:"required"` // 登录日志ID
}

type UserLoginLogPageListReq struct {
	gobject.PageQuery
	TenantID string `json:"tenantID" form:"tenantID"` // 租户ID
	UserID   string `json:"userID" form:"userID"`     // 用户ID
	LoginIP  string `json:"loginIP" form:"loginIP"`   // 登录IP地址
}

type UserLoginLogByUserReq struct {
	UserID string `json:"-" uri:"userID" binding:"required"` // 用户ID
}

type UserDepartmentByUserReq struct {
	UserID string `json:"-" uri:"userID" binding:"required"` // 用户ID
}

type AssignDepartmentsReq struct {
	UserID        string   `json:"-" uri:"userID" binding:"required"` // 用户ID（path）
	DepartmentIDs []string `json:"departmentIDs" binding:"required"`  // 部门ID列表
}

type RoleUserListReq struct {
	RoleID string `json:"-" uri:"roleID" binding:"required"`
}

type AssignRoleUsersReq struct {
	RoleID  string   `json:"-" uri:"roleID" binding:"required"`
	UserIDs []string `json:"userIDs" binding:"required,min=1"`
}

type RemoveRoleUserReq struct {
	RoleID string `json:"-" uri:"roleID" binding:"required"`
	UserID string `json:"-" uri:"userID" binding:"required"`
}

type RoleApplicationListReq struct {
	RoleID string `json:"roleID" form:"roleID" binding:"required"`
}

type AssignRoleApplicationsReq struct {
	RoleID string   `json:"roleID" binding:"required"`
	AppIDs []string `json:"appIDs" binding:"required,min=1"`
}
