package dtotenantapplication

type TenantApplicationCreateResp struct {
	TenantAppID uint `json:"tenantAppID"` // 租户应用订阅ID
}

type TenantApplicationDetailResp struct {
	TenantAppID  uint   `json:"tenantAppID"`  // 租户应用订阅ID
	TenantID     uint   `json:"tenantID"`     // 租户ID
	AppID        uint   `json:"appID"`        // 应用ID
	Status       string `json:"status"`       // 状态
	Config       string `json:"config"`       // 租户级应用配置(JSON)
	GrantedScope string `json:"grantedScope"` // 租户级scope授权(JSON)
	CreatedAt    string `json:"createdAt"`    // 创建时间
}

type PageListItem struct {
	TenantAppID uint   `json:"tenantAppID"` // 租户应用订阅ID
	TenantID    uint   `json:"tenantID"`    // 租户ID
	AppID       uint   `json:"appID"`       // 应用ID
	Status      string `json:"status"`      // 状态
	CreatedAt   string `json:"createdAt"`   // 创建时间
}

type TenantApplicationPageListResp struct {
	List  []PageListItem `json:"list"`  // 列表数据
	Total int64          `json:"total"` // 总数
}
