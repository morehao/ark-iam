package svcoidc

import (
	"context"
	"crypto/rsa"
	"fmt"
	"strconv"
	"strings"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/oidc/v3/pkg/op"
)

type AuthRequest struct {
	ID            string              `json:"id"`
	ClientID      string              `json:"client_id"`
	RedirectURI   string              `json:"redirect_uri"`
	State         string              `json:"state"`
	Scopes        []string            `json:"scopes"`
	ResponseType  oidc.ResponseType   `json:"response_type"`
	ResponseMode  oidc.ResponseMode   `json:"response_mode"`
	Nonce         string              `json:"nonce"`
	CodeChallenge *oidc.CodeChallenge `json:"code_challenge,omitempty"`
	Subject       string              `json:"subject"`
	AuthTime      time.Time           `json:"auth_time"`
	AMR           []string            `json:"amr"`
	ACR           string              `json:"acr"`
	Audience      []string            `json:"audience"`
	DoneFlag      bool                `json:"done_flag"`
	ExpiresAt     time.Time           `json:"expires_at"`
	TenantID      uint                `json:"tenant_id"`
}

func (a *AuthRequest) GetID() string                          { return a.ID }
func (a *AuthRequest) GetACR() string                         { return a.ACR }
func (a *AuthRequest) GetAMR() []string                       { return a.AMR }
func (a *AuthRequest) GetAudience() []string                  { return a.Audience }
func (a *AuthRequest) GetAuthTime() time.Time                 { return a.AuthTime }
func (a *AuthRequest) GetClientID() string                    { return a.ClientID }
func (a *AuthRequest) GetCodeChallenge() *oidc.CodeChallenge  { return a.CodeChallenge }
func (a *AuthRequest) GetNonce() string                       { return a.Nonce }
func (a *AuthRequest) GetRedirectURI() string                 { return a.RedirectURI }
func (a *AuthRequest) GetResponseType() oidc.ResponseType     { return a.ResponseType }
func (a *AuthRequest) GetResponseMode() oidc.ResponseMode     { return a.ResponseMode }
func (a *AuthRequest) GetScopes() []string                    { return a.Scopes }
func (a *AuthRequest) GetState() string                       { return a.State }
func (a *AuthRequest) GetSubject() string                     { return a.Subject }
func (a *AuthRequest) Done() bool                             { return a.DoneFlag }
func (a *AuthRequest) GetTenantID() uint                    { return a.TenantID }

type OIDCStorage struct {
	protocolStore   ProtocolStateStore
	persistentStore *PersistentStore
	signingKey      *rsa.PrivateKey
	signingKeyID    string
}

func NewOIDCStorage(protocolStore ProtocolStateStore, persistentStore *PersistentStore, signingKey *rsa.PrivateKey, keyID string) *OIDCStorage {
	return &OIDCStorage{
		protocolStore:   protocolStore,
		persistentStore: persistentStore,
		signingKey:      signingKey,
		signingKeyID:    keyID,
	}
}

var _ op.Storage = (*OIDCStorage)(nil)

func (s *OIDCStorage) Health(ctx context.Context) error {
	return s.protocolStore.Health(ctx)
}

func (s *OIDCStorage) GetClientByClientID(ctx context.Context, clientID string) (op.Client, error) {
	return s.persistentStore.GetClientByClientID(ctx, clientID)
}

func (s *OIDCStorage) AuthorizeClientIDSecret(ctx context.Context, clientID, clientSecret string) error {
	return s.persistentStore.AuthorizeClientIDSecret(ctx, clientID, clientSecret)
}

func (s *OIDCStorage) SetUserinfoFromScopes(ctx context.Context, userinfo *oidc.UserInfo, userID, clientID string, scopes []string) error {
	return s.persistentStore.SetUserinfoFromScopes(ctx, userinfo, userID, clientID, scopes)
}

func (s *OIDCStorage) SetUserinfoFromToken(ctx context.Context, userinfo *oidc.UserInfo, tokenID, subject, origin string) error {
	return s.persistentStore.SetUserinfoFromToken(ctx, userinfo, tokenID, subject, origin)
}

