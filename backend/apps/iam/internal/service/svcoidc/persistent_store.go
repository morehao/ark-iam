package svcoidc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/oidc/v3/pkg/op"

	"github.com/morehao/ark-iam/iam/dao"
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/token"
	"github.com/morehao/golib/glog"
)

type PersistentStore struct {
	oauthClientDao       func() *dao.OAuthClientDao
	oauthClientSecretDao func() *dao.OAuthClientSecretDao
	personDao            func() *dao.PersonDao
	userDao              func() *dao.UserDao
	refreshTokenDao      func() *dao.RefreshTokenDao
	apiKeyDao            func() *dao.ApiKeyDao
}

func NewPersistentStore() *PersistentStore {
	return &PersistentStore{
		oauthClientDao:       dao.NewOAuthClientDao,
		oauthClientSecretDao: dao.NewOAuthClientSecretDao,
		personDao:            dao.NewPersonDao,
		userDao:              dao.NewUserDao,
		refreshTokenDao:      dao.NewRefreshTokenDao,
		apiKeyDao:            dao.NewApiKeyDao,
	}
}

func (s *PersistentStore) LookupApiKeyByRawKey(ctx context.Context, rawKey string) (*model.ApiKeyEntity, error) {
	sum := sha256.Sum256([]byte(rawKey))
	hash := hex.EncodeToString(sum[:])
	entity, err := s.apiKeyDao().GetByCond(ctx, &dao.ApiKeyCond{KeyHash: hash})
	if err != nil || entity == nil || entity.ID == 0 {
		return nil, nil
	}
	if entity.RevokedAt != nil && !entity.RevokedAt.IsZero() {
		return nil, nil
	}
	if entity.ExpiredAt != nil && entity.ExpiredAt.Before(time.Now()) {
		return nil, nil
	}
	if err := s.apiKeyDao().UpdateMap(ctx, entity.ID, map[string]any{
		"last_used_at": time.Now(),
	}); err != nil {
		glog.Warnf(ctx, "[PersistentStore.LookupApiKeyByRawKey] update last_used_at fail, apiKeyID:%d, err:%v", entity.ID, err)
	}
	return entity, nil
}

func (s *PersistentStore) GetApiKeyClientByRawKey(ctx context.Context, rawKey string) (op.Client, error) {
	entity, err := s.LookupApiKeyByRawKey(ctx, rawKey)
	if err != nil || entity == nil {
		return nil, oidc.ErrInvalidClient()
	}
	return NewApiKeyOpClient(entity), nil
}

func (s *PersistentStore) GetClientByClientID(ctx context.Context, clientID string) (op.Client, error) {
	clientEntity, err := s.oauthClientDao().GetByCond(ctx, &dao.OAuthClientCond{ClientID: clientID})
	if err != nil || clientEntity == nil || clientEntity.ID == 0 {
		return nil, fmt.Errorf("client not found: %s", clientID)
	}
	return NewOIDCClient(clientEntity), nil
}

