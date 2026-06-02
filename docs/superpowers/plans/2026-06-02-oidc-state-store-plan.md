# OIDC 协议状态存储重构 — 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**目标:** 将 OIDC 授权流程中短期协议状态（AuthRequest、authorization code）从进程内存迁移到 Redis，同时将 `OIDCStorage` 按职责拆分为三层适配架构

**架构:** `ProtocolStateStore`（Redis 短期状态） + `PersistentStore`（DAO/DB 长期数据） + `OIDCStorage`（op.Storage 适配层编排）

**Tech Stack:** Go, Redis, Gin, GORM, zitadel/oidc v3

---

### 文件映射

| 操作 | 文件 | 说明 |
|------|------|------|
| 创建 | `backend/apps/iam/internal/service/svcoidc/protocol_state_store.go` | ProtocolStateStore 接口 + error + 序列化 + Redis 实现 |
| 创建 | `backend/apps/iam/internal/service/svcoidc/persistent_store.go` | PersistentStore 结构体 + DAO 委托实现 |
| 创建 | `backend/apps/iam/internal/service/svcoidc/protocol_state_store_test.go` | ProtocolStateStore 基于真实 Redis 的测试 |
| 修改 | `backend/apps/iam/config/config.go` | OIDC 结构体新增 TTL 配置 |
| 修改 | `backend/apps/iam/config/config.yaml` | 新增 TTL 配置默认值 |
| 修改 | `backend/apps/iam/internal/service/svcoidc/oidc.go` | SetupOIDCProvider 传入新依赖 |
| 修改 | `backend/apps/iam/internal/service/svcoidc/storage.go` | 移除 authRequestStore，注入两 Store，删除函数变量 |
| 修改 | `backend/apps/iam/internal/service/svcoidc/storage_test.go` | 删除 map 依赖测试，保留纯函数测试 |
| 修改 | `backend/apps/iam/internal/service/svcoidc/oidc_login_test.go` | 适配新构造函数 |
| 修改 | `backend/apps/iam/internal/service/svcoidc/provider_flow_test.go` | 改用真实 Redis |

---

### Task 1: 添加 OIDC TTL 配置

**Files:**
- Modify: `backend/apps/iam/config/config.go:33-42`
- Modify: `backend/apps/iam/config/config.yaml` (oidc 段)

- [ ] **Step 1: config.go 新增 TTL 字段**

```go
type OIDC struct {
	Issuer                string `yaml:"issuer"`
	FrontendLoginURL      string `yaml:"frontendLoginURL"`
	SigningKeyID          string `yaml:"signingKeyID"`
	SigningPrivateKeyPath string `yaml:"signingPrivateKeyPath"`
	SigningPrivateKeyPEM  string `yaml:"signingPrivateKeyPEM"`
	EncryptionKey         string `yaml:"encryptionKey"`
	EncryptionKeyID       string `yaml:"encryptionKeyID"`
	AllowInsecure         bool   `yaml:"allowInsecure"`
	AuthRequestTTL        int    `yaml:"authRequestTTL"`   // 秒，默认 600
	AuthCodeTTL           int    `yaml:"authCodeTTL"`      // 秒，默认 300
	SpentCodeTTL          int    `yaml:"spentCodeTTL"`     // 秒，默认 86400
}
```

- [ ] **Step 2: config.yaml oidc 段新增 TTL**

```yaml
oidc:
  issuer: "http://localhost:8099/v1/iam/oidc"
  frontendLoginURL: "http://localhost:3000/oidc/login"
  signingKeyID: "dev-oidc-key"
  signingPrivateKeyPath: "config/oidc-dev-key.pem"
  encryptionKey: "oidc-dev-encryption-key-32bytes"
  encryptionKeyID: "dev-enc-key"
  allowInsecure: true
  authRequestTTL: 600
  authCodeTTL: 300
  spentCodeTTL: 86400
```

- [ ] **Step 3: Commit**

```bash
git add backend/apps/iam/config/config.go backend/apps/iam/config/config.yaml
git commit -m "feat(oidc): add TTL config for protocol state store"
```

---

### Task 2: 定义 ProtocolStateStore 接口和错误

**Files:**
- Create: `backend/apps/iam/internal/service/svcoidc/protocol_state_store.go` (上半部分)

- [ ] **Step 1: 编写 error 定义和接口**

