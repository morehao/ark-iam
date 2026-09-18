package dtotenantapplication

import "github.com/morehao/ark-iam/pkg/model"

type TenantApplicationCreateResp struct {
	TenantAppID string `json:"tenantAppID"` // 租户应用订阅ID
}

type TenantApplicationDetailResp struct {
	TenantAppID  string                        `json:"tenantAppID"`  // 租户应用订阅ID
	TenantID     string                        `json:"tenantID"`     // 租户ID
	TenantName   string                        `json:"tenantName"`   // 租户名称
	AppID        string                        `json:"appID"`        // 应用ID
	AppName      string                        `json:"appName"`      // 应用名称
	AppSource    model.AppSource               `json:"appSource"`    // 所属应用来源（builtin=内置应用，其订阅由系统开通、不可删除）
	Status       model.TenantApplicationStatus `json:"status"`       // 状态
	CreatedAt    int64                         `json:"createdAt"`    // 创建时间(unix 秒)
}

type PageListItem struct {
	TenantAppID string                        `json:"tenantAppID"` // 租户应用订阅ID
	TenantID    string                        `json:"tenantID"`    // 租户ID
	TenantName  string                        `json:"tenantName"`  // 租户名称
	AppID       string                        `json:"appID"`       // 应用ID
	AppName     string                        `json:"appName"`     // 应用名称
	AppSource   model.AppSource               `json:"appSource"`   // 所属应用来源（builtin=内置应用，其订阅由系统开通、不可删除）
	Status      model.TenantApplicationStatus `json:"status"`      // 状态
	CreatedAt   int64                         `json:"createdAt"`   // 创建时间(unix 秒)
	UpdatedAt   int64                         `json:"updatedAt"`   // 更新时间(unix 秒)
}

type TenantApplicationPageListResp struct {
	List  []PageListItem `json:"list"`  // 列表数据
	Total int64          `json:"total"` // 总数
}
