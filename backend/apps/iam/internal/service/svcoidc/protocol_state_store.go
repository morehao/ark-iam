package svcoidc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	appconfig "github.com/morehao/ark-iam/iam/config"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/redis/go-redis/v9"
	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/oidc/v3/pkg/op"
)

var (
	ErrStoreUnavailable    = errors.New("oidc protocol state store unavailable")
	ErrSessionNotFound     = errors.New("auth request not found")
	ErrCodeInvalid         = errors.New("authorization code invalid")
	ErrCodeAlreadyUsed     = errors.New("authorization code already used")
	ErrCodeCollision       = errors.New("authorization code collision")
	ErrSessionNotCompleted = errors.New("auth request not completed")
)

type ProtocolStateStore interface {
	CreateAuthRequest(ctx context.Context, authReq *oidc.AuthRequest, userID string) (op.AuthRequest, error)
	AuthRequestByID(ctx context.Context, id string) (op.AuthRequest, error)
	AuthRequestByCode(ctx context.Context, code string) (op.AuthRequest, error)
	SaveAuthCode(ctx context.Context, id, code string) error
	CompleteAuthRequest(id string, subject string, authTime time.Time, amr []string, acr string) error
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

type RedisProtocolStateStore struct {
	client *redis.Client
}

var _ ProtocolStateStore = (*RedisProtocolStateStore)(nil)

func NewRedisProtocolStateStore() *RedisProtocolStateStore {
	if dbclient.RedisCli == nil {
		panic("redis client not initialized for OIDC protocol state store")
	}
	return &RedisProtocolStateStore{client: dbclient.RedisCli}
}

func (s *RedisProtocolStateStore) Health(ctx context.Context) error {
	if err := s.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("%w: %v", ErrStoreUnavailable, err)
	}
	return nil
}

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

func (s *RedisProtocolStateStore) SaveAuthCode(ctx context.Context, id, code string) error {
	if err := s.client.SetArgs(ctx, authCodeKey(code), id, redis.SetArgs{
		Mode: "NX",
		TTL:  defaultAuthCodeTTL(),
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
		if err := s.client.Set(ctx, spentCodeKey(code), "1", defaultSpentCodeTTL()).Err(); err != nil {
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
