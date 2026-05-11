package dtodomain

type DomainCreateResp struct {
	ID uint `json:"id"`
}

type DomainPageListItem struct {
	ID         uint   `json:"id"`
	Domain     string `json:"domain"`
	IsVerified int8   `json:"isVerified"`
	VerifiedAt string `json:"verifiedAt"`
	CreatedAt  string `json:"createdAt"`
}

type DomainPageListResp struct {
	List  []DomainPageListItem `json:"list"`
	Total int64                `json:"total"`
}