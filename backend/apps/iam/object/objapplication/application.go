package objapplication

type ApplicationBaseInfo struct {
	TenantID     uint   `json:"tenantID"`
	ClientID     string `json:"clientID"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	LogoURL      string `json:"logoURL"`
	HomepageURL  string `json:"homepageURL"`
	Type         string `json:"type"`
	Status       string `json:"status"`
	IsThirdParty int8   `json:"isThirdParty"`
}