func (s *OIDCStorage) SetIntrospectionFromToken(ctx context.Context, introspection *oidc.IntrospectionResponse, tokenID, subject, clientID string) error {
	return s.persistentStore.SetIntrospectionFromToken(ctx, introspection, tokenID, subject, clientID)
}

func (s *OIDCStorage) GetPrivateClaimsFromScopes(ctx context.Context, userID, clientID string, scopes []string) (map[string]any, error) {
	return s.persistentStore.GetPrivateClaimsFromScopes(ctx, userID, clientID, scopes)
}

var _ op.CanGetPrivateClaimsFromRequest = (*OIDCStorage)(nil)

func (s *OIDCStorage) GetPrivateClaimsFromRequest(ctx context.Context, request op.TokenRequest, restrictedScopes []string) (map[string]any, error) {
	if ccReq, ok := request.(*clientCredentialsTokenRequest); ok {
		return map[string]any{
			"client_id": ccReq.ClientID(),
		}, nil
	}
	return s.GetPrivateClaimsFromScopes(ctx, request.GetSubject(), getClientIDFromRequest(request), restrictedScopes)
}

func getClientIDFromRequest(request op.TokenRequest) string {
	if c, ok := request.(interface{ GetClientID() string }); ok {
		return c.GetClientID()
	}
	return ""
}

// resolveAudienceFromScopes 从请求 scope 中挑出 resource indicator（如 urn:...），作为 access token 的 aud。
// 返回第一个非 OIDC 标准 scope 的值；无则返回空串。
func resolveAudienceFromScopes(scopes []string) string {
	for _, s := range scopes {
		switch s {
		case "openid", "profile", "email", "phone", "offline_access":
			continue
		default:
			return s
		}
	}
	return ""
}

func (s *OIDCStorage) GetKeyByIDAndClientID(ctx context.Context, keyID, clientID string) (*jose.JSONWebKey, error) {
	return s.persistentStore.GetKeyByIDAndClientID(ctx, keyID, clientID)
}

func (s *OIDCStorage) ValidateJWTProfileScopes(ctx context.Context, userID string, scopes []string) ([]string, error) {
	return s.persistentStore.ValidateJWTProfileScopes(ctx, userID, scopes)
}

func (s *OIDCStorage) CreateAuthRequest(ctx context.Context, authReq *oidc.AuthRequest, userID string) (op.AuthRequest, error) {
	return s.protocolStore.CreateAuthRequest(ctx, authReq, userID)
}

func (s *OIDCStorage) AuthRequestByID(ctx context.Context, id string) (op.AuthRequest, error) {
	return s.protocolStore.AuthRequestByID(ctx, id)
}

func (s *OIDCStorage) AuthRequestByCode(ctx context.Context, code string) (op.AuthRequest, error) {
	return s.protocolStore.AuthRequestByCode(ctx, code)
}

func (s *OIDCStorage) CompleteAuthRequest(id string, subject string, authTime time.Time, amr []string, acr string) error {
	return s.protocolStore.CompleteAuthRequest(id, subject, authTime, amr, acr)
}

func (s *OIDCStorage) SaveAuthCode(ctx context.Context, id, code string) error {
	return s.protocolStore.SaveAuthCode(ctx, id, code)
}

func (s *OIDCStorage) DeleteAuthRequest(ctx context.Context, id string) error {
	return s.protocolStore.DeleteAuthRequest(ctx, id)
}

func (s *OIDCStorage) CreateAccessToken(ctx context.Context, request op.TokenRequest) (accessTokenID string, expiration time.Time, err error) {
	return s.persistentStore.CreateAccessToken(ctx, request)
}

func (s *OIDCStorage) CreateAccessAndRefreshTokens(ctx context.Context, request op.TokenRequest, currentRefreshToken string) (accessTokenID string, newRefreshToken string, expiration time.Time, err error) {
	return s.persistentStore.CreateAccessAndRefreshTokens(ctx, request, currentRefreshToken)
}

func (s *OIDCStorage) TokenRequestByRefreshToken(ctx context.Context, refreshToken string) (op.RefreshTokenRequest, error) {
	return s.persistentStore.TokenRequestByRefreshToken(ctx, refreshToken)
}

