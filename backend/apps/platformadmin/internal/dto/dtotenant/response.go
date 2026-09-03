package dtotenant

import (
	"github.com/morehao/ark-iam/pkg/iam/object/objaudit"
	"github.com/morehao/ark-iam/pkg/iam/object/objtenant"
	"github.com/morehao/golib/biz/gobject"
)

type TenantCreateResp struct {
	TenantID string `json:"tenantID"` // 租户ID
}

type TenantDetailResp struct {
	TenantID string `json:"tenantID"` // 租户ID
	objtenant.TenantBaseInfo
	gobject.OperatorBaseInfo
}

type TenantPageListItem struct {
	TenantID string `json:"tenantID"` // 租户ID
	objtenant.TenantBaseInfo
	gobject.OperatorBaseInfo
}

type TenantPageListResp struct {
	List  []TenantPageListItem `json:"list"`  // 数据列表
	Total int64                `json:"total"` // 数据总条数
}

// ---------- 日志（审计记录） ----------

type LogDetailResp struct {
	LogID string `json:"logID"` // 日志ID
	objaudit.LogBaseInfo
	gobject.OperatorBaseInfo
}

type LogPageListItem struct {
	LogID string `json:"logID"` // 日志ID
	objaudit.LogBaseInfo
	gobject.OperatorBaseInfo
}

type LogPageListResp struct {
	List  []LogPageListItem `json:"list"`  // 数据列表
	Total int64             `json:"total"` // 数据总条数
}