```go
package svcoidc

import (
	"context"
	"time"

	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/oidc/v3/pkg/op"
)

var (
	ErrStoreUnavailable        = errors.New("oidc protocol state store unavailable")
	ErrSessionNotFound         = errors.New("auth request not found")
	ErrCodeInvalid             = errors.New("authorization code invalid")
	ErrCodeAlreadyUsed         = errors.New("authorization code already used")
	ErrCodeCollision           = errors.New("authorization code collision")
	ErrSessionNotCompleted     = errors.New("auth request not completed")
)

type ProtocolStateStore interface {
	CreateAuthRequest(ctx context.Context, authReq *oidc.AuthRequest, userID string) (op.AuthRequest, error)
	AuthRequestByID(ctx context.Context, id string) (op.AuthRequest, error)
	AuthRequestByCode(ctx context.Context, code string) (op.AuthRequest, error)
	SaveAuthCode(ctx context.Context, id, code string) error
	CompleteAuthRequest(id string, subject string, authTime time.Time, amr []string, acr string) error
	DeleteAuthRequest(ctx context.Context, id string) error
	Health(ctx context.Context) error
}
```

- [ ] **Step 2: 给现有 AuthRequest 结构体添加 json tags（storage.go）**

改为：

```go
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
}
```

- [ ] **Step 3: 确认 `ExpiresAt` 和 `DoneFlag` 字段的 Get/Set 方法不受新字段影响**

`AuthRequest` 已有的 `GetID()` / `Done()` / `GetSubject()` 等方法不需要改动，只新增了一个 `ExpiresAt` 字段。

- [ ] **Step 4: Commit**

```bash
git add backend/apps/iam/internal/service/svcoidc/protocol_state_store.go backend/apps/iam/internal/service/svcoidc/storage.go
git commit -m "feat(oidc): define ProtocolStateStore interface and errors"
```

---

### Task 3: 实现 RedisProtocolStateStore

**Files:**
- Create: `backend/apps/iam/internal/service/svcoidc/protocol_state_store.go` (下半部分)

- [ ] **Step 1: 编写 key 构造辅助函数**

```go
const (
	authRequestKeyPrefix = "iam:oidc:auth_req:"
	authCodeKeyPrefix    = "iam:oidc:auth_code:"
	spentCodeKeyPrefix   = "iam:oidc:auth_code:spent:"
)

func authRequestKey(id string) string { return authRequestKeyPrefix + id }
func authCodeKey(code string) string  { return authCodeKeyPrefix + code }
func spentCodeKey(code string) string { return spentCodeKeyPrefix + code }
```

- [ ] **Step 2: 编写 defaultTTL 辅助函数**

```go
func defaultAuthRequestTTL() time.Duration {
	if appconfig.Conf != nil && appconfig.Conf.OIDC.AuthRequestTTL > 0 {
		return time.Duration(appconfig.Conf.OIDC.AuthRequestTTL) * time.Second
	}
	return 10 * time.Minute
}

func defaultAuthCodeTTL() time.Duration {
	if appconfig.Conf != nil && appconfig.Conf.OIDC.AuthCodeTTL > 0 {
		return time.Duration(appconfig.Conf.OIDC.AuthCodeTTL) * time.Second
	}
	return 5 * time.Minute
}

func defaultSpentCodeTTL() time.Duration {
	if appconfig.Conf != nil && appconfig.Conf.OIDC.SpentCodeTTL > 0 {
		return time.Duration(appconfig.Conf.OIDC.SpentCodeTTL) * time.Second
	}
	return 24 * time.Hour
}
```

- [ ] **Step 3: 编写 RedisProtocolStateStore 结构体和构造函数**

```go
type RedisProtocolStateStore struct {
	client *redis.Client
}

func NewRedisProtocolStateStore() *RedisProtocolStateStore {
	if dbclient.RedisCli == nil {
		panic("redis client not initialized for OIDC protocol state store")
	}
	return &RedisProtocolStateStore{client: dbclient.RedisCli}
}
```

- [ ] **Step 4: 实现 Health**

```go
func (s *RedisProtocolStateStore) Health(ctx context.Context) error {
	if err := s.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("%w: %v", ErrStoreUnavailable, err)
	}
	return nil
}
```

- [ ] **Step 5: 实现 CreateAuthRequest**

```go
func (s *RedisProtocolStateStore) CreateAuthRequest(ctx context.Context, authReq *oidc.AuthRequest, userID string) (op.AuthRequest, error) {
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
		ExpiresAt:    time.Now().Add(defaultAuthRequestTTL()),
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
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal auth request: %w", err)
	}
	if err := s.client.Set(ctx, authRequestKey(req.ID), data, defaultAuthRequestTTL()).Err(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrStoreUnavailable, err)
	}
	return req, nil
}
```

- [ ] **Step 6: 实现 AuthRequestByID**

```go
func (s *RedisProtocolStateStore) AuthRequestByID(ctx context.Context, id string) (op.AuthRequest, error) {
	data, err := s.client.Get(ctx, authRequestKey(id)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrStoreUnavailable, err)
	}
	var req AuthRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("unmarshal auth request: %w", err)
	}
	if time.Now().After(req.ExpiresAt) {
		return nil, ErrSessionNotFound
	}
	return &req, nil
}
```

