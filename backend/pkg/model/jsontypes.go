package model

// 本文件承载 JSON 列的载具类型：每列一个具名类型 + gorm:"serializer:json"。
// 列表元素不做闭合枚举的列（redirect_uris/allowed_origins/scopes/amr）只给具名切片，
// 不引入会倒退功能的白名单；需要跨 OIDC 库边界时走 Strings()，避免裸转换散落各处。

type RedirectURIList []string

type PostLogoutRedirectURIList []string

type AllowedOriginList []string

type DefaultScopeList []string

// ScopeList refresh_token.scopes（可空：无 default）。
type ScopeList []string

// AuthMethodList refresh_token.amr（RFC 8176，开放集合，元素不做闭合枚举）。
type AuthMethodList []string

// ConnectorScopeList connector.config.scopes。
type ConnectorScopeList []string

// DomainList ConnectorDomainPolicy 内部专用（allowed/blocked 同义，结构体内复用）。
type DomainList []string

// ResponseType OIDC 响应类型（闭合集合，对应 op.Client.ResponseTypes 的映射源）。
type ResponseType string

const (
	ResponseTypeCode         ResponseType = "code"
	ResponseTypeIDToken      ResponseType = "id_token"
	ResponseTypeIDTokenToken ResponseType = "id_token token"
)

type ResponseTypeList []ResponseType

// GrantTypeList 元素复用已存在的 GrantType 具名类型。
type GrantTypeList []GrantType

func (l RedirectURIList) Strings() []string { return []string(l) }

func (l PostLogoutRedirectURIList) Strings() []string { return []string(l) }

func (l AllowedOriginList) Strings() []string { return []string(l) }

func (l DefaultScopeList) Strings() []string { return []string(l) }

func (l ScopeList) Strings() []string { return []string(l) }

func (l AuthMethodList) Strings() []string { return []string(l) }

func (l ConnectorScopeList) Strings() []string { return []string(l) }

// ConnectorConfig 连接器配置：显式字段取代原 map[string]any + Raw/Extra 双逃生舱。
// Microsoft Entra ID 的 tenant 参数从 Raw["tenant"] 提升为一等字段（json tag 保持 "tenant"）。
type ConnectorConfig struct {
	Protocol     ConnectorProtocol  `json:"protocol"`
	Provider     ConnectorProvider  `json:"provider"`
	Issuer       string             `json:"issuer,omitempty"`
	AuthURL      string             `json:"authUrl,omitempty"`
	TokenURL     string             `json:"tokenUrl,omitempty"`
	UserInfoURL  string             `json:"userInfoUrl,omitempty"`
	ClientID     string             `json:"clientID,omitempty"`
	ClientSecret string             `json:"clientSecret,omitempty"`
	RedirectURI  string             `json:"redirectUri,omitempty"`
	Scopes       ConnectorScopeList `json:"scopes,omitempty"`
	Tenant       string             `json:"tenant,omitempty"`
}

// ConnectorClaimMapping 声明映射：外部 IdP 的 claim 名 → 标准身份字段。
type ConnectorClaimMapping struct {
	Subject     string `json:"subject,omitempty"`
	Email       string `json:"email,omitempty"`
	Username    string `json:"username,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	AvatarURL   string `json:"avatarUrl,omitempty"`
}

// ConnectorDomainPolicy 域策略：允许/拒绝登录的邮箱域。
// 两个列表共用 DomainList 是结构体内部的同义复用，与「跨列共用载具类型」是两件事。
type ConnectorDomainPolicy struct {
	AllowedDomains DomainList `json:"allowedDomains,omitempty"`
	BlockedDomains DomainList `json:"blockedDomains,omitempty"`
}

// UserIdentityDetail 第三方身份明细（落 user_identity.detail）。
// 显式白名单：外部 IdP 的非白名单 claim 在写入边界（svcauth identityMapper）丢弃，
// 持久化结构必须完全可枚举。
type UserIdentityDetail struct {
	Issuer        string                 `json:"issuer,omitempty"`
	Subject       string                 `json:"subject,omitempty"`
	Email         string                 `json:"email,omitempty"`
	EmailVerified EmailVerificationState `json:"emailVerified,omitempty"`
	Username      string                 `json:"username,omitempty"`
	DisplayName   string                 `json:"displayName,omitempty"`
	AvatarURL     string                 `json:"avatarUrl,omitempty"`
	GivenName     string                 `json:"givenName,omitempty"`
	FamilyName    string                 `json:"familyName,omitempty"`
	Locale        string                 `json:"locale,omitempty"`
}
