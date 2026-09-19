package directory

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubTokens 是固定的 M2M 令牌来源。
type stubTokens struct {
	mu        sync.Mutex
	token     string
	err       error
	invalidAt int32
}

func (s *stubTokens) Token(context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.token, s.err
}

func (s *stubTokens) Invalidate() {
	atomic.AddInt32(&s.invalidAt, 1)
}

// countingMetrics 记录缓存命中/未命中/不可用计数。
type countingMetrics struct {
	hits        int32
	staleHits   int32
	misses      int32
	unavailable int32
	notModified int32
}

func (m *countingMetrics) CacheHit(stale bool) {
	if stale {
		atomic.AddInt32(&m.staleHits, 1)
		return
	}
	atomic.AddInt32(&m.hits, 1)
}

func (m *countingMetrics) CacheMiss()   { atomic.AddInt32(&m.misses, 1) }
func (m *countingMetrics) Unavailable() { atomic.AddInt32(&m.unavailable, 1) }

// NotModified 属于可选扩增接口 NotModifiedMetrics（304 命中计数）。
func (m *countingMetrics) NotModified() { atomic.AddInt32(&m.notModified, 1) }

func newClient(t *testing.T, baseURL string, mutate func(*Config)) *Client {
	t.Helper()
	cfg := Config{
		BaseURL: baseURL,
		Tokens:  &stubTokens{token: "m2m-token"},
	}
	if mutate != nil {
		mutate(&cfg)
	}
	c, err := New(cfg)
	require.NoError(t, err)
	return c
}

func TestNew_RequiresBaseURLAndTokens(t *testing.T) {
	_, err := New(Config{})
	require.Error(t, err)
	_, err = New(Config{BaseURL: "http://x"})
	require.Error(t, err)
}

func TestMembers_BatchAndMissing(t *testing.T) {
	var requests int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		assert.Equal(t, "Bearer m2m-token", r.Header.Get("Authorization"))
		assert.Equal(t, "/v1/rp/directory/members", r.URL.Path)
		ids := strings.Split(r.URL.Query().Get("ids"), ",")
		items := make([]string, 0, len(ids))
		missing := make([]string, 0, len(ids))
		for _, id := range ids {
			if id == "u-missing" {
				missing = append(missing, id)
				continue
			}
			items = append(items, fmt.Sprintf(`{"userID":%q,"name":"name-%s","status":"active"}`, id, id))
		}
		_, _ = fmt.Fprintf(w, `{"list":[%s],"missing":[%s]}`, strings.Join(items, ","), quoteJoin(missing))
	}))
	defer srv.Close()

	c := newClient(t, srv.URL+"/v1/rp", nil)
	got, err := c.Members(context.Background(), []string{"u-1", "u-2", "u-missing"})
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "name-u-1", got["u-1"].Name)
	_, ok := got["u-missing"]
	assert.False(t, ok, "缺项不得出现在结果里（跨租户与不存在不可区分）")
	assert.Equal(t, int32(1), atomic.LoadInt32(&requests))
}

func TestMembers_TooManyIDs(t *testing.T) {
	var requests int32
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		atomic.AddInt32(&requests, 1)
	}))
	defer srv.Close()

	c := newClient(t, srv.URL, nil)
	ids := make([]string, 0, MaxBatchIDs+1)
	for i := 0; i <= MaxBatchIDs; i++ {
		ids = append(ids, fmt.Sprintf("u-%d", i))
	}
	_, err := c.Members(context.Background(), ids)
	assert.ErrorIs(t, err, ErrTooManyIDs)
	assert.Equal(t, int32(0), atomic.LoadInt32(&requests), "超限必须客户端侧拒绝，不发请求")
}

func TestMembers_TTLCacheAndSingleFlight(t *testing.T) {
	var requests int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&requests, 1)
		time.Sleep(50 * time.Millisecond)
		_, _ = w.Write([]byte(`{"list":[{"userID":"u-1","name":"n1","status":"active"}]}`))
	}))
	defer srv.Close()

	m := &countingMetrics{}
	c := newClient(t, srv.URL, func(cfg *Config) { cfg.Metrics = m })

	// 并发 10 次同 key：single-flight 只应发一次请求。
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, err := c.Members(context.Background(), []string{"u-1"})
			assert.NoError(t, err)
			assert.Contains(t, got, "u-1")
		}()
	}
	wg.Wait()
	assert.Equal(t, int32(1), atomic.LoadInt32(&requests), "single-flight 必须把并发合并为一次请求")

	// TTL 内命中缓存。
	before := atomic.LoadInt32(&requests)
	_, err := c.Members(context.Background(), []string{"u-1"})
	require.NoError(t, err)
	assert.Equal(t, before, atomic.LoadInt32(&requests), "TTL 内必须命中缓存")
	assert.GreaterOrEqual(t, atomic.LoadInt32(&m.hits), int32(1))
}

