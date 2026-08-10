package dtouser

import (
	"github.com/morehao/ark-iam/pkg/iam/object/objuser"
	"github.com/morehao/golib/biz/gobject"
)

type UserCreateReq struct {
	objuser.UserBaseInfo
}

type UserUpdateReq struct {
	UserID uint `json:"userID" binding:"required"` // 用户ID
	objuser.UserBaseInfo
}

type UserDetailReq struct {
	UserID uint `json:"userID" form:"userID" binding:"required"` // 用户ID
}

type UserPageListReq struct {
	gobject.PageQuery
	TenantID     uint   `json:"tenantID"`     // 租户ID
	Username     string `json:"username"`     // 用户名
	PrimaryEmail string `json:"primaryEmail"` // 主要邮箱
	PrimaryPhone string `json:"primaryPhone"` // 主要手机号
	Name         string `json:"name"`         // 姓名
	IsSuspended  *int8  `json:"isSuspended"`  // 是否挂起
}

type UserDeleteReq struct {
	UserID uint `json:"userID" binding:"required"` // 用户ID
}

type UserPasswordUpdateReq struct {
	UserID uint `json:"userID" binding:"required"` // 用户ID
	objuser.UserPasswordInfo
}

type UserStatusUpdateReq struct {
	UserID       uint `json:"userID" binding:"required"` // 用户ID
	IsSuspended int8 `json:"isSuspended"`              // 是否挂起
}

type UserIdentityCreateReq struct {
	TenantID   uint   `json:"tenantID" binding:"required"`   // 租户ID
	UserID     uint   `json:"userID" binding:"required"`     // 用户ID
	Issuer     string `json:"issuer" binding:"required"`     // 身份提供商
	IdentityID string `json:"identityID" binding:"required"` // 第三方用户ID
	Detail     any    `json:"detail"`                         // 详细信息
}

type UserIdentityUpdateReq struct {
	UserIdentityID uint `json:"userIdentityID" binding:"required"` // 用户身份ID
	TenantID       uint   `json:"tenantID"`   // 租户ID
	UserID         uint   `json:"userID"`     // 用户ID
	Issuer         string `json:"issuer"`     // 身份提供商
	IdentityID     string `json:"identityID"` // 第三方用户ID
	Detail         any    `json:"detail"`       // 详细信息
}

type UserIdentityDetailReq struct {
	UserIdentityID uint `json:"userIdentityID" form:"userIdentityID" binding:"required"` // 用户身份ID
}

type UserIdentityPageListReq struct {
	gobject.PageQuery
	TenantID   uint   `json:"tenantID"`   // 租户ID
	UserID     uint   `json:"userID"`     // 用户ID
	Issuer     string `json:"issuer"`     // 身份提供商
	IdentityID string `json:"identityID"` // 第三方用户ID
}

type UserIdentityDeleteReq struct {
	UserIdentityID uint `json:"userIdentityID" binding:"required"` // 用户身份ID
}

type UserIdentityByUserReq struct {
	UserID uint `json:"userID" form:"userID" binding:"required"` // 用户ID
}

type UserLoginLogDetailReq struct {
	UserLoginLogID uint `json:"userLoginLogID" form:"userLoginLogID" binding:"required"` // 登录日志ID
}

type UserLoginLogPageListReq struct {
	gobject.PageQuery
	TenantID uint   `json:"tenantID"` // 租户ID
	UserID   uint   `json:"userID"`   // 用户ID
	LoginIP  string `json:"loginIP"`  // 登录IP地址
}

type UserLoginLogByUserReq struct {
	UserID uint `json:"userID" form:"userID" binding:"required"` // 用户ID
}

type UserDepartmentByUserReq struct {
	UserID uint `json:"userID" form:"userID" binding:"required"` // 用户ID
}

type AssignDepartmentsReq struct {
	UserID        uint   `json:"userID" binding:"required"`         // 用户ID
	DepartmentIDs []uint `json:"departmentIDs" binding:"required"`   // 部门ID列表
}

type RoleUserListReq struct {
	RoleID uint64 `json:"roleId" form:"roleId" binding:"required"`
}

type AssignRoleUsersReq struct {
	RoleID  uint64   `json:"roleId" binding:"required"`
	UserIDs []uint64 `json:"userIds" binding:"required,min=1"`
}

type RemoveRoleUserReq struct {
	RoleID uint64 `json:"roleId" uri:"roleId" binding:"required"`
	UserID uint64 `json:"userId" uri:"userId" binding:"required"`
}

type RoleApplicationListReq struct {
	RoleID uint64 `json:"roleId" form:"roleId" binding:"required"`
}

type AssignRoleApplicationsReq struct {
	RoleID         uint64   `json:"roleId" binding:"required"`
	AppIDs []uint64 `json:"appIds" binding:"required,min=1"`
}
