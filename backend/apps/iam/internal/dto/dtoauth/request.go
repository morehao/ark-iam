package dtoauth

import (
	"github.com/morehao/ark-iam/iam/object/objauth"
	"github.com/morehao/golib/biz/gobject"
)

type LoginReq struct {
	TenantID   uint   `json:"tenantID" binding:"required"`     // 租户ID
	Identifier string `json:"identifier" binding:"required"`   // 用户名/邮箱/手机号
	Password   string `json:"password" binding:"required"`     // 密码
}

type RegisterReq struct {
	TenantID     uint   `json:"tenantID" binding:"required"`       // 租户ID
	Username     string `json:"username" binding:"required"`         // 用户名
	PrimaryEmail string `json:"primaryEmail"`                       // 主要邮箱
	PrimaryPhone string `json:"primaryPhone"`                       // 主要手机号
	Password     string `json:"password" binding:"required"`       // 密码
	Name         string `json:"name"`                             // 姓名
}

type RefreshTokenReq struct {
	RefreshToken string `json:"refreshToken" binding:"required"` // 刷新令牌
}

type LogoutReq struct {
	RefreshToken string `json:"refreshToken"` // 刷新令牌
}

type UserinfoReq struct {
}

type SsoAuthorizationUrlReq struct {
	ConnectorID string `json:"connectorId" binding:"required"` // 连接器ID
}

type SsoCallbackReq struct {
	ConnectorID string `json:"connectorId" binding:"required"` // 连接器ID
	Code        string `json:"code" binding:"required"`       // 授权码
	State       string `json:"state"`                         // 状态
}

type AssignDepartmentsReq struct {
	UserID        uint   `json:"userID" binding:"required"`         // 用户ID
	DepartmentIDs []uint `json:"departmentIDs" binding:"required"`   // 部门ID列表
}

type ConnectorDeleteReq struct {
	ConnectorID uint `json:"connectorID" binding:"required"` // 连接器ID
}

type SsoConnectorCreateReq struct {
	objauth.SsoConnectorBaseInfo
}

type SsoConnectorUpdateReq struct {
	SsoConnectorID uint `json:"ssoConnectorID" binding:"required"` // SSO连接器ID
	objauth.SsoConnectorBaseInfo
}

type SsoConnectorDetailReq struct {
	SsoConnectorID uint `json:"ssoConnectorID" binding:"required"` // SSO连接器ID
}

type SsoConnectorPageListReq struct {
	gobject.PageQuery
	TenantID      uint   `json:"tenantID"`      // 租户ID
	ProviderName  string `json:"providerName"`  // 提供商名称
	ConnectorName string `json:"connectorName"` // 连接器名称
}

type SsoConnectorDeleteReq struct {
	SsoConnectorID uint `json:"ssoConnectorID" binding:"required"` // SSO连接器ID
}

type ConnectorCreateReq struct {
	objauth.ConnectorBaseInfo
}

type ConnectorUpdateReq struct {
	ConnectorID uint `json:"connectorID" binding:"required"` // 连接器ID
	objauth.ConnectorBaseInfo
}

type ConnectorDetailReq struct {
	ConnectorID uint `json:"connectorID" binding:"required"` // 连接器ID
}

type ConnectorPageListReq struct {
	gobject.PageQuery
	TenantID    uint   `json:"tenantID"`    // 租户ID
	ConnectorID string `json:"connectorID"` // 连接器ID
}
