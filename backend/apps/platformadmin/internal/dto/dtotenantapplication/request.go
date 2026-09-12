package dtotenantapplication

import "github.com/morehao/ark-iam/pkg/model"

type TenantApplicationCreateReq struct {
	TenantID     string                        `json:"tenantID" binding:"required"` // 租户ID（平台侧指定归属租户）
	AppID        string                        `json:"appID" binding:"required"`    // 应用ID
	Status       model.TenantApplicationStatus `json:"status"`                      // 状态: enable-启用, disable-停用
	Config       string                        `json:"config"`                      // 租户级应用配置(JSON)
	GrantedScope string                        `json:"grantedScope"`                // 租户级scope授权(JSON)
}

type TenantApplicationUpdateReq struct {
	TenantAppID  string                        `json:"-" uri:"tenantAppID" binding:"required"` // 租户应用订阅ID
	Status       model.TenantApplicationStatus `json:"status"`                                 // 状态
	Config       string                        `json:"config"`                                 // 租户级应用配置(JSON)
	GrantedScope string                        `json:"grantedScope"`                           // 租户级scope授权(JSON)
}

type TenantApplicationDetailReq struct {
	TenantAppID string `json:"-" uri:"tenantAppID" binding:"required"` // 租户应用订阅ID
}

type TenantApplicationDeleteReq struct {
	TenantAppID string `json:"-" uri:"tenantAppID" binding:"required"` // 租户应用订阅ID
}

type TenantApplicationPageListReq struct {
	Page     int                           `json:"page" form:"page"`         // 页码
	PageSize int                           `json:"pageSize" form:"pageSize"` // 每页条数
	TenantID string                        `json:"tenantID" form:"tenantID"` // 租户ID（筛选，留空为全部租户）
	Status   model.TenantApplicationStatus `json:"status" form:"status"`     // 状态
}