- [ ] **Step 7: 实现 CompleteAuthRequest**

```go
func (s *RedisProtocolStateStore) CompleteAuthRequest(id string, subject string, authTime time.Time, amr []string, acr string) error {
	ctx := context.Background()
	data, err := s.client.Get(ctx, authRequestKey(id)).Bytes()
	if errors.Is(err, redis.Nil) {
		return ErrSessionNotFound
	}
	if err != nil {
		return fmt.Errorf("%w: %w", ErrStoreUnavailable, err)
	}
	var req AuthRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return fmt.Errorf("unmarshal auth request: %w", err)
	}
	req.Subject = subject
	req.AuthTime = authTime
	req.AMR = append([]string(nil), amr...)
	req.ACR = acr
	req.DoneFlag = true
	updated, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal auth request: %w", err)
	}
	ttl := time.Until(req.ExpiresAt)
	if ttl <= 0 {
		return ErrSessionNotFound
	}
	if err := s.client.Set(ctx, authRequestKey(id), updated, ttl).Err(); err != nil {
		return fmt.Errorf("%w: %w", ErrStoreUnavailable, err)
	}
	return nil
}
```

- [ ] **Step 8: 实现 SaveAuthCode**

```go
func (s *RedisProtocolStateStore) SaveAuthCode(ctx context.Context, id, code string) error {
	ok, err := s.client.SetNX(ctx, authCodeKey(code), id, defaultAuthCodeTTL()).Result()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrStoreUnavailable, err)
	}
	if !ok {
		return ErrCodeCollision
	}
	return nil
}
```

- [ ] **Step 9: 实现 AuthRequestByCode**

```go
func (s *RedisProtocolStateStore) AuthRequestByCode(ctx context.Context, code string) (op.AuthRequest, error) {
	return s.consumeAuthCode(ctx, code, false)
}

func (s *RedisProtocolStateStore) consumeAuthCode(ctx context.Context, code string, deleteCode bool) (op.AuthRequest, error) {
	var requestID string
	if deleteCode {
		val, err := s.client.GetDel(ctx, authCodeKey(code)).Result()
		if errors.Is(err, redis.Nil) {
			return nil, s.checkSpentOrInvalid(ctx, code)
		}
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrStoreUnavailable, err)
		}
		requestID = val
	} else {
		val, err := s.client.Get(ctx, authCodeKey(code)).Result()
		if errors.Is(err, redis.Nil) {
			return nil, s.checkSpentOrInvalid(ctx, code)
		}
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrStoreUnavailable, err)
		}
		requestID = val
	}
	data, err := s.client.Get(ctx, authRequestKey(requestID)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrStoreUnavailable, err)
	}
	var req AuthRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("unmarshal auth request: %w", err)
	}
	if !req.DoneFlag {
		return nil, ErrSessionNotCompleted
	}
	if time.Now().After(req.ExpiresAt) {
		return nil, ErrSessionNotFound
	}
	return &req, nil
}

func (s *RedisProtocolStateStore) checkSpentOrInvalid(ctx context.Context, code string) error {
	exists, err := s.client.Exists(ctx, spentCodeKey(code)).Result()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrStoreUnavailable, err)
	}
	if exists == 1 {
		return ErrCodeAlreadyUsed
	}
	return ErrCodeInvalid
}
```

- [ ] **Step 10: 实现 DeleteAuthRequest**

```go
func (s *RedisProtocolStateStore) DeleteAuthRequest(ctx context.Context, id string) error {
	if err := s.client.Del(ctx, authRequestKey(id)).Err(); err != nil {
		return fmt.Errorf("%w: %w", ErrStoreUnavailable, err)
	}
	return nil
}
```

- [ ] **Step 11: 实现 ConsumeAuthCode（专用方法，用于 token 端点的消费）**

```go
func (s *RedisProtocolStateStore) ConsumeAuthCode(ctx context.Context, code string) (op.AuthRequest, error) {
	return s.consumeAuthCode(ctx, code, true)
}
```

在 `AuthRequestByCode` 上添加 `ConsumeAuthCode` 方法到 `ProtocolStateStore` 接口中：

```go
type ProtocolStateStore interface {
	// ... 前面的方法 ...
	ConsumeAuthCode(ctx context.Context, code string) (op.AuthRequest, error)
}
```

- [ ] **Step 12: Commit**

```bash
git add backend/apps/iam/internal/service/svcoidc/protocol_state_store.go
git commit -m "feat(oidc): implement RedisProtocolStateStore"
```

---

### Task 4: 定义 PersistentStore

**Files:**
- Create: `backend/apps/iam/internal/service/svcoidc/persistent_store.go`

- [ ] **Step 1: 编写 PersistentStore 结构体和构造函数**

```go
package svcoidc

import (
	"context"
	"fmt"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/oidc/v3/pkg/op"

	"github.com/morehao/ark-iam/iam/dao"
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/pkg/token"
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
```

