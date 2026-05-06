package dtoauth

import (
	"github.com/morehao/ark-iam/iam/object/objauth"
	"github.com/morehao/golib/biz/gobject"
)

type LoginResp struct {
	objauth.TokenInfo
}

type RegisterResp struct {
	UserID uint `json:"userID"` // 用户ID
}

type RefreshTokenResp struct {
	objauth.TokenInfo
}

type UserinfoResp struct {
	objauth.UserInfo
}

type SsoAuthorizationUrlResp struct {
	AuthorizationUrl string `json:"authorizationUrl"` // 授权地址
}

type SsoCallbackResp struct {
	objauth.TokenInfo
}

type SsoConnectorCreateResp struct {
	SsoConnectorID uint `json:"ssoConnectorID"` // SSO连接器ID
}

type SsoConnectorDetailResp struct {
	SsoConnectorID uint `json:"ssoConnectorID"` // SSO连接器ID
	objauth.SsoConnectorBaseInfo
	gobject.OperatorBaseInfo
}

type SsoConnectorPageListItem struct {
	SsoConnectorID uint `json:"ssoConnectorID"` // SSO连接器ID
	objauth.SsoConnectorBaseInfo
	gobject.OperatorBaseInfo
}

type SsoConnectorPageListResp struct {
	List  []SsoConnectorPageListItem `json:"list"`  // 数据列表
	Total int64                     `json:"total"` // 数据总条数
}

type ConnectorCreateResp struct {
	ConnectorID uint `json:"connectorID"` // 连接器ID
}

type ConnectorDetailResp struct {
	ConnectorID uint `json:"connectorID"` // 连接器ID
	objauth.ConnectorBaseInfo
	gobject.OperatorBaseInfo
}

type ConnectorPageListItem struct {
	ConnectorID uint `json:"connectorID"` // 连接器ID
	objauth.ConnectorBaseInfo
	gobject.OperatorBaseInfo
}

type ConnectorPageListResp struct {
	List  []ConnectorPageListItem `json:"list"`  // 数据列表
	Total int64                  `json:"total"` // 数据总条数
}
