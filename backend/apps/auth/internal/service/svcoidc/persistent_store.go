package svcoidc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/oidc/v3/pkg/op"

	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/iam/object/objauth"
	"github.com/morehao/ark-iam/pkg/iam/sso"
	"github.com/morehao/ark-iam/pkg/token"
	"github.com/morehao/golib/glog"
)

type PersistentStore struct {
	applicationClientDao       func(opts ...dao.DaoOption) *dao.ApplicationClientDao
	applicationClientSecretDao func(opts ...dao.DaoOption) *dao.ApplicationClientSecretDao
	personDao                  func(opts ...dao.DaoOption) *dao.PersonDao
	userDao                    func(opts ...dao.DaoOption) *dao.UserDao
	refreshTokenDao            func(opts ...dao.DaoOption) *dao.RefreshTokenDao
	apiKeyDao                  func(opts ...dao.DaoOption) *dao.ApiKeyDao
}

func NewPersistentStore() *PersistentStore {
	return &PersistentStore{
		applicationClientDao:       func(opts ...dao.DaoOption) *dao.ApplicationClientDao { return dao.NewApplicationClientDao() },
		applicationClientSecretDao: dao.NewApplicationClientSecretDao,
		personDao:                  dao.NewPersonDao,
		userDao:                    dao.NewUserDao,
		refreshTokenDao:            dao.NewRefreshTokenDao,
		apiKeyDao:                  func(opts ...dao.DaoOption) *dao.ApiKeyDao { return dao.NewApiKeyDao() },
	}
}

