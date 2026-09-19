// Package directory 是 IAM 只读目录 API（/v1/rp/directory/*）的客户端。
//
// 纪律（设计文档关键决策六）：
//   - **绝不把目录客户端放进鉴权中间件或审计写入的关键路径**（可用性硬依赖）；
//   - TTL 缓存 + single-flight + 负缓存 + 批量合并，把 QPS 压到可忽略；
//   - 4xx（401/403/404）一律透传、**绝不降级**；仅连接失败/超时/5xx 时可返回 stale；
//   - 缓存里的 Member.Status **不得**用于放行/拒绝判定（只服务展示）。
package directory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

// 目录 API 的状态码语义（固定四类，见设计文档「接口契约与错误语义」）。
var (
	// ErrUnauthorized 表示 401：无令牌或令牌无效。
	ErrUnauthorized = errors.New("directory: unauthorized")
	// ErrForbidden 表示 403：缺 directory.read scope，或令牌无 tenant_id。
	ErrForbidden = errors.New("directory: forbidden")
	// ErrNotFound 表示 404：跨租户或不存在（二者不可区分）。
	ErrNotFound = errors.New("directory: not found")
	// ErrUnavailable 表示目录 API 不可用（连接失败/超时/5xx 且无可用缓存）。
	ErrUnavailable = errors.New("directory: unavailable")
	// ErrTooManyIDs 表示批量查询超过上限（客户端侧提前拒绝，不静默截断）。
	ErrTooManyIDs = errors.New("directory: too many ids")
)

// MaxBatchIDs 是批量查询的 id 上限（与服务端一致，超过即 400）。
const MaxBatchIDs = 100

// ScopeRead 是目录 API 要求的标准 scope。
const ScopeRead = "directory.read"

// Member 是成员摘要。
type Member struct {
	UserID          string   `json:"userID"`
	Name            string   `json:"name"`
	Avatar          string   `json:"avatar"`
	Status          string   `json:"status"`
	UserType        string   `json:"userType"`
	DepartmentNames []string `json:"departmentNames"`
}

// Department 是部门节点。
type Department struct {
	ID       string       `json:"id"`
	Name     string       `json:"name"`
	ParentID string       `json:"parentID"`
	Children []Department `json:"children,omitempty"`
}

// Role 是角色清单项。
type Role struct {
	Code  string `json:"code"`
	Name  string `json:"name"`
	AppID string `json:"appID"`
}

// TokenProvider 提供 M2M access token（实现见 rp.TokenClient）。
type TokenProvider interface {
	Token(ctx context.Context) (string, error)
	Invalidate()
}

// Metrics 是目录客户端的最小计数接口（配合设计文档要求的监控项：
// stale 命中率、ErrDirectoryUnavailable 计数）。
type Metrics interface {
	// CacheHit 记录一次缓存命中（stale=true 表示降级返回的过期副本）。
	CacheHit(stale bool)
	// CacheMiss 记录一次未命中。
	CacheMiss()
	// Unavailable 记录一次目录不可用。
	Unavailable()
}

// NotModifiedMetrics 是 Metrics 的**可选扩增**：实现它的 Metrics 会额外收到
// 304（ETag 命中）计数。刻意不做成 Metrics 的必选方法——Metrics 是外部实现
// 的公开接口，加方法会直接破坏既有实现的编译。
type NotModifiedMetrics interface {
	// NotModified 记录一次 ETag 命中（304，省下响应体传输与反序列化）。
	NotModified()
}

// Config 是目录客户端配置。
type Config struct {
	// BaseURL 是目录 API 根地址（如 http://iam:8084/v1/rp），不带尾斜杠。
	BaseURL string
	// Tokens 是 M2M 令牌来源。
	Tokens TokenProvider
	// HTTPClient 可注入自定义客户端；默认 connect 1s / 总超时 3s。
	HTTPClient *http.Client
	// MemberTTL 是成员缓存 TTL（默认 60s）。
	MemberTTL time.Duration
	// DepartmentTTL 是部门树缓存 TTL（默认 300s）。
	DepartmentTTL time.Duration
	// RoleTTL 是角色清单缓存 TTL（默认 300s）。
	RoleTTL time.Duration
	// NegativeTTL 是 404 负缓存 TTL（默认 30s，挡穿透）。
	NegativeTTL time.Duration
	// StaleOnError 允许在连接失败/超时/5xx 时返回过期缓存；0 表示关闭（默认关闭）。
	// 建议上限 5 分钟。
	StaleOnError time.Duration
	// Retry 是 GET 失败后的重试次数（带退避）。**零值即不重试**：
	// 0 是合法取值（"不重试"），因此这里不做"零值填默认"，需要重试请显式设为 1。
	Retry int
	// Metrics 可选注入计数器。
	Metrics Metrics
}

