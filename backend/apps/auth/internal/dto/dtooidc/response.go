package dtooidc

import "github.com/morehao/ark-iam/pkg/iam/object/objauth"

type OIDCLoginResp struct {
	ContinueURL             string                 `json:"continueURL"`
	SessionID               string                 `json:"sessionID,omitempty"`
	TenantID                string                 `json:"tenantID,omitempty"`
	PersonID                string                 `json:"personID,omitempty"`
	RequiresTenantSelection bool                   `json:"requiresTenantSelection,omitempty"`
	Tenants                 []objauth.TenantOption `json:"tenants,omitempty"`
	AllowPersonCreateTenant bool                   `json:"allowPersonCreateTenant"`
}