var _ op.ClientCredentialsStorage = (*OIDCStorage)(nil)

func (s *OIDCStorage) ClientCredentials(ctx context.Context, clientID, clientSecret string) (op.Client, error) {
	if err := s.AuthorizeClientIDSecret(ctx, clientID, clientSecret); err != nil {
		return nil, oidc.ErrInvalidClient()
	}
	client, err := s.GetClientByClientID(ctx, clientID)
	if err != nil {
		return nil, oidc.ErrInvalidClient()
	}
	if client.AuthMethod() == oidc.AuthMethodNone {
		return nil, oidc.ErrInvalidClient()
	}
	return client, nil
}

func (s *OIDCStorage) ClientCredentialsTokenRequest(ctx context.Context, clientID string, scopes []string) (op.TokenRequest, error) {
	aud := resolveAudienceFromScopes(scopes)
	if aud == "" {
		aud = clientID
	}
	return &clientCredentialsTokenRequest{
		subject:  clientID,
		audience: []string{aud},
		scopes:   scopes,
		clientID: clientID,
	}, nil
}

type clientCredentialsTokenRequest struct {
	subject  string
	audience []string
	scopes   []string
	clientID string
}

func (r *clientCredentialsTokenRequest) GetSubject() string   { return r.subject }
func (r *clientCredentialsTokenRequest) GetAudience() []string { return r.audience }
func (r *clientCredentialsTokenRequest) GetScopes() []string  { return r.scopes }

func (r *clientCredentialsTokenRequest) ClientID() string { return r.clientID }

func (s *OIDCStorage) TerminateSession(ctx context.Context, userID string, clientID string) error {
	return s.persistentStore.TerminateSession(ctx, userID, clientID)
}

func (s *OIDCStorage) RevokeToken(ctx context.Context, tokenOrTokenID string, userID string, clientID string) *oidc.Error {
	return s.persistentStore.RevokeToken(ctx, tokenOrTokenID, userID, clientID)
}

func (s *OIDCStorage) GetRefreshTokenInfo(ctx context.Context, clientID string, tokenValue string) (userID string, tokenID string, err error) {
	return s.persistentStore.GetRefreshTokenInfo(ctx, clientID, tokenValue)
}



func buildOIDCSubject(personID uint) string {
	return fmt.Sprintf("person:%d", personID)
}

func parseOIDCSubject(subject string) (uint, error) {
	const prefix = "person:"
	if !strings.HasPrefix(subject, prefix) {
		return 0, fmt.Errorf("invalid oidc subject: %s", subject)
	}
	rawID := strings.TrimPrefix(subject, prefix)
	personID, err := strconv.ParseUint(rawID, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid oidc subject: %s", subject)
	}
	return uint(personID), nil
}

func (s *OIDCStorage) SigningKey(ctx context.Context) (op.SigningKey, error) {
	return &oidcSigningKey{
		id:  s.signingKeyID,
		alg: jose.RS256,
		key: s.signingKey,
	}, nil
}

func (s *OIDCStorage) SignatureAlgorithms(ctx context.Context) ([]jose.SignatureAlgorithm, error) {
	return []jose.SignatureAlgorithm{jose.RS256}, nil
}

func (s *OIDCStorage) KeySet(ctx context.Context) ([]op.Key, error) {
	return []op.Key{
		&oidcKey{
			id:  s.signingKeyID,
			alg: jose.RS256,
			use: "sig",
			key: &s.signingKey.PublicKey,
		},
	}, nil
}

type oidcSigningKey struct {
	id  string
	alg jose.SignatureAlgorithm
	key any
}

func (k *oidcSigningKey) SignatureAlgorithm() jose.SignatureAlgorithm { return k.alg }
func (k *oidcSigningKey) Key() any                                    { return k.key }
func (k *oidcSigningKey) ID() string                                  { return k.id }

type oidcKey struct {
	id  string
	alg jose.SignatureAlgorithm
	use string
	key any
}

func (k *oidcKey) ID() string                           { return k.id }
func (k *oidcKey) Algorithm() jose.SignatureAlgorithm    { return k.alg }
func (k *oidcKey) Use() string                           { return k.use }
func (k *oidcKey) Key() any                              { return k.key }
