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
}

func NewPersistentStore() *PersistentStore {
	return &PersistentStore{
		oauthClientDao:       dao.NewOAuthClientDao,
		oauthClientSecretDao: dao.NewOAuthClientSecretDao,
		personDao:            dao.NewPersonDao,
		userDao:              dao.NewUserDao,
		refreshTokenDao:      dao.NewRefreshTokenDao,
	}
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
	return fmt.Sprintf("at-%d", time.Now().UnixNano()), time.Now().Add(time.Hour), nil
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
	userEntity := &users[0]

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
