package dtotenant

// 租户端 API 密钥响应 DTO。

type ApiKeyPageListItem struct {
	KeyID       string `json:"keyID"`       // API密钥ID
	Name        string `json:"name"`        // 密钥名称
	KeyPrefix   string `json:"keyPrefix"`   // 密钥前缀
	OwnerUserID string `json:"ownerUserID"` // 归属服务账号ID
	OwnerType   string `json:"ownerType"`   // 归属类型(machine服务账号;兼容历史member数据)
	OwnerName   string `json:"ownerName"`   // 归属用户名称
	CreatedBy   string `json:"createdBy"`   // 创建人ID
	CreatorName string `json:"creatorName"` // 创建人名称
	ExpiredAt   *int64 `json:"expiredAt"`   // 过期时间(unix秒,永不过期=null)
	LastUsedAt  *int64 `json:"lastUsedAt"`  // 最后使用时间(null=从未使用)
	RevokedAt   *int64 `json:"revokedAt"`   // 吊销时间(null=未吊销)
	CreatedAt   int64  `json:"createdAt"`   // 创建时间
}

type ApiKeyPageListResp struct {
	List  []ApiKeyPageListItem `json:"list"`  // 数据列表
	Total int64                `json:"total"` // 数据总条数
}

type ApiKeyCreateResp struct {
	ID        string `json:"id"`        // API密钥ID
	Name      string `json:"name"`      // 密钥名称
	Key       string `json:"key"`       // 明文密钥(仅此一次展示,请妥善保存)
	KeyPrefix string `json:"keyPrefix"` // 密钥前缀
	ExpiredAt int64  `json:"expiredAt"` // 过期时间(unix秒,0=永不过期)
}
