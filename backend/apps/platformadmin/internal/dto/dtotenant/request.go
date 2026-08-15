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






