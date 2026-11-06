package dtoauth

import (
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/object/objauth"
	"github.com/morehao/ark-iam/pkg/object/objidentity"
	"github.com/morehao/golib/biz/gobject"
)

type MyTenantsReq struct {
	// Operator 由 controller 从鉴权结果填入（json:"-"，客户端不可伪造）。
	Operator objidentity.Operator
}

// LogoutReq 无请求体字段：Logout 为 person 级全局登出（撤销全部 refresh token + SSO 会话），
// person 身份取自 access token。曾有的 refreshToken 字段从未被使用，已删除。
type LogoutReq struct {
	// Operator 由 controller 从鉴权结果填入（json:"-"，客户端不可伪造）。
	Operator objidentity.Operator
}

// LogoutAllReq 与 LogoutReq 语义一致（全局登出）。
type LogoutAllReq struct {
	// Operator 由 controller 从鉴权结果填入（json:"-"，客户端不可伪造）。
	Operator objidentity.Operator
}

// JoinTenantReq 加入已有租户请求。落哪个租户不由用户指定，而是通过
// inviteCode（邀请码）或应用定向的默认租户决定，禁止裸 tenantID 直入。
type JoinTenantReq struct {
	InviteCode string `json:"inviteCode,omitempty"` // 邀请码（主门禁）
	// ClientID 是本次请求所用 OIDC 客户端（= access token 的 client_id），
	// 由 controller 从鉴权结果填入，用于解析调用方所属应用的门禁开关。
	ClientID string `json:"-"`
	// Operator 由 controller 从鉴权结果填入（json:"-"，客户端不可伪造）。
	Operator objidentity.Operator
}

type UserinfoReq struct {
	// Operator 由 controller 从鉴权结果填入（json:"-"，客户端不可伪造）。
	Operator objidentity.Operator
}

type ConnectorDeleteReq struct {
	ConnectorID string `json:"-" uri:"connectorID" binding:"required"` // 连接器ID
	// Operator 由 controller 从鉴权结果填入（json:"-"，客户端不可伪造）。
	Operator objidentity.Operator
}

type ConnectorCreateReq struct {
	objauth.ConnectorBaseInfo
	// Operator 由 controller 从鉴权结果填入（json:"-"，客户端不可伪造）。
	Operator objidentity.Operator
}

type ConnectorUpdateReq struct {
	ConnectorID string `json:"-" uri:"connectorID" binding:"required"` // 连接器ID
	objauth.ConnectorBaseInfo
	// Operator 由 controller 从鉴权结果填入（json:"-"，客户端不可伪造）。
	Operator objidentity.Operator
}

type ConnectorDetailReq struct {
	ConnectorID string `json:"-" uri:"connectorID" binding:"required"` // 连接器ID
}

type ConnectorPageListReq struct {
	gobject.PageQuery
	TenantID    string                  `json:"tenantID" form:"tenantID"`       // 租户ID
	Protocol    model.ConnectorProtocol `json:"protocol" form:"protocol"`       // 协议类型
	Provider    model.ConnectorProvider `json:"provider" form:"provider"`       // 提供商
	Status      model.ConnectorStatus   `json:"status" form:"status"`           // 状态
	Name        string                  `json:"name" form:"name"`               // 名称
	DisplayName string                  `json:"displayName" form:"displayName"` // 显示名称
}
