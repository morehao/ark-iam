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

	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/object/objauth"
	"github.com/morehao/ark-iam/pkg/iam/sso"
	"github.com/morehao/golib/glog"
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
	SessionID     string              `json:"session_id"`
}

func (a *AuthRequest) GetID() string                         { return a.ID }
func (a *AuthRequest) GetACR() string                        { return a.ACR }
func (a *AuthRequest) GetAMR() []string                      { return a.AMR }
func (a *AuthRequest) GetAudience() []string                 { return a.Audience }
func (a *AuthRequest) GetAuthTime() time.Time                { return a.AuthTime }
func (a *AuthRequest) GetClientID() string                   { return a.ClientID }
func (a *AuthRequest) GetCodeChallenge() *oidc.CodeChallenge { return a.CodeChallenge }
func (a *AuthRequest) GetNonce() string                      { return a.Nonce }
func (a *AuthRequest) GetRedirectURI() string                { return a.RedirectURI }
func (a *AuthRequest) GetResponseType() oidc.ResponseType    { return a.ResponseType }
func (a *AuthRequest) GetResponseMode() oidc.ResponseMode    { return a.ResponseMode }
func (a *AuthRequest) GetScopes() []string                   { return a.Scopes }
func (a *AuthRequest) GetState() string                      { return a.State }
func (a *AuthRequest) GetSubject() string                    { return a.Subject }
func (a *AuthRequest) Done() bool                            { return a.DoneFlag }
func (a *AuthRequest) GetTenantID() uint                     { return a.TenantID }
func (a *AuthRequest) GetSessionID() string                  { return a.SessionID }

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
		if ccReq.isApiKey {
			return (objauth.TokenClaims{
				TenantID:   ccReq.ownerTenantID,
				UserID:     ccReq.ownerUserID,
				TokenUsage: objauth.TokenUsageMachine,
			}).OIDCPrivateClaims(), nil
		}
		return objauth.TokenClaims{ClientID: ccReq.ClientID()}.OIDCPrivateClaims(), nil
	}
	// authorization_code：优先用认证流程确定的租户
	if authReq, ok := request.(*AuthRequest); ok {
		if tid := authReq.GetTenantID(); tid > 0 {
			if pid, perr := parseOIDCSubject(authReq.GetSubject()); perr == nil {
				if users, uerr := s.persistentStore.userDao().GetListByCond(ctx, &dao.UserCond{PersonID: pid, TenantID: tid}); uerr == nil && len(users) > 0 {
					claims := objauth.TokenClaims{TenantID: tid}.OIDCPrivateClaims()
					// sid：注入 SSO 会话标识，使 ID token / access token 携带 sid，
					// 供 RP 匹配与会话粒度的 back-channel 登出（M4）。
					if authReq.SessionID != "" {
						claims["sid"] = authReq.SessionID
					}
					return claims, nil
				}
			}
		}
	}
	// refresh token 轮换：保留存储的租户，避免重新落到 users[0]
	if rr, ok := request.(*refreshTokenRequest); ok {
		if tid := rr.GetTenantID(); tid > 0 {
			if pid, perr := parseOIDCSubject(rr.GetSubject()); perr == nil {
				if users, uerr := s.persistentStore.userDao().GetListByCond(ctx, &dao.UserCond{PersonID: pid, TenantID: tid}); uerr == nil && len(users) > 0 {
					return objauth.TokenClaims{TenantID: tid}.OIDCPrivateClaims(), nil
				}
			}
		}
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
	var tenantID uint
	if v, ok := ctx.Value(ctxTenantHintKey).(uint); ok {
		tenantID = v
	}
	return s.protocolStore.CreateAuthRequest(ctx, authReq, userID, tenantID)
}

func (s *OIDCStorage) AuthRequestByID(ctx context.Context, id string) (op.AuthRequest, error) {
	return s.protocolStore.AuthRequestByID(ctx, id)
}

func (s *OIDCStorage) AuthRequestByCode(ctx context.Context, code string) (op.AuthRequest, error) {
	return s.protocolStore.AuthRequestByCode(ctx, code)
}

func (s *OIDCStorage) CompleteAuthRequest(id string, subject string, authTime time.Time, amr []string, acr string, tenantID uint, done bool) error {
	return s.protocolStore.CompleteAuthRequest(id, subject, authTime, amr, acr, tenantID, done)
}

func (s *OIDCStorage) AssociateSession(ctx context.Context, id string, sessionID string) error {
	return s.protocolStore.AssociateSession(ctx, id, sessionID)
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
	if err := s.AuthorizeClientIDSecret(ctx, clientID, clientSecret); err == nil {
		client, cErr := s.GetClientByClientID(ctx, clientID)
		if cErr == nil {
			if client.AuthMethod() == oidc.AuthMethodNone {
				return nil, oidc.ErrInvalidClient()
			}
			return client, nil
		}
	}
	// fallback: API Key 作为 client credential（clientID==clientSecret==rawKey）
	if clientID != clientSecret {
		return nil, oidc.ErrInvalidClient()
	}
	apiKey, err := s.persistentStore.LookupApiKeyByRawKey(ctx, clientID)
	if err != nil || apiKey == nil {
		return nil, oidc.ErrInvalidClient()
	}
	return &apiKeyOpClient{entity: apiKey}, nil
}

