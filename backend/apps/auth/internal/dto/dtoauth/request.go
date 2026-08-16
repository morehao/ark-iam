package dtoauth

import (
	"github.com/morehao/ark-iam/pkg/iam/object/objauth"
	"github.com/morehao/golib/biz/gobject"
)

type MyTenantsReq struct {
}

// LogoutReq 无请求体字段：Logout 为 person 级全局登出（撤销全部 refresh token + SSO 会话），
// person 身份取自 access token。曾有的 refreshToken 字段从未被使用，已删除。
type LogoutReq struct {
}

// LogoutAllReq 与 LogoutReq 语义一致（全局登出）。
type LogoutAllReq struct {
}

type JoinTenantReq struct {
	TenantID string `json:"tenantID" binding:"required"` // 租户ID
}

// RegisterReq 注册请求。Username/PrimaryEmail/PrimaryPhone 至少提供一个；
// 字段长度上限与全局标识一致（防超长字符串入库）。
type RegisterReq struct {
	TenantID     string `json:"tenantID" binding:"required"`         // 租户ID
	Username     string `json:"username" binding:"max=128"`          // 用户名
	PrimaryEmail string `json:"primaryEmail" binding:"max=128"`      // 主要邮箱
	PrimaryPhone string `json:"primaryPhone" binding:"max=32"`       // 主要手机号
	Password     string `json:"password" binding:"required,max=128"` // 密码
	Name         string `json:"name" binding:"max=128"`              // 姓名
}

type UserinfoReq struct {
}

type AssignDepartmentsReq struct {
	UserID        string   `json:"userID" binding:"required"`        // 用户ID
	DepartmentIDs []string `json:"departmentIDs" binding:"required"` // 部门ID列表
}

type ConnectorDeleteReq struct {
	ConnectorID string `json:"-" uri:"connectorID" binding:"required"` // 连接器ID
}

type ConnectorCreateReq struct {
	objauth.ConnectorBaseInfo
}

type ConnectorUpdateReq struct {
	ConnectorID string `json:"-" uri:"connectorID" binding:"required"` // 连接器ID
	objauth.ConnectorBaseInfo
}

type ConnectorDetailReq struct {
	ConnectorID string `json:"-" uri:"connectorID" binding:"required"` // 连接器ID
}

type ConnectorPageListReq struct {
	gobject.PageQuery
	TenantID    string `json:"tenantID" form:"tenantID"`       // 租户ID
	Protocol    string `json:"protocol" form:"protocol"`       // 协议类型
	Provider    string `json:"provider" form:"provider"`       // 提供商
	Status      string `json:"status" form:"status"`           // 状态
	Name        string `json:"name" form:"name"`               // 名称
	DisplayName string `json:"displayName" form:"displayName"` // 显示名称
}