- [ ] **Step 2: 将 OIDCStorage 中所有 DAO 相关方法复制到 PersistentStore**

```go
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
	secrets, _, err := s.oauthClientSecretDao().GetPageListByCond(ctx, &dao.OAuthClientSecretCond{OAuthClientID: clientEntity.ID})
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
	return nil, nil
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
			s.refreshTokenDao().UpdateMap(ctx, oldToken.ID, map[string]any{
				"revoked_at": &dbt,
			})
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
	}, nil
}

func (s *PersistentStore) TerminateSession(ctx context.Context, userID string, clientID string) error {
	return nil
}

func (s *PersistentStore) RevokeToken(ctx context.Context, tokenOrTokenID string, userID string, clientID string) *oidc.Error {
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
```

- [ ] **Step 2: Commit**

```bash
git add backend/apps/iam/internal/service/svcoidc/persistent_store.go
git commit -m "feat(oidc): define PersistentStore with DAO operations"
```

---

### Task 5: 重构 OIDCStorage

**Files:**
- Modify: `backend/apps/iam/internal/service/svcoidc/storage.go`

- [ ] **Step 1: 替换 OIDCStorage 结构体**

```go
type OIDCStorage struct {
	protocolStore  ProtocolStateStore
	persistentStore *PersistentStore
	signingKey     *rsa.PrivateKey
	signingKeyID   string
}
```

- [ ] **Step 2: 替换构造函数**

```go
func NewOIDCStorage(protocolStore ProtocolStateStore, persistentStore *PersistentStore, signingKey *rsa.PrivateKey, keyID string) *OIDCStorage {
	return &OIDCStorage{
		protocolStore:    protocolStore,
		persistentStore:  persistentStore,
		signingKey:       signingKey,
		signingKeyID:     keyID,
	}
}
```

- [ ] **Step 3: 删除 authRequestStore 结构体和 authRequestStore 字段**

删除 `authRequestStore struct`（第 25-29 行）、`AuthRequest` 改为只保留 json tags 的版本（已在 Task 2 做）。

- [ ] **Step 4: 删除 `newOAuthClientDao` 和 `newOAuthClientSecretDao` 函数变量**

删除第 99-105 行。

- [ ] **Step 5: 将各方法改为委托调用**

- `CreateAuthRequest` → `s.protocolStore.CreateAuthRequest(ctx, authReq, userID)`
- `AuthRequestByID` → `s.protocolStore.AuthRequestByID(ctx, id)`
- `AuthRequestByCode` → `s.protocolStore.AuthRequestByCode(ctx, code)` （查询语义，不消费）
- `SaveAuthCode` → `s.protocolStore.SaveAuthCode(ctx, id, code)`
- `CompleteAuthRequest` → `s.protocolStore.CompleteAuthRequest(id, subject, authTime, amr, acr)`
- `DeleteAuthRequest` → `s.protocolStore.DeleteAuthRequest(ctx, id)`
- `Health` → `s.protocolStore.Health(ctx)` （检查 Redis）
- `GetClientByClientID` → `s.persistentStore.GetClientByClientID(ctx, clientID)`
- `AuthorizeClientIDSecret` → `s.persistentStore.AuthorizeClientIDSecret(ctx, clientID, clientSecret)`
- `SetUserinfoFromScopes` → `s.persistentStore.SetUserinfoFromScopes(ctx, userinfo, userID, clientID, scopes)`
- `SetUserinfoFromToken` → `s.persistentStore.SetUserinfoFromToken(ctx, userinfo, tokenID, subject, origin)`
- `SetIntrospectionFromToken` → `s.persistentStore.SetIntrospectionFromToken(ctx, introspection, tokenID, subject, clientID)`
- `GetPrivateClaimsFromScopes` → `s.persistentStore.GetPrivateClaimsFromScopes(ctx, userID, clientID, scopes)`
- `GetKeyByIDAndClientID` → `s.persistentStore.GetKeyByIDAndClientID(ctx, keyID, clientID)`
- `ValidateJWTProfileScopes` → `s.persistentStore.ValidateJWTProfileScopes(ctx, userID, scopes)`
- `CreateAccessToken` → `s.persistentStore.CreateAccessToken(ctx, request)`
- `CreateAccessAndRefreshTokens` → `s.persistentStore.CreateAccessAndRefreshTokens(ctx, request, currentRefreshToken)`
- `TokenRequestByRefreshToken` → `s.persistentStore.TokenRequestByRefreshToken(ctx, refreshToken)`
- `TerminateSession` → `s.persistentStore.TerminateSession(ctx, userID, clientID)`
- `RevokeToken` → `s.persistentStore.RevokeToken(ctx, tokenOrTokenID, userID, clientID)`
- `GetRefreshTokenInfo` → `s.persistentStore.GetRefreshTokenInfo(ctx, clientID, tokenValue)`
- `SigningKey` → 保持不变（自己持有 key）
- `SignatureAlgorithms` → 保持不变
- `KeySet` → 保持不变

