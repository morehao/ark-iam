package svcoidc

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/oidc/v3/pkg/op"

	"github.com/morehao/ark-iam/iam/dao"
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/pkg/token"
)

type authRequestStore struct {
	mu       sync.RWMutex
	requests map[string]*AuthRequest
	codes    map[string]string
}

type AuthRequest struct {
	ID              string
	ClientID        string
	RedirectURI     string
	State           string
	Scopes          []string
	ResponseType    oidc.ResponseType
	ResponseMode    oidc.ResponseMode
	Nonce           string
	CodeChallenge   *oidc.CodeChallenge
	Subject         string
	AuthTime        time.Time
	AMR             []string
	ACR             string
	Audience        []string
	DoneFlag        bool
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

type OIDCStorage struct {
	authRequests *authRequestStore
	signingKey   *rsa.PrivateKey
	signingKeyID string
}

func NewOIDCStorage() *OIDCStorage {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	return &OIDCStorage{
		authRequests: &authRequestStore{
			requests: make(map[string]*AuthRequest),
			codes:    make(map[string]string),
		},
		signingKey:   privateKey,
		signingKeyID: "default-key",
	}
}

func NewOIDCStorageWithKey(privateKey *rsa.PrivateKey, keyID string) *OIDCStorage {
	return &OIDCStorage{
		authRequests: &authRequestStore{
			requests: make(map[string]*AuthRequest),
			codes:    make(map[string]string),
		},
		signingKey:   privateKey,
		signingKeyID: keyID,
	}
}

var _ op.Storage = (*OIDCStorage)(nil)

var newOAuthClientDao = func() *dao.OAuthClientDao {
	return dao.NewOAuthClientDao()
}

var newOAuthClientSecretDao = func() *dao.OAuthClientSecretDao {
	return dao.NewOAuthClientSecretDao()
}

func (s *OIDCStorage) Health(ctx context.Context) error {
	return nil
}

func (s *OIDCStorage) GetClientByClientID(ctx context.Context, clientID string) (op.Client, error) {
	clientEntity, err := newOAuthClientDao().GetByCond(ctx, &dao.OAuthClientCond{ClientID: clientID})
	if err != nil || clientEntity == nil || clientEntity.ID == 0 {
		return nil, fmt.Errorf("client not found: %s", clientID)
	}
	return NewOIDCClient(clientEntity), nil
}

func (s *OIDCStorage) AuthorizeClientIDSecret(ctx context.Context, clientID, clientSecret string) error {
	secretHash := sha256.Sum256([]byte(clientSecret))
	clientHash := hex.EncodeToString(secretHash[:])

	clientEntity, err := newOAuthClientDao().GetByCond(ctx, &dao.OAuthClientCond{ClientID: clientID})
	if err != nil || clientEntity == nil || clientEntity.ID == 0 {
		return oidc.ErrInvalidClient()
	}
	secrets, _, err := newOAuthClientSecretDao().GetPageListByCond(ctx, &dao.OAuthClientSecretCond{OAuthClientID: clientEntity.ID})
	if err != nil {
		return oidc.ErrInvalidClient()
	}
	for _, sec := range secrets {
		if sec.ValueHash == clientHash && sec.RevokedAt == nil {
			if sec.ExpiredAt == nil || sec.ExpiredAt.After(time.Now()) {
				return nil
			}
		}
	}
	return oidc.ErrInvalidClient()
}

func (s *OIDCStorage) SetUserinfoFromScopes(ctx context.Context, userinfo *oidc.UserInfo, userID, clientID string, scopes []string) error {
	var pid uint
	fmt.Sscanf(userID, "%d", &pid)
	personDao := dao.NewPersonDao()
	person, err := personDao.GetByID(ctx, pid)
	if err != nil || person == nil || person.ID == 0 {
		return nil
	}
	for _, scope := range scopes {
		switch scope {
		case oidc.ScopeProfile:
			userinfo.Name = person.Name
			userinfo.PreferredUsername = person.Username
		case oidc.ScopeEmail:
			userinfo.Email = person.PrimaryEmail
			userinfo.EmailVerified = true
		case oidc.ScopePhone:
			userinfo.PhoneNumber = person.PrimaryPhone
			userinfo.PhoneNumberVerified = false
		}
	}
	return nil
}

func (s *OIDCStorage) SetUserinfoFromToken(ctx context.Context, userinfo *oidc.UserInfo, tokenID, subject, origin string) error {
	var pid uint
	fmt.Sscanf(subject, "%d", &pid)
	personDao := dao.NewPersonDao()
	person, err := personDao.GetByID(ctx, pid)
	if err != nil || person == nil || person.ID == 0 {
		return nil
	}
	userinfo.Name = person.Name
	userinfo.PreferredUsername = person.Username
	userinfo.Email = person.PrimaryEmail
	userinfo.EmailVerified = true
	return nil
}

func (s *OIDCStorage) SetIntrospectionFromToken(ctx context.Context, introspection *oidc.IntrospectionResponse, tokenID, subject, clientID string) error {
	return nil
}

func (s *OIDCStorage) GetPrivateClaimsFromScopes(ctx context.Context, userID, clientID string, scopes []string) (map[string]any, error) {
	return nil, nil
}

func (s *OIDCStorage) GetKeyByIDAndClientID(ctx context.Context, keyID, clientID string) (*jose.JSONWebKey, error) {
	return nil, errors.New("key not found")
}

func (s *OIDCStorage) ValidateJWTProfileScopes(ctx context.Context, userID string, scopes []string) ([]string, error) {
	return scopes, nil
}

func (s *OIDCStorage) CreateAuthRequest(ctx context.Context, authReq *oidc.AuthRequest, userID string) (op.AuthRequest, error) {
	req := &AuthRequest{
		ID:           fmt.Sprintf("ar-%d", time.Now().UnixNano()),
		ClientID:     authReq.ClientID,
		RedirectURI:  authReq.RedirectURI,
		State:        authReq.State,
		Scopes:       authReq.Scopes,
		ResponseType: authReq.ResponseType,
		ResponseMode: authReq.ResponseMode,
		Nonce:        authReq.Nonce,
		Subject:      userID,
		AuthTime:     time.Now(),
		Audience:     []string{authReq.ClientID},
	}
	if authReq.CodeChallenge != "" {
		method := oidc.CodeChallengeMethodS256
		if authReq.CodeChallengeMethod == "plain" {
			method = oidc.CodeChallengeMethodPlain
		}
		req.CodeChallenge = &oidc.CodeChallenge{
			Challenge: authReq.CodeChallenge,
			Method:    method,
		}
	}
	s.authRequests.mu.Lock()
	s.authRequests.requests[req.ID] = req
	s.authRequests.mu.Unlock()
	return req, nil
}

func (s *OIDCStorage) AuthRequestByID(ctx context.Context, id string) (op.AuthRequest, error) {
	s.authRequests.mu.RLock()
	req, ok := s.authRequests.requests[id]
	s.authRequests.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("auth request not found: %s", id)
	}
	return req, nil
}

