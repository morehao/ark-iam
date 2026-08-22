package dtotenantapplication

type TenantApplicationCreateResp struct {
	TenantAppID string `json:"tenantAppID"` // 租户应用订阅ID
}

type TenantApplicationDetailResp struct {
	TenantAppID  string `json:"tenantAppID"`  // 租户应用订阅ID
	TenantID     string `json:"tenantID"`     // 租户ID
	AppID        string `json:"appID"`        // 应用ID
	Status       string `json:"status"`       // 状态
	Config       string `json:"config"`       // 租户级应用配置(JSON)
	GrantedScope string `json:"grantedScope"` // 租户级scope授权(JSON)
	CreatedAt    int64  `json:"createdAt"`    // 创建时间(unix 秒)
}

type PageListItem struct {
	TenantAppID string `json:"tenantAppID"` // 租户应用订阅ID
	TenantID    string `json:"tenantID"`    // 租户ID
	AppID       string `json:"appID"`       // 应用ID
	Status      string `json:"status"`      // 状态
	CreatedAt   int64  `json:"createdAt"`   // 创建时间(unix 秒)
}

type TenantApplicationPageListResp struct {
	List  []PageListItem `json:"list"`  // 列表数据
	Total int64          `json:"total"` // 总数
}
