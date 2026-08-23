package dtoauth

import (
	"github.com/morehao/ark-iam/pkg/iam/object/objauth"
	"github.com/morehao/golib/biz/gobject"
)

type LoginResp struct {
	SSOSessionID string                 `json:"ssoSessionID"`
	Tenants      []objauth.TenantOption `json:"tenants"`
}

type MyTenantsResp struct {
	List []objauth.TenantOption `json:"list"`
}

type RegisterResp struct {
	UserID    string `json:"userID"`    // 用户ID
	TenantID  string `json:"tenantID"`  // 新开通的租户ID
	SessionID string `json:"sessionID"` // 注册即登录的 SSO 会话ID（可能为空）
}

type UserinfoResp struct {
	PersonInfo objauth.PersonInfo     `json:"personInfo"`
	UserInfo   objauth.TenantUserInfo `json:"userInfo"`
}

type ConnectorCreateResp struct {
	ConnectorID string `json:"connectorID"` // 连接器ID
}

type ConnectorDetailResp struct {
	ConnectorID string `json:"connectorID"` // 连接器ID
	objauth.ConnectorBaseInfo
	gobject.OperatorBaseInfo
}

type ConnectorPageListItem struct {
	ConnectorID string `json:"connectorID"` // 连接器ID
	objauth.ConnectorBaseInfo
	gobject.OperatorBaseInfo
}

type ConnectorPageListResp struct {
	List  []ConnectorPageListItem `json:"list"`  // 数据列表
	Total int64                   `json:"total"` // 数据总条数
}

type JoinTenantResp struct {
	UserID string `json:"userID"` // 用户ID
}
