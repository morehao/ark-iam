package dtodomain

type DomainCreateResp struct {
	ID uint `json:"id"` // 域名ID
}

type DomainDetailResp struct {
	ID         uint   `json:"id"`         // 域名ID
	Domain     string `json:"domain"`     // 域名
	IsVerified int8   `json:"isVerified"` // 是否验证(0-未验证 1-已验证)
	VerifiedAt string `json:"verifiedAt"` // 验证时间
	CreatedAt  string `json:"createdAt"`  // 创建时间
	UpdatedAt  string `json:"updatedAt"`  // 更新时间
}

type DomainPageListItem struct {
	ID         uint   `json:"id"`         // 域名ID
	Domain     string `json:"domain"`     // 域名
	IsVerified int8   `json:"isVerified"` // 是否验证(0-未验证 1-已验证)
	VerifiedAt string `json:"verifiedAt"` // 验证时间
	CreatedAt  string `json:"createdAt"`  // 创建时间
}

type DomainPageListResp struct {
	List  []DomainPageListItem `json:"list"`
	Total int64                `json:"total"`
}