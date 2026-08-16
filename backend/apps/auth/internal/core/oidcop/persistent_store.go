package oidcop

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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
	"gorm.io/gorm"
)

// errRefreshTokenReused 标记 refresh token 复用（已被并发轮换或撤销）。
// 按 RFC 9706 §4.1 检测到复用时应撤销整个 token 家族。
var errRefreshTokenReused = errors.New("refresh token reused")

// persistentStoreOption 承载持久化存储的可注入配置。
type persistentStoreOption struct {
	// issuer 为 OP 的 issuer，构造 OIDCClient 时注入，用于生成 LoginURL。
	issuer string
}

// PersistentStoreOption 允许调用方为持久化存储注入配置。
type PersistentStoreOption func(*persistentStoreOption)

// WithIssuer 注入 OP issuer，使签发的 OIDCClient 能构造正确的 LoginURL。
func WithIssuer(issuer string) PersistentStoreOption {
	return func(o *persistentStoreOption) { o.issuer = issuer }
}

type PersistentStore struct {
	applicationClientDao       func(opts ...dao.DaoOption) *dao.ApplicationClientDao
	applicationClientSecretDao func(opts ...dao.DaoOption) *dao.ApplicationClientSecretDao
	personDao                  func(opts ...dao.DaoOption) *dao.PersonDao
	userDao                    func(opts ...dao.DaoOption) *dao.UserDao
	refreshTokenDao            func(opts ...dao.DaoOption) *dao.RefreshTokenDao
	apiKeyDao                  func(opts ...dao.DaoOption) *dao.ApiKeyDao
	issuer                     string
	// db 返回用于事务的 DB 句柄（轮换原子性等）。默认全局 iam 库，
	// 测试可注入独立 SQLite 连接。
	db func(ctx context.Context) *gorm.DB
}

func NewPersistentStore(opts ...PersistentStoreOption) *PersistentStore {
	cfg := &persistentStoreOption{}
	for _, opt := range opts {
		opt(cfg)
	}
	return &PersistentStore{
		applicationClientDao:       func(opts ...dao.DaoOption) *dao.ApplicationClientDao { return dao.NewApplicationClientDao() },
		applicationClientSecretDao: dao.NewApplicationClientSecretDao,
		personDao:                  dao.NewPersonDao,
		userDao:                    dao.NewUserDao,
		refreshTokenDao:            dao.NewRefreshTokenDao,
		apiKeyDao:                  func(opts ...dao.DaoOption) *dao.ApiKeyDao { return dao.NewApiKeyDao() },
		issuer:                     cfg.issuer,
		db:                         dbclient.IamDB,
	}
}

