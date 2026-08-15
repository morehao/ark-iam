package objaudit

type LogBaseInfo struct {
	TenantID string `json:"tenantID" form:"tenantID"` // 租户ID
	Key      string `json:"key" form:"key"`           // 日志键
	Payload  any    `json:"payload" form:"payload"`   // 日志内容
}
