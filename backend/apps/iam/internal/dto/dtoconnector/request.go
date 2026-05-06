package dtoconnector

import (
	"github.com/morehao/ark-iam/iam/object/objconnector"
	"github.com/morehao/golib/biz/gobject"
)

type ConnectorCreateReq struct {
	objconnector.ConnectorBaseInfo
}

type ConnectorUpdateReq struct {
	ConnectorID uint `json:"connectorID" binding:"required"` // 连接器ID
	objconnector.ConnectorBaseInfo
}

type ConnectorDetailReq struct {
	ConnectorID uint `json:"connectorID" binding:"required"` // 连接器ID
}

type ConnectorPageListReq struct {
	gobject.PageQuery
	TenantID    uint   `json:"tenantID"`    // 租户ID
	ConnectorID string `json:"connectorID"` // 连接器ID
}

type ConnectorDeleteReq struct {
	ConnectorID uint `json:"connectorID" binding:"required"` // 连接器ID
}