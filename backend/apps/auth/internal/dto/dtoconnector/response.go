package dtoconnector

import "github.com/morehao/ark-iam/pkg/iam/model"

type ConnectorFactoryResp struct {
	FactoryID     string                      `json:"factoryID"`
	Protocol      model.ConnectorProtocol     `json:"protocol"`
	Provider      model.ConnectorProvider     `json:"provider"`
	DisplayName   string                      `json:"displayName"`
	IsStandard    bool                        `json:"isStandard"`
	DefaultScopes []string                    `json:"defaultScopes"`
	Capabilities  []model.ConnectorCapability `json:"capabilities"`
	ConfigSchema  any                         `json:"configSchema"`
}

type ConnectorFactoryListResp struct {
	List []ConnectorFactoryResp `json:"list"`
}

type ConnectorTestResp struct {
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
