package dtoapplication

type ApplicationCreateReq struct {
	TenantID    uint   `json:"tenantID" form:"tenantID"`       // 租户ID
	Name        string `json:"name" form:"name"`               // 应用名称
	Secret      string `json:"secret" form:"secret"`           // 应用密钥
	Description string `json:"description" form:"description"` // 应用描述
	Type        string `json:"type" form:"type"`               // 应用类型
	IsThirdParty int8  `json:"isThirdParty" form:"isThirdParty"` // 是否第三方应用
}

type ApplicationUpdateReq struct {
	ApplicationID uint   `json:"applicationID" form:"applicationID"` // 应用ID
	TenantID      uint   `json:"tenantID" form:"tenantID"`           // 租户ID
	Name          string `json:"name" form:"name"`                   // 应用名称
	Description   string `json:"description" form:"description"`     // 应用描述
	Type          string `json:"type" form:"type"`                   // 应用类型
	IsThirdParty  int8   `json:"isThirdParty" form:"isThirdParty"`   // 是否第三方应用
}

type ApplicationDeleteReq struct {
	ApplicationID uint `json:"applicationID" form:"applicationID"` // 应用ID
}

type ApplicationDetailReq struct {
	ApplicationID uint `json:"applicationID" form:"applicationID"` // 应用ID
}

type ApplicationPageListReq struct {
	Page        int    `json:"page" form:"page"`         // 页码
	PageSize    int    `json:"pageSize" form:"pageSize"` // 每页数量
	TenantID    uint   `json:"tenantID" form:"tenantID"` // 租户ID
	Name        string `json:"name" form:"name"`         // 应用名称
	Type        string `json:"type" form:"type"`         // 应用类型
	IsThirdParty int8  `json:"isThirdParty" form:"isThirdParty"` // 是否第三方应用
}