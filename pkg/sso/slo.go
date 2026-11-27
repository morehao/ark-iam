// Package sso 提供 SSO 中心会话（认证态）的存储与校验能力，
// 以及基于会话粒度的 OIDC back-channel logout 登出登记。
//
// 定位：跨应用共享的基础会话层，供 auth（OP）等持有会话的一侧使用。
package sso

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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
	PersonID             string `json:"personID"`
	OIDCSessionID        string `json:"oidcSessionID"`
	ClientID             string `json:"clientID"`
	UserID               string `json:"userID"`
	BackChannelLogoutURI string `json:"backChannelLogoutURI"`
	// Issuer 该登出所属 issuer（worker 可按 issuer 分组/审计）；空=未记录。
	Issuer    string    `json:"issuer,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

func (j LogoutJob) encode() string {
	b, _ := json.Marshal(j)
	return string(b)
}

// EnqueueLogout 将一条登出通知任务推入 Redis FIFO 队列，供背信道 worker 异步消费。
func EnqueueLogout(ctx context.Context, job LogoutJob) error {
	return enqueueLogout(ctx, ssoLogoutQueueKey, job)
}

// EnqueueLogoutsByPersonID 将该自然人名下已登记的全部 client 的 back-channel logout 任务入队，
// 返回成功入队的任务数。
//
// 必须在撤销 SSO 会话之前调用：登记查询 ListByPersonID 依赖 sso_user_sessions 索引，
// 会话一旦撤销索引即被清除，通知就再也发不出去（这是「先入队、再撤销」顺序的根因）。
//
// 查询和批量入队的错误均返回调用方。
func EnqueueLogoutsByPersonID(ctx context.Context, personID string) (int, error) {
	return enqueueLogoutsByPersonID(ctx, ssoLogoutQueueKey, personID)
}

// enqueueLogoutsByPersonID 与 EnqueueLogoutsByPersonID 逻辑相同，但允许指定队列键，
// 供测试使用独立队列避免与运行中的 worker 共享消费（与 enqueueLogout 的拆法一致）。
func enqueueLogoutsByPersonID(ctx context.Context, queueKey string, personID string) (int, error) {
	if personID == "" {
		return 0, nil
	}
	regs, err := NewSLOStore().ListByPersonID(ctx, personID)
	if err != nil {
		return 0, fmt.Errorf("list logout registrations: %w", err)
	}
	if len(regs) == 0 {
		return 0, nil
	}
	payloads := make([]any, 0, len(regs))
	now := time.Now()
	for _, reg := range regs {
		payloads = append(payloads, (LogoutJob{
			SessionID:            reg.SessionID,
			PersonID:             personID,
			OIDCSessionID:        reg.OIDCSessionID,
			ClientID:             reg.ClientID,
			UserID:               reg.UserID,
			BackChannelLogoutURI: reg.BackChannelLogoutURI,
			Issuer:               reg.Issuer,
			CreatedAt:            now,
		}).encode())
	}
	if err := dbclient.RedisCli.LPush(ctx, queueKey, payloads...).Err(); err != nil {
		return 0, fmt.Errorf("enqueue logout registrations: %w", err)
	}
	return len(regs), nil
}

// enqueueLogout 允许指定队列键，
// 供测试使用独立队列避免与运行中的 worker 共享消费。
func enqueueLogout(ctx context.Context, queueKey string, job LogoutJob) error {
	if dbclient.RedisCli == nil {
		return fmt.Errorf("redis client not available")
	}
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now()
	}
	return dbclient.RedisCli.LPush(ctx, queueKey, job.encode()).Err()
}

// Atomically recover one timed-out delivery or claim the oldest queued job.
var claimLogoutScript = redis.NewScript(`
local now = redis.call('TIME')
local ms = tonumber(now[1]) * 1000 + math.floor(tonumber(now[2]) / 1000)
local expired = redis.call('ZRANGEBYSCORE', KEYS[2], '-inf', ms, 'LIMIT', 0, 1)
local payload
if expired[1] then
    redis.call('ZREM', KEYS[2], expired[1])
    payload = string.sub(expired[1], string.find(expired[1], ':', 1, true) + 1)
