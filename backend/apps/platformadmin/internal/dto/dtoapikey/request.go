package dtoapikey

import "github.com/morehao/golib/biz/gobject"

type ApiKeyCreateReq struct {
	Name      string `json:"name" binding:"required"`
	Scope     string `json:"scope"`
	ExpiredAt int64  `json:"expiresAt"` // 过期时间(unix 秒)
}

type RevokeApiKeyReq struct {
	ApiKeyID string `json:"-" uri:"apiKeyID" binding:"required"`
}

type ApiKeyDeleteReq struct {
	ApiKeyID string `json:"-" uri:"apiKeyID" binding:"required"`
}

type ApiKeyPageListReq struct {
	gobject.PageQuery
	Name string `json:"name" form:"name"`
}

// ApiKeySupervisionPageListReq 全租户只读监督列表：忽略当前上下文租户，跨租户检索（平台排查视角）。
type ApiKeySupervisionPageListReq struct {
	gobject.PageQuery
	TenantID string `json:"tenantID" form:"tenantID"` // 租户ID（可选过滤）
	Name     string `json:"name" form:"name"`
}
