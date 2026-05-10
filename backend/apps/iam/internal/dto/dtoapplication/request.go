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

type ApplicationRoleListReq struct {
	ApplicationID uint `json:"applicationId" form:"applicationId" binding:"required"`
}

type AssignApplicationRolesReq struct {
	ApplicationID uint64   `json:"applicationId" binding:"required"`
	RoleIDs       []uint64 `json:"roleIds" binding:"required,min=1"`
}

type RemoveApplicationRoleReq struct {
	ApplicationID uint64 `json:"applicationId" form:"applicationId" binding:"required"`
	RoleID        uint64 `json:"roleId" uri:"roleId" binding:"required"`
}

type ApplicationSecretListReq struct {
	ApplicationID uint `json:"applicationId" form:"applicationId" binding:"required"`
}

type CreateApplicationSecretReq struct {
	ApplicationID uint   `json:"applicationId" binding:"required"`
	Name          string `json:"name" binding:"required"`
	ExpiresAt     string `json:"expiresAt"`
}

type DeleteApplicationSecretReq struct {
	SecretID uint64 `json:"secretId" uri:"secretId" binding:"required"`
}