// txDB 返回事务用 DB；未注入时回退全局 iam 库。
func (s *PersistentStore) txDB(ctx context.Context) *gorm.DB {
	if s.db != nil {
		if db := s.db(ctx); db != nil {
			return db
		}
	}
	return dbclient.IamDB(ctx)
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
	// last_used_at 写入降频：一分钟窗口内不重复写，避免每个请求都触发一次 DB 写
	if !entity.LastUsedAt.Valid || time.Since(entity.LastUsedAt.Time) > time.Minute {
		if err := s.apiKeyDao().UpdateMap(ctx, entity.ID, map[string]any{
			"last_used_at": time.Now(),
		}); err != nil {
			glog.Warnf(ctx, "[PersistentStore.LookupApiKeyByRawKey] update last_used_at fail, apiKeyID:%s, err:%v", entity.ID, err)
		}
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
	// H4：仅返回启用状态的 client，管理员停用后 authorize/token/client_credentials 立即失效
	clientEntity, err := s.applicationClientDao().GetByCond(ctx, &dao.ApplicationClientCond{Code: clientID, Status: model.ApplicationClientStatusEnable})
	if err != nil || clientEntity == nil || clientEntity.ID == "" {
		return nil, fmt.Errorf("client not found: %s", clientID)
	}
	return NewOIDCClient(clientEntity, s.issuer), nil
}

func (s *PersistentStore) AuthorizeClientIDSecret(ctx context.Context, clientID, clientSecret string) error {
	secretHash := sha256.Sum256([]byte(clientSecret))
	clientHash := hex.EncodeToString(secretHash[:])

	// H4：停用 client 的 secret 一律拒绝
	clientEntity, err := s.applicationClientDao().GetByCond(ctx, &dao.ApplicationClientCond{Code: clientID, Status: model.ApplicationClientStatusEnable})
	if err != nil || clientEntity == nil || clientEntity.ID == "" {
		return oidc.ErrInvalidClient()
	}
	secrets, err := s.applicationClientSecretDao().GetListByCond(ctx, &dao.ApplicationClientSecretCond{ApplicationClientID: clientEntity.ID})
	if err != nil {
		return oidc.ErrInvalidClient()
	}
	for _, sec := range secrets {
		// H14：哈希比较使用恒定时间算法，避免时序侧信道
		if subtle.ConstantTimeCompare([]byte(sec.ValueHash), []byte(clientHash)) == 1 && sec.RevokedAt == nil {
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
	pid, err := ParseSubject(userID)
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
// 已被撤销（黑名单）或元数据不可得（Redis 不可用 / 未知 token）时返回 error，
// 由 op 层拒绝请求（403），避免绕过 scope 授权泄露 email/name 等声明。
func (s *PersistentStore) SetUserinfoFromToken(ctx context.Context, userinfo *oidc.UserInfo, tokenID, subject, origin string) error {
	userinfo.Subject = subject
	if isAccessTokenRevoked(ctx, tokenID) {
		return errors.New("access token revoked")
	}
	meta := loadAccessTokenMeta(ctx, tokenID)
	if meta == nil {
		return errors.New("access token meta not found")
	}
	pid, err := ParseSubject(subject)
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
// 元数据不可得或 token 已被撤销时返回 error，op 层将保持 active=false。
func (s *PersistentStore) SetIntrospectionFromToken(ctx context.Context, introspection *oidc.IntrospectionResponse, tokenID, subject, clientID string) error {
	if isAccessTokenRevoked(ctx, tokenID) {
		return errors.New("access token revoked")
	}
	meta := loadAccessTokenMeta(ctx, tokenID)
	if meta == nil {
		return errors.New("access token meta not found")
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
		if pid, perr := ParseSubject(meta.Subject); perr == nil {
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
	pid, err := ParseSubject(userID)
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

	// sessionID：code 流且 client 未启用 refresh_token grant 时也走本方法，
	// 此时从 AuthRequest 提取会话标识，保证这类 client 也能收到 back-channel 登出通知。
	sessionID := ""
	if authReq, ok := request.(*AuthRequest); ok {
		sessionID = authReq.SessionID
	}

	ttl := 15 * time.Minute
	var backChannelLogoutURI string
	var clientID string
	if ccReq, ok := request.(*clientCredentialsTokenRequest); ok {
		clientID = ccReq.ClientID()
		if entity, e := s.applicationClientDao().GetByCond(ctx, &dao.ApplicationClientCond{Code: clientID}); e == nil && entity != nil && entity.AccessTokenTTL > 0 {
			ttl = time.Duration(entity.AccessTokenTTL) * time.Second
		} else if e != nil {
			glog.Warnf(ctx, "[PersistentStore.CreateAccessToken] load client ttl fail, clientID:%s, err:%v", clientID, e)
		}
	} else if authReq, ok := request.(*AuthRequest); ok {
		clientID = authReq.GetClientID()
		if clientID != "" {
			if entity, e := s.applicationClientDao().GetByCond(ctx, &dao.ApplicationClientCond{Code: clientID}); e == nil && entity != nil {
				backChannelLogoutURI = entity.BackChannelLogoutURI
				if entity.AccessTokenTTL > 0 {
					ttl = time.Duration(entity.AccessTokenTTL) * time.Second
				}
			}
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
		SessionID:  sessionID,
		TokenUsage: tokenUsageFromRequest(request),
	})
	// back-channel 登记：与 CreateAccessAndRefreshTokens 对齐，覆盖"仅 access token"的 code 流
	if sessionID != "" && backChannelLogoutURI != "" {
		pid, perr := ParseSubject(request.GetSubject())
		if perr == nil {
			_ = sso.NewSLOStore().Register(ctx, sessionID, sso.LogoutRegistration{
				OIDCSessionID:        accessTokenID,
				ClientID:             clientID,
				UserID:               BuildSubject(pid),
				SessionID:            sessionID,
				BackChannelLogoutURI: backChannelLogoutURI,
			})
		}
	}
	return accessTokenID, expiration, nil
}

func (s *PersistentStore) CreateAccessAndRefreshTokens(ctx context.Context, request op.TokenRequest, currentRefreshToken string) (accessTokenID string, newRefreshToken string, expiration time.Time, err error) {
	accessTokenID, err = randomTokenID("at")
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("generate access token id: %w", err)
	}

	var personID string
	personID, err = ParseSubject(request.GetSubject())
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
	if rr, ok := request.(*refreshTokenRequest); ok {
		// 刷新轮换必须还原授权时持久化的会话标识（M4），否则刷新后的 token 丢失 sid，
		// 无法再按会话粒度关联背信道登出与 id_token 的 sid 声明。
		sessionID = rr.GetSessionID()
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

	// S7：新行插入与旧行撤销放同一事务，且旧行撤销采用条件 UPDATE
	// （WHERE token=? AND revoked_at IS NULL，行数=1 才算成功），
	// 杜绝并发刷新时两个请求都通过校验、各自产出一套新 token 的分裂。
	// 条件撤销命中 0 行说明旧 token 已被并发轮换/撤销 → 视为复用攻击
	// （RFC 9706 §4.1），撤销该 person 全部 refresh token（token 家族）。
	txErr := s.txDB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(refreshEntity).Error; err != nil {
			return err
		}
		if currentRefreshToken != "" {
			oldTokenHash := token.HashToken(currentRefreshToken)
			res := tx.Model(&model.RefreshTokenEntity{}).Table(model.TableNameRefreshToken).
				Where("token = ? AND revoked_at IS NULL", oldTokenHash).
				Update("revoked_at", &now)
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return errRefreshTokenReused
			}
		}
		return nil
	})
	if txErr != nil {
		if errors.Is(txErr, errRefreshTokenReused) {
			// 复用：旧 token 家族全部作废（独立于回滚的事务执行）
			glog.Warnf(ctx, "[PersistentStore.CreateAccessAndRefreshTokens] refresh token reused, revoke family, personID:%s", personID)
			famErr := s.txDB(ctx).Model(&model.RefreshTokenEntity{}).Table(model.TableNameRefreshToken).
				Where("person_id = ? AND revoked_at IS NULL", personID).
				Update("revoked_at", &now).Error
			if famErr != nil {
				glog.Warnf(ctx, "[PersistentStore.CreateAccessAndRefreshTokens] revoke token family fail, personID:%s, err:%v", personID, famErr)
			}
			return "", "", time.Time{}, op.ErrInvalidRefreshToken
		}
		return "", "", time.Time{}, txErr
	}

	// 登出登记：有 SSO 会话 且 client 配置了 back_channel_logout_uri 时，登记该会话对该 client 的通知关系。
	// 无 sid（服务账号/Client Credentials）或未配置背信道 URI 则跳过，对齐 OIDC Back-Channel 注册要求。
	if sessionID != "" && backChannelLogoutURI != "" {
		_ = sso.NewSLOStore().Register(ctx, sessionID, sso.LogoutRegistration{
			OIDCSessionID:        accessTokenID,
			ClientID:             clientID,
			UserID:               BuildSubject(personID),
			SessionID:            sessionID,
			BackChannelLogoutURI: backChannelLogoutURI,
		})
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
		subject:   BuildSubject(storedToken.PersonID),
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
	personID, err := ParseSubject(userID)
	if err != nil {
		glog.Warnf(ctx, "[PersistentStore.TerminateSession] ParseSubject fail, userID:%s, err:%v", userID, err)
		return nil
	}
	glog.Infof(ctx, "[PersistentStore.TerminateSession] terminating session, userID:%s, personID:%s, clientID:%s", userID, personID, clientID)

	if ssoErr := sso.NewSSOSessionStore().RevokeSessionsByPersonID(ctx, personID); ssoErr != nil {
		glog.Warnf(ctx, "[PersistentStore.TerminateSession] revoke SSO sessions fail, err:%v", ssoErr)
	}

	now := time.Now()
	dbErr := s.txDB(ctx).Model(&model.RefreshTokenEntity{}).Table(model.TableNameRefreshToken).
		Where("person_id = ?", personID).Where("revoked_at IS NULL").
		Update("revoked_at", &now).Error
	if dbErr != nil {
		glog.Warnf(ctx, "[PersistentStore.TerminateSession] revoke refresh tokens fail, personID:%s, err:%v", personID, dbErr)
	}
	return nil
}

func (s *PersistentStore) RevokeToken(ctx context.Context, tokenOrTokenID string, userID string, clientID string) *oidc.Error {
	if tokenOrTokenID == "" {
		return nil
	}
	// access token jti（at- 前缀）→ 加入 Redis 黑名单，使 OP 侧 userinfo/introspection 拒绝
	if strings.HasPrefix(tokenOrTokenID, "at-") {
		revokeAccessToken(ctx, tokenOrTokenID)
		return nil
	}
	// refresh token 行 ID → 按主键撤销（RFC 7009：GetRefreshTokenInfo 返回行 ID 后由 op 传入）
	q := s.txDB(ctx).Model(&model.RefreshTokenEntity{}).Table(model.TableNameRefreshToken).
		Where("id = ?", tokenOrTokenID)
	if userID != "" {
		if pid, perr := ParseSubject(userID); perr == nil {
			q = q.Where("person_id = ?", pid)
		}
	}
	if clientID != "" {
		if entity, cErr := s.applicationClientDao().GetByCond(ctx, &dao.ApplicationClientCond{Code: clientID}); cErr == nil && entity != nil && entity.ID != "" {
			q = q.Where("application_client_id = ?", entity.ID)
		}
	}
	now := time.Now()
	if updateErr := q.Update("revoked_at", &now).Error; updateErr != nil {
		glog.Warnf(ctx, "[PersistentStore.RevokeToken] update revoked_at fail, tokenID:%s, err:%v", tokenOrTokenID, updateErr)
	}
	return nil
}

func (s *PersistentStore) GetRefreshTokenInfo(ctx context.Context, clientID string, tokenValue string) (userID string, tokenID string, err error) {
	refreshTokenHash := token.HashToken(tokenValue)
	storedToken, err := s.refreshTokenDao().GetByCond(ctx, &dao.RefreshTokenCond{Token: refreshTokenHash})
	if err != nil || storedToken == nil || storedToken.ID == "" {
		return "", "", op.ErrInvalidRefreshToken
	}
	if storedToken.RevokedAt != nil {
		return "", "", op.ErrInvalidRefreshToken
	}
	if storedToken.ExpiredAt == nil || !storedToken.ExpiredAt.After(time.Now()) {
		return "", "", op.ErrInvalidRefreshToken
	}
	// RFC 7009 §2.1：token 必须属于发起撤销请求的 client
	if clientID != "" {
		if entity, cErr := s.applicationClientDao().GetByID(ctx, storedToken.ApplicationClientID); cErr == nil && entity != nil && entity.Code != "" {
			if entity.Code != clientID {
				return "", "", op.ErrInvalidRefreshToken
			}
		}
	}
	// 返回行 ID（而非空串）：zitadel 撤销流程会用它作为待撤销 token 传给 RevokeToken
	return BuildSubject(storedToken.PersonID), storedToken.ID, nil
}
