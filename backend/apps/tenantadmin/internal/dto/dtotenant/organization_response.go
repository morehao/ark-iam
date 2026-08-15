package dtotenant

import "github.com/morehao/ark-iam/pkg/iam/object/objtenant"

type OrganizationCreateResp struct {
	OrganizationID string `json:"organizationID"` // 组织ID
}

type OrganizationDetailResp struct {
	OrganizationID string `json:"organizationID"` // 组织ID
	objtenant.OrganizationBaseInfo
}

type OrganizationPageListResp struct {
	List  []OrganizationPageListItem `json:"list"`  // 组织列表
	Total int64                      `json:"total"` // 总数
}

type OrganizationPageListItem struct {
	OrganizationID string `json:"organizationID"` // 组织ID
	objtenant.OrganizationBaseInfo
}

type OrganizationRoleCreateResp struct {
	OrganizationRoleID string `json:"organizationRoleID"` // 组织角色ID
}

type OrganizationRoleDetailResp struct {
	OrganizationRoleID string `json:"organizationRoleID"` // 组织角色ID
	OrganizationRoleBaseInfo
}

type OrganizationRolePageListResp struct {
	List  []OrganizationRolePageListItem `json:"list"`  // 组织角色列表
	Total int64                          `json:"total"` // 总数
}

type OrganizationRolePageListItem struct {
	OrganizationRoleID string `json:"organizationRoleID"` // 组织角色ID
	OrganizationRoleBaseInfo
}

type OrganizationRoleBaseInfo struct {
	TenantID       string `json:"tenantID" form:"tenantID"`             // 租户ID
	OrganizationID string `json:"organizationID" form:"organizationID"` // 组织ID
	Name           string `json:"name" form:"name"`                     // 角色名称
	Description    string `json:"description" form:"description"`       // 角色描述
	Type           string `json:"type" form:"type"`                     // 角色类型
}

type OrganizationUserCreateResp struct {
}

type OrganizationUserPageListResp struct {
	List  []OrganizationUserPageListItem `json:"list"`  // 组织用户列表
	Total int64                          `json:"total"` // 总条数
}

type OrganizationUserPageListItem struct {
	OrganizationID string `json:"organizationID"` // 组织ID
	UserID         string `json:"userID"`         // 用户ID
	TenantID       string `json:"tenantID"`       // 租户ID
}

type OrganizationRoleUserCreateResp struct {
}

type OrganizationRoleUserPageListResp struct {
	List  []OrganizationRoleUserPageListItem `json:"list"`  // 组织角色用户列表
	Total int64                              `json:"total"` // 总条数
}

type OrganizationRoleUserPageListItem struct {
	OrganizationID     string `json:"organizationID"`     // 组织ID
	OrganizationRoleID string `json:"organizationRoleID"` // 组织角色ID
	UserID             string `json:"userID"`             // 用户ID
	TenantID           string `json:"tenantID"`           // 租户ID
}
