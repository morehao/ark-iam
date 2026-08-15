package dtotenant

type OrganizationCreateReq struct {
	TenantID      string `json:"tenantID" form:"tenantID"`           // 租户ID
	Name          string `json:"name" form:"name"`                   // 组织名称
	Description   string `json:"description" form:"description"`     // 组织描述
	IsMFARequired int8   `json:"isMFARequired" form:"isMFARequired"` // 是否需要MFA
}

type OrganizationUpdateReq struct {
	OrganizationID string `json:"-" uri:"organizationID"`             // 组织ID
	TenantID       string `json:"tenantID" form:"tenantID"`           // 租户ID
	Name           string `json:"name" form:"name"`                   // 组织名称
	Description    string `json:"description" form:"description"`     // 组织描述
	IsMFARequired  int8   `json:"isMFARequired" form:"isMFARequired"` // 是否需要MFA
}

type OrganizationDeleteReq struct {
	OrganizationID string `json:"-" uri:"organizationID"` // 组织ID
}

type OrganizationDetailReq struct {
	OrganizationID string `json:"-" uri:"organizationID"` // 组织ID
}

type OrganizationPageListReq struct {
	Page     int    `json:"page" form:"page"`         // 页码
	PageSize int    `json:"pageSize" form:"pageSize"` // 每页数量
	TenantID string `json:"tenantID" form:"tenantID"` // 租户ID
	Name     string `json:"name" form:"name"`         // 组织名称
}

type OrganizationRoleCreateReq struct {
	TenantID       string `json:"tenantID" form:"tenantID"`                                // 租户ID
	OrganizationID string `json:"organizationID" form:"organizationID" binding:"required"` // 组织ID
	Name           string `json:"name" form:"name"`                                        // 角色名称
	Description    string `json:"description" form:"description"`                          // 角色描述
	Type           string `json:"type" form:"type"`                                        // 角色类型
}

type OrganizationRoleUpdateReq struct {
	OrganizationRoleID string `json:"-" binding:"required" uri:"organizationRoleID"`           // 组织角色ID
	TenantID           string `json:"tenantID" form:"tenantID"`                                // 租户ID
	OrganizationID     string `json:"organizationID" form:"organizationID" binding:"required"` // 组织ID
	Name               string `json:"name" form:"name"`                                        // 角色名称
	Description        string `json:"description" form:"description"`                          // 角色描述
	Type               string `json:"type" form:"type"`                                        // 角色类型
}

type OrganizationRoleDeleteReq struct {
	OrganizationRoleID string `json:"-" uri:"organizationRoleID"` // 组织角色ID
}

type OrganizationRoleDetailReq struct {
	OrganizationRoleID string `json:"-" uri:"organizationRoleID"` // 组织角色ID
}

type OrganizationRolePageListReq struct {
	Page           int    `json:"page" form:"page"`                     // 页码
	PageSize       int    `json:"pageSize" form:"pageSize"`             // 每页数量
	TenantID       string `json:"tenantID" form:"tenantID"`             // 租户ID
	OrganizationID string `json:"organizationID" form:"organizationID"` // 组织ID
	Name           string `json:"name" form:"name"`                     // 角色名称
}

type OrganizationUserCreateReq struct {
	OrganizationID string `json:"organizationID" form:"organizationID" binding:"required"` // 组织ID
	UserID         string `json:"userID" form:"userID" binding:"required"`                 // 用户ID
}

type OrganizationUserDeleteReq struct {
	OrganizationID string `json:"-" uri:"organizationID" binding:"required"` // 组织ID
	UserID         string `json:"-" uri:"userID" binding:"required"`         // 用户ID
}

type OrganizationUserPageListReq struct {
	Page           int    `json:"page" form:"page"`                     // 页码
	PageSize       int    `json:"pageSize" form:"pageSize"`             // 每页数量
	TenantID       string `json:"tenantID" form:"tenantID"`             // 租户ID
	OrganizationID string `json:"organizationID" form:"organizationID"` // 组织ID
	UserID         string `json:"userID" form:"userID"`                 // 用户ID
}

type OrganizationRoleUserCreateReq struct {
	OrganizationID     string `json:"organizationID" form:"organizationID" binding:"required"`         // 组织ID
	OrganizationRoleID string `json:"organizationRoleID" form:"organizationRoleID" binding:"required"` // 组织角色ID
	UserID             string `json:"userID" form:"userID" binding:"required"`                         // 用户ID
}

type OrganizationRoleUserDeleteReq struct {
	OrganizationID     string `json:"organizationID" form:"organizationID"`          // 组织ID
	OrganizationRoleID string `json:"-" uri:"organizationRoleID" binding:"required"` // 组织角色ID
	UserID             string `json:"-" uri:"userID" binding:"required"`             // 用户ID
}

type OrganizationRoleUserPageListReq struct {
	Page               int    `json:"page" form:"page"`                             // 页码
	PageSize           int    `json:"pageSize" form:"pageSize"`                     // 每页数量
	TenantID           string `json:"tenantID" form:"tenantID"`                     // 租户ID
	OrganizationID     string `json:"organizationID" form:"organizationID"`         // 组织ID
	OrganizationRoleID string `json:"organizationRoleID" form:"organizationRoleID"` // 组织角色ID
	UserID             string `json:"userID" form:"userID"`                         // 用户ID
}
