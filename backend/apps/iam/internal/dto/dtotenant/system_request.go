package dtotenant

import (
	"github.com/morehao/ark-iam/iam/object/objtenant"
	"github.com/morehao/golib/biz/gobject"
)

type SystemCreateReq struct {
	objtenant.SystemBaseInfo
}

type SystemUpdateReq struct {
	SystemID uint `json:"systemID" binding:"required"` // 自增ID
	objtenant.SystemBaseInfo
}

type SystemDetailReq struct {
	SystemID uint `json:"systemID" binding:"required"` // 自增ID
}

type SystemPageListReq struct {
	gobject.PageQuery
	TenantID uint   `json:"tenantID"` // 租户ID
	Key      string `json:"key"`     // 配置键
}

type SystemDeleteReq struct {
	SystemID uint `json:"systemID" binding:"required"` // 自增ID
}

type LogDetailReq struct {
	LogID uint `json:"logID" binding:"required"` // 日志ID
}

type LogPageListReq struct {
	gobject.PageQuery
	TenantID uint   `json:"tenantID"` // 租户ID
	Key      string `json:"key"`     // 日志键
}
