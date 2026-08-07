package dtotenant

import "github.com/morehao/ark-iam/iam/object/objtenant"

type OrganizationCreateResp struct {
	OrganizationID uint `json:"organizationID"` // 组织ID
}

type OrganizationDetailResp struct {
	OrganizationID      uint `json:"organizationID"`        // 组织ID
	objtenant.OrganizationBaseInfo `json:"organizationBaseInfo"` // 组织基础信息
}

type OrganizationPageListResp struct {
	List  []OrganizationPageListItem `json:"list"`  // 组织列表
	Total int64                      `json:"total"` // 总数
}

type OrganizationPageListItem struct {
	OrganizationID      uint `json:"organizationID"`        // 组织ID
	objtenant.OrganizationBaseInfo `json:"organizationBaseInfo"` // 组织基础信息
}

type OrganizationRoleCreateResp struct {
	OrganizationRoleID uint `json:"organizationRoleID"` // 组织角色ID
}

type OrganizationRoleDetailResp struct {
	OrganizationRoleID uint `json:"organizationRoleID"`        // 组织角色ID
	OrganizationRoleBaseInfo
}

type OrganizationRolePageListResp struct {
	List  []OrganizationRolePageListItem `json:"list"`  // 组织角色列表
	Total int64                          `json:"total"` // 总数
}

type OrganizationRolePageListItem struct {
	OrganizationRoleID uint `json:"organizationRoleID"`        // 组织角色ID
	OrganizationRoleBaseInfo
}

type OrganizationRoleBaseInfo struct {
	TenantID       uint   `json:"tenantID" form:"tenantID"`         // 租户ID
	OrganizationID uint   `json:"organizationID" form:"organizationID"` // 组织ID
	Name           string `json:"name" form:"name"`                 // 角色名称
	Description    string `json:"description" form:"description"`   // 角色描述
	Type           string `json:"type" form:"type"`                 // 角色类型
}

type OrganizationUserCreateResp struct {
}

type OrganizationUserPageListResp struct {
	List  []OrganizationUserPageListItem `json:"list"`  // 组织用户列表
	Total int64                                  `json:"total"` // 总条数
}

type OrganizationUserPageListItem struct {
	OrganizationID uint `json:"organizationID"` // 组织ID
	UserID         uint `json:"userID"`         // 用户ID
	TenantID       uint `json:"tenantID"`       // 租户ID
}

type OrganizationRoleUserCreateResp struct {
}

type OrganizationRoleUserPageListResp struct {
	List  []OrganizationRoleUserPageListItem `json:"list"`  // 组织角色用户列表
	Total int64                                       `json:"total"` // 总条数
}

type OrganizationRoleUserPageListItem struct {
	OrganizationID     uint `json:"organizationID"`     // 组织ID
	OrganizationRoleID uint `json:"organizationRoleID"` // 组织角色ID
	UserID             uint `json:"userID"`             // 用户ID
	TenantID           uint `json:"tenantID"`           // 租户ID
}