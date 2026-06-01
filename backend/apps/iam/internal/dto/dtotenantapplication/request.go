package dtotenantapplication

type CreateReq struct {
	AppID  uint   `json:"appId" binding:"required"`  // 应用ID
	Status string `json:"status"`                     // 状态: enable-启用, disable-停用
	Config string `json:"config"`                     // 租户级应用配置(JSON)
}

type UpdateReq struct {
	TenantAppID uint   `json:"tenantAppId" binding:"required"` // 租户应用订阅ID
	Status      string `json:"status"`                          // 状态
	Config      string `json:"config"`                          // 租户级应用配置(JSON)
}

type DetailReq struct {
	TenantAppID uint `form:"tenantAppId" binding:"required"` // 租户应用订阅ID
}

type DeleteReq struct {
	TenantAppID uint `json:"tenantAppId" binding:"required"` // 租户应用订阅ID
}

type PageListReq struct {
	Page     int    `form:"page"`      // 页码
	PageSize int    `form:"pageSize"`  // 每页条数
	Status   string `form:"status"`    // 状态
}
