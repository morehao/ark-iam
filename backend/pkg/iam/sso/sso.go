// Package sso 提供 SSO 中心会话（认证态）的存储与校验能力。
//
// 定位：跨应用共享的基础会话层，供 auth（OP）等持有会话的一侧使用。
// 业务应用（RP）作为无状态侧不直接依赖本包（见 docs/design/oidc-slo-unified-logout.md）。
package sso

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"

	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/glog"
)

const (
	sessionAuditStatusActive = "active"
)

const (
	ssoSessionKeyPrefix      = "iam:oidc:sso_session:"
	ssoUserSessionsKeyPrefix = "iam:oidc:sso_user_sessions:"
)

// 默认会话 TTL。调用方可经 WithSessionTTL 覆盖（推荐传入应用配置读取到的值）。
const defaultSessionTTLDuration = 24 * time.Hour

func ssoSessionKey(sessionID string) string {
	return ssoSessionKeyPrefix + sessionID
}

func ssoUserSessionsKey(personID uint) string {
	return fmt.Sprintf("%s%d", ssoUserSessionsKeyPrefix, personID)
}

type ssoSessionData struct {
	PersonID  uint      `json:"personID"`
	CreatedAt time.Time `json:"createdAt"`
}

// SessionTTLOption 允许调用方为会话 TTL 注入自定义值，取代对具体应用 config 的依赖。
type SessionTTLOption struct {
	sessionTTL time.Duration
}

// WithSessionTTL 设置会话有效期。
func WithSessionTTL(ttl time.Duration) SessionTTLOption {
	return SessionTTLOption{sessionTTL: ttl}
}

// SSOSessionStore 定义 SSO 中心会话的行为。
type SSOSessionStore interface {
	CreateSession(ctx context.Context, personID uint) (string, error)
	ValidateSession(ctx context.Context, sessionID string) (uint, error)
	RevokeSession(ctx context.Context, sessionID string) error
	RevokeSessionsByPersonID(ctx context.Context, personID uint) error
	HasActiveSession(ctx context.Context, personID uint) (bool, error)
}

type redisSSOSessionStore struct {
	client     *redis.Client
	sessionTTL time.Duration
}

var _ SSOSessionStore = (*redisSSOSessionStore)(nil)

// NewSSOSessionStore 构造 SSOSessionStore。默认读取全局 dbclient.RedisCli，
// 调用方可经 WithSessionTTL 注入会话有效期。
func NewSSOSessionStore(opts ...SessionTTLOption) SSOSessionStore {
	store := &redisSSOSessionStore{sessionTTL: defaultSessionTTLDuration}
	for _, opt := range opts {
		if opt.sessionTTL > 0 {
			store.sessionTTL = opt.sessionTTL
		}
	}
	if dbclient.RedisCli != nil {
		store.client = dbclient.RedisCli
	}
	return store
}

// sessionAuditWriter 落库 session 审计，返回 error 表示成功写入。默认写入 session 审计表，
// 测试可通过覆盖该变量注入桩实现，避免污染真实数据库。
var sessionAuditWriter = func(ctx context.Context, entity *model.SessionAuditEntity) error {
	return dao.NewSessionAuditDao().Insert(ctx, entity)
}

// recordSessionAuditBestEffort Session 创建后尽力落库一条 session 审计记录。
// 审计写入失败仅记录日志，绝不阻断 SSO 会话本身（Redis 会话必须照常可用）。
// CreateSession 无 gin 上下文，仅能记录 person_id/session_id/tenant_id/login_time/status，
// client_ip 与 user_agent 暂留空。
func recordSessionAuditBestEffort(ctx context.Context, sid string, personID uint) {
	tenantID := uint(0)
	if v := ctx.Value(gcontext.KeyTenantID); v != nil {
		if t, ok := v.(uint); ok {
			tenantID = t
		}
	}
	entity := &model.SessionAuditEntity{
		PersonID:  personID,
		SessionID: sid,
		TenantID:  tenantID,
		LoginTime: time.Now(),
		Status:    sessionAuditStatusActive,
		CreatedBy: personID,
	}
	if err := sessionAuditWriter(ctx, entity); err != nil {
		glog.Errorf(ctx, "[sso.recordSessionAudit] write session audit fail, err:%v, sessionId:%s, personId:%d", err, sid, personID)
	}
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
	ttl := s.sessionTTL
	if err := s.client.Set(ctx, ssoSessionKey(sessionID), encoded, ttl).Err(); err != nil {
		return "", fmt.Errorf("failed to store session: %w", err)
	}
	if err := s.client.SAdd(ctx, ssoUserSessionsKey(personID), sessionID).Err(); err != nil {
		// 索引写入失败时清理已写入的 session，避免产生无法随全局登出撤销的孤儿会话
		s.client.Del(ctx, ssoSessionKey(sessionID))
		return "", fmt.Errorf("failed to index session: %w", err)
	}
	s.client.Expire(ctx, ssoUserSessionsKey(personID), ttl)
	recordSessionAuditBestEffort(ctx, sessionID, personID)
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
	ttl := s.sessionTTL
	for _, sessionID := range sessionIDs {
		if n, err := s.client.Exists(ctx, ssoSessionKey(sessionID)).Result(); err == nil && n > 0 {
			// 滑动续期：保持活跃会话不被会话 TTL 淘汰
			s.client.Expire(ctx, ssoSessionKey(sessionID), ttl)
			return true, nil
		}
	}
	return false, nil
}
