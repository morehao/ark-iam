package dtoapplication

import "encoding/json"

type ApplicationCreateResp struct {
	AppID string `json:"appID"` // 应用ID
	Code  string `json:"code"`  // 应用编码
}

type ApplicationDetailResp struct {
	AppID        string          `json:"appID"`        // 应用ID
	Code         string          `json:"code"`         // 应用编码
	Name         string          `json:"name"`         // 应用名称
	Description  string          `json:"description"`  // 应用描述
	LogoURL      string          `json:"logoUrl"`      // 应用logo
	HomepageURL  string          `json:"homepageUrl"`  // 应用主页
	Type         string          `json:"type"`         // 应用类型
	Status       string          `json:"status"`       // 状态
	Visibility   string          `json:"visibility"`   // 可见性
	Sort         int             `json:"sort"`         // 排序
	TenantPolicy json.RawMessage `json:"tenantPolicy"` // 租户策略
	CreatedAt    int64           `json:"createdAt"`    // 创建时间(unix 秒)
}

type PageListItem struct {
	AppID        string          `json:"appID"`        // 应用ID
	Code         string          `json:"code"`         // 应用编码
	Name         string          `json:"name"`         // 应用名称
	Description  string          `json:"description"`  // 应用描述
	Type         string          `json:"type"`         // 应用类型
	Status       string          `json:"status"`       // 状态
	Visibility   string          `json:"visibility"`   // 可见性
	Sort         int             `json:"sort"`         // 排序
	TenantPolicy json.RawMessage `json:"tenantPolicy"` // 租户策略
	CreatedAt    int64           `json:"createdAt"`    // 创建时间(unix 秒)
}

type ApplicationPageListResp struct {
	List  []PageListItem `json:"list"`  // 列表数据
	Total int64          `json:"total"` // 总数
}
