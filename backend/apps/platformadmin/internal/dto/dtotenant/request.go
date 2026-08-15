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

type TenantCreateAsOwnerReq struct {
	// PersonID 自然人ID（由服务端从认证上下文注入，不信任客户端请求体）
	PersonID string `json:"-"`
	Name     string `json:"name" binding:"required"` // 租户名称
	AppID    string `json:"appID"`                   // 应用ID（可选，订阅该应用）
}






