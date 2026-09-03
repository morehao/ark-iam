package dtoapikey

// ApiKeySupervisionItem 全租户监督视图条目：明文密钥永不可见（仅前缀），
// 展示归属主体（服务账号/真实用户）、租户与创建人等信息用于泄漏风险排查。
type ApiKeySupervisionItem struct {
	ID          string `json:"id"`
	TenantID    string `json:"tenantID"`
	TenantName  string `json:"tenantName"`
	OwnerUserID string `json:"ownerUserID"` // 归属用户ID（真实用户本人或服务账号）
	OwnerType   string `json:"ownerType"`   // 归属类型(member真实用户/machine服务账号)
	OwnerName   string `json:"ownerName"`   // 归属用户名称
	CreatedBy   string `json:"createdBy"`
	CreatorName string `json:"creatorName"`
	Name        string `json:"name"`
	KeyPrefix   string `json:"keyPrefix"`
	Scope       string `json:"scope"`
	ExpiredAt   int64  `json:"expiresAt"`  // 过期时间(unix 秒)
	LastUsedAt  int64  `json:"lastUsedAt"` // 最后使用时间(unix 秒)
	RevokedAt   int64  `json:"revokedAt"`  // 撤销时间(unix 秒)
	CreatedAt   int64  `json:"createdAt"`  // 创建时间(unix 秒)
}

type ApiKeySupervisionPageListResp struct {
	List  []ApiKeySupervisionItem `json:"list"`
	Total int64                   `json:"total"`
}
