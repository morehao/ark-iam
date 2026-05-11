package dtoapikey

type CreateApiKeyResp struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	Key       string `json:"key"`
	KeyPrefix string `json:"keyPrefix"`
	ExpiresAt string `json:"expiresAt"`
}

type ApiKeyPageListItem struct {
	ID         uint   `json:"id"`
	Name       string `json:"name"`
	KeyPrefix  string `json:"keyPrefix"`
	Scope      string `json:"scope"`
	ExpiresAt  string `json:"expiresAt"`
	LastUsedAt string `json:"lastUsedAt"`
	RevokedAt  string `json:"revokedAt"`
	CreatedAt  string `json:"createdAt"`
}

type ApiKeyPageListResp struct {
	List  []ApiKeyPageListItem `json:"list"`
	Total int64                `json:"total"`
}