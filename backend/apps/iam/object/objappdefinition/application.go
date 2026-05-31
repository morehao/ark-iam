package objappdefinition

type AppDefinitionBaseInfo struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	LogoURL     string `json:"logoUrl"`
	HomepageURL string `json:"homepageUrl"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	Sort        int    `json:"sort"`
}
