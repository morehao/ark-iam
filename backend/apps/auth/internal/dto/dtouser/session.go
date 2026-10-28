package dtouser

import (
	"github.com/morehao/ark-iam/pkg/object/objidentity"
	"github.com/morehao/golib/biz/gobject"
)

type SessionListReq struct {
	gobject.PageQuery
	// Operator 由 controller 从鉴权结果填入（json:"-"，客户端不可伪造）。
	Operator objidentity.Operator
}

type SessionRevokeReq struct {
	SessionID string `json:"-" uri:"sessionID" binding:"required"`
	// Operator 由 controller 从鉴权结果填入（json:"-"，客户端不可伪造）。
	Operator objidentity.Operator
}

// SessionRevokeAllReq 撤销所有会话（无业务请求参数，仅携带操作人）。
type SessionRevokeAllReq struct {
	// Operator 由 controller 从鉴权结果填入（json:"-"，客户端不可伪造）。
	Operator objidentity.Operator
}

type SessionResp struct {
	ID         string `json:"id"`
	SessionID  string `json:"sessionID"`
	AppID      string `json:"appID"`
	TenantID   string `json:"tenantID"`
	ClientType string `json:"clientType"`
	ClientIP   string `json:"clientIP"`
	UserAgent  string `json:"userAgent"`
	ExpiredAt  *int64 `json:"expiresAt"` // 过期时间(unix 秒)
	CreatedAt  int64  `json:"createdAt"` // 创建时间(unix 秒)
	IsActive   bool   `json:"isActive"`
}

type SessionListResp struct {
	List  []SessionResp `json:"list"`
	Total int64         `json:"total"`
}
