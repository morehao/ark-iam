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

// JoinTenantReq 加入已有租户请求。落哪个租户不由用户指定，而是通过
// inviteCode（邀请码）或应用定向的默认租户决定，禁止裸 tenantID 直入。
type JoinTenantReq struct {
	InviteCode string `json:"inviteCode,omitempty"` // 邀请码（主门禁）
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
