package svcauth

import "github.com/morehao/ark-iam/pkg/model"

// 连接器字典常量的 svcauth 包内短名别名：取值事实源在 model
// （model.ConnectorProtocol* / ConnectorProvider* / ConnectorCapability*），
// 此处保留短名供驱动注册、工厂描述与测试复用，禁止再出现裸字面量。
const (
	connectorDriverTypeOIDC   = model.ConnectorProtocolOIDC
	connectorDriverTypeOAuth2 = model.ConnectorProtocolOAuth2

	connectorProviderGoogle    = model.ConnectorProviderGoogle
	connectorProviderGithub    = model.ConnectorProviderGithub
	connectorProviderMicrosoft = model.ConnectorProviderMicrosoft
	connectorProviderWechat    = model.ConnectorProviderWechat

	connectorCapabilityAuthorize    = model.ConnectorCapabilityAuthorize
	connectorCapabilityCallback     = model.ConnectorCapabilityCallback
	connectorCapabilityClaimMapping = model.ConnectorCapabilityClaimMapping
	connectorCapabilityDomainPolicy = model.ConnectorCapabilityDomainPolicy
	connectorCapabilityProfileSync  = model.ConnectorCapabilityProfileSync
)

// ConnectorConfig 连接器配置载具：结构定义在 model（对应 connector.config 列，serializer:json），
// 包内保留短名别名以复用驱动签名。
type ConnectorConfig = model.ConnectorConfig

type StandardIdentity struct {
	Issuer        string         `json:"issuer"`
	Subject       string         `json:"subject"`
	Email         string         `json:"email"`
	Username      string         `json:"username"`
	DisplayName   string         `json:"displayName"`
	AvatarURL     string         `json:"avatarUrl"`
	EmailVerified bool           `json:"emailVerified"`
	Claims        map[string]any `json:"claims"`
}

type ConnectorAuthorizeInput struct {
	Config       ConnectorConfig
	ConnectorID  string
	RedirectURI  string
	State        string
	LoginHint    string
	ResponseMode string
}

type ConnectorAuthorizeOutput struct {
	AuthorizationURL string
	Nonce            string
	// CodeVerifier 为 PKCE S256 verifier，由驱动生成，需随 state 持久化以便回调回填。
	CodeVerifier string
}

type ConnectorCallbackInput struct {
	Config      ConnectorConfig
	ConnectorID string
	Code        string
	State       string
	Nonce       string
	// CodeVerifier 为授权阶段生成的 PKCE verifier，换 code 时回填。
	CodeVerifier string
	RedirectURI  string
}

type ConnectorCallbackOutput struct {
	Identity     StandardIdentity
	AccessToken  string
	RefreshToken string
}

type ConnectorTestInput struct {
	Config ConnectorConfig
}

type ConnectorTestOutput struct {
	Success bool
	Message string
}
