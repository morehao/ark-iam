package svcauth

import "github.com/morehao/ark-iam/platformadmin/internal/dto/dtoconnector"

func defaultConnectorFactories() []dtoconnector.ConnectorFactoryResp {
	return []dtoconnector.ConnectorFactoryResp{
		{
			FactoryID:     "oidc-google",
			Protocol:      connectorDriverTypeOIDC,
			Provider:      connectorProviderGoogle,
			DisplayName:   "Google",
			IsStandard:    true,
			DefaultScopes: []string{"openid", "profile", "email"},
			Capabilities:  []string{connectorCapabilityAuthorize, connectorCapabilityCallback, connectorCapabilityClaimMapping, connectorCapabilityDomainPolicy},
			ConfigSchema: map[string]any{
				"type":     "object",
				"required": []string{"issuer", "clientId", "clientSecret", "redirectUri"},
			},
		},
		{
			FactoryID:     "oauth2-github",
			Protocol:      connectorDriverTypeOAuth2,
			Provider:      connectorProviderGithub,
			DisplayName:   "GitHub",
			IsStandard:    true,
			DefaultScopes: []string{"read:user", "user:email"},
			Capabilities:  []string{connectorCapabilityAuthorize, connectorCapabilityCallback, connectorCapabilityProfileSync},
			ConfigSchema: map[string]any{
				"type":     "object",
				"required": []string{"authUrl", "tokenUrl", "userInfoUrl", "clientId", "clientSecret", "redirectUri"},
			},
		},
		{
			FactoryID:     "oidc-microsoft-entra",
			Protocol:      connectorDriverTypeOIDC,
			Provider:      connectorProviderMicrosoft,
			DisplayName:   "Microsoft Entra ID",
			IsStandard:    true,
			DefaultScopes: []string{"openid", "profile", "email"},
			Capabilities:  []string{connectorCapabilityAuthorize, connectorCapabilityCallback, connectorCapabilityClaimMapping, connectorCapabilityDomainPolicy},
			ConfigSchema: map[string]any{
				"type":     "object",
				"required": []string{"issuer", "clientId", "clientSecret", "redirectUri", "tenant"},
			},
		},
	}
}

func selectConnectorFactories(req *dtoconnector.ConnectorFactoryListReq, factories []dtoconnector.ConnectorFactoryResp) []dtoconnector.ConnectorFactoryResp {
	filtered := make([]dtoconnector.ConnectorFactoryResp, 0, len(factories))
	for _, factory := range factories {
		if req != nil && req.Protocol != "" && factory.Protocol != req.Protocol {
			continue
		}
		if req != nil && req.Provider != "" && factory.Provider != req.Provider {
			continue
		}
		filtered = append(filtered, factory)
	}
	return filtered
}
