package dtoconnector

type ConnectorFactoryResp struct {
	FactoryID   string `json:"factoryId"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Logo        string `json:"logo"`
	IsStandard  bool   `json:"isStandard"`
}

type ConnectorFactoryListResp struct {
	Factories []ConnectorFactoryResp `json:"factories"`
}

type TestConnectorResp struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type AuthorizationUriResp struct {
	AuthorizationUri string `json:"authorizationUri"`
}
