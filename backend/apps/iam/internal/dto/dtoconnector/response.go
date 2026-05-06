package dtoconnector

import (
	"github.com/morehao/ark-iam/iam/object/objconnector"
	"github.com/morehao/golib/biz/gobject"
)

type ConnectorCreateResp struct {
	ConnectorID uint `json:"connectorID"` // 连接器ID
}

type ConnectorDetailResp struct {
	ConnectorID uint `json:"connectorID"` // 连接器ID
	objconnector.ConnectorBaseInfo
	gobject.OperatorBaseInfo
}

type ConnectorPageListItem struct {
	ConnectorID uint `json:"connectorID"` // 连接器ID
	objconnector.ConnectorBaseInfo
	gobject.OperatorBaseInfo
}

type ConnectorPageListResp struct {
	List  []ConnectorPageListItem `json:"list"`  // 数据列表
	Total int64                  `json:"total"` // 数据总条数
}