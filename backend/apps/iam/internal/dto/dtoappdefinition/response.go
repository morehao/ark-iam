package dtoappdefinition

type CreateResp struct {
	AppDefID uint   `json:"appDefId"`
	Code     string `json:"code"`
}

type DetailResp struct {
	AppDefID    uint   `json:"appDefId"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	LogoURL     string `json:"logoUrl"`
	HomepageURL string `json:"homepageUrl"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	Sort        int    `json:"sort"`
	CreatedAt   string `json:"createdAt"`
}

type PageListItem struct {
	AppDefID    uint   `json:"appDefId"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	Sort        int    `json:"sort"`
	CreatedAt   string `json:"createdAt"`
}

type PageListResp struct {
	List  []PageListItem `json:"list"`
	Total int64          `json:"total"`
}