// Client 是目录 API 客户端（内置缓存/批量/降级）。
type Client struct {
	cfg Config
	hc  *http.Client
	now func() time.Time

	mu        sync.Mutex
	members   map[string]memberEntry
	depts     cacheEntry[[]Department]
	roles     cacheEntry[rolesResult]
	negatives map[string]time.Time
	// inflight 按 key 去重并发请求（single-flight，挡惊群）。
	inflight map[string]chan struct{}
	// batchETags 记录批量查询 path 对应的 ETag：成员缓存按 id 存，
	// 而 ETag 属于整批响应，粒度不同故单独存（只保留最近若干批）。
	batchETags map[string]string
}

type memberEntry struct {
	member  *Member
	fetched time.Time
}

type cacheEntry[T any] struct {
	value   T
	fetched time.Time
	valid   bool
	// etag 是服务端上次返回的响应体哈希（设计文档「缓存契约」）。
	// 下一次拉取带上 If-None-Match，304 时直接复用 value 并把 fetched 续到当前，
	// 省下带宽与反序列化（**不能省 DB 查询**，服务端要算哈希就得先取数）。
	etag string
}

type rolesResult struct {
	roles     []Role
	truncated bool
}

// New 构造目录客户端。
func New(cfg Config) (*Client, error) {
	if cfg.BaseURL == "" {
		return nil, errors.New("directory: baseURL is required")
	}
	if cfg.Tokens == nil {
		return nil, errors.New("directory: token provider is required")
	}
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{
			Timeout: 3 * time.Second,
			Transport: &http.Transport{
				DialContext:           (&net.Dialer{Timeout: time.Second}).DialContext,
				ResponseHeaderTimeout: 3 * time.Second,
			},
		}
	}
	if cfg.MemberTTL <= 0 {
		cfg.MemberTTL = 60 * time.Second
	}
	if cfg.DepartmentTTL <= 0 {
		cfg.DepartmentTTL = 300 * time.Second
	}
	if cfg.RoleTTL <= 0 {
		cfg.RoleTTL = 300 * time.Second
	}
	if cfg.NegativeTTL <= 0 {
		cfg.NegativeTTL = 30 * time.Second
	}
	return &Client{
		cfg:        cfg,
		hc:         hc,
		now:        time.Now,
		members:    map[string]memberEntry{},
		negatives:  map[string]time.Time{},
		inflight:   map[string]chan struct{}{},
		batchETags: map[string]string{},
	}, nil
}

// Member 取单个成员摘要（带 TTL 缓存；不存在/跨租户返回 ErrNotFound）。
func (c *Client) Member(ctx context.Context, userID string) (*Member, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, ErrNotFound
	}
	if m, err := c.cachedMember(userID); err == nil {
		return m, nil
	} else if !errors.Is(err, errCacheMiss) {
		return nil, err
	}
	got, err := c.Members(ctx, []string{userID})
	if err != nil {
		return nil, err
	}
	if m, ok := got[userID]; ok {
		return m, nil
	}
	return nil, ErrNotFound
}

