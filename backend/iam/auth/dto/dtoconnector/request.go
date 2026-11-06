package dtoconnector

import "github.com/morehao/ark-iam/pkg/model"

type ConnectorIDReq struct {
	ConnectorID string `json:"-" uri:"connectorID" binding:"required"`
}

type ConnectorAuthorizeReq struct {
	ConnectorID  string `json:"-" uri:"connectorID" binding:"required"`
	RedirectURI  string `json:"redirectUri" binding:"required"`
	State        string `json:"state"`
	LoginHint    string `json:"loginHint"`
	ResponseMode string `json:"responseMode"`
}

type ConnectorCallbackReq struct {
	ConnectorID string `json:"connectorID"`
	Code        string `json:"code" binding:"required"`
	State       string `json:"state" binding:"required"`
}

type ConnectorFactoryListReq struct {
	Protocol model.ConnectorProtocol `json:"protocol" form:"protocol"`
	Provider model.ConnectorProvider `json:"provider" form:"provider"`
}
