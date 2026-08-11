package dtoconnector

type ConnectorFactoryResp struct {
	FactoryID     string   `json:"factoryId"`
	Protocol      string   `json:"protocol"`
	Provider      string   `json:"provider"`
	DisplayName   string   `json:"displayName"`
	IsStandard    bool     `json:"isStandard"`
	DefaultScopes []string `json:"defaultScopes"`
	Capabilities  []string `json:"capabilities"`
	ConfigSchema  any      `json:"configSchema"`
}

type ConnectorFactoryListResp struct {
	List []ConnectorFactoryResp `json:"list"`
}

type TestConnectorResp struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type ConnectorAuthorizeResp struct {
	AuthorizationURL string `json:"authorizationUrl"`
}

type ConnectorCallbackResp struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}