func (s *OIDCStorage) AuthRequestByCode(ctx context.Context, code string) (op.AuthRequest, error) {
	s.authRequests.mu.RLock()
	id, ok := s.authRequests.codes[code]
	s.authRequests.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("auth code not found: %s", code)
	}
	return s.AuthRequestByID(ctx, id)
}

func (s *OIDCStorage) CompleteAuthRequest(id string, subject string, authTime time.Time, amr []string, acr string) error {
	s.authRequests.mu.Lock()
	defer s.authRequests.mu.Unlock()

	req, ok := s.authRequests.requests[id]
	if !ok {
		return fmt.Errorf("auth request not found: %s", id)
	}
	req.Subject = subject
	req.AuthTime = authTime
	req.AMR = append([]string(nil), amr...)
	req.ACR = acr
	req.DoneFlag = true
	return nil
}

func (s *OIDCStorage) SaveAuthCode(ctx context.Context, id, code string) error {
	s.authRequests.mu.Lock()
	s.authRequests.codes[code] = id
	s.authRequests.mu.Unlock()
	return nil
}

func (s *OIDCStorage) DeleteAuthRequest(ctx context.Context, id string) error {
	s.authRequests.mu.Lock()
	delete(s.authRequests.requests, id)
	for code, reqID := range s.authRequests.codes {
		if reqID == id {
			delete(s.authRequests.codes, code)
		}
	}
	s.authRequests.mu.Unlock()
	return nil
}

func (s *OIDCStorage) CreateAccessToken(ctx context.Context, request op.TokenRequest) (accessTokenID string, expiration time.Time, err error) {
	return fmt.Sprintf("at-%d", time.Now().UnixNano()), time.Now().Add(time.Hour), nil
}

