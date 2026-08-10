package dtooidc

type OIDCLoginReq struct {
	AuthRequestID string `json:"authRequestID" binding:"required"`
	Identifier    string `json:"identifier" binding:"required"`
	Password      string `json:"password" binding:"required"`
}

type OIDCSelectTenantReq struct {
	AuthRequestID string `json:"authRequestID" binding:"required"`
	TenantID      uint   `json:"tenantID" binding:"required"`
}
