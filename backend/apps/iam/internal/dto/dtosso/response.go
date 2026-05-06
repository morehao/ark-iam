package dtosso

import (
	"github.com/morehao/ark-iam/iam/object/objsso"
	"github.com/morehao/golib/biz/gobject"
)

type SsoConnectorCreateResp struct {
	SsoConnectorID uint `json:"ssoConnectorID"` // SSO连接器ID
}

type SsoConnectorDetailResp struct {
	SsoConnectorID uint `json:"ssoConnectorID"` // SSO连接器ID
	objsso.SsoConnectorBaseInfo
	gobject.OperatorBaseInfo
}

type SsoConnectorPageListItem struct {
	SsoConnectorID uint `json:"ssoConnectorID"` // SSO连接器ID
	objsso.SsoConnectorBaseInfo
	gobject.OperatorBaseInfo
}

type SsoConnectorPageListResp struct {
	List  []SsoConnectorPageListItem `json:"list"`  // 数据列表
	Total int64                     `json:"total"` // 数据总条数
}