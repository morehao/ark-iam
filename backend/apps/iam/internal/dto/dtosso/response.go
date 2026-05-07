package dtosso

type SsoProviderResp struct {
	ProviderName string `json:"providerName"`
	DisplayName  string `json:"displayName"`
	Logo        string `json:"logo"`
}

type SsoProviderListResp struct {
	Providers []SsoProviderResp `json:"providers"`
}

type SsoIdpConfigResp struct {
	ClientID    string   `json:"clientId"`
	AuthURL    string   `json:"authUrl"`
	TokenURL   string   `json:"tokenUrl"`
	UserInfoURL string   `json:"userInfoUrl"`
	Scopes     []string `json:"scopes"`
}