- [ ] **Step 6: 删除不再需要的 imports**

删除不必要的 `sync`、`strconv`、`strings` 等不再需要的包。

- [ ] **Step 7: 确认 storage.go 编译通过**

Run: `go build ./backend/apps/iam/...`
Expected: PASS

- [ ] **Step 8: Commit**

```bash
git add backend/apps/iam/internal/service/svcoidc/storage.go
git commit -m "refactor(oidc): delegate OIDCStorage to ProtocolStateStore and PersistentStore"
```

---

### Task 6: 更新 OIDCProvider 构造

**Files:**
- Modify: `backend/apps/iam/internal/service/svcoidc/oidc.go`

- [ ] **Step 1: 修改 SetupOIDCProvider**

```go
func SetupOIDCProvider(issuer string) (*OIDCProvider, error) {
	privateKey, keyID, err := loadSigningKey()
	if err != nil {
		return nil, fmt.Errorf("failed to load OIDC signing key: %w", err)
	}

	protocolStore := NewRedisProtocolStateStore()
	persistentStore := NewPersistentStore()
	storage := NewOIDCStorage(protocolStore, persistentStore, privateKey, keyID)

	// ... 后续不变，storage 已经是一个 *OIDCStorage ...
}
```

- [ ] **Step 2: 验证编译**

Run: `go build ./backend/apps/iam/...`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add backend/apps/iam/internal/service/svcoidc/oidc.go
git commit -m "refactor(oidc): inject ProtocolStateStore and PersistentStore into OIDCStorage"
```

---

### Task 7: 编写 ProtocolStateStore 测试

**Files:**
- Create: `backend/apps/iam/internal/service/svcoidc/protocol_state_store_test.go`

- [ ] **Step 1: 编写测试文件框架**

```go
package svcoidc

