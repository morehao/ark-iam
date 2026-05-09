package svcauth

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

type fakeRedisStatusCmd struct {
	err error
}

func (c fakeRedisStatusCmd) Err() error {
	return c.err
}

type fakeRedisStringCmd struct {
	value []byte
	err   error
}

func (c fakeRedisStringCmd) Bytes() ([]byte, error) {
	return c.value, c.err
}

type fakeRedisClient struct {
	lastSetKey    string
	lastSetValue  any
	lastSetTTL    time.Duration
	setErr        error
	getValue      []byte
	getErr        error
	getDelValue   []byte
	getDelErr     error
	lastGetKey    string
	lastGetDelKey string
}

func (c *fakeRedisClient) Set(_ context.Context, key string, value any, expiration time.Duration) redisStatusCommander {
	c.lastSetKey = key
	c.lastSetValue = value
	c.lastSetTTL = expiration
	return fakeRedisStatusCmd{err: c.setErr}
}

func (c *fakeRedisClient) Get(_ context.Context, key string) redisStringCommander {
	c.lastGetKey = key
	return fakeRedisStringCmd{value: c.getValue, err: c.getErr}
}

func (c *fakeRedisClient) GetDel(_ context.Context, key string) redisStringCommander {
	c.lastGetDelKey = key
	return fakeRedisStringCmd{value: c.getDelValue, err: c.getDelErr}
}

func TestStateStoreSaveLoadConsume(t *testing.T) {
	store := NewInMemoryConnectorStateStore()
	ctx := context.Background()
	state := &ConnectorState{
		State:       "state-1",
		Nonce:       "nonce-1",
		ConnectorID: 12,
		TenantID:    34,
		RedirectURI: "https://console.example.com/callback",
		ExpiresAt:   time.Now().Add(time.Minute),
	}

	if err := store.Save(ctx, state); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	loaded, err := store.Load(ctx, state.State)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if loaded.State != state.State || loaded.Nonce != state.Nonce {
		t.Fatalf("Load should return persisted state, got %#v", loaded)
	}
	if loaded.ConnectorID != state.ConnectorID || loaded.TenantID != state.TenantID {
		t.Fatalf("Load should preserve connector and tenant ids, got %#v", loaded)
	}
	if loaded.RedirectURI != state.RedirectURI {
		t.Fatalf("Load should preserve redirect uri, got %#v", loaded)
	}

	consumed, err := store.Consume(ctx, state.State)
	if err != nil {
		t.Fatalf("Consume returned error: %v", err)
	}
	if consumed.State != state.State || consumed.Nonce != state.Nonce {
		t.Fatalf("Consume should return persisted state, got %#v", consumed)
	}

	_, err = store.Load(ctx, state.State)
	if !errors.Is(err, ErrConnectorStateNotFound) {
		t.Fatalf("Load after consume should return not found, got %v", err)
	}

	_, err = store.Consume(ctx, state.State)
	if !errors.Is(err, ErrConnectorStateNotFound) {
		t.Fatalf("Consume after consume should return not found, got %v", err)
	}
}

func TestStateStoreLoadMissingState(t *testing.T) {
	store := NewInMemoryConnectorStateStore()

	_, err := store.Load(context.Background(), "missing-state")
	if !errors.Is(err, ErrConnectorStateNotFound) {
		t.Fatalf("Load missing state should return not found, got %v", err)
	}
}

func TestStateStoreLoadExpiredState(t *testing.T) {
	store := NewInMemoryConnectorStateStore()
	ctx := context.Background()
	state := &ConnectorState{
		State:       "expired-state",
		Nonce:       "nonce-1",
		ConnectorID: 12,
		TenantID:    34,
		RedirectURI: "https://console.example.com/callback",
		ExpiresAt:   time.Now().Add(time.Second),
	}

	if err := store.Save(ctx, state); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	memoryStore := store.(*inMemoryConnectorStateStore)
	memoryStore.mu.Lock()
	stored := memoryStore.states[state.State]
	stored.ExpiresAt = time.Now().Add(-time.Second)
	memoryStore.states[state.State] = stored
	memoryStore.mu.Unlock()

	_, err := store.Load(ctx, state.State)
	if !errors.Is(err, ErrConnectorStateNotFound) {
		t.Fatalf("Load expired state should return not found, got %v", err)
	}

	_, err = store.Consume(ctx, state.State)
	if !errors.Is(err, ErrConnectorStateNotFound) {
		t.Fatalf("Consume expired state should return not found, got %v", err)
	}

	_, err = store.Load(ctx, state.State)
	if !errors.Is(err, ErrConnectorStateNotFound) {
		t.Fatalf("Load after expired state cleanup should return not found, got %v", err)
	}

	_, err = store.Consume(ctx, state.State)
	if !errors.Is(err, ErrConnectorStateNotFound) {
		t.Fatalf("Consume after expired state cleanup should return not found, got %v", err)
	}
}

