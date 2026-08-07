package svcoidc

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	appconfig "github.com/morehao/ark-iam/iam/config"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/redis/go-redis/v9"
)

const (
	ssoSessionKeyPrefix      = "iam:oidc:sso_session:"
	ssoUserSessionsKeyPrefix = "iam:oidc:sso_user_sessions:"
)

func ssoSessionKey(sessionID string) string {
	return ssoSessionKeyPrefix + sessionID
}

func ssoUserSessionsKey(personID uint) string {
	return fmt.Sprintf("%s%d", ssoUserSessionsKeyPrefix, personID)
}

func defaultSessionTTL() time.Duration {
	if appconfig.Conf != nil && appconfig.Conf.OIDC.SessionTTL > 0 {
		return time.Duration(appconfig.Conf.OIDC.SessionTTL) * time.Second
	}
	return 24 * time.Hour
}

type ssoSessionData struct {
	PersonID  uint      `json:"personID"`
	CreatedAt time.Time `json:"createdAt"`
}

type SSOSessionStore interface {
	CreateSession(ctx context.Context, personID uint) (string, error)
	ValidateSession(ctx context.Context, sessionID string) (uint, error)
	RevokeSession(ctx context.Context, sessionID string) error
	RevokeSessionsByPersonID(ctx context.Context, personID uint) error
	HasActiveSession(ctx context.Context, personID uint) (bool, error)
}

type redisSSOSessionStore struct {
	client *redis.Client
}

var _ SSOSessionStore = (*redisSSOSessionStore)(nil)

func NewSSOSessionStore() SSOSessionStore {
	if dbclient.RedisCli == nil {
		return &redisSSOSessionStore{}
	}
	return &redisSSOSessionStore{client: dbclient.RedisCli}
}

// RevokeSSOSessionsByPersonID 撤销指定 person 的全部 SSO session。
// 供跨包（如 svcauth 登出）调用，实现全局登出语义。
func RevokeSSOSessionsByPersonID(ctx context.Context, personID uint) error {
	return NewSSOSessionStore().RevokeSessionsByPersonID(ctx, personID)
}

func generateSessionID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate session id: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func (s *redisSSOSessionStore) CreateSession(ctx context.Context, personID uint) (string, error) {
	if s.client == nil {
		return "", fmt.Errorf("redis client not available")
	}
	sessionID, err := generateSessionID()
	if err != nil {
		return "", err
	}
	data := ssoSessionData{
		PersonID:  personID,
		CreatedAt: time.Now(),
	}
	encoded, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal session data: %w", err)
	}
	ttl := defaultSessionTTL()
	if err := s.client.Set(ctx, ssoSessionKey(sessionID), encoded, ttl).Err(); err != nil {
		return "", fmt.Errorf("failed to store session: %w", err)
	}
	if err := s.client.SAdd(ctx, ssoUserSessionsKey(personID), sessionID).Err(); err != nil {
		// 索引写入失败时清理已写入的 session，避免产生无法随全局登出撤销的孤儿会话
		s.client.Del(ctx, ssoSessionKey(sessionID))
		return "", fmt.Errorf("failed to index session: %w", err)
	}
	s.client.Expire(ctx, ssoUserSessionsKey(personID), ttl)
	return sessionID, nil
}

func (s *redisSSOSessionStore) ValidateSession(ctx context.Context, sessionID string) (uint, error) {
	if s.client == nil {
		return 0, fmt.Errorf("redis client not available")
	}
	encoded, err := s.client.Get(ctx, ssoSessionKey(sessionID)).Bytes()
	if err == redis.Nil {
		return 0, fmt.Errorf("session not found or expired")
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get session: %w", err)
	}
	var data ssoSessionData
	if err := json.Unmarshal(encoded, &data); err != nil {
		return 0, fmt.Errorf("failed to unmarshal session data: %w", err)
	}
	return data.PersonID, nil
}

func (s *redisSSOSessionStore) RevokeSession(ctx context.Context, sessionID string) error {
	if s.client == nil {
		return fmt.Errorf("redis client not available")
	}
	encoded, err := s.client.Get(ctx, ssoSessionKey(sessionID)).Bytes()
	if err != nil {
		return s.client.Del(ctx, ssoSessionKey(sessionID)).Err()
	}
	var data ssoSessionData
	if err := json.Unmarshal(encoded, &data); err != nil {
		return s.client.Del(ctx, ssoSessionKey(sessionID)).Err()
	}
	s.client.SRem(ctx, ssoUserSessionsKey(data.PersonID), sessionID)
	return s.client.Del(ctx, ssoSessionKey(sessionID)).Err()
}

func (s *redisSSOSessionStore) RevokeSessionsByPersonID(ctx context.Context, personID uint) error {
	if s.client == nil {
		return fmt.Errorf("redis client not available")
	}
	userSessionsKey := ssoUserSessionsKey(personID)
	sessionIDs, err := s.client.SMembers(ctx, userSessionsKey).Result()
	if err != nil {
		return fmt.Errorf("failed to get user sessions: %w", err)
	}
	for _, sessionID := range sessionIDs {
		s.client.Del(ctx, ssoSessionKey(sessionID))
	}
	s.client.Del(ctx, userSessionsKey)
	return nil
}

// HasActiveSession 返回该自然人是否存在至少一个仍然有效的 SSO 会话。
// 全局登出（RevokeSessionsByPersonID）会清空对应 sso_user_sessions 索引，
// 之后此处将返回 false，从而让该自然人的既有 OIDC 访问令牌失效（必须重新认证）。
// 找到任一有效会话时顺带刷新其 TTL，实现滑动续期，避免活跃用户因会话过期被误登出。
func (s *redisSSOSessionStore) HasActiveSession(ctx context.Context, personID uint) (bool, error) {
	if s.client == nil {
		return false, fmt.Errorf("redis client not available")
	}
	userSessionsKey := ssoUserSessionsKey(personID)
	sessionIDs, err := s.client.SMembers(ctx, userSessionsKey).Result()
	if err != nil {
		return false, fmt.Errorf("failed to get user sessions: %w", err)
	}
	if len(sessionIDs) == 0 {
		return false, nil
	}
	ttl := defaultSessionTTL()
	for _, sessionID := range sessionIDs {
		if n, err := s.client.Exists(ctx, ssoSessionKey(sessionID)).Result(); err == nil && n > 0 {
			// 滑动续期：保持活跃会话不被会话 TTL 淘汰
			s.client.Expire(ctx, ssoSessionKey(sessionID), ttl)
			return true, nil
		}
	}
	return false, nil
}
