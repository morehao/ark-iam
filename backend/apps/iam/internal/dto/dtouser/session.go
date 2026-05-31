package dtouser

import "github.com/morehao/golib/biz/gobject"

type SessionListReq struct {
	gobject.PageQuery
}

type SessionRevokeReq struct {
	SessionID uint64 `json:"sessionId" uri:"sessionId" binding:"required"`
}

type SessionResp struct {
	ID            uint64     `json:"id"`
	SessionID     string     `json:"sessionId"`
	AppID uint64     `json:"appId"`
	TenantID      uint64     `json:"tenantId"`
	ClientType    string     `json:"clientType"`
	ClientIP      string     `json:"clientIP"`
	UserAgent     string     `json:"userAgent"`
	ExpiredAt     *string    `json:"expiresAt"`
	CreatedAt     string     `json:"createdAt"`
	IsActive      bool       `json:"isActive"`
}

type SessionListResp struct {
	List  []SessionResp `json:"list"`
	Total int64         `json:"total"`
}
