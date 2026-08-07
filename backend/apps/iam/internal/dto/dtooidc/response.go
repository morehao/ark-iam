package dtooidc

import "github.com/morehao/ark-iam/iam/object/objauth"

type OIDCLoginResp struct {
	ContinueURL string                 `json:"continueURL"`
	SessionID   string                 `json:"sessionID,omitempty"`
	TenantID    uint                   `json:"tenantID,omitempty"`
	Tenants     []objauth.TenantOption `json:"tenants,omitempty"`
}