// Members 批量取成员摘要：命中进 map，未命中（含跨租户/不存在）不出现在结果里。
//
// 个别 id 查不到不影响整体成功；超过 MaxBatchIDs 时返回 ErrTooManyIDs
// （不静默截断）。跨租户 id 与不存在 id 在语义上不可区分。
func (c *Client) Members(ctx context.Context, userIDs []string) (map[string]*Member, error) {
	unique := normalizeIDs(userIDs)
	if len(unique) > MaxBatchIDs {
		return nil, fmt.Errorf("%w: %d > %d", ErrTooManyIDs, len(unique), MaxBatchIDs)
	}
	out := make(map[string]*Member, len(unique))
	if len(unique) == 0 {
		return out, nil
	}
	pending, err := c.collectCached(unique, out)
	if err != nil {
		return nil, err
	}
	if len(pending) == 0 {
		c.countHit(false)
		return out, nil
	}
	c.countMiss()

	key := "members:" + strings.Join(pending, ",")
	release, wait, leader := c.acquire(key)
	if !leader {
		defer release()
		select {
		case <-wait:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		// 复用 leader 的结果：结果已并入缓存，这里只重读缓存。
		c.mu.Lock()
		defer c.mu.Unlock()
		for _, id := range pending {
			if entry, ok := c.members[id]; ok {
				out[id] = entry.member
			}
		}
		return out, nil
	}
	// leader 必须在**所有**退出路径上关闭 flight，否则该 key 会永久悬挂，
	// 后续同 key 调用全部退化为 follower 并永久阻塞。
	defer func() {
		release()
		c.finishFlight(key)
	}()

	prevETag := c.batchETag(key)
	fetched, newETag, notModified, fErr := c.fetchMembers(ctx, pending, prevETag)
	if fErr != nil {
		if stale, ok := c.staleMembers(pending, fErr); ok {
			c.countHit(true)
			return stale, nil
		}
		c.countUnavailable()
		return nil, fErr
	}
	stamp := c.now()
	if notModified {
		// 304：内容未变，直接延长本地副本的有效期（不省 DB 查询，省带宽与反序列化）。
		c.countNotModified()
		c.mu.Lock()
		for _, id := range pending {
			if entry, ok := c.members[id]; ok {
				entry.fetched = stamp
				c.members[id] = entry
				out[id] = entry.member
			}
		}
		c.mu.Unlock()
		return out, nil
	}
	c.mu.Lock()
	c.rememberBatchETagLocked(key, newETag)
	for _, id := range pending {
		if m, ok := fetched[id]; ok {
			c.members[id] = memberEntry{member: m, fetched: stamp}
		} else {
			// 跨租户 id 与不存在 id 不可区分：都进负缓存（防枚举 + 挡穿透）。
			c.negatives[id] = stamp.Add(c.cfg.NegativeTTL)
		}
	}
	c.mu.Unlock()
	for id, m := range fetched {
		out[id] = m
	}
	return out, nil
}

// batchETag 读取某批查询上次的 ETag（无则空串）。
func (c *Client) batchETag(key string) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.batchETags[key]
}

// rememberBatchETagLocked 记录 ETag 并限制 map 规模（调用方需持有 c.mu）。
// 批量查询的 key 组合近乎无限，因此只保留最近 maxBatchETags 条，
// 超出时按 map 随机顺序淘汰——ETag 只是优化，丢失只会退化为 200 全量响应。
func (c *Client) rememberBatchETagLocked(key, etag string) {
	if etag == "" {
		delete(c.batchETags, key)
		return
	}
	if len(c.batchETags) >= maxBatchETags {
		for k := range c.batchETags {
			if k == key {
				continue
			}
			delete(c.batchETags, k)
			break
		}
	}
	c.batchETags[key] = etag
}

// maxBatchETags 是批量 ETag 表的容量上限。
const maxBatchETags = 256

// DepartmentTree 取部门树（TTL 缓存 + 可选 stale 降级）。
func (c *Client) DepartmentTree(ctx context.Context) ([]Department, error) {
	c.mu.Lock()
	entry := c.depts
	c.mu.Unlock()
	if entry.valid && c.now().Sub(entry.fetched) < c.cfg.DepartmentTTL {
		c.countHit(false)
		return entry.value, nil
	}
	c.countMiss()
	var out []Department
	etag, notModified, err := c.getJSON(ctx, "/directory/departments/tree", &out, entry.etag)
	if err != nil {
		if stale, ok := staleValue(c, entry, err); ok {
			return stale, nil
		}
		c.countUnavailable()
		return nil, err
	}
	if notModified {
		out = entry.value
		c.countNotModified()
	}
	c.mu.Lock()
	c.depts = cacheEntry[[]Department]{value: out, fetched: c.now(), valid: true, etag: pickETag(etag, entry.etag)}
	c.mu.Unlock()
	return out, nil
}

// Roles 取角色清单（TTL 缓存 + 可选 stale 降级）。
func (c *Client) Roles(ctx context.Context) ([]Role, error) {
	c.mu.Lock()
	entry := c.roles
	c.mu.Unlock()
	if entry.valid && c.now().Sub(entry.fetched) < c.cfg.RoleTTL {
		c.countHit(false)
		return entry.value.roles, nil
	}
	c.countMiss()
	var payload struct {
		List      []Role `json:"list"`
		Truncated bool   `json:"truncated"`
	}
	etag, notModified, err := c.getJSON(ctx, "/directory/roles", &payload, entry.etag)
	if err != nil {
		if stale, ok := staleValue(c, entry, err); ok {
			return stale.roles, nil
		}
		c.countUnavailable()
		return nil, err
	}
	if notModified {
		c.countNotModified()
		c.mu.Lock()
		c.roles = cacheEntry[rolesResult]{value: entry.value, fetched: c.now(), valid: true, etag: pickETag(etag, entry.etag)}
		c.mu.Unlock()
		return entry.value.roles, nil
	}
	c.mu.Lock()
	c.roles = cacheEntry[rolesResult]{
		value:   rolesResult{roles: payload.List, truncated: payload.Truncated},
		fetched: c.now(),
		valid:   true,
		etag:    pickETag(etag, entry.etag),
	}
	c.mu.Unlock()
	return payload.List, nil
}

