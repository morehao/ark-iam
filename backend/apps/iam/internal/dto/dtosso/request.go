package dtosso

type SsoConnectorIDReq struct {
	ConnectorID uint `json:"connectorId" path:"connectorId" binding:"required"`
}

type SsoIdpConfigReq struct {
	ClientID     string   `json:"clientId"`
	ClientSecret string   `json:"clientSecret"`
	AuthURL      string   `json:"authUrl"`
	TokenURL     string   `json:"tokenUrl"`
	UserInfoURL  string   `json:"userInfoUrl"`
	Scopes       []string `json:"scopes"`
}