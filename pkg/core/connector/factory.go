package connector

import "github.com/morehao/ark-iam/pkg/model"

// Factory 描述一个内置连接器工厂（可创建哪些协议的连接器），随列表下发给前端展示。
// 与具体 app 的 DTO 解耦：auth/platformadmin 各自把 Factory 映射为自己的响应类型。
type Factory struct {
	FactoryID     string
	Protocol      model.ConnectorProtocol
	Provider      model.ConnectorProvider
	DisplayName   string
	IsStandard    bool
	DefaultScopes []string
	Capabilities  []model.ConnectorCapability
	ConfigSchema  map[string]any
}

var defaultFactoriesOnce = []Factory{
	{
		FactoryID:     "oidc-google",
		Protocol:      DriverTypeOIDC,
		Provider:      ProviderGoogle,
		DisplayName:   "Google",
		IsStandard:    true,
		DefaultScopes: []string{model.ScopeOpenID, model.ScopeProfile, model.ScopeEmail},
		Capabilities:  []model.ConnectorCapability{CapabilityAuthorize, CapabilityCallback, CapabilityClaimMapping, CapabilityDomainPolicy},
		ConfigSchema: map[string]any{
			"type":     "object",
			"required": []string{"issuer", "clientId", "clientSecret", "redirectUri"},
		},
	},
	{
		FactoryID:     "oauth2-github",
		Protocol:      DriverTypeOAuth2,
		Provider:      ProviderGithub,
		DisplayName:   "GitHub",
		IsStandard:    true,
		DefaultScopes: []string{"read:user", "user:email"},
		Capabilities:  []model.ConnectorCapability{CapabilityAuthorize, CapabilityCallback, CapabilityProfileSync},
		ConfigSchema: map[string]any{
			"type":     "object",
			"required": []string{"authUrl", "tokenUrl", "userInfoUrl", "clientId", "clientSecret", "redirectUri"},
		},
	},
	{
		FactoryID:     "oidc-microsoft-entra",
		Protocol:      DriverTypeOIDC,
		Provider:      ProviderMicrosoft,
		DisplayName:   "Microsoft Entra ID",
		IsStandard:    true,
		DefaultScopes: []string{model.ScopeOpenID, model.ScopeProfile, model.ScopeEmail},
		Capabilities:  []model.ConnectorCapability{CapabilityAuthorize, CapabilityCallback, CapabilityClaimMapping, CapabilityDomainPolicy},
		ConfigSchema: map[string]any{
			"type":     "object",
			"required": []string{"issuer", "clientId", "clientSecret", "redirectUri", "tenant"},
		},
	},
}

// Factories 返回内置工厂列表（只读共享）。
func Factories() []Factory {
	out := make([]Factory, len(defaultFactoriesOnce))
	copy(out, defaultFactoriesOnce)
	return out
}

// SelectFactories 按 protocol/provider 过滤工厂列表（空值不过滤）。
func SelectFactories(protocol model.ConnectorProtocol, provider model.ConnectorProvider) []Factory {
	filtered := make([]Factory, 0, len(defaultFactoriesOnce))
	for _, factory := range defaultFactoriesOnce {
		if protocol != "" && factory.Protocol != protocol {
			continue
		}
		if provider != "" && factory.Provider != provider {
			continue
		}
		filtered = append(filtered, factory)
	}
	return filtered
}
