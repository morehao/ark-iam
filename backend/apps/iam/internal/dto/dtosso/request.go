package dtosso

import (
	"github.com/morehao/ark-iam/iam/object/objsso"
	"github.com/morehao/golib/biz/gobject"
)

type SsoConnectorCreateReq struct {
	objsso.SsoConnectorBaseInfo
}

type SsoConnectorUpdateReq struct {
	SsoConnectorID uint `json:"ssoConnectorID" binding:"required"` // SSO连接器ID
	objsso.SsoConnectorBaseInfo
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