func (s *PersistentStore) AuthorizeClientIDSecret(ctx context.Context, clientID, clientSecret string) error {
	secretHash := sha256.Sum256([]byte(clientSecret))
	clientHash := hex.EncodeToString(secretHash[:])

	clientEntity, err := s.oauthClientDao().GetByCond(ctx, &dao.OAuthClientCond{ClientID: clientID})
	if err != nil || clientEntity == nil || clientEntity.ID == 0 {
		return oidc.ErrInvalidClient()
	}
	secrets, err := s.oauthClientSecretDao().GetListByCond(ctx, &dao.OAuthClientSecretCond{OAuthClientID: clientEntity.ID})
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

func (s *PersistentStore) SetUserinfoFromScopes(ctx context.Context, userinfo *oidc.UserInfo, userID, clientID string, scopes []string) error {
	userinfo.Subject = userID
	pid, err := parseOIDCSubject(userID)
	if err != nil {
		return nil
	}
	person, err := s.personDao().GetByID(ctx, pid)
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

func (s *PersistentStore) SetUserinfoFromToken(ctx context.Context, userinfo *oidc.UserInfo, tokenID, subject, origin string) error {
	userinfo.Subject = subject
	pid, err := parseOIDCSubject(subject)
	if err != nil {
		return nil
	}
	person, err := s.personDao().GetByID(ctx, pid)
	if err != nil || person == nil || person.ID == 0 {
		return nil
	}
	userinfo.Name = person.Name
	userinfo.PreferredUsername = person.Username
	userinfo.Email = person.PrimaryEmail
	userinfo.EmailVerified = true
	return nil
}

func (s *PersistentStore) SetIntrospectionFromToken(ctx context.Context, introspection *oidc.IntrospectionResponse, tokenID, subject, clientID string) error {
	return nil
}

func (s *PersistentStore) GetPrivateClaimsFromScopes(ctx context.Context, userID, clientID string, scopes []string) (map[string]any, error) {
	pid, err := parseOIDCSubject(userID)
	if err != nil {
		return nil, nil
	}
	users, err := s.userDao().GetListByCond(ctx, &dao.UserCond{PersonID: pid})
	if err != nil || len(users) == 0 {
		return nil, nil
	}
	tenantID := users[0].TenantID
	return map[string]any{
		"tenant_id": tenantID,
	}, nil
}

func (s *PersistentStore) GetKeyByIDAndClientID(ctx context.Context, keyID, clientID string) (*jose.JSONWebKey, error) {
	return nil, errors.New("key not found")
}

func (s *PersistentStore) ValidateJWTProfileScopes(ctx context.Context, userID string, scopes []string) ([]string, error) {
	return scopes, nil
}

func (s *PersistentStore) CreateAccessToken(ctx context.Context, request op.TokenRequest) (accessTokenID string, expiration time.Time, err error) {
	accessTokenID = fmt.Sprintf("at-%d", time.Now().UnixNano())

	ttl := time.Hour
	if ccReq, ok := request.(*clientCredentialsTokenRequest); ok {
		if entity, e := s.oauthClientDao().GetByCond(ctx, &dao.OAuthClientCond{ClientID: ccReq.ClientID()}); e == nil && entity != nil && entity.AccessTokenTTL > 0 {
			ttl = time.Duration(entity.AccessTokenTTL) * time.Second
		} else if e != nil {
			glog.Warnf(ctx, "[PersistentStore.CreateAccessToken] load client ttl fail, clientID:%s, err:%v", ccReq.ClientID(), e)
		}
	}
	expiration = time.Now().Add(ttl)
	return accessTokenID, expiration, nil
}

func (s *PersistentStore) CreateAccessAndRefreshTokens(ctx context.Context, request op.TokenRequest, currentRefreshToken string) (accessTokenID string, newRefreshToken string, expiration time.Time, err error) {
	accessTokenID = fmt.Sprintf("at-%d", time.Now().UnixNano())
	expiration = time.Now().Add(time.Hour)

	var personID uint
	personID, err = parseOIDCSubject(request.GetSubject())
	if err != nil {
		return "", "", time.Time{}, err
	}

	users, err := s.userDao().GetListByCond(ctx, &dao.UserCond{PersonID: personID})
	if err != nil || len(users) == 0 {
		return "", "", time.Time{}, fmt.Errorf("user not found for person %d", personID)
	}

	selectedTenantID := uint(0)
	if authReq, ok := request.(*AuthRequest); ok {
		selectedTenantID = authReq.GetTenantID()
	}
	var userEntity *model.UserEntity
	if selectedTenantID > 0 {
		for i := range users {
			if users[i].TenantID == selectedTenantID {
				userEntity = &users[i]
				break
			}
		}
	}
	if userEntity == nil {
		userEntity = &users[0]
	}

	clientID := ""
	if authReq, ok := request.(op.AuthRequest); ok {
		clientID = authReq.GetClientID()
	}

	var oauthClientID uint
	if clientID != "" {
		clientEntity, err := s.oauthClientDao().GetByCond(ctx, &dao.OAuthClientCond{ClientID: clientID})
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
	if err := s.refreshTokenDao().Insert(ctx, refreshEntity); err != nil {
		return "", "", time.Time{}, err
	}

	if currentRefreshToken != "" {
		oldTokenHash := token.HashToken(currentRefreshToken)
		oldToken, err := s.refreshTokenDao().GetByCond(ctx, &dao.RefreshTokenCond{Token: oldTokenHash})
		if err == nil && oldToken != nil && oldToken.ID != 0 {
			dbt := time.Now()
			if err := s.refreshTokenDao().UpdateMap(ctx, oldToken.ID, map[string]any{
				"revoked_at": &dbt,
			}); err != nil {
				return "", "", time.Time{}, err
			}
		}
	}

	return accessTokenID, refreshTokenValue, expiration, nil
}

func (s *PersistentStore) TokenRequestByRefreshToken(ctx context.Context, refreshToken string) (op.RefreshTokenRequest, error) {
	refreshTokenHash := token.HashToken(refreshToken)
	storedToken, err := s.refreshTokenDao().GetByCond(ctx, &dao.RefreshTokenCond{Token: refreshTokenHash})
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
		clientEntity, err := s.oauthClientDao().GetByID(ctx, storedToken.OAuthClientID)
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
		tenantID: storedToken.TenantID,
	}, nil
}

type refreshTokenRequest struct {
	subject  string
	audience []string
	scopes   []string
	clientID string
	amr      []string
	authTime time.Time
	tenantID uint
}

func (r *refreshTokenRequest) GetAMR() []string                 { return r.amr }
func (r *refreshTokenRequest) GetAudience() []string            { return r.audience }
func (r *refreshTokenRequest) GetAuthTime() time.Time           { return r.authTime }
func (r *refreshTokenRequest) GetClientID() string              { return r.clientID }
func (r *refreshTokenRequest) GetScopes() []string              { return r.scopes }
func (r *refreshTokenRequest) GetSubject() string               { return r.subject }
func (r *refreshTokenRequest) SetCurrentScopes(scopes []string) { r.scopes = scopes }
func (r *refreshTokenRequest) GetTenantID() uint                { return r.tenantID }

func (s *PersistentStore) TerminateSession(ctx context.Context, userID string, clientID string) error {
	personID, err := parseOIDCSubject(userID)
	if err != nil {
		glog.Warnf(ctx, "[PersistentStore.TerminateSession] parseOIDCSubject fail, userID:%s, err:%v", userID, err)
		return nil
	}
	glog.Infof(ctx, "[PersistentStore.TerminateSession] terminating session, userID:%s, personID:%d, clientID:%s", userID, personID, clientID)

	if ssoErr := NewSSOSessionStore().RevokeSessionsByPersonID(ctx, personID); ssoErr != nil {
		glog.Warnf(ctx, "[PersistentStore.TerminateSession] revoke SSO sessions fail, err:%v", ssoErr)
	}

	now := time.Now()
	dbErr := dbclient.IamDB(ctx).Model(&model.RefreshTokenEntity{}).Table(model.TableNameRefreshToken).
		Where("person_id = ?", personID).Where("revoked_at IS NULL").
		Update("revoked_at", &now).Error
	if dbErr != nil {
		glog.Warnf(ctx, "[PersistentStore.TerminateSession] revoke refresh tokens fail, personID:%d, err:%v", personID, dbErr)
	}
	return nil
}

func (s *PersistentStore) RevokeToken(ctx context.Context, tokenOrTokenID string, userID string, clientID string) *oidc.Error {
	refreshTokenHash := token.HashToken(tokenOrTokenID)
	storedToken, err := s.refreshTokenDao().GetByCond(ctx, &dao.RefreshTokenCond{Token: refreshTokenHash})
	if err != nil || storedToken == nil || storedToken.ID == 0 {
		return nil
	}
	now := time.Now()
	if updateErr := s.refreshTokenDao().UpdateMap(ctx, storedToken.ID, map[string]any{
		"revoked_at": &now,
	}); updateErr != nil {
		glog.Warnf(ctx, "[PersistentStore.RevokeToken] update revoked_at fail, tokenID:%d, err:%v", storedToken.ID, updateErr)
		return nil
	}
	return nil
}

func (s *PersistentStore) GetRefreshTokenInfo(ctx context.Context, clientID string, tokenValue string) (userID string, tokenID string, err error) {
	refreshTokenHash := token.HashToken(tokenValue)
	storedToken, err := s.refreshTokenDao().GetByCond(ctx, &dao.RefreshTokenCond{Token: refreshTokenHash})
	if err != nil || storedToken == nil || storedToken.ID == 0 {
		return "", "", op.ErrInvalidRefreshToken
	}
	return buildOIDCSubject(storedToken.PersonID), "", nil
}
