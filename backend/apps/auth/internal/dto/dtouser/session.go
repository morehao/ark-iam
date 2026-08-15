package dtouser

import "github.com/morehao/golib/biz/gobject"

type SessionListReq struct {
	gobject.PageQuery
}

type SessionRevokeReq struct {
	SessionID string `json:"-" uri:"sessionID" binding:"required"`
}

type SessionResp struct {
	ID         string  `json:"id"`
	SessionID  string  `json:"sessionID"`
	AppID      string  `json:"appID"`
	TenantID   string  `json:"tenantID"`
	ClientType string  `json:"clientType"`
	ClientIP   string  `json:"clientIP"`
	UserAgent  string  `json:"userAgent"`
	ExpiredAt  *string `json:"expiresAt"`
	CreatedAt  string  `json:"createdAt"`
	IsActive   bool    `json:"isActive"`
}

type SessionListResp struct {
	List  []SessionResp `json:"list"`
	Total int64         `json:"total"`
}
