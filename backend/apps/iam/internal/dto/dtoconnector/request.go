package dtoconnector

type ConnectorIDReq struct {
	ConnectorID uint64 `json:"connectorId" uri:"connectorId" binding:"required"`
}

type ConnectorAuthorizeReq struct {
	ConnectorID  uint   `json:"connectorId" uri:"connectorId" binding:"required"`
	RedirectURI  string `json:"redirectUri" binding:"required"`
	State        string `json:"state"`
	LoginHint    string `json:"loginHint"`
	ResponseMode string `json:"responseMode"`
}

type ConnectorCallbackReq struct {
	ConnectorID uint   `json:"connectorId"`
	Code        string `json:"code" binding:"required"`
	State       string `json:"state" binding:"required"`
}

type ConnectorFactoryListReq struct {
	Protocol string `json:"protocol" form:"protocol"`
	Provider string `json:"provider" form:"provider"`
}