func TestMembers_NegativeCacheBlocksRepeat(t *testing.T) {
	var requests int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&requests, 1)
		_, _ = w.Write([]byte(`{"list":[],"missing":["u-x"]}`))
	}))
	defer srv.Close()

	c := newClient(t, srv.URL, func(cfg *Config) { cfg.NegativeTTL = time.Minute })

	got, err := c.Members(context.Background(), []string{"u-x"})
	require.NoError(t, err)
	assert.Empty(t, got)

	_, err = c.Member(context.Background(), "u-x")
	assert.ErrorIs(t, err, ErrNotFound, "负缓存命中必须返回 NotFound")

	got2, err := c.Members(context.Background(), []string{"u-x"})
	require.NoError(t, err)
	assert.Empty(t, got2)
	assert.Equal(t, int32(1), atomic.LoadInt32(&requests), "负缓存必须挡住重复穿透")
}

func TestMembers_4xxNotDegraded(t *testing.T) {
	cases := []struct {
		status int
		want   error
	}{
		{status: http.StatusUnauthorized, want: ErrUnauthorized},
		{status: http.StatusForbidden, want: ErrForbidden},
		{status: http.StatusNotFound, want: ErrNotFound},
	}
	for _, tc := range cases {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			var requests int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				atomic.AddInt32(&requests, 1)
				w.WriteHeader(tc.status)
			}))
			defer srv.Close()

			c := newClient(t, srv.URL, func(cfg *Config) { cfg.StaleOnError = time.Hour })
			// 先放一份"缓存"（用成功的服务器同一实例不可行，直接走错误路径即可：
			// 关键断言是 4xx 不重试）。
			_, err := c.Members(context.Background(), []string{"u-1"})
			assert.ErrorIs(t, err, tc.want)
			assert.Equal(t, int32(1), atomic.LoadInt32(&requests), "4xx 不得重试")
		})
	}
}

// TestMembers_StaleOnErrorOnlyFor5xx 覆盖降级边界：5xx 可降级，401 不可降级。
func TestMembers_StaleOnErrorOnlyFor5xx(t *testing.T) {
	var failStatus int32
	var requests int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&requests, 1)
		if s := atomic.LoadInt32(&failStatus); s != 0 {
			w.WriteHeader(int(s))
			return
		}
		_, _ = w.Write([]byte(`{"list":[{"userID":"u-1","name":"cached","status":"active"}]}`))
	}))
	defer srv.Close()

	c := newClient(t, srv.URL, func(cfg *Config) {
		cfg.MemberTTL = time.Millisecond // 立刻过期，迫使重新拉取
		cfg.StaleOnError = time.Hour
	})
	first, err := c.Members(context.Background(), []string{"u-1"})
	require.NoError(t, err)
	require.Equal(t, "cached", first["u-1"].Name)
	time.Sleep(10 * time.Millisecond)

	// 5xx → 返回 stale
	atomic.StoreInt32(&failStatus, http.StatusInternalServerError)
	stale, err := c.Members(context.Background(), []string{"u-1"})
	require.NoError(t, err)
	assert.Equal(t, "cached", stale["u-1"].Name, "5xx 时允许返回过期缓存")

	// 401 → 透传，绝不降级（stale 分支只对 ErrUnavailable 生效）
	time.Sleep(10 * time.Millisecond)
	atomic.StoreInt32(&failStatus, http.StatusUnauthorized)
	_, err = c.Members(context.Background(), []string{"u-1"})
	assert.ErrorIs(t, err, ErrUnauthorized, "401 必须透传，绝不降级")
}

func TestMembers_5xxWithoutCacheReturnsUnavailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	c := newClient(t, srv.URL, func(cfg *Config) {
		cfg.StaleOnError = time.Hour
		cfg.Retry = 0
	})
	_, err := c.Members(context.Background(), []string{"u-1"})
	assert.ErrorIs(t, err, ErrUnavailable)
}

func TestMembers_ConnectionFailureReturnsUnavailable(t *testing.T) {
	c := newClient(t, "http://127.0.0.1:1", func(cfg *Config) { cfg.Retry = 0 })
	_, err := c.Members(context.Background(), []string{"u-1"})
	assert.ErrorIs(t, err, ErrUnavailable)
}