func (s *PersistentStore) LookupApiKeyByRawKey(ctx context.Context, rawKey string) (*model.ApiKeyEntity, error) {
	sum := sha256.Sum256([]byte(rawKey))
	hash := hex.EncodeToString(sum[:])
	entity, err := s.apiKeyDao().GetByCond(ctx, &dao.ApiKeyCond{KeyHash: hash})
	if err != nil || entity == nil || entity.ID == "" {
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
		glog.Warnf(ctx, "[PersistentStore.LookupApiKeyByRawKey] update last_used_at fail, apiKeyID:%s, err:%v", entity.ID, err)
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
	clientEntity, err := s.applicationClientDao().GetByCond(ctx, &dao.ApplicationClientCond{Code: clientID})
	if err != nil || clientEntity == nil || clientEntity.ID == "" {
		return nil, fmt.Errorf("client not found: %s", clientID)
	}
	return NewOIDCClient(clientEntity), nil
}

func (s *PersistentStore) AuthorizeClientIDSecret(ctx context.Context, clientID, clientSecret string) error {
	secretHash := sha256.Sum256([]byte(clientSecret))
	clientHash := hex.EncodeToString(secretHash[:])

	clientEntity, err := s.applicationClientDao().GetByCond(ctx, &dao.ApplicationClientCond{Code: clientID})
	if err != nil || clientEntity == nil || clientEntity.ID == "" {
		return oidc.ErrInvalidClient()
	}
	secrets, err := s.applicationClientSecretDao().GetListByCond(ctx, &dao.ApplicationClientSecretCond{ApplicationClientID: clientEntity.ID})
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

// fillUserInfoByScopes 按 scope 填充 userinfo 标准声明。
// email_verified 恒为 false：当前无邮箱验证流程，不得宣称已验证（H5）。
func fillUserInfoByScopes(userinfo *oidc.UserInfo, person *model.PersonEntity, scopes []string) {
	for _, scope := range scopes {
		switch scope {
		case oidc.ScopeProfile:
			userinfo.Name = person.Name
			userinfo.PreferredUsername = model.DerefStr(person.Username)
		case oidc.ScopeEmail:
			userinfo.Email = model.DerefStr(person.PrimaryEmail)
			if userinfo.Email != "" {
				userinfo.EmailVerified = false
			}
		case oidc.ScopePhone:
			userinfo.PhoneNumber = model.DerefStr(person.PrimaryPhone)
			userinfo.PhoneNumberVerified = false
		}
	}
}

func (s *PersistentStore) SetUserinfoFromScopes(ctx context.Context, userinfo *oidc.UserInfo, userID, clientID string, scopes []string) error {
	userinfo.Subject = userID
	pid, err := parseOIDCSubject(userID)
	if err != nil {
		return nil
	}
	person, err := s.personDao().GetByID(ctx, pid)
	if err != nil || person == nil || person.ID == "" {
		return nil
	}
	fillUserInfoByScopes(userinfo, person, scopes)
	return nil
}

// SetUserinfoFromToken 按 access token 签发时记录的 scope 裁剪 userinfo 声明（M2）。
// 元数据不可得（Redis 不可用 / 未知 token）时仅返回 sub，不返回任何个人信息，
// 避免绕过 scope 授权泄露 email/name 等声明。
func (s *PersistentStore) SetUserinfoFromToken(ctx context.Context, userinfo *oidc.UserInfo, tokenID, subject, origin string) error {
	userinfo.Subject = subject
	meta := loadAccessTokenMeta(ctx, tokenID)
	if meta == nil {
		return nil
	}
	pid, err := parseOIDCSubject(subject)
	if err != nil {
		return nil
	}
	person, err := s.personDao().GetByID(ctx, pid)
	if err != nil || person == nil || person.ID == "" {
		return nil
	}
	fillUserInfoByScopes(userinfo, person, meta.Scopes)
	return nil
}

// SetIntrospectionFromToken 返回 RFC 7662 规定的完整 introspection 响应（M1）：
// scope/client_id/sub/exp/iat/token_type/username 及私有声明。
// 元数据不可得时保持空响应（active 由 op 层在 SetIntrospectionFromToken 成功后置 true）。
func (s *PersistentStore) SetIntrospectionFromToken(ctx context.Context, introspection *oidc.IntrospectionResponse, tokenID, subject, clientID string) error {
	meta := loadAccessTokenMeta(ctx, tokenID)
	if meta == nil {
		return nil
	}
	introspection.Scope = oidc.SpaceDelimitedArray(meta.Scopes)
	introspection.ClientID = meta.ClientID
	introspection.TokenType = oidc.BearerToken
	introspection.Expiration = oidc.FromTime(meta.ExpiresAt)
	introspection.IssuedAt = oidc.FromTime(meta.IssuedAt)
	introspection.Subject = meta.Subject
	introspection.Audience = oidc.Audience{meta.ClientID}
	introspection.Username = meta.Username
	// 人 token 补充 username（person 的用户名）；机器 token 直接用 client 标识。
	if introspection.Username == "" {
		if pid, perr := parseOIDCSubject(meta.Subject); perr == nil {
			if person, perr2 := s.personDao().GetByID(ctx, pid); perr2 == nil && person != nil && person.ID != "" {
				introspection.Username = model.DerefStr(person.Username)
			}
		} else {
			introspection.Username = meta.ClientID
		}
	}
	claims := make(map[string]any, 3)
	if meta.TenantID != "" {
		claims["tenant_id"] = meta.TenantID
	}
	if meta.TokenUsage != "" {
		claims["token_usage"] = meta.TokenUsage
	}
	if meta.SessionID != "" {
		claims["sid"] = meta.SessionID
	}
	if len(claims) > 0 {
		introspection.Claims = claims
	}
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
	// L4：请求未携带明确租户上下文时，只有单租户才可确定 tenant_id；
	// 多租户（users>1）存在歧义，宁可不产出 claim 也不静默取 users[0]。
	if len(users) != 1 {
		return nil, nil
	}
	return objauth.TokenClaims{TenantID: users[0].TenantID}.OIDCPrivateClaims(), nil
}

func (s *PersistentStore) GetKeyByIDAndClientID(ctx context.Context, keyID, clientID string) (*jose.JSONWebKey, error) {
	return nil, errors.New("key not found")
}

func (s *PersistentStore) ValidateJWTProfileScopes(ctx context.Context, userID string, scopes []string) ([]string, error) {
	return scopes, nil
}

func (s *PersistentStore) CreateAccessToken(ctx context.Context, request op.TokenRequest) (accessTokenID string, expiration time.Time, err error) {
	accessTokenID, err = randomTokenID("at")
	if err != nil {
		return "", time.Time{}, fmt.Errorf("generate access token id: %w", err)
	}

	ttl := 15 * time.Minute
	if ccReq, ok := request.(*clientCredentialsTokenRequest); ok {
		if entity, e := s.applicationClientDao().GetByCond(ctx, &dao.ApplicationClientCond{Code: ccReq.ClientID()}); e == nil && entity != nil && entity.AccessTokenTTL > 0 {
			ttl = time.Duration(entity.AccessTokenTTL) * time.Second
		} else if e != nil {
			glog.Warnf(ctx, "[PersistentStore.CreateAccessToken] load client ttl fail, clientID:%s, err:%v", ccReq.ClientID(), e)
		}
	}
	expiration = time.Now().Add(ttl)
	storeAccessTokenMeta(ctx, accessTokenID, accessTokenMeta{
		Subject:    request.GetSubject(),
		ClientID:   getClientIDFromRequest(request),
		Scopes:     request.GetScopes(),
		IssuedAt:   time.Now(),
		ExpiresAt:  expiration,
		TenantID:   tenantIDFromRequest(request),
		TokenUsage: tokenUsageFromRequest(request),
	})
	return accessTokenID, expiration, nil
}

func (s *PersistentStore) CreateAccessAndRefreshTokens(ctx context.Context, request op.TokenRequest, currentRefreshToken string) (accessTokenID string, newRefreshToken string, expiration time.Time, err error) {
	accessTokenID, err = randomTokenID("at")
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("generate access token id: %w", err)
	}

	var personID string
	personID, err = parseOIDCSubject(request.GetSubject())
	if err != nil {
		return "", "", time.Time{}, err
	}

	users, err := s.userDao().GetListByCond(ctx, &dao.UserCond{PersonID: personID})
	if err != nil || len(users) == 0 {
		return "", "", time.Time{}, fmt.Errorf("user not found for person %s", personID)
	}

	selectedTenantID := selectedTenantFromRequest(request)
	var userEntity *model.UserEntity
	if selectedTenantID != "" {
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

	var applicationClientID string
	var clientAccessTokenTTL time.Duration
	var clientRefreshTokenTTL time.Duration
	backChannelLogoutURI := ""
	if clientID != "" {
		clientEntity, err := s.applicationClientDao().GetByCond(ctx, &dao.ApplicationClientCond{Code: clientID})
		if err == nil && clientEntity != nil {
			applicationClientID = clientEntity.ID
			backChannelLogoutURI = clientEntity.BackChannelLogoutURI
			if clientEntity.AccessTokenTTL > 0 {
				clientAccessTokenTTL = time.Duration(clientEntity.AccessTokenTTL) * time.Second
			}
			if clientEntity.RefreshTokenTTL > 0 {
				clientRefreshTokenTTL = time.Duration(clientEntity.RefreshTokenTTL) * time.Second
			}
		}
	}
	if clientAccessTokenTTL <= 0 {
		clientAccessTokenTTL = 15 * time.Minute
	}
	expiration = time.Now().Add(clientAccessTokenTTL)

	// H2：refresh token 持久化授权时授予的 scope / amr / auth_time，
	// 刷新时原样还原（RFC 6749 §6 要求刷新 token 的 scope 与原始授权一致）。
	scopes := append([]string(nil), request.GetScopes()...)
	// op.TokenRequest 不含 amr/auth_time（仅 IDTokenRequest 有），
	// 授权码流/刷新流的具体请求类型均实现之，此处用接口断言提取。
	var amr []string
	if r, ok := request.(interface{ GetAMR() []string }); ok {
		amr = append([]string(nil), r.GetAMR()...)
	}
	authTime := time.Time{}
	if r, ok := request.(interface{ GetAuthTime() time.Time }); ok {
		authTime = r.GetAuthTime()
	}
	if authTime.IsZero() {
		authTime = time.Now()
	}

	sessionID := ""
	if authReq, ok := request.(*AuthRequest); ok {
		sessionID = authReq.SessionID
	}

	now := time.Now()
	refreshTokenExp := now.Add(30 * 24 * time.Hour)
	if clientRefreshTokenTTL > 0 {
		refreshTokenExp = now.Add(clientRefreshTokenTTL)
	}
	// refresh token 是长期凭证，值必须为密码学随机（不可由时间戳+ID 推断）。
	refreshTokenValue, err := randomTokenID("rt")
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("generate refresh token: %w", err)
	}

	refreshTokenHash := token.HashToken(refreshTokenValue)
	scopesJSON, _ := json.Marshal(scopes)
	amrJSON, _ := json.Marshal(amr)
	refreshEntity := &model.RefreshTokenEntity{
		PersonID:            personID,
		TenantID:            userEntity.TenantID,
		UserID:              userEntity.ID,
		ApplicationClientID: applicationClientID,
		SessionID:           sessionID,
		Token:               refreshTokenHash,
		Scopes:              scopesJSON,
		AMR:                 amrJSON,
		AuthTime:            &authTime,
		ExpiredAt:           &refreshTokenExp,
		CreatedBy:           userEntity.ID,
	}
	if currentRefreshToken != "" {
		refreshEntity.LastRotatedAt = &now
	}
	if err := s.refreshTokenDao().Insert(ctx, refreshEntity); err != nil {
		return "", "", time.Time{}, err
	}

	// 登出登记：有 SSO 会话 且 client 配置了 back_channel_logout_uri 时，登记该会话对该 client 的通知关系。
	// 无 sid（服务账号/Client Credentials）或未配置背信道 URI 则跳过，对齐 OIDC Back-Channel 注册要求。
	if sessionID != "" && backChannelLogoutURI != "" {
		_ = sso.NewSLOStore().Register(ctx, sessionID, sso.LogoutRegistration{
			OIDCSessionID:        accessTokenID,
			ClientID:             clientID,
			UserID:               buildOIDCSubject(personID),
			BackChannelLogoutURI: backChannelLogoutURI,
		})
	}

	if currentRefreshToken != "" {
		oldTokenHash := token.HashToken(currentRefreshToken)
		oldToken, err := s.refreshTokenDao().GetByCond(ctx, &dao.RefreshTokenCond{Token: oldTokenHash})
		if err == nil && oldToken != nil && oldToken.ID != "" {
			dbt := time.Now()
			if err := s.refreshTokenDao().UpdateMap(ctx, oldToken.ID, map[string]any{
				"revoked_at": &dbt,
			}); err != nil {
				return "", "", time.Time{}, err
			}
		}
	}

	// access token 元数据（M1/M2：introspection 与 userinfo 按 scope 裁剪）。
	storeAccessTokenMeta(ctx, accessTokenID, accessTokenMeta{
		Subject:    request.GetSubject(),
		ClientID:   clientID,
		Scopes:     scopes,
		IssuedAt:   time.Now(),
		ExpiresAt:  expiration,
		TenantID:   userEntity.TenantID,
		SessionID:  sessionID,
		TokenUsage: "",
	})

	return accessTokenID, refreshTokenValue, expiration, nil
}

// tenantIDFromRequest / tokenUsageFromRequest 供 client_credentials 的 access token 元数据使用。
func tenantIDFromRequest(request op.TokenRequest) string {
	if ccReq, ok := request.(*clientCredentialsTokenRequest); ok {
		return ccReq.ownerTenantID
	}
	return ""
}

func tokenUsageFromRequest(request op.TokenRequest) string {
	if ccReq, ok := request.(*clientCredentialsTokenRequest); ok && ccReq.isApiKey {
		return objauth.TokenUsageMachine
	}
	return ""
}

func (s *PersistentStore) TokenRequestByRefreshToken(ctx context.Context, refreshToken string) (op.RefreshTokenRequest, error) {
	refreshTokenHash := token.HashToken(refreshToken)
	storedToken, err := s.refreshTokenDao().GetByCond(ctx, &dao.RefreshTokenCond{Token: refreshTokenHash})
	if err != nil || storedToken == nil || storedToken.ID == "" {
		return nil, op.ErrInvalidRefreshToken
	}
	if storedToken.RevokedAt != nil {
		return nil, op.ErrInvalidRefreshToken
	}
	if storedToken.ExpiredAt == nil || !storedToken.ExpiredAt.After(time.Now()) {
		return nil, op.ErrInvalidRefreshToken
	}

	clientID := ""
	if storedToken.ApplicationClientID != "" {
		clientEntity, err := s.applicationClientDao().GetByID(ctx, storedToken.ApplicationClientID)
		if err == nil && clientEntity != nil {
			clientID = clientEntity.Code
		}
	}

	// H2：还原授权时持久化的 scope / amr / auth_time / session_id，
	// 保证刷新后的 token 与原始授权一致（RFC 6749 §6），并让刷新后的 token 携带 sid。
	scopes := decodeJSONStringSlice(storedToken.Scopes)
	if len(scopes) == 0 {
		scopes = []string{oidc.ScopeOpenID, oidc.ScopeProfile}
	}
	amr := decodeJSONStringSlice(storedToken.AMR)
	if len(amr) == 0 {
		amr = []string{"pwd"}
	}
	authTime := storedToken.CreatedAt
	if storedToken.AuthTime != nil && !storedToken.AuthTime.IsZero() {
		authTime = *storedToken.AuthTime
	}

	return &refreshTokenRequest{
		subject:   buildOIDCSubject(storedToken.PersonID),
		audience:  []string{clientID},
		scopes:    scopes,
		clientID:  clientID,
		amr:       amr,
		authTime:  authTime,
		tenantID:  storedToken.TenantID,
		sessionID: storedToken.SessionID,
	}, nil
}

type refreshTokenRequest struct {
	subject   string
	audience  []string
	scopes    []string
	clientID  string
	amr       []string
	authTime  time.Time
	tenantID  string
	sessionID string
}

func (r *refreshTokenRequest) GetAMR() []string                 { return r.amr }
func (r *refreshTokenRequest) GetAudience() []string            { return r.audience }
func (r *refreshTokenRequest) GetAuthTime() time.Time           { return r.authTime }
func (r *refreshTokenRequest) GetClientID() string              { return r.clientID }
func (r *refreshTokenRequest) GetScopes() []string              { return r.scopes }
func (r *refreshTokenRequest) GetSubject() string               { return r.subject }
func (r *refreshTokenRequest) SetCurrentScopes(scopes []string) { r.scopes = scopes }
func (r *refreshTokenRequest) GetTenantID() string              { return r.tenantID }
func (r *refreshTokenRequest) GetSessionID() string             { return r.sessionID }

// decodeJSONStringSlice 解析 JSON 字符串数组；解析失败返回 nil。
func decodeJSONStringSlice(raw []byte) []string {
	if len(raw) == 0 {
		return nil
	}
	var out []string
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return out
}

// selectedTenantFromRequest 返回请求携带的租户 ID（authorization code 或 refresh token 轮换时从其存储的 tenant 读取）。
// 未设置（TenantID == ""）时返回 0，由调用方决定回退逻辑。
func selectedTenantFromRequest(request op.TokenRequest) string {
	if ar, ok := request.(*AuthRequest); ok {
		return ar.GetTenantID()
	}
	if rr, ok := request.(*refreshTokenRequest); ok {
		return rr.GetTenantID()
	}
	return ""
}

func (s *PersistentStore) TerminateSession(ctx context.Context, userID string, clientID string) error {
	personID, err := parseOIDCSubject(userID)
	if err != nil {
		glog.Warnf(ctx, "[PersistentStore.TerminateSession] parseOIDCSubject fail, userID:%s, err:%v", userID, err)
		return nil
	}
	glog.Infof(ctx, "[PersistentStore.TerminateSession] terminating session, userID:%s, personID:%s, clientID:%s", userID, personID, clientID)

	if ssoErr := sso.NewSSOSessionStore().RevokeSessionsByPersonID(ctx, personID); ssoErr != nil {
		glog.Warnf(ctx, "[PersistentStore.TerminateSession] revoke SSO sessions fail, err:%v", ssoErr)
	}

	now := time.Now()
	dbErr := dbclient.IamDB(ctx).Model(&model.RefreshTokenEntity{}).Table(model.TableNameRefreshToken).
		Where("person_id = ?", personID).Where("revoked_at IS NULL").
		Update("revoked_at", &now).Error
	if dbErr != nil {
		glog.Warnf(ctx, "[PersistentStore.TerminateSession] revoke refresh tokens fail, personID:%s, err:%v", personID, dbErr)
	}
	return nil
}

func (s *PersistentStore) RevokeToken(ctx context.Context, tokenOrTokenID string, userID string, clientID string) *oidc.Error {
	refreshTokenHash := token.HashToken(tokenOrTokenID)
	storedToken, err := s.refreshTokenDao().GetByCond(ctx, &dao.RefreshTokenCond{Token: refreshTokenHash})
	if err != nil || storedToken == nil || storedToken.ID == "" {
		return nil
	}
	now := time.Now()
	if updateErr := s.refreshTokenDao().UpdateMap(ctx, storedToken.ID, map[string]any{
		"revoked_at": &now,
	}); updateErr != nil {
		glog.Warnf(ctx, "[PersistentStore.RevokeToken] update revoked_at fail, tokenID:%s, err:%v", storedToken.ID, updateErr)
		return nil
	}
	return nil
}

func (s *PersistentStore) GetRefreshTokenInfo(ctx context.Context, clientID string, tokenValue string) (userID string, tokenID string, err error) {
	refreshTokenHash := token.HashToken(tokenValue)
	storedToken, err := s.refreshTokenDao().GetByCond(ctx, &dao.RefreshTokenCond{Token: refreshTokenHash})
	if err != nil || storedToken == nil || storedToken.ID == "" {
		return "", "", op.ErrInvalidRefreshToken
	}
	return buildOIDCSubject(storedToken.PersonID), "", nil
}
