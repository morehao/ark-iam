package svcauth

const (
	connectorDriverTypeOIDC   = "oidc"
	connectorDriverTypeOAuth2 = "oauth2"

	connectorProviderGoogle    = "google"
	connectorProviderGithub    = "github"
	connectorProviderMicrosoft = "microsoft"

	connectorCapabilityAuthorize    = "authorize"
	connectorCapabilityCallback     = "callback"
	connectorCapabilityClaimMapping = "claim_mapping"
	connectorCapabilityDomainPolicy = "domain_policy"
	connectorCapabilityProfileSync  = "profile_sync"
)

type ConnectorConfig struct {
	Protocol     string         `json:"protocol"`
	Provider     string         `json:"provider"`
	Issuer       string         `json:"issuer"`
	AuthURL      string         `json:"authUrl"`
	TokenURL     string         `json:"tokenUrl"`
	UserInfoURL  string         `json:"userInfoUrl"`
	ClientID     string         `json:"clientID"`
	ClientSecret string         `json:"clientSecret"`
	RedirectURI  string         `json:"redirectUri"`
	Scopes       []string       `json:"scopes"`
	Raw          map[string]any `json:"-"`
	Extra        map[string]any `json:"extra"`
}

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
