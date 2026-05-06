package dtotenant

import (
	"github.com/morehao/ark-iam/iam/object/objtenant"
	"github.com/morehao/golib/biz/gobject"
)

type TenantCreateResp struct {
	TenantID uint `json:"tenantID"` // 租户ID
}

type TenantDetailResp struct {
	TenantID uint `json:"tenantID"` // 租户ID
	objtenant.TenantBaseInfo
	gobject.OperatorBaseInfo
}

type TenantPageListItem struct {
	TenantID uint `json:"tenantID"` // 租户ID
	objtenant.TenantBaseInfo
	gobject.OperatorBaseInfo
}

type TenantPageListResp struct {
	List  []TenantPageListItem `json:"list"`  // 数据列表
	Total int64               `json:"total"` // 数据总条数
}