func TestRedisStateStoreUnavailable(t *testing.T) {
	store := &redisConnectorStateStore{}
	ctx := context.Background()
	state := &ConnectorState{
		State:       "state-1",
		Nonce:       "nonce-1",
		ConnectorID: 12,
		TenantID:    34,
		RedirectURI: "https://console.example.com/callback",
		ExpiresAt:   time.Now().Add(time.Minute),
	}

	if err := store.Save(ctx, state); !errors.Is(err, ErrConnectorStateStoreUnavailable) {
		t.Fatalf("Save with nil client should return unavailable, got %v", err)
	}

	_, err := store.Load(ctx, "state-1")
	if !errors.Is(err, ErrConnectorStateStoreUnavailable) {
		t.Fatalf("Load with nil client should return unavailable, got %v", err)
	}

	_, err = store.Consume(ctx, "state-1")
	if !errors.Is(err, ErrConnectorStateStoreUnavailable) {
		t.Fatalf("Consume with nil client should return unavailable, got %v", err)
	}
}

func TestRedisStateStoreUsesTTL(t *testing.T) {
	fakeClient := &fakeRedisClient{}
	store := &redisConnectorStateStore{client: fakeClient}
	ctx := context.Background()
	state := &ConnectorState{
		State:       "state-1",
		Nonce:       "nonce-1",
		ConnectorID: 12,
		TenantID:    34,
		RedirectURI: "https://console.example.com/callback",
		ExpiresAt:   time.Now().Add(time.Minute),
	}

	if err := store.Save(ctx, state); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	if fakeClient.lastSetKey != connectorStateRedisKey(state.State) {
		t.Fatalf("Save should use state redis key, got %q", fakeClient.lastSetKey)
	}
	if fakeClient.lastSetTTL <= 0 {
		t.Fatalf("Save should pass positive ttl, got %v", fakeClient.lastSetTTL)
	}
	if _, ok := fakeClient.lastSetValue.([]byte); !ok {
		t.Fatalf("Save should marshal state into bytes, got %T", fakeClient.lastSetValue)
	}

	fakeClient.getErr = redis.Nil
	_, err := store.Load(ctx, "missing-state")
	if !errors.Is(err, ErrConnectorStateNotFound) {
		t.Fatalf("Load missing redis state should return not found, got %v", err)
	}

	fakeClient.getDelErr = redis.Nil
	_, err = store.Consume(ctx, "missing-state")
	if !errors.Is(err, ErrConnectorStateNotFound) {
		t.Fatalf("Consume missing redis state should return not found, got %v", err)
	}
}

func TestRedisStateStoreRejectsEmptyState(t *testing.T) {
	payload, err := json.Marshal(&ConnectorState{
		State:       "state-1",
		Nonce:       "nonce-1",
		ConnectorID: 12,
		TenantID:    34,
		RedirectURI: "https://console.example.com/callback",
		ExpiresAt:   time.Now().Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}

	fakeClient := &fakeRedisClient{getValue: payload, getDelValue: payload}
	store := &redisConnectorStateStore{client: fakeClient}

	_, err = store.Load(context.Background(), "")
	if !errors.Is(err, ErrConnectorStateNotFound) {
		t.Fatalf("Load empty state should return stable error, got %v", err)
	}
	if fakeClient.lastGetKey != "" {
		t.Fatalf("Load empty state should not access redis, got key %q", fakeClient.lastGetKey)
	}

	_, err = store.Consume(context.Background(), "")
	if !errors.Is(err, ErrConnectorStateNotFound) {
		t.Fatalf("Consume empty state should return stable error, got %v", err)
	}
	if fakeClient.lastGetDelKey != "" {
		t.Fatalf("Consume empty state should not access redis, got key %q", fakeClient.lastGetDelKey)
	}
}
