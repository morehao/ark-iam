package dtolog

import (
	"github.com/morehao/golib/biz/gobject"
)

type LogDetailReq struct {
	LogID uint `json:"logID" binding:"required"` // 日志ID
}

type LogPageListReq struct {
	gobject.PageQuery
	TenantID uint   `json:"tenantID"` // 租户ID
	Key      string `json:"key"`     // 日志键
}