package dtoapikey

type ApiKeyCreateResp struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Key       string `json:"key"`
	KeyPrefix string `json:"keyPrefix"`
	ExpiredAt int64 `json:"expiresAt"` // 过期时间(unix 秒)
}

type ApiKeyPageListItem struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	KeyPrefix  string `json:"keyPrefix"`
	Scope      string `json:"scope"`
	ExpiredAt  int64  `json:"expiresAt"`  // 过期时间(unix 秒)
	LastUsedAt int64  `json:"lastUsedAt"` // 最后使用时间(unix 秒)
	RevokedAt  int64  `json:"revokedAt"`  // 撤销时间(unix 秒)
	CreatedAt  int64  `json:"createdAt"`  // 创建时间(unix 秒)
}

type ApiKeyPageListResp struct {
	List  []ApiKeyPageListItem `json:"list"`
	Total int64                `json:"total"`
}
