package dtotenant

import (
	"github.com/morehao/ark-iam/iam/object/objtenant"
	"github.com/morehao/golib/biz/gobject"
)

type TenantCreateReq struct {
	objtenant.TenantBaseInfo
}

type TenantUpdateReq struct {
	TenantID uint `json:"tenantID" binding:"required"` // 租户ID
	objtenant.TenantBaseInfo
}

type TenantDetailReq struct {
	TenantID uint `json:"tenantID" binding:"required"` // 租户ID
}

type TenantPageListReq struct {
	gobject.PageQuery
}

type TenantDeleteReq struct {
	TenantID uint `json:"tenantID" binding:"required"` // 租户ID
}