else
    payload = redis.call('RPOP', KEYS[1])
end
if not payload then return false end
local delivery = ARGV[1] .. ':' .. payload
redis.call('ZADD', KEYS[2], ms + tonumber(ARGV[2]), delivery)
return delivery
`)

const logoutDeliveryLease = time.Minute

// LogoutDelivery identifies a single claim. An expired claimant cannot acknowledge a newer claim.
type LogoutDelivery struct {
	LogoutJob
	queueKey string
	receipt  string
}

func ClaimLogout(ctx context.Context) (*LogoutDelivery, error) {
	return claimLogout(ctx, ssoLogoutQueueKey, logoutDeliveryLease)
}

func claimLogout(ctx context.Context, queueKey string, lease time.Duration) (*LogoutDelivery, error) {
	if dbclient.RedisCli == nil {
		return nil, fmt.Errorf("redis client not available")
	}
	id, err := generateSessionID()
	if err != nil {
		return nil, err
	}
	receipt, err := claimLogoutScript.Run(ctx, dbclient.RedisCli, []string{queueKey, queueKey + ":pending"}, id, lease.Milliseconds()).Text()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	delivery := &LogoutDelivery{queueKey: queueKey, receipt: receipt}
	if err := json.Unmarshal([]byte(strings.TrimPrefix(receipt, id+":")), &delivery.LogoutJob); err != nil {
		return nil, errors.Join(fmt.Errorf("unmarshal logout job: %w", err), delivery.Ack(ctx))
	}
	return delivery, nil
}

func (d *LogoutDelivery) Ack(ctx context.Context) error {
	return dbclient.RedisCli.ZRem(ctx, d.queueKey+":pending", d.receipt).Err()
}

// LogoutRegistration 记录一次会话在其某个 client 上签发过令牌，登出时应向该 client 发 back-channel 通知。
type LogoutRegistration struct {
	// SessionID 是中心 SSO 会话 ID（sid）。登记时回填，使 person 级登出路径
	// 也能构造携带 sid 的 logout_token，并让通知成功后 Delete 幂等可执行。
	SessionID            string `json:"sessionID"`
	OIDCSessionID        string `json:"oidcSessionID"`
	ClientID             string `json:"clientID"`
	UserID               string `json:"userID"`
	BackChannelLogoutURI string `json:"backChannelLogoutURI"`
	// Issuer 该登记所属 issuer；空=未记录（旧登记）。
	Issuer string `json:"issuer,omitempty"`
}

func (r LogoutRegistration) memberValue() string {
	b, _ := json.Marshal(r)
	return string(b)
}

// SLOStore 定义登出登记的读写行为，供第三方 RP 的 back-channel logout 通知使用。
type SLOStore interface {
	Register(ctx context.Context, sessionID string, reg LogoutRegistration) error
	ListBySessionID(ctx context.Context, sessionID string) ([]LogoutRegistration, error)
	ListByPersonID(ctx context.Context, personID string) ([]LogoutRegistration, error)
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
// 登记集合键带 TTL（ssoLogoutRegTTL），避免登出后无人清理导致 Redis 键无限堆积。
func (s *redisSLOStore) Register(ctx context.Context, sessionID string, reg LogoutRegistration) error {
	if s.client == nil {
		return fmt.Errorf("redis client not available")
	}
	if reg.SessionID == "" {
		reg.SessionID = sessionID
	}
	if err := s.client.SAdd(ctx, ssoLogoutRegKey(sessionID), reg.memberValue()).Err(); err != nil {
		return err
	}
	return s.client.Expire(ctx, ssoLogoutRegKey(sessionID), ssoLogoutRegTTL).Err()
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
func (s *redisSLOStore) ListByPersonID(ctx context.Context, personID string) ([]LogoutRegistration, error) {
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
