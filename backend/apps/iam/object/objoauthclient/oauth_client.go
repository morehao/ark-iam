package objoauthclient

type OAuthClientBaseInfo struct {
	TenantID        uint   `json:"tenantID"`
	AppID   uint   `json:"appId"`
	ClientID        string `json:"clientID"`
	Name            string `json:"name"`
	Type            string `json:"type"`
	Status          string `json:"status"`
	IsThirdParty    int8   `json:"isThirdParty"`
}