// RolesTruncated 报告最近一次 roles 响应是否被服务端截断（>500 条）。
func (c *Client) RolesTruncated() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.roles.value.truncated
}

// InvalidateMember 丢弃指定成员的缓存（调用方在写路径改名后可主动失效）。
func (c *Client) InvalidateMember(userID string) {
	c.mu.Lock()
	delete(c.members, userID)
	delete(c.negatives, userID)
	c.mu.Unlock()
}

// ---------- 内部实现 ----------

var errCacheMiss = errors.New("directory: cache miss")

// collectCached 把命中缓存的 id 填入 out，返回需要远端查询的 id。
func (c *Client) collectCached(ids []string, out map[string]*Member) ([]string, error) {
	now := c.now()
	pending := make([]string, 0, len(ids))
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, id := range ids {
		if exp, ok := c.negatives[id]; ok {
			if now.After(exp) {
				delete(c.negatives, id)
			} else {
				// 负缓存命中：该 id 在租户内不可见，不进入结果也不进入 pending。
				continue
			}
		}
		if entry, ok := c.members[id]; ok && now.Sub(entry.fetched) < c.cfg.MemberTTL {
			out[id] = entry.member
			continue
		}
		pending = append(pending, id)
	}
	return pending, nil
}

func (c *Client) cachedMember(userID string) (*Member, error) {
	now := c.now()
	c.mu.Lock()
	defer c.mu.Unlock()
	if exp, ok := c.negatives[userID]; ok {
		if now.After(exp) {
			delete(c.negatives, userID)
		} else {
			return nil, ErrNotFound
		}
	}
	entry, ok := c.members[userID]
	if !ok || now.Sub(entry.fetched) >= c.cfg.MemberTTL {
		return nil, errCacheMiss
	}
	c.countHit(false)
	return entry.member, nil
}

// staleValue 在 5xx/超时/连接失败且缓存年龄在上限内时返回过期副本。
// 401/403/404 一律透传，绝不降级——否则 M2M 客户端被撤销或 scope 被收紧后，
// SDK 会用旧缓存继续"放行"。
func staleValue[T any](c *Client, entry cacheEntry[T], err error) (T, bool) {
	var zero T
	if !entry.valid || c.cfg.StaleOnError <= 0 {
		return zero, false
	}
	if !errors.Is(err, ErrUnavailable) {
		return zero, false
	}
	if c.now().Sub(entry.fetched) > c.cfg.StaleOnError {
		return zero, false
	}
	c.countHit(true)
	return entry.value, true
}

// staleMembers 返回过期成员副本，**仅当失败原因是"目录暂时不可用"**
// （连接失败/超时/5xx）。401/403/404 属授权与可见性结论，必须透传、绝不降级。
func (c *Client) staleMembers(ids []string, cause error) (map[string]*Member, bool) {
	if c.cfg.StaleOnError <= 0 || !errors.Is(cause, ErrUnavailable) {
		return nil, false
	}
	now := c.now()
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make(map[string]*Member, len(ids))
	for _, id := range ids {
		entry, ok := c.members[id]
		if !ok || now.Sub(entry.fetched) > c.cfg.StaleOnError {
			continue
		}
		out[id] = entry.member
	}
	if len(out) == 0 {
		return nil, false
	}
	return out, true
}

func (c *Client) fetchMembers(ctx context.Context, ids []string, etag string) (map[string]*Member, string, bool, error) {
	path := "/directory/members?ids=" + url.QueryEscape(strings.Join(ids, ","))
	var payload struct {
		List    []Member `json:"list"`
		Missing []string `json:"missing"`
	}
	newETag, notModified, err := c.getJSON(ctx, path, &payload, etag)
	if err != nil {
		return nil, "", false, err
	}
	if notModified {
		return nil, newETag, true, nil
	}
	out := make(map[string]*Member, len(payload.List))
	for i := range payload.List {
		m := payload.List[i]
		if m.UserID == "" {
			continue
		}
		out[m.UserID] = &m
	}
	return out, newETag, false, nil
}

