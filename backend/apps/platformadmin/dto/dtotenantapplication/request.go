package dtotenantapplication

import (
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/object/objidentity"
)

type TenantApplicationCreateReq struct {
	TenantID string                        `json:"tenantID" binding:"required"` // 租户ID（平台侧指定归属租户）
	AppID    string                        `json:"appID" binding:"required"`    // 应用ID
	Status   model.TenantApplicationStatus `json:"status"`                      // 状态: enable-启用, disable-停用
	// Operator 由 controller 从鉴权结果填入（json:"-"，客户端不可伪造）。
	Operator objidentity.Operator
}

type TenantApplicationUpdateReq struct {
	TenantAppID string                        `json:"-" uri:"tenantAppID" binding:"required"` // 租户应用订阅ID
	Status      model.TenantApplicationStatus `json:"status"`                                 // 状态
	// Operator 由 controller 从鉴权结果填入（json:"-"，客户端不可伪造）。
	Operator objidentity.Operator
}

type TenantApplicationDetailReq struct {
	TenantAppID string `json:"-" uri:"tenantAppID" binding:"required"` // 租户应用订阅ID
}

type TenantApplicationDeleteReq struct {
	TenantAppID string `json:"-" uri:"tenantAppID" binding:"required"` // 租户应用订阅ID
	// Operator 由 controller 从鉴权结果填入（json:"-"，客户端不可伪造）。
	Operator objidentity.Operator
}

type TenantApplicationPageListReq struct {
	Page     int                           `json:"page" form:"page"`         // 页码
	PageSize int                           `json:"pageSize" form:"pageSize"` // 每页条数
	TenantID string                        `json:"tenantID" form:"tenantID"` // 租户ID（筛选，留空为全部租户）
	Status   model.TenantApplicationStatus `json:"status" form:"status"`     // 状态
}
