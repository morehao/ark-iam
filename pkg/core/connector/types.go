// Package connector 承载连接器（平台信任的外部 IdP）的领域不变式：
// 驱动注册与构造、配置/声明/域策略校验、出站 URL 与回调 URL 校验、配置脱敏、secret 保留、
// 开关归一化与状态校验、工厂定义与筛选。
//
// 归属：连接器是平台级全局对象（一次配置、服务多租户），管理面在 platformadmin（跨租户），
// 运行时（Authorize/Callback）在 auth。两端共用本包，避免复制驱动与校验逻辑。
// 本包只放领域不变式，不放 HTTP/DB 编排（见 AGENTS.md「公共层约定」）。
package connector

import "github.com/morehao/ark-iam/pkg/model"

// 包内短名别名：取值事实源在 model（model.ConnectorProtocol* 等），
// 供驱动注册、工厂描述复用，禁止再出现裸字面量。
const (
	DriverTypeOIDC   = model.ConnectorProtocolOIDC
	DriverTypeOAuth2 = model.ConnectorProtocolOAuth2

	ProviderGoogle    = model.ConnectorProviderGoogle
	ProviderGithub    = model.ConnectorProviderGithub
	ProviderMicrosoft = model.ConnectorProviderMicrosoft
	ProviderWechat    = model.ConnectorProviderWechat

	CapabilityAuthorize    = model.ConnectorCapabilityAuthorize
	CapabilityCallback     = model.ConnectorCapabilityCallback
	CapabilityClaimMapping = model.ConnectorCapabilityClaimMapping
	CapabilityDomainPolicy = model.ConnectorCapabilityDomainPolicy
	CapabilityProfileSync  = model.ConnectorCapabilityProfileSync
)

// Config 连接器配置载具：结构定义在 model（对应 connector.config 列，serializer:json）。
type Config = model.ConnectorConfig

// StandardIdentity 驱动解析出的标准身份（与具体 IdP 无关的中间态）。
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

type AuthorizeInput struct {
	Config       Config
	ConnectorID  string
	RedirectURI  string
	State        string
	LoginHint    string
	ResponseMode string
}

type AuthorizeOutput struct {
	AuthorizationURL string
	Nonce            string
	// CodeVerifier 为 PKCE S256 verifier，由驱动生成，需随 state 持久化以便回调回填。
	CodeVerifier string
}

type CallbackInput struct {
	Config       Config
	ConnectorID  string
	Code         string
	State        string
	Nonce        string
	// CodeVerifier 为授权阶段生成的 PKCE verifier，换 code 时回填。
	CodeVerifier string
	RedirectURI  string
}

type CallbackOutput struct {
	Identity     StandardIdentity
	AccessToken  string
	RefreshToken string
}