import (
	"context"
	"testing"
	"time"

	"github.com/morehao/ark-iam/iam/testutil"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/testsetup"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zitadel/oidc/v3/pkg/oidc"
)
```

- [ ] **Step 2: 编写 TestCreateAndGetAuthRequest**

```go
func TestCreateAndGetAuthRequest(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	store := NewRedisProtocolStateStore()
	req, err := store.CreateAuthRequest(context.Background(), &oidc.AuthRequest{
		ClientID:     "client-1",
		RedirectURI:  "https://client.example.com/callback",
		State:        "state-1",
		Scopes:       []string{oidc.ScopeOpenID, oidc.ScopeProfile},
		ResponseType: oidc.ResponseTypeCode,
		ResponseMode: oidc.ResponseModeQuery,
		Nonce:        "nonce-1",
	}, "")
	require.NoError(t, err)
	require.NotEmpty(t, req.GetID())
	assert.False(t, req.Done())

	found, err := store.AuthRequestByID(context.Background(), req.GetID())
	require.NoError(t, err)
	assert.Equal(t, req.GetID(), found.GetID())
	assert.Equal(t, "client-1", found.GetClientID())
}
```

Run: `go test -count=1 -v ./internal/service/svcoidc/... -run TestCreateAndGetAuthRequest`
Expected: PASS

- [ ] **Step 3: 编写 TestAuthRequestByIDNotFound**

```go
func TestAuthRequestByIDNotFound(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	store := NewRedisProtocolStateStore()
	_, err := store.AuthRequestByID(context.Background(), "nonexistent")
	assert.ErrorIs(t, err, ErrSessionNotFound)
}
```

Run: `go test -count=1 -v ./internal/service/svcoidc/... -run TestAuthRequestByIDNotFound`
Expected: PASS

- [ ] **Step 4: 编写 TestCompleteAuthRequest**

```go
func TestCompleteAuthRequest(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	store := NewRedisProtocolStateStore()
	req, err := store.CreateAuthRequest(context.Background(), &oidc.AuthRequest{
		ClientID:     "client-1",
		RedirectURI:  "https://client.example.com/callback",
		State:        "state-1",
		Scopes:       []string{oidc.ScopeOpenID, oidc.ScopeProfile},
		ResponseType: oidc.ResponseTypeCode,
		ResponseMode: oidc.ResponseModeQuery,
	}, "")
	require.NoError(t, err)

	authTime := time.Unix(1710000000, 0)
	err = store.CompleteAuthRequest(req.GetID(), "person:88", authTime, []string{"pwd"}, "")
	require.NoError(t, err)

	found, err := store.AuthRequestByID(context.Background(), req.GetID())
	require.NoError(t, err)
	assert.True(t, found.Done())
	assert.Equal(t, "person:88", found.GetSubject())
	assert.Equal(t, authTime.Unix(), found.GetAuthTime().Unix())
	assert.Equal(t, []string{"pwd"}, found.GetAMR())
}
```

Run: `go test -count=1 -v ./internal/service/svcoidc/... -run TestCompleteAuthRequest`
Expected: PASS

- [ ] **Step 5: 编写 TestSaveAndConsumeCode**

```go
func TestSaveAndConsumeCode(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	store := NewRedisProtocolStateStore()
	req, err := store.CreateAuthRequest(context.Background(), &oidc.AuthRequest{
		ClientID:     "client-1",
		RedirectURI:  "https://client.example.com/callback",
		State:        "state-1",
		Scopes:       []string{oidc.ScopeOpenID, oidc.ScopeProfile},
		ResponseType: oidc.ResponseTypeCode,
		ResponseMode: oidc.ResponseModeQuery,
	}, "")
	require.NoError(t, err)

	store.CompleteAuthRequest(req.GetID(), "person:88", time.Now(), []string{"pwd"}, "")

	code := "auth-code-123"
	err = store.SaveAuthCode(context.Background(), req.GetID(), code)
	require.NoError(t, err)

	found, err := store.AuthRequestByCode(context.Background(), code)
	require.NoError(t, err)
	assert.Equal(t, req.GetID(), found.GetID())
	assert.True(t, found.Done())

	// Consume 后再次读取应失败
	found, err = store.ConsumeAuthCode(context.Background(), code)
	require.NoError(t, err)
	assert.Equal(t, req.GetID(), found.GetID())

	_, err = store.AuthRequestByCode(context.Background(), code)
	assert.ErrorIs(t, err, ErrCodeAlreadyUsed)
}
```

Run: `go test -count=1 -v ./internal/service/svcoidc/... -run TestSaveAndConsumeCode`
Expected: PASS

- [ ] **Step 6: 编写 TestDeleteAuthRequest**

```go
func TestDeleteAuthRequest(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	store := NewRedisProtocolStateStore()
	req, err := store.CreateAuthRequest(context.Background(), &oidc.AuthRequest{
		ClientID:     "client-1",
		RedirectURI:  "https://client.example.com/callback",
		State:        "state-1",
		Scopes:       []string{oidc.ScopeOpenID, oidc.ScopeProfile},
		ResponseType: oidc.ResponseTypeCode,
		ResponseMode: oidc.ResponseModeQuery,
	}, "")
	require.NoError(t, err)

	err = store.DeleteAuthRequest(context.Background(), req.GetID())
	require.NoError(t, err)

	_, err = store.AuthRequestByID(context.Background(), req.GetID())
	assert.ErrorIs(t, err, ErrSessionNotFound)
}
```

Run: `go test -count=1 -v ./internal/service/svcoidc/... -run TestDeleteAuthRequest`
Expected: PASS

- [ ] **Step 7: 编写 TestConsumeCodeNotCompleted**

```go
func TestConsumeCodeNotCompleted(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	store := NewRedisProtocolStateStore()
	req, err := store.CreateAuthRequest(context.Background(), &oidc.AuthRequest{
		ClientID:     "client-1",
		RedirectURI:  "https://client.example.com/callback",
		State:        "state-1",
		Scopes:       []string{oidc.ScopeOpenID, oidc.ScopeProfile},
		ResponseType: oidc.ResponseTypeCode,
		ResponseMode: oidc.ResponseModeQuery,
	}, "")
	require.NoError(t, err)

	// 不 CompleteAuthRequest 直接 SaveAuthCode
	code := "code-not-completed"
	err = store.SaveAuthCode(context.Background(), req.GetID(), code)
	require.NoError(t, err)

	_, err = store.AuthRequestByCode(context.Background(), code)
	assert.ErrorIs(t, err, ErrSessionNotCompleted)
}
```

Run: `go test -count=1 -v ./internal/service/svcoidc/... -run TestConsumeCodeNotCompleted`
Expected: PASS

- [ ] **Step 8: 编写 TestHealthCheck**

```go
func TestHealthCheck(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	store := NewRedisProtocolStateStore()
	err := store.Health(context.Background())
	require.NoError(t, err)
}
```

Run: `go test -count=1 -v ./internal/service/svcoidc/... -run TestHealthCheck`
Expected: PASS

- [ ] **Step 9: Commit**

```bash
git add backend/apps/iam/internal/service/svcoidc/protocol_state_store_test.go
git commit -m "test(oidc): add ProtocolStateStore tests with real Redis"
```

---

### Task 8: 适配现有存储测试

**Files:**
- Modify: `backend/apps/iam/internal/service/svcoidc/storage_test.go`

- [ ] **Step 1: 删除已迁移到 ProtocolStateStore 测试的用例**

删除 `TestCompleteAuthRequestMarksRequestDone`（已在 Task 7 覆盖）。

保留 `TestBuildOIDCSubject`、`TestParseOIDCSubject`、`TestParseOIDCSubjectRejectsInvalidFormat`、`TestOIDCClientLoginURLUsesConfiguredFrontend`。

- [ ] **Step 2: 更新 TestSigningKeyUsesAsymmetricPrivateKey**

```go
func TestSigningKeyUsesAsymmetricPrivateKey(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	protocolStore := NewRedisProtocolStateStore()
	persistentStore := NewPersistentStore()
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	storage := NewOIDCStorage(protocolStore, persistentStore, key, "test-key")

	signingKey, err := storage.SigningKey(context.Background())
	require.NoError(t, err)
	_, ok := signingKey.Key().(*rsa.PrivateKey)
	assert.True(t, ok)
}
```

- [ ] **Step 3: 更新 TestKeySetExposesPublicKeyOnly**

```go
func TestKeySetExposesPublicKeyOnly(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	protocolStore := NewRedisProtocolStateStore()
	persistentStore := NewPersistentStore()
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	storage := NewOIDCStorage(protocolStore, persistentStore, key, "test-key")

	keys, err := storage.KeySet(context.Background())
	require.NoError(t, err)
	require.Len(t, keys, 1)
	_, ok := keys[0].Key().(*rsa.PublicKey)
	assert.True(t, ok)
}
```

- [ ] **Step 4: 验证测试通过**

Run: `go test -count=1 -v ./internal/service/svcoidc/... -run 'Test(Build|Parse|LoginURL|Signing|KeySet|Client)'`
Expected: 过滤掉已删除的测试，余下全部 PASS

- [ ] **Step 5: Commit**

```bash
git add backend/apps/iam/internal/service/svcoidc/storage_test.go
git commit -m "test(oidc): adapt storage tests for refactored OIDCStorage"
```

---

### Task 9: 适配完整的流程测试

**Files:**
- Modify: `backend/apps/iam/internal/service/svcoidc/oidc_login_test.go`
- Modify: `backend/apps/iam/internal/service/svcoidc/provider_flow_test.go`

- [ ] **Step 1: 更新 oidc_login_test.go 中 SetupOIDCProvider 的测试**

当前测试已使用 `SetupOIDCProvider`，自动获得 Redis 实现。但需要一个运行中的 Redis。

```go
func TestCompleteLoginReturnsContinueURLAndCompletesRequest(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	appconfig.Conf = &appconfig.Config{
		JWT: appconfig.JWT{SignKey: "test-sign-key"},
		OIDC: appconfig.OIDC{
			Issuer:           "http://localhost:8099/v1/iam/oidc",
			FrontendLoginURL: "http://localhost:3000/oidc/login",
			AllowInsecure:    true,
		},
	}
	provider, err := SetupOIDCProvider(appconfig.Conf.OIDC.Issuer)
	require.NoError(t, err)
	// ... 后续不变 ...
}
```

- [ ] **Step 2: 更新 provider_flow_test.go**

```go
func TestFullOIDCCodeFlow(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	issuer := "http://localhost:8099/v1/iam/oidc"
	appconfig.Conf = &appconfig.Config{
		JWT: appconfig.JWT{SignKey: "test-sign-key"},
		OIDC: appconfig.OIDC{
			Issuer:        issuer,
			AllowInsecure: true,
		},
	}

	provider, err := SetupOIDCProvider(issuer)
	require.NoError(t, err)
	// ... 后续请求变为通过 provider.Storage 进行，底层已经是 Redis ...
}
```

注意：`provider_flow_test.go` 中的 `newOAuthClientDao` 和 `newOAuthClientSecretDao` 函数变量不再使用（已在 `OIDCStorage` 中删除），改用 `NewPersistentStore()`。需要创建一个在测试中 mock DAO 的 `PersistentStore`，或直接使用真实 DAO。

简便做法：保留 OIDCStorage 之前的测试 mock 路径，改为 PersistentStore 的注入：

```go
origNewOAuthClientDao := dao.NewOAuthClientDao
origNewOAuthClientSecretDao := dao.NewOAuthClientSecretDao
t.Cleanup(func() {
	dao.NewOAuthClientDao = origNewOAuthClientDao
	dao.NewOAuthClientSecretDao = origNewOAuthClientSecretDao
})
dao.NewOAuthClientDao = func() *dao.OAuthClientDao { return &dao.OAuthClientDao{} }
dao.NewOAuthClientSecretDao = func() *dao.OAuthClientSecretDao { return &dao.OAuthClientSecretDao{} }
```

然后将 `NewOIDCStorage` 的调用改为使用 mock DAO 的 PersistentStore：

```go
persistentStore := &PersistentStore{
	oauthClientDao:       func() *dao.OAuthClientDao { return &dao.OAuthClientDao{} },
	oauthClientSecretDao: func() *dao.OAuthClientSecretDao { return &dao.OAuthClientSecretDao{} },
	personDao:            dao.NewPersonDao,
	userDao:              dao.NewUserDao,
	refreshTokenDao:      dao.NewRefreshTokenDao,
}
```

但这样 PersistentStore 需要导出字段或提供配置方法。更好的方案：向 `PersistentStore` 添加 With 选项或测试专用构造函数。

```go
func NewTestPersistentStore(
	oauthClientDao func() *dao.OAuthClientDao,
	oauthClientSecretDao func() *dao.OAuthClientSecretDao,
) *PersistentStore {
	return &PersistentStore{
		oauthClientDao:       oauthClientDao,
		oauthClientSecretDao: oauthClientSecretDao,
		personDao:            dao.NewPersonDao,
		userDao:              dao.NewUserDao,
		refreshTokenDao:      dao.NewRefreshTokenDao,
	}
}
```

或者在 `persistent_store.go` 中新增：

```go
func (s *PersistentStore) WithMockOAuthClientDao(fn func() *dao.OAuthClientDao) *PersistentStore {
	s.oauthClientDao = fn
	return s
}
```

推荐使用第二个方案（fluent setter），对生产代码无侵入。

- [ ] **Step 3: 更新 provider_flow_test.go 中的 CompleteLogin 测试**

将 `fakePasswordAuthenticator` 保持不变，但 provider 的创建改为通过 `SetupOIDCProvider` + mock PersistentStore：

```go
func TestFullOIDCCodeFlow(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	issuer := "http://localhost:8099/v1/iam/oidc"
	appconfig.Conf = &appconfig.Config{
		JWT: appconfig.JWT{SignKey: "test-sign-key"},
		OIDC: appconfig.OIDC{
			Issuer:        issuer,
			AllowInsecure: true,
		},
	}

	protocolStore := NewRedisProtocolStateStore()
	persistentStore := NewPersistentStore()
	persistentStore.WithMockOAuthClientDao(func() *dao.OAuthClientDao { return &dao.OAuthClientDao{} })
	persistentStore.WithMockOAuthClientSecretDao(func() *dao.OAuthClientSecretDao { return &dao.OAuthClientSecretDao{} })

	privateKey, keyID, err := loadSigningKey()
	require.NoError(t, err)
	storage := NewOIDCStorage(protocolStore, persistentStore, privateKey, keyID)

	// ... rest of test using storage directly ...
}
```

- [ ] **Step 4: 运行完整测试**

Run: `go test -count=1 -v ./internal/service/svcoidc/... 2>&1 | grep -E '(PASS|FAIL|---)'`
Expected: 所有测试 PASS

- [ ] **Step 5: Commit**

```bash
git add backend/apps/iam/internal/service/svcoidc/oidc_login_test.go backend/apps/iam/internal/service/svcoidc/provider_flow_test.go
git commit -m "test(oidc): adapt flow tests for Redis-backed state store"
```

---

### Task 10: 全量验证与收尾

**Files:** 无

- [ ] **Step 1: 编译检查**

```bash
go build ./backend/apps/iam/... && go vet ./backend/apps/iam/...
```
Expected: 无错误

- [ ] **Step 2: 跑全部 svcoidc 测试**

```bash
go test -count=1 -v ./internal/service/svcoidc/... 2>&1 | grep -E '(PASS|FAIL|---)'
```
Expected: 全部 PASS

- [ ] **Step 3: 跑全部 router 测试（OIDC 路由）**

```bash
go test -count=1 -v ./internal/router/... 2>&1 | grep -E '(PASS|FAIL|---)'
```
Expected: 全部 PASS

- [ ] **Step 4: 检查未使用的 import 和变量**

```bash
go vet ./...
```
Expected: 无 warning

- [ ] **Step 5: 最终提交**

```bash
git add -A
git commit -m "refactor(oidc): complete OIDC state store refactoring to Redis-backed architecture"
```

---

### 自检清单

- [ ] 每个 spec 需求都能对应到一个任务
  - [x] 三层架构 → Task 2, 4, 5
  - [x] Redis key 设计 → Task 3（协议状态 store 实现）
  - [x] TTL 配置 → Task 1
  - [x] 消费语义和防重放 → Task 3
  - [x] 错误处理 → Task 2（sentinel errors）+ Task 3（各方法实现）
  - [x] Redis 不可用时严格失败 → Constructor panic + Health check
  - [x] 测试策略（真实 Redis） → Task 7, 8, 9
  - [x] 清理 `newOAuthClientDao` 等函数变量 → Task 5
- [ ] 没有 "TBD"/"TODO" 占位符
- [ ] 类型和方法签名在各任务间一致
- [ ] 范围聚焦在 OIDC 状态存储重构，没有膨胀
