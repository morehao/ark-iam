package dtoapikey

import "github.com/morehao/golib/biz/gobject"

// ApiKeySupervisionPageListReq 全租户只读监督列表：忽略当前上下文租户，跨租户检索（平台排查视角）。
type ApiKeySupervisionPageListReq struct {
	gobject.PageQuery
	TenantID string `json:"tenantID" form:"tenantID"` // 租户ID（可选过滤）
	Name     string `json:"name" form:"name"`
}
