package dtoapplication

import "github.com/morehao/ark-iam/iam/object/objapplication"

type ApplicationCreateResp struct {
	ApplicationID uint `json:"applicationID"` // 应用ID
}

type ApplicationDetailResp struct {
	ApplicationID       uint `json:"applicationID"`        // 应用ID
	objapplication.ApplicationBaseInfo `json:"applicationBaseInfo"` // 应用基础信息
}

type ApplicationPageListResp struct {
	List  []ApplicationPageListItem `json:"list"`  // 应用列表
	Total int64                     `json:"total"` // 总数
}

type ApplicationPageListItem struct {
	ApplicationID       uint `json:"applicationID"`        // 应用ID
	objapplication.ApplicationBaseInfo `json:"applicationBaseInfo"` // 应用基础信息
}

type ApplicationRoleResp struct {
	RoleID        uint64 `json:"roleId"`
	RoleName      string `json:"roleName"`
	RoleCode      string `json:"roleCode"`
	ApplicationID uint64 `json:"applicationId"`
	CreatedAt     string `json:"createdAt"`
}

type ApplicationRoleListResp struct {
	Total  int64                 `json:"total"`
	Roles  []ApplicationRoleResp `json:"roles"`
}

type ApplicationSecretResp struct {
	ID            uint64  `json:"id"`
	ApplicationID uint64  `json:"applicationId"`
	Name          string  `json:"name"`
	Value         string  `json:"value,omitempty"`
	ExpiresAt     *string `json:"expiresAt"`
	CreatedAt     string  `json:"createdAt"`
}

type ApplicationSecretListResp struct {
	Total   int64                    `json:"total"`
	Secrets []ApplicationSecretResp `json:"secrets"`
}

type CreateApplicationSecretResp struct {
	ID     uint64 `json:"id"`
	Name   string `json:"name"`
	Secret string `json:"secret"`
}