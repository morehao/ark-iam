package oidcop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/redis/go-redis/v9"
	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/oidc/v3/pkg/op"
)

var (
	ErrStoreUnavailable               = errors.New("oidc protocol state store unavailable")
	ErrSessionNotFound                = errors.New("auth request not found")
	ErrCodeInvalid                    = errors.New("authorization code invalid")
	ErrCodeAlreadyUsed                = errors.New("authorization code already used")
	ErrCodeCollision                  = errors.New("authorization code collision")
	ErrSessionNotCompleted            = errors.New("auth request not completed")
	ErrCodeChallengeMethodUnsupported = errors.New("unsupported code challenge method")
)

type ProtocolStateStore interface {
	CreateAuthRequest(ctx context.Context, authReq *oidc.AuthRequest, userID string, tenantID string) (op.AuthRequest, error)
	AuthRequestByID(ctx context.Context, id string) (op.AuthRequest, error)
	AuthRequestByCode(ctx context.Context, code string) (op.AuthRequest, error)
	SaveAuthCode(ctx context.Context, id, code string) error
	CompleteAuthRequest(ctx context.Context, id string, subject string, authTime time.Time, amr []string, acr string, tenantID string, done bool) error
	AssociateSession(ctx context.Context, id string, sessionID string) error
	DeleteAuthRequest(ctx context.Context, id string) error
	ConsumeAuthCode(ctx context.Context, code string) (op.AuthRequest, error)
	Health(ctx context.Context) error
}

const (
	authRequestKeyPrefix = "iam:oidc:auth_req:"
	authCodeKeyPrefix    = "iam:oidc:auth_code:"
	spentCodeKeyPrefix   = "iam:oidc:auth_code:spent:"
)

func authRequestKey(id string) string { return authRequestKeyPrefix + id }
func authCodeKey(code string) string  { return authCodeKeyPrefix + code }
func spentCodeKey(code string) string { return spentCodeKeyPrefix + code }

// 协议态默认 TTL（可经 NewRedisProtocolStateStore 的选项覆盖，由组装方注入应用配置值）。
const (
	defaultAuthRequestTTL = 10 * time.Minute
	defaultAuthCodeTTL    = 5 * time.Minute
	defaultSpentCodeTTL   = 24 * time.Hour
)

// protocolStateStoreOption 承载 Redis 协议态存储的可注入配置。
type protocolStateStoreOption struct {
	authRequestTTL time.Duration
	authCodeTTL    time.Duration
	spentCodeTTL   time.Duration
}

// ProtocolStateOption 允许调用方为协议态存储注入 TTL 等配置。
type ProtocolStateOption func(*protocolStateStoreOption)

// WithAuthRequestTTL 设置授权票据 TTL。
func WithAuthRequestTTL(ttl time.Duration) ProtocolStateOption {
	return func(o *protocolStateStoreOption) { o.authRequestTTL = ttl }
}

// WithAuthCodeTTL 设置授权码 TTL。
func WithAuthCodeTTL(ttl time.Duration) ProtocolStateOption {
	return func(o *protocolStateStoreOption) { o.authCodeTTL = ttl }
}

// WithSpentCodeTTL 设置已消费授权码防重放 TTL。
func WithSpentCodeTTL(ttl time.Duration) ProtocolStateOption {
	return func(o *protocolStateStoreOption) { o.spentCodeTTL = ttl }
}

type RedisProtocolStateStore struct {
	client         *redis.Client
	authRequestTTL time.Duration
	authCodeTTL    time.Duration
	spentCodeTTL   time.Duration
}

var _ ProtocolStateStore = (*RedisProtocolStateStore)(nil)

