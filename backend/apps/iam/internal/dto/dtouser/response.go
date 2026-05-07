package dtouser

import (
	"github.com/morehao/ark-iam/iam/object/objuser"
	"github.com/morehao/golib/biz/gobject"
)

type UserCreateResp struct {
	UserID uint `json:"userID"` // 用户ID
}

type UserDetailResp struct {
	UserID uint `json:"userID"` // 用户ID
	objuser.UserBaseInfo
	gobject.OperatorBaseInfo
}

type UserPageListItem struct {
	UserID uint `json:"userID"` // 用户ID
	objuser.UserBaseInfo
	gobject.OperatorBaseInfo
}

type UserPageListResp struct {
	List  []UserPageListItem `json:"list"`  // 数据列表
	Total int64            `json:"total"` // 数据总条数
}

type UserIdentityCreateResp struct {
	UserIdentityID uint `json:"userIdentityID"` // 用户身份ID
}

type UserIdentityDetailResp struct {
	UserIdentityID uint `json:"userIdentityID"` // 用户身份ID
	TenantID      uint   `json:"tenantID"`     // 租户ID
	UserID        uint   `json:"userID"`       // 用户ID
	Issuer        string `json:"issuer"`       // 身份提供商
	IdentityID    string `json:"identityID"`  // 第三方用户ID
	Detail        any    `json:"detail"`       // 详细信息
	gobject.OperatorBaseInfo
}

type UserIdentityPageListItem struct {
	UserIdentityID uint `json:"userIdentityID"` // 用户身份ID
	TenantID      uint   `json:"tenantID"`     // 租户ID
	UserID        uint   `json:"userID"`       // 用户ID
	Issuer        string `json:"issuer"`       // 身份提供商
	IdentityID    string `json:"identityID"`  // 第三方用户ID
	Detail        any    `json:"detail"`       // 详细信息
	gobject.OperatorBaseInfo
}

type UserIdentityPageListResp struct {
	List  []UserIdentityPageListItem `json:"list"`  // 数据列表
	Total int64                   `json:"total"` // 数据总条数
}

type UserLoginLogDetailResp struct {
	UserLoginLogID uint `json:"userLoginLogID"` // 登录日志ID
	TenantID       uint   `json:"tenantID"`       // 租户ID
	UserID         uint   `json:"userID"`         // 用户ID
	LoginIP        string `json:"loginIP"`       // 登录IP地址
	UserAgent      string `json:"userAgent"`     // 用户代理信息
	LoginTime      int64  `json:"loginTime"`     // 登录时间
	gobject.OperatorBaseInfo
}

type UserLoginLogPageListItem struct {
	UserLoginLogID uint `json:"userLoginLogID"` // 登录日志ID
	TenantID       uint   `json:"tenantID"`       // 租户ID
	UserID         uint   `json:"userID"`         // 用户ID
	LoginIP        string `json:"loginIP"`       // 登录IP地址
	UserAgent      string `json:"userAgent"`     // 用户代理信息
	LoginTime      int64  `json:"loginTime"`     // 登录时间
	gobject.OperatorBaseInfo
}

type UserLoginLogPageListResp struct {
	List  []UserLoginLogPageListItem `json:"list"`  // 数据列表
	Total int64                   `json:"total"` // 数据总条数
}

type UserDepartmentRelationCreateResp struct {
	UserDepartmentRelationID uint `json:"userDepartmentRelationID"` // 用户部门关系ID
}

type UserDepartmentRelationDetailResp struct {
	UserDepartmentRelationID uint `json:"userDepartmentRelationID"` // 用户部门关系ID
	TenantID                 uint   `json:"tenantID"`         // 租户ID
	UserID                   uint   `json:"userID"`           // 用户ID
	DepartmentID             uint   `json:"departmentID"`     // 部门ID
	IsPrimary                int8   `json:"isPrimary"`         // 是否主部门
	gobject.OperatorBaseInfo
}

type UserDepartmentRelationPageListItem struct {
	UserDepartmentRelationID uint `json:"userDepartmentRelationID"` // 用户部门关系ID
	TenantID                 uint   `json:"tenantID"`         // 租户ID
	UserID                   uint   `json:"userID"`           // 用户ID
	DepartmentID             uint   `json:"departmentID"`     // 部门ID
	IsPrimary                int8   `json:"isPrimary"`         // 是否主部门
	gobject.OperatorBaseInfo
}

type UserDepartmentRelationPageListResp struct {
	List  []UserDepartmentRelationPageListItem `json:"list"`  // 数据列表
	Total int64                            `json:"total"` // 数据总条数
}

type RoleUserResp struct {
	UserID    uint64 `json:"userId"`
	Username  string `json:"username"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	RoleID    uint64 `json:"roleId"`
	CreatedAt string `json:"createdAt"`
}

type RoleUserListResp struct {
	Total int64           `json:"total"`
	Users []RoleUserResp `json:"users"`
}

type RoleApplicationResp struct {
	ApplicationID uint64 `json:"applicationId"`
	AppName       string `json:"appName"`
	AppType       string `json:"appType"`
	RoleID        uint64 `json:"roleId"`
	CreatedAt     string `json:"createdAt"`
}

type RoleApplicationListResp struct {
	Total        int64                   `json:"total"`
	Applications []RoleApplicationResp `json:"applications"`
}