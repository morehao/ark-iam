package svcauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/morehao/ark-iam/pkg/dbclient"
)

const connectorStateRedisKeyPrefix = "iam:connector:state:"

var (
	ErrConnectorStateNotFound         = errors.New("connector state not found")
	ErrConnectorStateStoreUnavailable = errors.New("connector state store unavailable")
)

type ConnectorState struct {
	State string `json:"state"`
	Nonce string `json:"nonce"`
	// CodeVerifier 是 OAuth2/OIDC 授权码模式生成的 PKCE verifier（S256），
	// 回调换 code 时回填，防止授权码被第三方截获后兑换（H10）。
	CodeVerifier string    `json:"codeVerifier"`
	ConnectorID  string    `json:"connectorID"`
	TenantID     string    `json:"tenantID"`
	RedirectURI  string    `json:"redirectUri"`
	ExpiredAt    time.Time `json:"expiresAt"`
}

type ConnectorStateStore interface {
	Save(ctx context.Context, state *ConnectorState) error
	Load(ctx context.Context, state string) (*ConnectorState, error)
	Consume(ctx context.Context, state string) (*ConnectorState, error)
}

type redisStatusCommander interface {
	Err() error
}

type redisStringCommander interface {
	Bytes() ([]byte, error)
}

type redisStateClient interface {
	Set(ctx context.Context, key string, value any, expiration time.Duration) redisStatusCommander
	Get(ctx context.Context, key string) redisStringCommander
	GetDel(ctx context.Context, key string) redisStringCommander
}

type redisStateClientAdapter struct {
	client *redis.Client
}

type redisConnectorStateStore struct {
	client redisStateClient
}

type inMemoryConnectorStateStore struct {
	mu     sync.Mutex
	states map[string]ConnectorState
}

var _ ConnectorStateStore = (*redisConnectorStateStore)(nil)
var _ ConnectorStateStore = (*inMemoryConnectorStateStore)(nil)

func NewRedisConnectorStateStore() ConnectorStateStore {
	if dbclient.RedisCli == nil {
		return &redisConnectorStateStore{}
	}
	return &redisConnectorStateStore{client: &redisStateClientAdapter{client: dbclient.RedisCli}}
}

func NewInMemoryConnectorStateStore() ConnectorStateStore {
	return &inMemoryConnectorStateStore{states: make(map[string]ConnectorState)}
}

func (s *redisConnectorStateStore) Save(ctx context.Context, state *ConnectorState) error {
	if s.client == nil {
		return ErrConnectorStateStoreUnavailable
	}
	if err := validateConnectorState(state); err != nil {
		return err
	}
	payload, err := json.Marshal(state)
	if err != nil {
		return err
	}
	if err := s.client.Set(ctx, connectorStateRedisKey(state.State), payload, time.Until(state.ExpiredAt)).Err(); err != nil {
		return err
	}
	return nil
}

func (s *redisConnectorStateStore) Load(ctx context.Context, state string) (*ConnectorState, error) {
	if s.client == nil {
		return nil, ErrConnectorStateStoreUnavailable
	}
	if state == "" {
		return nil, ErrConnectorStateNotFound
	}
	stored, err := s.client.Get(ctx, connectorStateRedisKey(state)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, ErrConnectorStateNotFound
	}
	if err != nil {
		return nil, err
	}
	return decodeConnectorState(stored)
}

func (s *redisConnectorStateStore) Consume(ctx context.Context, state string) (*ConnectorState, error) {
	if s.client == nil {
		return nil, ErrConnectorStateStoreUnavailable
	}
	if state == "" {
		return nil, ErrConnectorStateNotFound
	}
	stored, err := s.client.GetDel(ctx, connectorStateRedisKey(state)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, ErrConnectorStateNotFound
	}
	if err != nil {
		return nil, err
	}
	return decodeConnectorState(stored)
}

func (s *inMemoryConnectorStateStore) Save(_ context.Context, state *ConnectorState) error {
	if err := validateConnectorState(state); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.states[state.State] = *state
	return nil
}

func (s *inMemoryConnectorStateStore) Load(_ context.Context, state string) (*ConnectorState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	stored, err := s.loadLocked(state)
	if err != nil {
		return nil, err
	}
	return stored, nil
}

func (s *inMemoryConnectorStateStore) Consume(_ context.Context, state string) (*ConnectorState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	stored, err := s.loadLocked(state)
	if err != nil {
		return nil, err
	}
	delete(s.states, state)
	return stored, nil
}

func (s *inMemoryConnectorStateStore) loadLocked(state string) (*ConnectorState, error) {
	stored, ok := s.states[state]
	if !ok {
		return nil, ErrConnectorStateNotFound
	}
	if time.Now().After(stored.ExpiredAt) {
		delete(s.states, state)
		return nil, ErrConnectorStateNotFound
	}
	copyState := stored
	return &copyState, nil
}

func validateConnectorState(state *ConnectorState) error {
	if state == nil {
		return fmt.Errorf("connector state is nil")
	}
	if state.State == "" {
		return fmt.Errorf("connector state is required")
	}
	if state.ExpiredAt.IsZero() {
		return fmt.Errorf("connector state expiresAt is required")
	}
	if time.Until(state.ExpiredAt) <= 0 {
		return fmt.Errorf("connector state is expired")
	}
	return nil
}

func decodeConnectorState(payload []byte) (*ConnectorState, error) {
	var state ConnectorState
	if err := json.Unmarshal(payload, &state); err != nil {
		return nil, err
	}
	if time.Now().After(state.ExpiredAt) {
		return nil, ErrConnectorStateNotFound
	}
	return &state, nil
}

func connectorStateRedisKey(state string) string {
	return connectorStateRedisKeyPrefix + state
}

func (a *redisStateClientAdapter) Set(ctx context.Context, key string, value any, expiration time.Duration) redisStatusCommander {
	return a.client.Set(ctx, key, value, expiration)
}

func (a *redisStateClientAdapter) Get(ctx context.Context, key string) redisStringCommander {
	return a.client.Get(ctx, key)
}

func (a *redisStateClientAdapter) GetDel(ctx context.Context, key string) redisStringCommander {
	return a.client.GetDel(ctx, key)
}
