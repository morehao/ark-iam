package dtotenantapplication

type TenantApplicationCreateReq struct {
	AppID        uint   `json:"appID" binding:"required"` // 应用ID
	Status       string `json:"status"`                   // 状态: enable-启用, disable-停用
	Config       string `json:"config"`                   // 租户级应用配置(JSON)
	GrantedScope string `json:"grantedScope"`             // 租户级scope授权(JSON)
}

type TenantApplicationUpdateReq struct {
	TenantAppID  uint   `json:"tenantAppID" binding:"required"` // 租户应用订阅ID
	Status       string `json:"status"`                         // 状态
	Config       string `json:"config"`                         // 租户级应用配置(JSON)
	GrantedScope string `json:"grantedScope"`                   // 租户级scope授权(JSON)
}

type TenantApplicationDetailReq struct {
	TenantAppID uint `form:"tenantAppID" binding:"required"` // 租户应用订阅ID
}

type TenantApplicationDeleteReq struct {
	TenantAppID uint `json:"tenantAppID" binding:"required"` // 租户应用订阅ID
}

type TenantApplicationPageListReq struct {
	Page     int    `json:"page"`     // 页码
	PageSize int    `json:"pageSize"` // 每页条数
	Status   string `json:"status"`   // 状态
}
