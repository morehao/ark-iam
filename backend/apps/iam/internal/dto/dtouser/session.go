package dtouser

import "github.com/morehao/golib/biz/gobject"

type SessionListReq struct {
	gobject.PageQuery
}

type SessionRevokeReq struct {
	SessionID uint64 `json:"sessionId" path:"sessionId" binding:"required"`
}

type SessionResp struct {
	ID            uint64     `json:"id"`
	ApplicationID uint64     `json:"applicationId"`
	TenantID      uint64     `json:"tenantId"`
	ExpiresAt     *string    `json:"expiresAt"`
	CreatedAt     string     `json:"createdAt"`
	IsActive      bool       `json:"isActive"`
}

type SessionListResp struct {
	List  []SessionResp `json:"list"`
	Total int64         `json:"total"`
}