// getJSON 发起带令牌的 GET，按状态码语义分类错误（4xx 不重试、不降级）。
//
// 返回值 (etag, notModified, err)：
//   - 200 → (响应 ETag, false, nil)，out 已被填充；
//   - 304 → (原 ETag 或响应头值, true, nil)，调用方必须复用本地副本；
//   - 其余错误按状态码分类，4xx 一律透传、绝不降级。
func (c *Client) getJSON(ctx context.Context, path string, out any, etag string) (string, bool, error) {
	attempts := c.cfg.Retry + 1
	if attempts < 1 {
		attempts = 1
	}
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			select {
			case <-time.After(time.Duration(attempt) * 100 * time.Millisecond):
			case <-ctx.Done():
				return "", false, ctx.Err()
			}
		}
		body, respETag, status, err := c.doGet(ctx, path, etag)
		if err != nil {
			lastErr = fmt.Errorf("%w: %v", ErrUnavailable, err)
			continue
		}
		switch {
		case status == http.StatusOK:
			if err := json.Unmarshal(body, out); err != nil {
				return "", false, fmt.Errorf("directory: decode response: %w", err)
			}
			return respETag, false, nil
		case status == http.StatusNotModified:
			// 服务端确认内容未变：不反序列化，调用方复用本地副本。
			return pickETag(respETag, etag), true, nil
		case status == http.StatusUnauthorized:
			c.invalidateTokenOnce()
			return "", false, ErrUnauthorized
		case status == http.StatusForbidden:
			return "", false, ErrForbidden
		case status == http.StatusNotFound:
			return "", false, ErrNotFound
		case status == http.StatusBadRequest:
			return "", false, fmt.Errorf("directory: bad request (400): %s", truncateBody(body))
		case status >= 500:
			lastErr = fmt.Errorf("%w: status %d", ErrUnavailable, status)
			continue
		default:
			return "", false, fmt.Errorf("directory: unexpected status %d", status)
		}
	}
	if lastErr == nil {
		lastErr = ErrUnavailable
	}
	return "", false, lastErr
}

// pickETag 优先取新值，其次沿用旧值（服务端未回 ETag 时不丢已有缓存标识）。
func pickETag(fresh, prev string) string {
	if strings.TrimSpace(fresh) != "" {
		return strings.TrimSpace(fresh)
	}
	return prev
}

// doGet 发起一次带令牌的 GET，返回 (响应体, ETag, 状态码)。
// etag 非空时带上 If-None-Match，服务端可直接以 304 结束（body 为空）。
func (c *Client) doGet(ctx context.Context, path, etag string) ([]byte, string, int, error) {
	token, err := c.cfg.Tokens.Token(ctx)
	if err != nil {
		return nil, "", 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(c.cfg.BaseURL, "/")+path, nil)
	if err != nil {
		return nil, "", 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, "", 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, "", 0, err
	}
	return body, resp.Header.Get("ETag"), resp.StatusCode, nil
}

// invalidateTokenOnce 在 401 时让令牌缓存失效，下一次调用会重新取 M2M 令牌。
func (c *Client) invalidateTokenOnce() {
	if c.cfg.Tokens != nil {
		c.cfg.Tokens.Invalidate()
	}
}

// acquire 实现 single-flight：leader 返回 (release, nil, true)，
// follower 返回 (release, wait, false) 并等待 leader 完成。
func (c *Client) acquire(key string) (release func(), wait <-chan struct{}, leader bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if ch, ok := c.inflight[key]; ok {
		return func() {}, ch, false
	}
	ch := make(chan struct{})
	c.inflight[key] = ch
	return func() {}, ch, true
}

func (c *Client) finishFlight(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if ch, ok := c.inflight[key]; ok {
		close(ch)
		delete(c.inflight, key)
	}
}

func (c *Client) countHit(stale bool) {
	if c.cfg.Metrics != nil {
		c.cfg.Metrics.CacheHit(stale)
	}
}

func (c *Client) countMiss() {
	if c.cfg.Metrics != nil {
		c.cfg.Metrics.CacheMiss()
	}
}

func (c *Client) countUnavailable() {
	if c.cfg.Metrics != nil {
		c.cfg.Metrics.Unavailable()
	}
}

func (c *Client) countNotModified() {
	if m, ok := c.cfg.Metrics.(NotModifiedMetrics); ok {
		m.NotModified()
	}
}

func normalizeIDs(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	sort.Strings(out) // 稳定顺序：single-flight key 可复用
	return out
}

func truncateBody(body []byte) string {
	const max = 256
	if len(body) <= max {
		return string(body)
	}
	return string(body[:max])
}
