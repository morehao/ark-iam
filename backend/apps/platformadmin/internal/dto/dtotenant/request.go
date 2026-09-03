package dtotenant

import (
	"github.com/morehao/ark-iam/pkg/iam/object/objtenant"
	"github.com/morehao/golib/biz/gobject"
)

type TenantCreateReq struct {
	objtenant.TenantBaseInfo
}

type TenantUpdateReq struct {
	TenantID string `json:"-" uri:"tenantID" binding:"required"` // 租户ID
	objtenant.TenantBaseInfo
}

type TenantDetailReq struct {
	TenantID string `json:"-" uri:"tenantID" binding:"required"` // 租户ID
}

type TenantPageListReq struct {
	gobject.PageQuery
}

type TenantDeleteReq struct {
	TenantID string `json:"-" uri:"tenantID" binding:"required"` // 租户ID
}

// ---------- 日志（审计记录） ----------

type LogDetailReq struct {
	LogID string `json:"-" uri:"logID" binding:"required"` // 日志ID
}

type LogPageListReq struct {
	gobject.PageQuery
	TenantID string `json:"tenantID" form:"tenantID"` // 租户ID
	Key      string `json:"key" form:"key"`           // 日志键
}
