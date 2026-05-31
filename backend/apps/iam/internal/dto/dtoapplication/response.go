package dtoapplication

import "github.com/morehao/ark-iam/iam/object/objapplication"

type ApplicationCreateResp struct {
	ApplicationID uint   `json:"applicationID"`
	ClientID      string `json:"clientID"`
}

type ApplicationDetailResp struct {
	ApplicationID uint `json:"applicationID"`
	objapplication.ApplicationBaseInfo
}

type ApplicationPageListResp struct {
	List  []ApplicationPageListItem `json:"list"`
	Total int64                     `json:"total"`
}

type ApplicationPageListItem struct {
	ApplicationID uint `json:"applicationID"`
	objapplication.ApplicationBaseInfo
}

type ApplicationRoleResp struct {
	RoleID        uint64 `json:"roleId"`
	RoleName      string `json:"roleName"`
	RoleCode      string `json:"roleCode"`
	ApplicationID uint64 `json:"applicationId"`
	CreatedAt     string `json:"createdAt"`
}

type ApplicationRoleListResp struct {
	Total int64                 `json:"total"`
	Roles []ApplicationRoleResp `json:"roles"`
}

type ApplicationSecretResp struct {
	ID            uint64  `json:"id"`
	ApplicationID uint64  `json:"applicationId"`
	Name          string  `json:"name"`
	ValuePrefix   string  `json:"valuePrefix"`
	ExpiredAt     *string `json:"expiresAt"`
	CreatedAt     string  `json:"createdAt"`
}

type ApplicationSecretListResp struct {
	Total   int64                    `json:"total"`
	Secrets []ApplicationSecretResp `json:"secrets"`
}

type CreateApplicationSecretResp struct {
	ID          uint64 `json:"id"`
	Name        string `json:"name"`
	ValuePrefix string `json:"valuePrefix"`
	Secret      string `json:"secret"`
}