func (s *OIDCStorage) CreateAccessAndRefreshTokens(ctx context.Context, request op.TokenRequest, currentRefreshToken string) (accessTokenID string, newRefreshToken string, expiration time.Time, err error) {
	accessTokenID = fmt.Sprintf("at-%d", time.Now().UnixNano())
	expiration = time.Now().Add(time.Hour)

	var personID uint
	personID, err = parseOIDCSubject(request.GetSubject())
	if err != nil {
		return "", "", time.Time{}, err
	}

	userDao := dao.NewUserDao()
	users, err := userDao.GetListByCond(ctx, &dao.UserCond{PersonID: personID})
	if err != nil || len(users) == 0 {
		return "", "", time.Time{}, fmt.Errorf("user not found for person %d", personID)
	}
	userEntity := &users[0]

	clientID := ""
	if authReq, ok := request.(op.AuthRequest); ok {
		clientID = authReq.GetClientID()
	}

	var oauthClientID uint
	if clientID != "" {
		clientEntity, err := newOAuthClientDao().GetByCond(ctx, &dao.OAuthClientCond{ClientID: clientID})
		if err == nil && clientEntity != nil {
			oauthClientID = clientEntity.ID
		}
	}

	now := time.Now()
	refreshTokenExp := now.Add(30 * 24 * time.Hour)
	refreshTokenValue := fmt.Sprintf("rt-%d-%d", time.Now().UnixNano(), userEntity.ID)

	refreshTokenHash := token.HashToken(refreshTokenValue)
	refreshEntity := &model.RefreshTokenEntity{
		PersonID:      personID,
		TenantID:      userEntity.TenantID,
		UserID:        userEntity.ID,
		OAuthClientID: oauthClientID,
		Token:         refreshTokenHash,
		ExpiredAt:     &refreshTokenExp,
		CreatedBy:     userEntity.ID,
	}
	if err := dao.NewRefreshTokenDao().Insert(ctx, refreshEntity); err != nil {
		return "", "", time.Time{}, err
	}

	if currentRefreshToken != "" {
		oldTokenHash := token.HashToken(currentRefreshToken)
		oldToken, err := dao.NewRefreshTokenDao().GetByCond(ctx, &dao.RefreshTokenCond{Token: oldTokenHash})
		if err == nil && oldToken != nil && oldToken.ID != 0 {
			dbt := time.Now()
			dao.NewRefreshTokenDao().UpdateMap(ctx, oldToken.ID, map[string]any{
				"revoked_at": &dbt,
			})
		}
	}

	return accessTokenID, refreshTokenValue, expiration, nil
}

func (s *OIDCStorage) TokenRequestByRefreshToken(ctx context.Context, refreshToken string) (op.RefreshTokenRequest, error) {
	refreshTokenHash := token.HashToken(refreshToken)
	storedToken, err := dao.NewRefreshTokenDao().GetByCond(ctx, &dao.RefreshTokenCond{Token: refreshTokenHash})
	if err != nil || storedToken == nil || storedToken.ID == 0 {
		return nil, op.ErrInvalidRefreshToken
	}
	if storedToken.RevokedAt != nil {
		return nil, op.ErrInvalidRefreshToken
	}
	if storedToken.ExpiredAt == nil || !storedToken.ExpiredAt.After(time.Now()) {
		return nil, op.ErrInvalidRefreshToken
	}

	clientID := ""
	if storedToken.OAuthClientID != 0 {
		clientEntity, err := newOAuthClientDao().GetByID(ctx, storedToken.OAuthClientID)
		if err == nil && clientEntity != nil {
			clientID = clientEntity.ClientID
		}
	}

	return &refreshTokenRequest{
		subject:  buildOIDCSubject(storedToken.PersonID),
		audience: []string{clientID},
		scopes:   []string{oidc.ScopeOpenID, oidc.ScopeProfile},
		clientID: clientID,
		amr:      []string{"pwd"},
		authTime: storedToken.CreatedAt,
	}, nil
}

type refreshTokenRequest struct {
	subject  string
	audience []string
	scopes   []string
	clientID string
	amr      []string
	authTime time.Time
}

func (r *refreshTokenRequest) GetAMR() []string                           { return r.amr }
func (r *refreshTokenRequest) GetAudience() []string                      { return r.audience }
func (r *refreshTokenRequest) GetAuthTime() time.Time                     { return r.authTime }
func (r *refreshTokenRequest) GetClientID() string                        { return r.clientID }
func (r *refreshTokenRequest) GetScopes() []string                        { return r.scopes }
func (r *refreshTokenRequest) GetSubject() string                         { return r.subject }
func (r *refreshTokenRequest) SetCurrentScopes(scopes []string)           { r.scopes = scopes }

func (s *OIDCStorage) TerminateSession(ctx context.Context, userID string, clientID string) error {
	return nil
}

func (s *OIDCStorage) RevokeToken(ctx context.Context, tokenOrTokenID string, userID string, clientID string) *oidc.Error {
	return nil
}

func (s *OIDCStorage) GetRefreshTokenInfo(ctx context.Context, clientID string, tokenValue string) (userID string, tokenID string, err error) {
	refreshTokenHash := token.HashToken(tokenValue)
	storedToken, err := dao.NewRefreshTokenDao().GetByCond(ctx, &dao.RefreshTokenCond{Token: refreshTokenHash})
	if err != nil || storedToken == nil || storedToken.ID == 0 {
		return "", "", op.ErrInvalidRefreshToken
	}
	return buildOIDCSubject(storedToken.PersonID), "", nil
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