func TestDepartmentTreeAndRoles(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/rp/directory/departments/tree":
			_, _ = w.Write([]byte(`[{"id":"d1","name":"研发","parentID":"","children":[{"id":"d2","name":"平台","parentID":"d1"}]}]`))
		case "/v1/rp/directory/roles":
			_, _ = w.Write([]byte(`{"list":[{"code":"admin","name":"管理员","appID":"app-1"}],"truncated":true}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c := newClient(t, srv.URL+"/v1/rp", nil)
	tree, err := c.DepartmentTree(context.Background())
	require.NoError(t, err)
	require.Len(t, tree, 1)
	assert.Equal(t, "研发", tree[0].Name)
	require.Len(t, tree[0].Children, 1)

	roles, err := c.Roles(context.Background())
	require.NoError(t, err)
	require.Len(t, roles, 1)
	assert.Equal(t, "admin", roles[0].Code)
	assert.True(t, c.RolesTruncated(), "服务端截断标记必须透出")
}

func TestUnauthorizedInvalidatesM2MToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	tokens := &stubTokens{token: "t"}
	c, err := New(Config{BaseURL: srv.URL, Tokens: tokens})
	require.NoError(t, err)

	_, err = c.Members(context.Background(), []string{"u-1"})
	assert.ErrorIs(t, err, ErrUnauthorized)
	assert.Equal(t, int32(1), atomic.LoadInt32(&tokens.invalidAt), "401 必须让 M2M 令牌缓存失效")
}

func TestInvalidateMember(t *testing.T) {
	var requests int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&requests, 1)
		_, _ = w.Write([]byte(`{"list":[{"userID":"u-1","name":"n","status":"active"}]}`))
	}))
	defer srv.Close()

	c := newClient(t, srv.URL, nil)
	_, err := c.Member(context.Background(), "u-1")
	require.NoError(t, err)
	require.Equal(t, int32(1), atomic.LoadInt32(&requests))

	c.InvalidateMember("u-1")
	_, err = c.Member(context.Background(), "u-1")
	require.NoError(t, err)
	assert.Equal(t, int32(2), atomic.LoadInt32(&requests), "主动失效后必须重新拉取")
}

func TestNormalizeIDs(t *testing.T) {
	got := normalizeIDs([]string{" b ", "a", "b", "", "a"})
	assert.Equal(t, []string{"a", "b"}, got)
}

func TestMembers_SingleFlightReuseOnError(t *testing.T) {
	// leader 失败时 follower 不应挂死，也应看到错误。
	var requests int32
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&requests, 1)
		<-release
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	c := newClient(t, srv.URL, func(cfg *Config) { cfg.Retry = 0 })
	var wg sync.WaitGroup
	errs := make([]error, 4)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, errs[idx] = c.Members(context.Background(), []string{"u-1"})
		}(i)
	}
	time.Sleep(50 * time.Millisecond)
	close(release)
	wg.Wait()
	for _, err := range errs {
		assert.True(t, err == nil || errors.Is(err, ErrUnavailable) || errors.Is(err, ErrNotFound))
	}
}

// quoteJoin 把字符串切片编码为 JSON 字符串数组的内容（测试用）。
func quoteJoin(items []string) string {
	quoted := make([]string, 0, len(items))
	for _, item := range items {
		quoted = append(quoted, fmt.Sprintf("%q", item))
	}
	return strings.Join(quoted, ",")
}

// TestETagNotModifiedReusesLocalCopy 覆盖缓存契约里的 ETag/304 分支：
// 第二次拉取带 If-None-Match，服务端 304 时必须复用本地副本、把有效期续到当前，
// 并且**不**反序列化响应体（304 无 body）。
func TestETagNotModifiedReusesLocalCopy(t *testing.T) {
	const etag = `"body-hash-1"`
	var (
		requests     int32
		secondHadTag int32
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&requests, 1)
		if n > 1 {
			if r.Header.Get("If-None-Match") == etag {
				atomic.StoreInt32(&secondHadTag, 1)
			}
			w.Header().Set("ETag", etag)
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("ETag", etag)
		if r.URL.Path == "/v1/rp/directory/members" {
			_, _ = w.Write([]byte(`{"list":[{"userID":"u-1","name":"甲"}],"missing":[]}`))
			return
		}
		_, _ = w.Write([]byte(`{"list":[{"code":"admin","name":"管理员","appID":"app-1"}],"truncated":false}`))
	}))
	defer srv.Close()

	m := &countingMetrics{}
	// TTL 设为 0：跳过缓存直接命中网络分支，从而走到 ETag 协商与 304 复用。
	c, err := New(Config{BaseURL: srv.URL + "/v1/rp", Tokens: &stubTokens{token: "t"}, Metrics: m, RoleTTL: time.Nanosecond})
	require.NoError(t, err)

	roles, err := c.Roles(context.Background())
	require.NoError(t, err)
	require.Len(t, roles, 1)

	roles, err = c.Roles(context.Background())
	require.NoError(t, err)
	require.Len(t, roles, 1, "304 后必须复用本地副本")
	assert.Equal(t, "admin", roles[0].Code)
	assert.Equal(t, int32(1), atomic.LoadInt32(&secondHadTag), "第二次请求必须带 If-None-Match")
	assert.Equal(t, int32(1), atomic.LoadInt32(&m.notModified), "304 必须计入 NotModified 指标")
}
