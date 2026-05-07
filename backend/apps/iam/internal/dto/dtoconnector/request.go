package dtoconnector

type ConnectorIDReq struct {
	ConnectorID uint64 `json:"connectorId" path:"connectorId" binding:"required"`
}

type AuthorizationUriReq struct {
	RedirectURI string `json:"redirectUri" binding:"required"`
	State       string `json:"state"`
}
