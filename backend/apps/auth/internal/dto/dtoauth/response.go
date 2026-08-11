package dtoauth

import (
	"github.com/morehao/ark-iam/pkg/iam/object/objauth"
	"github.com/morehao/golib/biz/gobject"
)

type LoginResp struct {
	SSOSessionID string                `json:"ssoSessionID"`
	Tenants      []objauth.TenantOption `json:"tenants"`
}

type MyTenantsResp struct {
	List []objauth.TenantOption `json:"list"`
}

type RegisterResp struct {
	UserID uint `json:"userID"` // 用户ID
}

type UserinfoResp struct {
	PersonInfo objauth.PersonInfo     `json:"personInfo"`
	UserInfo   objauth.TenantUserInfo `json:"userInfo"`
}

type ConnectorCreateResp struct {
	ConnectorID uint `json:"connectorId"` // 连接器ID
}

type ConnectorDetailResp struct {
	ConnectorID uint `json:"connectorId"` // 连接器ID
	objauth.ConnectorBaseInfo
	gobject.OperatorBaseInfo
}

type ConnectorPageListItem struct {
	ConnectorID uint `json:"connectorId"` // 连接器ID
	objauth.ConnectorBaseInfo
	gobject.OperatorBaseInfo
}

type ConnectorPageListResp struct {
	List  []ConnectorPageListItem `json:"list"`  // 数据列表
	Total int64                  `json:"total"` // 数据总条数
}

type JoinTenantResp struct {
	UserID uint `json:"userID"` // 用户ID
}
