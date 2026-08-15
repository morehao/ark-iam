package dtotenant

import (
	"github.com/morehao/ark-iam/pkg/iam/object/objtenant"
	"github.com/morehao/golib/biz/gobject"
)

type SystemCreateReq struct {
	objtenant.SystemBaseInfo
}

type SystemUpdateReq struct {
	SystemID string `json:"-" uri:"systemID" binding:"required"` // 自增ID
	objtenant.SystemBaseInfo
}

type SystemDetailReq struct {
	SystemID string `json:"-" uri:"systemID" binding:"required"` // 自增ID
}

type SystemPageListReq struct {
	gobject.PageQuery
	TenantID string `json:"tenantID" form:"tenantID"` // 租户ID
	Key      string `json:"key" form:"key"`           // 配置键
}

type SystemDeleteReq struct {
	SystemID string `json:"-" uri:"systemID" binding:"required"` // 自增ID
}

type LogDetailReq struct {
	LogID string `json:"-" uri:"logID" binding:"required"` // 日志ID
}

type LogPageListReq struct {
	gobject.PageQuery
	TenantID string `json:"tenantID" form:"tenantID"` // 租户ID
	Key      string `json:"key" form:"key"`           // 日志键
}