// NewRedisProtocolStateStore 构造 Redis 协议态存储，默认读取全局 dbclient.RedisCli，
// 调用方可经 ProtocolStateOption 注入 TTL 等配置。
func NewRedisProtocolStateStore(opts ...ProtocolStateOption) *RedisProtocolStateStore {
	if dbclient.RedisCli == nil {
		panic("redis client not initialized for OIDC protocol state store")
	}
	cfg := protocolStateStoreOption{
		authRequestTTL: defaultAuthRequestTTL,
		authCodeTTL:    defaultAuthCodeTTL,
		spentCodeTTL:   defaultSpentCodeTTL,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return &RedisProtocolStateStore{
		client:         dbclient.RedisCli,
		authRequestTTL: cfg.authRequestTTL,
		authCodeTTL:    cfg.authCodeTTL,
		spentCodeTTL:   cfg.spentCodeTTL,
	}
}

func (s *RedisProtocolStateStore) Health(ctx context.Context) error {
	if err := s.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("%w: %v", ErrStoreUnavailable, err)
	}
	return nil
}

func (s *RedisProtocolStateStore) CreateAuthRequest(ctx context.Context, authReq *oidc.AuthRequest, userID string, tenantID string) (op.AuthRequest, error) {
	reqID, err := randomTokenID("ar")
	if err != nil {
		return nil, fmt.Errorf("generate auth request id: %w", err)
	}
	req := &AuthRequest{
		ID:           reqID,
		ClientID:     authReq.ClientID,
		RedirectURI:  authReq.RedirectURI,
		State:        authReq.State,
		Scopes:       authReq.Scopes,
		ResponseType: authReq.ResponseType,
		ResponseMode: authReq.ResponseMode,
		Nonce:        authReq.Nonce,
		Subject:      userID,
		TenantID:     tenantID,
		AuthTime:     time.Now(),
		Audience:     []string{authReq.ClientID},
		ExpiresAt:    time.Now().Add(s.authRequestTTL),
	}
	if authReq.CodeChallenge != "" {
		// 仅接受 S256：discovery 的 code_challenge_methods_supported 只声明 S256，
		// plain 会被 RFC 7636 认为可被窃听的降级通道，一律拒绝。
		if authReq.CodeChallengeMethod != "" && authReq.CodeChallengeMethod != oidc.CodeChallengeMethodS256 {
			return nil, ErrCodeChallengeMethodUnsupported
		}
		req.CodeChallenge = &oidc.CodeChallenge{
			Challenge: authReq.CodeChallenge,
			Method:    oidc.CodeChallengeMethodS256,
		}
	}
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal auth request: %w", err)
	}
	if err := s.client.Set(ctx, authRequestKey(req.ID), data, s.authRequestTTL).Err(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrStoreUnavailable, err)
	}
	return req, nil
}

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

func (s *RedisProtocolStateStore) CompleteAuthRequest(ctx context.Context, id string, subject string, authTime time.Time, amr []string, acr string, tenantID string, done bool) error {
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
	req.TenantID = tenantID
	req.DoneFlag = done
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

// AssociateSession 将登录环节创建的 SSO 会话 ID 回写到授权票据，作为背信道登出的 sid 锚点。
func (s *RedisProtocolStateStore) AssociateSession(ctx context.Context, id string, sessionID string) error {
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
	req.SessionID = sessionID
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

func (s *RedisProtocolStateStore) SaveAuthCode(ctx context.Context, id, code string) error {
	if err := s.client.SetArgs(ctx, authCodeKey(code), id, redis.SetArgs{
		Mode: "NX",
		TTL:  s.authCodeTTL,
	}).Err(); err != nil {
		if errors.Is(err, redis.Nil) {
			return ErrCodeCollision
		}
		return fmt.Errorf("%w: %w", ErrStoreUnavailable, err)
	}
	return nil
}

func (s *RedisProtocolStateStore) AuthRequestByCode(ctx context.Context, code string) (op.AuthRequest, error) {
	return s.consumeAuthCode(ctx, code, false)
}

func (s *RedisProtocolStateStore) ConsumeAuthCode(ctx context.Context, code string) (op.AuthRequest, error) {
	return s.consumeAuthCode(ctx, code, true)
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

	if deleteCode {
		if err := s.client.Set(ctx, spentCodeKey(code), "1", s.spentCodeTTL).Err(); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrStoreUnavailable, err)
		}
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

func (s *RedisProtocolStateStore) DeleteAuthRequest(ctx context.Context, id string) error {
	if err := s.client.Del(ctx, authRequestKey(id)).Err(); err != nil {
		return fmt.Errorf("%w: %w", ErrStoreUnavailable, err)
	}
	return nil
}
