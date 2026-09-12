package dtooidc

import "github.com/morehao/ark-iam/pkg/object/objauth"

type OIDCLoginResp struct {
	ContinueURL             string `json:"continueURL"`
	SessionID               string `json:"sessionID,omitempty"`
	TenantID                string `json:"tenantID,omitempty"`
	PersonID                string `json:"personID,omitempty"`
	RequiresTenantSelection bool   `json:"requiresTenantSelection,omitempty"`
	// RequiresPasswordChange 该自然人当前持临时密码（person.must_change_password），
	// 必须先用 POST /oidc/login/changePassword 设置新密码后才能继续登录流程。
	RequiresPasswordChange  bool                   `json:"requiresPasswordChange,omitempty"`
	Tenants                 []objauth.TenantOption `json:"tenants,omitempty"`
	AllowPersonCreateTenant bool                   `json:"allowPersonCreateTenant"`
}
