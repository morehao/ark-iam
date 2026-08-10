package dtoauth

import (
	"github.com/morehao/ark-iam/iam/object/objauth"
	"github.com/morehao/golib/biz/gobject"
)

type MyTenantsReq struct {
}

type LogoutAllReq struct {
	RefreshToken string `json:"refreshToken"` // 刷新令牌
}

type RegisterReq struct {
	TenantID     uint   `json:"tenantID" binding:"required"` // 租户ID
	Username     string `json:"username"`                     // 用户名
	PrimaryEmail string `json:"primaryEmail"`                 // 主要邮箱
	PrimaryPhone string `json:"primaryPhone"`                 // 主要手机号
	Password     string `json:"password" binding:"required"` // 密码
	Name         string `json:"name"`                         // 姓名
}

type LogoutReq struct {
	RefreshToken string `json:"refreshToken"` // 刷新令牌
}

type JoinTenantReq struct {
	TenantID uint `json:"tenantID" binding:"required"` // 租户ID
}

type UserinfoReq struct {
}

type AssignDepartmentsReq struct {
	UserID        uint   `json:"userID" binding:"required"`         // 用户ID
	DepartmentIDs []uint `json:"departmentIDs" binding:"required"`   // 部门ID列表
}

type ConnectorDeleteReq struct {
	ConnectorID uint `json:"connectorId" binding:"required"` // 连接器ID
}

type ConnectorCreateReq struct {
	objauth.ConnectorBaseInfo
}

type ConnectorUpdateReq struct {
	ConnectorID uint `json:"connectorId" binding:"required"` // 连接器ID
	objauth.ConnectorBaseInfo
}

type ConnectorDetailReq struct {
	ConnectorID uint `json:"connectorId" form:"connectorId" binding:"required"` // 连接器ID
}

type ConnectorPageListReq struct {
	gobject.PageQuery
	TenantID    uint   `json:"tenantId"`    // 租户ID
	Protocol    string `json:"protocol"`    // 协议类型
	Provider    string `json:"provider"`    // 提供商
	Status      string `json:"status"`      // 状态
	Name        string `json:"name"`        // 名称
	DisplayName string `json:"displayName"` // 显示名称
}
