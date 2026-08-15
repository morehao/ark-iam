package dtotenant

type OrganizationCreateReq struct {
	TenantID      uint   `json:"tenantID" form:"tenantID"`           // 租户ID
	Name          string `json:"name" form:"name"`                   // 组织名称
	Description   string `json:"description" form:"description"`     // 组织描述
	IsMFARequired int8   `json:"isMFARequired" form:"isMFARequired"` // 是否需要MFA
}

type OrganizationUpdateReq struct {
	OrganizationID uint   `json:"-" uri:"organizationID"`             // 组织ID
	TenantID       uint   `json:"tenantID" form:"tenantID"`           // 租户ID
	Name           string `json:"name" form:"name"`                   // 组织名称
	Description    string `json:"description" form:"description"`     // 组织描述
	IsMFARequired  int8   `json:"isMFARequired" form:"isMFARequired"` // 是否需要MFA
}

type OrganizationDeleteReq struct {
	OrganizationID uint `json:"-" uri:"organizationID"` // 组织ID
}

type OrganizationDetailReq struct {
	OrganizationID uint `json:"-" uri:"organizationID"` // 组织ID
}

type OrganizationPageListReq struct {
	Page     int    `json:"page" form:"page"`         // 页码
	PageSize int    `json:"pageSize" form:"pageSize"` // 每页数量
	TenantID uint   `json:"tenantID" form:"tenantID"` // 租户ID
	Name     string `json:"name" form:"name"`         // 组织名称
}

type OrganizationRoleCreateReq struct {
	TenantID       uint   `json:"tenantID" form:"tenantID"`                                // 租户ID
	OrganizationID uint   `json:"organizationID" form:"organizationID" binding:"required"` // 组织ID
	Name           string `json:"name" form:"name"`                                        // 角色名称
	Description    string `json:"description" form:"description"`                          // 角色描述
	Type           string `json:"type" form:"type"`                                        // 角色类型
}

type OrganizationRoleUpdateReq struct {
	OrganizationRoleID uint   `json:"-" binding:"required" uri:"organizationRoleID"`           // 组织角色ID
	TenantID           uint   `json:"tenantID" form:"tenantID"`                                // 租户ID
	OrganizationID     uint   `json:"organizationID" form:"organizationID" binding:"required"` // 组织ID
	Name               string `json:"name" form:"name"`                                        // 角色名称
	Description        string `json:"description" form:"description"`                          // 角色描述
	Type               string `json:"type" form:"type"`                                        // 角色类型
}

type OrganizationRoleDeleteReq struct {
	OrganizationRoleID uint `json:"-" uri:"organizationRoleID"` // 组织角色ID
}

type OrganizationRoleDetailReq struct {
	OrganizationRoleID uint `json:"-" uri:"organizationRoleID"` // 组织角色ID
}

type OrganizationRolePageListReq struct {
	Page           int    `json:"page" form:"page"`                     // 页码
	PageSize       int    `json:"pageSize" form:"pageSize"`             // 每页数量
	TenantID       uint   `json:"tenantID" form:"tenantID"`             // 租户ID
	OrganizationID uint   `json:"organizationID" form:"organizationID"` // 组织ID
	Name           string `json:"name" form:"name"`                     // 角色名称
}

type OrganizationUserCreateReq struct {
	OrganizationID uint `json:"organizationID" form:"organizationID" binding:"required"` // 组织ID
	UserID         uint `json:"userID" form:"userID" binding:"required"`                 // 用户ID
}

type OrganizationUserDeleteReq struct {
	OrganizationID uint `json:"-" uri:"organizationID" binding:"required"` // 组织ID
	UserID         uint `json:"-" uri:"userID" binding:"required"`         // 用户ID
}

type OrganizationUserPageListReq struct {
	Page           int  `json:"page" form:"page"`                     // 页码
	PageSize       int  `json:"pageSize" form:"pageSize"`             // 每页数量
	TenantID       uint `json:"tenantID" form:"tenantID"`             // 租户ID
	OrganizationID uint `json:"organizationID" form:"organizationID"` // 组织ID
	UserID         uint `json:"userID" form:"userID"`                 // 用户ID
}

type OrganizationRoleUserCreateReq struct {
	OrganizationID     uint `json:"organizationID" form:"organizationID" binding:"required"`         // 组织ID
	OrganizationRoleID uint `json:"organizationRoleID" form:"organizationRoleID" binding:"required"` // 组织角色ID
	UserID             uint `json:"userID" form:"userID" binding:"required"`                         // 用户ID
}

type OrganizationRoleUserDeleteReq struct {
	OrganizationID     uint `json:"organizationID" form:"organizationID"`          // 组织ID
	OrganizationRoleID uint `json:"-" uri:"organizationRoleID" binding:"required"` // 组织角色ID
	UserID             uint `json:"-" uri:"userID" binding:"required"`             // 用户ID
}

type OrganizationRoleUserPageListReq struct {
	Page               int  `json:"page" form:"page"`                             // 页码
	PageSize           int  `json:"pageSize" form:"pageSize"`                     // 每页数量
	TenantID           uint `json:"tenantID" form:"tenantID"`                     // 租户ID
	OrganizationID     uint `json:"organizationID" form:"organizationID"`         // 组织ID
	OrganizationRoleID uint `json:"organizationRoleID" form:"organizationRoleID"` // 组织角色ID
	UserID             uint `json:"userID" form:"userID"`                         // 用户ID
}
