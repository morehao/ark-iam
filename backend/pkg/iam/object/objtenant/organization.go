package objtenant

type OrganizationBaseInfo struct {
	TenantID      uint   `json:"tenantID" form:"tenantID"`       // 租户ID
	Name          string `json:"name" form:"name"`               // 组织名称
	Description   string `json:"description" form:"description"` // 组织描述
	IsMFARequired int8   `json:"isMFARequired" form:"isMFARequired"` // 是否需要MFA
}