func (s *OIDCStorage) ClientCredentialsTokenRequest(ctx context.Context, clientID string, scopes []string) (op.TokenRequest, error) {
	aud := resolveAudienceFromScopes(scopes)
	if aud == "" {
		aud = clientID
	}
	req := &clientCredentialsTokenRequest{
		subject:  clientID,
		audience: []string{aud},
		scopes:   scopes,
		clientID: clientID,
	}
	if apiKey, err := s.persistentStore.LookupApiKeyByRawKey(ctx, clientID); err == nil && apiKey != nil {
		req.isApiKey = true
		req.ownerTenantID = apiKey.TenantID
		// sub 需用 owner user 的 person id 语义，以便 oidcauth parsePersonIDFromSub 识别（见 Task 4）。
		// apiKey.CreatedBy 是 owner user 的 user id，非 person id；需经 user 表解析出 person id。
		if user, uErr := s.persistentStore.userDao().GetByID(ctx, apiKey.CreatedBy); uErr == nil && user != nil && user.ID != 0 {
			req.ownerUserID = user.ID
			req.subject = buildOIDCSubject(user.PersonID)
		} else {
			// owner user 无法解析出 person id：不产出 person 上下文（保持 subject=clientID），
			// 仅携带 client_id 私有 claim，避免给出 personID=0 的误导性 sub。
			glog.Warnf(ctx, "[svcoidc.ClientCredentialsTokenRequest] api key owner user not resolved, createdBy:%d, err:%v", apiKey.CreatedBy, uErr)
			req.isApiKey = false
		}
	}
	return req, nil
}

type clientCredentialsTokenRequest struct {
	subject       string
	audience      []string
	scopes        []string
	clientID      string
	isApiKey      bool
	ownerTenantID uint
	ownerUserID   uint
}

func (r *clientCredentialsTokenRequest) GetSubject() string    { return r.subject }
func (r *clientCredentialsTokenRequest) GetAudience() []string { return r.audience }
func (r *clientCredentialsTokenRequest) GetScopes() []string   { return r.scopes }

func (r *clientCredentialsTokenRequest) ClientID() string { return r.clientID }

func (s *OIDCStorage) TerminateSession(ctx context.Context, userID string, clientID string) error {
	return s.persistentStore.TerminateSession(ctx, userID, clientID)
}

// SigningPrivateKey 暴露 OP 的令牌签名私钥，供 back-channel logout worker 构造 logout_token。
func (s *OIDCStorage) SigningPrivateKey() *rsa.PrivateKey { return s.signingKey }

// TerminateSessionFromRequest 实现 op.CanTerminateSessionFromRequest（RP-Initiated Logout）。
//
// 撤销该 person 的 SSO 会话与全部 refresh token，并将待通知的 back-channel 登出任务入队。
// M3 阶段 ID token 尚不携带 sid，统一按 person 级执行；M4 注入 sid 后按 id_token_hint 精确到中心会话。
func (s *OIDCStorage) TerminateSessionFromRequest(ctx context.Context, endSessionRequest *op.EndSessionRequest) (string, error) {
	personID, pErr := parseOIDCSubject(endSessionRequest.UserID)
	if pErr != nil {
		glog.Warnf(ctx, "[svcoidc.TerminateSessionFromRequest] parse subject fail, userID:%s, err:%v", endSessionRequest.UserID, pErr)
		return endSessionRequest.RedirectURI, nil
	}

	// 撤销 SSO 会话 + 吊销该 person 全部 refresh（D2=A 防续命兜底）
	if tErr := s.persistentStore.TerminateSession(ctx, endSessionRequest.UserID, endSessionRequest.ClientID); tErr != nil {
		glog.Warnf(ctx, "[svcoidc.TerminateSessionFromRequest] terminate session fail, personID:%d, err:%v", personID, tErr)
	}

	// 取待通知的登记：优先按 sid 精确（M4 后），否则按 person 全部
	slo := sso.NewSLOStore()
	var regs []sso.LogoutRegistration
	var err error
	if endSessionRequest.IDTokenHintClaims != nil && endSessionRequest.IDTokenHintClaims.SessionID != "" {
		regs, err = slo.ListBySessionID(ctx, endSessionRequest.IDTokenHintClaims.SessionID)
	} else {
		regs, err = slo.ListByPersonID(ctx, personID)
	}
	if err != nil {
		glog.Warnf(ctx, "[svcoidc.TerminateSessionFromRequest] list registrations fail, personID:%d, err:%v", personID, err)
		regs = nil
	}

	var jobSessionID string
	if endSessionRequest.IDTokenHintClaims != nil {
		jobSessionID = endSessionRequest.IDTokenHintClaims.SessionID
	}
	for _, reg := range regs {
		if err := sso.EnqueueLogout(ctx, sso.LogoutJob{
			SessionID:            jobSessionID,
			PersonID:             personID,
			OIDCSessionID:        reg.OIDCSessionID,
			ClientID:             reg.ClientID,
			UserID:               reg.UserID,
			BackChannelLogoutURI: reg.BackChannelLogoutURI,
		}); err != nil {
			glog.Warnf(ctx, "[svcoidc.TerminateSessionFromRequest] enqueue logout fail, clientID:%s, err:%v", reg.ClientID, err)
		}
	}

	return endSessionRequest.RedirectURI, nil
}

var _ op.CanTerminateSessionFromRequest = (*OIDCStorage)(nil)

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

func (k *oidcKey) ID() string                         { return k.id }
func (k *oidcKey) Algorithm() jose.SignatureAlgorithm { return k.alg }
func (k *oidcKey) Use() string                        { return k.use }
func (k *oidcKey) Key() any                           { return k.key }
