// Package sso 提供 SSO 中心会话（认证态）的存储与校验能力，
// 以及基于会话粒度的 OIDC back-channel logout 登出登记。
//
// 定位：跨应用共享的基础会话层，供 auth（OP）等持有会话的一侧使用。
package sso

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/morehao/ark-iam/pkg/dbclient"
)

const (
	ssoLogoutRegKeyPrefix = "iam:oidc:slo_reg:"
	ssoLogoutRegTTL       = 24 * time.Hour
	ssoLogoutQueueKey     = "iam:oidc:slo_queue"
)

func ssoLogoutRegKey(sessionID string) string {
	return ssoLogoutRegKeyPrefix + sessionID
}

// LogoutJob 一次待发送的 back-channel logout 通知任务。
type LogoutJob struct {
	SessionID            string `json:"sessionID"`
	PersonID             uint   `json:"personID"`
	OIDCSessionID        string `json:"oidcSessionID"`
	ClientID             string `json:"clientID"`
	UserID               string `json:"userID"`
	BackChannelLogoutURI string `json:"backChannelLogoutURI"`
}

func (j LogoutJob) encode() string {
	b, _ := json.Marshal(j)
	return string(b)
}

// EnqueueLogout 将一条登出通知任务推入 Redis FIFO 队列，供背信道 worker 异步消费。
func EnqueueLogout(ctx context.Context, job LogoutJob) error {
	if dbclient.RedisCli == nil {
		return fmt.Errorf("redis client not available")
	}
	return dbclient.RedisCli.LPush(ctx, ssoLogoutQueueKey, job.encode()).Err()
}

// DequeueLogout 阻塞地从队列取出（BRPOP）一条登出任务；无任务时返回 ok=false。
func DequeueLogout(ctx context.Context, timeout time.Duration) (LogoutJob, bool, error) {
	if dbclient.RedisCli == nil {
		return LogoutJob{}, false, fmt.Errorf("redis client not available")
	}
	vals, err := dbclient.RedisCli.BRPop(ctx, timeout, ssoLogoutQueueKey).Result()
	if err == redis.Nil || len(vals) < 2 {
		return LogoutJob{}, false, nil
	}
	if err != nil {
		return LogoutJob{}, false, err
	}
	var job LogoutJob
	if err := json.Unmarshal([]byte(vals[1]), &job); err != nil {
		return LogoutJob{}, false, fmt.Errorf("unmarshal logout job: %w", err)
	}
	return job, true, nil
}

// LogoutRegistration 记录一次会话在其某个 client 上签发过令牌，登出时应向该 client 发 back-channel 通知。
type LogoutRegistration struct {
	OIDCSessionID        string `json:"oidcSessionID"`
	ClientID             string `json:"clientID"`
	UserID               string `json:"userID"`
	BackChannelLogoutURI string `json:"backChannelLogoutURI"`
}

func (r LogoutRegistration) memberValue() string {
	b, _ := json.Marshal(r)
	return string(b)
}

// SLOStore 定义登出登记的读写行为，供第三方 RP 的 back-channel logout 通知使用。
type SLOStore interface {
	Register(ctx context.Context, sessionID string, reg LogoutRegistration) error
	ListBySessionID(ctx context.Context, sessionID string) ([]LogoutRegistration, error)
	ListByPersonID(ctx context.Context, personID uint) ([]LogoutRegistration, error)
	Delete(ctx context.Context, sessionID string, oidcSessionID string) error
}

type redisSLOStore struct {
	client *redis.Client
}

var _ SLOStore = (*redisSLOStore)(nil)

// NewSLOStore 构造登出登记存储，默认读取全局 dbclient.RedisCli。
func NewSLOStore() SLOStore {
	store := &redisSLOStore{}
	if dbclient.RedisCli != nil {
		store.client = dbclient.RedisCli
	}
	return store
}

// Register 为指定会话登记一条待通知的 client 信息。
func (s *redisSLOStore) Register(ctx context.Context, sessionID string, reg LogoutRegistration) error {
	if s.client == nil {
		return fmt.Errorf("redis client not available")
	}
	return s.client.SAdd(ctx, ssoLogoutRegKey(sessionID), reg.memberValue()).Err()
}

// ListBySessionID 返回指定会话下登记的全部 client。
func (s *redisSLOStore) ListBySessionID(ctx context.Context, sessionID string) ([]LogoutRegistration, error) {
	if s.client == nil {
		return nil, fmt.Errorf("redis client not available")
	}
	members, err := s.client.SMembers(ctx, ssoLogoutRegKey(sessionID)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get logout registrations: %w", err)
	}
	return decodeRegistrations(members), nil
}

// ListByPersonID 返回指定自然人（跨其全部 SSO 会话）登记的全部 client，
// 供无 id_token_hint 的 person 级全局登出使用。
func (s *redisSLOStore) ListByPersonID(ctx context.Context, personID uint) ([]LogoutRegistration, error) {
	if s.client == nil {
		return nil, fmt.Errorf("redis client not available")
	}
	userSessionsKey := ssoUserSessionsKey(personID)
	sessionIDs, err := s.client.SMembers(ctx, userSessionsKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get user sessions: %w", err)
	}
	result := make([]LogoutRegistration, 0, 8)
	for _, sessionID := range sessionIDs {
		members, err := s.client.SMembers(ctx, ssoLogoutRegKey(sessionID)).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to get logout registrations: %w", err)
		}
		result = append(result, decodeRegistrations(members)...)
	}
	return result, nil
}

// Delete 在背信道通知成功后删除某条登记，保证通知幂等。
func (s *redisSLOStore) Delete(ctx context.Context, sessionID string, oidcSessionID string) error {
	if s.client == nil {
		return fmt.Errorf("redis client not available")
	}
	regs, err := s.ListBySessionID(ctx, sessionID)
	if err != nil {
		return err
	}
	for _, reg := range regs {
		if reg.OIDCSessionID == oidcSessionID {
			s.client.SRem(ctx, ssoLogoutRegKey(sessionID), reg.memberValue())
		}
	}
	return nil
}

func decodeRegistrations(members []string) []LogoutRegistration {
	regs := make([]LogoutRegistration, 0, len(members))
	for _, m := range members {
		var reg LogoutRegistration
		if err := json.Unmarshal([]byte(m), &reg); err != nil {
			continue
		}
		if reg.ClientID == "" {
			continue
		}
		regs = append(regs, reg)
	}
	return regs
}
