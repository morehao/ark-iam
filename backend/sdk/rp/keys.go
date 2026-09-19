// Package rp 是 RP（业务应用）侧的框架无关 OIDC 令牌校验核心。
//
// 设计要点（详见 .dsh/docs/specs/2026-09-18-rp-sdk-sharing-design.md「关键决策二」）：
//   - 本地验签：启动预取一次 JWKS，之后请求路径零网络调用，只用内存缓存按 kid 验签；
//   - 未知 kid 立即刷新一次（限速），拉取失败保留旧 key（stale-while-error）；
//   - 后台定期刷新；
//   - 只依赖标准库 + jwt/v5，不含任何框架（gin/echo/grpc 绑定由消费者自写）。
package rp

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/morehao/ark-iam/sdk/contract"
)

// Logger 是 SDK 的最小日志接口（避免引入任何日志框架依赖）。
// 未注入时静默；调用方可传自己的日志实现。
type Logger interface {
	// Debugf 打印调试信息（如 JWKS 刷新成功）。
	Debugf(format string, args ...any)
	// Warnf 打印告警（如 JWKS 刷新失败、保留旧 key）。
	Warnf(format string, args ...any)
}

// nopLogger 是默认日志实现（静默）。
type nopLogger struct{}

func (nopLogger) Debugf(string, ...any) {}
func (nopLogger) Warnf(string, ...any)  {}

// KeySource 是验签密钥来源：按 kid 返回 RSA 公钥。
//
// 实现方（*Keys）内置 JWKS 拉取、内存缓存、单飞、未知 kid 立即刷新（限速）、
// 后台定期刷新、拉取失败保留旧 key 与可选落盘兜底。
type KeySource interface {
	// PublicKey 返回 kid 对应的 RSA 公钥。
	// 命中内存缓存直接返回；未命中时按限速触发一次远端刷新。
	// 找不到时报 contract.ErrUnknownKid；密钥来源整体不可用时
	// 报 contract.ErrKeySourceUnavailable。
	PublicKey(ctx context.Context, kid string) (*rsa.PublicKey, error)
}

const (
	// DefaultJWKSTTL 是 JWKS 内存缓存的后台刷新周期。
	DefaultJWKSTTL = 10 * time.Minute
	// DefaultMinRefreshInterval 是未知 kid 触发的即时刷新限速窗口（1 次/分钟）。
	DefaultMinRefreshInterval = time.Minute
	// DefaultMaxKeyAge 是「拉取失败保留旧 key」的最长容忍时长。
	// 超过它仍无法刷新则视为密钥源不可用（fail-closed），避免无限期使用陈旧密钥。
	DefaultMaxKeyAge = 24 * time.Hour
	// DefaultHTTPTimeout 是 JWKS 拉取的单次超时。
	DefaultHTTPTimeout = 3 * time.Second
	// keysEndpointPath 是 zitadel/oidc（本仓 OP 使用的库）的默认 JWKS 端点路径：
	// discovery 文档里 jwks_uri = {issuer}/keys。
	keysEndpointPath = "/keys"
	// wellKnownJWKSPath 是 {issuer}/.well-known/jwks.json：非标准 OP 的兼容路径，
	// 只作为 issuer 形态的**最后**候选，绝不能当成本仓 OP 的发布路径。
	wellKnownJWKSPath = "/.well-known/jwks.json"
	discoveryPath     = "/.well-known/openid-configuration"
)

// KeysOption 配置 *Keys。
type KeysOption func(*keysConfig)

type keysConfig struct {
	ttl                time.Duration
	minRefreshInterval time.Duration
	maxKeyAge          time.Duration
	httpClient         *http.Client
	logger             Logger
	fallbackFile       string
	now                func() time.Time
}

// WithJWKSTTL 设置后台刷新周期。
func WithJWKSTTL(d time.Duration) KeysOption {
	return func(c *keysConfig) {
		if d > 0 {
			c.ttl = d
		}
	}
}

// WithMinRefreshInterval 设置未知 kid 触发的即时刷新限速窗口。
func WithMinRefreshInterval(d time.Duration) KeysOption {
	return func(c *keysConfig) {
		if d >= 0 {
			c.minRefreshInterval = d
		}
	}
}

// WithMaxKeyAge 设置拉取失败时保留旧 key 的最长容忍时长（stale-while-error 上界）。
func WithMaxKeyAge(d time.Duration) KeysOption {
	return func(c *keysConfig) {
		if d > 0 {
			c.maxKeyAge = d
		}
	}
}

// WithHTTPClient 注入自定义 HTTP 客户端（测试与代理场景）。
func WithHTTPClient(hc *http.Client) KeysOption {
	return func(c *keysConfig) {
		if hc != nil {
			c.httpClient = hc
		}
	}
}

// WithLogger 注入日志实现。
func WithLogger(l Logger) KeysOption {
	return func(c *keysConfig) {
		if l != nil {
			c.logger = l
		}
	}
}

// WithJWKSFallback 开启落盘兜底：远端不可用时读取该文件中的 JWKS。
//
// 与目录客户端的 WithStaleOnError 区分命名（这里缓存的是公钥，不是目录数据）。
// 调用方应保证文件权限为 0600。
func WithJWKSFallback(path string) KeysOption {
	return func(c *keysConfig) {
		c.fallbackFile = path
	}
}

func withNow(f func() time.Time) KeysOption {
	return func(c *keysConfig) {
		if f != nil {
			c.now = f
		}
	}
}

func newKeysConfig(opts []KeysOption) *keysConfig {
	c := &keysConfig{
		ttl:                DefaultJWKSTTL,
		minRefreshInterval: DefaultMinRefreshInterval,
		maxKeyAge:          DefaultMaxKeyAge,
		httpClient:         &http.Client{Timeout: DefaultHTTPTimeout},
		logger:             nopLogger{},
		now:                time.Now,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Keys 是 KeySource 的默认实现。
type Keys struct {
	cfg *keysConfig
	// urls 是按优先级排列的候选 JWKS 端点（构造期由 jwksURLCandidates 解析）。
	// 首个成功拉取的端点会被固化为 resolvedURL，此后刷新只打这一个地址。
	urls []string
	// resolvedURL 是已成功拉到密钥的端点；空表示尚未确定。
	resolvedURL string
	keys        map[string]*rsa.PublicKey

	mu          sync.RWMutex
	fetchedAt   time.Time
	lastAttempt time.Time
	inflight    chan struct{}
	closeOnce   sync.Once
	stop        chan struct{}
}

var _ KeySource = (*Keys)(nil)

// NewKeys 构造密钥来源。
//
//	jwksURL：既可传 JWKS 端点完整地址（如 https://op.example.com/oidc/keys 或
//	         .../jwks.json，原样使用），也可传 issuer（如 http://localhost:8100/oidc）——
//	         后者按「标准 discovery 的 jwks_uri → {issuer}/keys → {issuer}/.well-known/jwks.json」
//	         顺序逐个尝试。**不能**假定 issuer 下的 JWKS 一定在 .well-known/jwks.json：
//	         zitadel/oidc（本仓 OP）发布在 {issuer}/keys。
//
// NewKeys 立即同步预取一次；拉取失败时若配置了 WithJWKSFallback 且文件有效则用文件内容，
// 否则返回错误（fail-fast，避免"起来了但全部 401"）。
func NewKeys(ctx context.Context, jwksURL string, opts ...KeysOption) (*Keys, error) {
	cfg := newKeysConfig(opts)
	k := &Keys{
		cfg:  cfg,
		urls: jwksURLCandidates(ctx, jwksURL, cfg),
		keys: map[string]*rsa.PublicKey{},
		stop: make(chan struct{}),
	}
	if err := k.refresh(ctx, true); err != nil {
		if fbErr := k.loadFallback(); fbErr != nil {
			return nil, fmt.Errorf("rp: prefetch JWKS from %s failed: %w", strings.Join(k.urls, ", "), err)
		}
		cfg.logger.Warnf("[rp.Keys] prefetch JWKS fail, using fallback file, err:%v", err)
	}
	go k.backgroundRefresh()
	return k, nil
}

// NewKeysFromSet 用已知公钥集合构造密钥来源（进程内形态，零网络调用）。
//
// gateway 单体部署时由 OP 直接注入全部公钥（多 key 并行发布），
// 内置应用因此不必挂载 OP 私钥、也不必发网络请求。
func NewKeysFromSet(keys map[string]*rsa.PublicKey) *Keys {
	return &Keys{
		cfg:       newKeysConfig(nil),
		keys:      cloneKeyMap(keys),
		fetchedAt: time.Now(),
		stop:      make(chan struct{}),
	}
}

// Close 停止后台刷新。
func (k *Keys) Close() {
	if k == nil {
		return
	}
	k.closeOnce.Do(func() { close(k.stop) })
}

// PublicKey 实现 KeySource。
func (k *Keys) PublicKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	if k == nil {
		return nil, contract.ErrKeySourceUnavailable
	}
	if kid == "" {
		return nil, contract.ErrMissingKid
	}
	if key, ok := k.lookup(kid); ok {
		return key, nil
	}
	// 未命中：按限速立即刷新一次（kid 可能来自刚轮换出的新 key）。
	if err := k.refresh(ctx, false); err == nil {
		if key, ok := k.lookup(kid); ok {
			return key, nil
		}
	}
	// 区分两种失败语义：密钥集整体不可用（无有效缓存且刷新失败）
	// 与"仅该 kid 不在已发布的密钥集里"（可能已被紧急轮换摘除）。
	if !k.isUsable() {
		return nil, contract.ErrKeySourceUnavailable
	}
	return nil, fmt.Errorf("%w: %s", contract.ErrUnknownKid, kid)
}

// isUsable 报告当前缓存是否仍在 maxKeyAge 内。
func (k *Keys) isUsable() bool {
	k.mu.RLock()
	defer k.mu.RUnlock()
	return k.usableLocked()
}

// Prime 在启动期预取一次；可选落盘兜底已在 NewKeys 内处理。
func (k *Keys) Prime(ctx context.Context) error {
	return k.refresh(ctx, true)
}

func (k *Keys) lookup(kid string) (*rsa.PublicKey, bool) {
	k.mu.RLock()
	defer k.mu.RUnlock()
	// 超过 maxKeyAge 的缓存视为不可用（fail-closed）：否则一次长时间故障后
	// 进程会无限期用陈旧公钥放行，紧急轮换摘除旧 key 的语义被静默绕过。
	if !k.usableLocked() {
		return nil, false
	}
	key, ok := k.keys[kid]
	return key, ok
}

// refresh 拉取并替换密钥集。
//
// force=true 表示启动预取/后台刷新（不受限速约束）；force=false 表示
// 未知 kid 触发的即时刷新（受限速窗口与单飞约束）。
// 拉取失败时保留旧 key（stale-while-error），但超过 maxKeyAge 视为不可用。
func (k *Keys) refresh(ctx context.Context, force bool) error {
	if len(k.urls) == 0 {
		return nil // 进程内形态：无远端可刷新
	}
	k.mu.Lock()
	if !force {
		if k.cfg.now().Sub(k.lastAttempt) < k.cfg.minRefreshInterval {
			fresh := k.usableLocked()
			k.mu.Unlock()
			if fresh {
				return nil
			}
			return contract.ErrKeySourceUnavailable
		}
	}
	// 单飞：同刻只允许一次拉取，其余等待复用结果。
	if k.inflight != nil {
		wait := k.inflight
		k.mu.Unlock()
		select {
		case <-wait:
		case <-ctx.Done():
			return ctx.Err()
		}
		return nil
	}
	done := make(chan struct{})
	k.inflight = done
	k.lastAttempt = k.cfg.now()
	k.mu.Unlock()

	keys, usedURL, err := k.fetch(ctx)

	k.mu.Lock()
	k.inflight = nil
	if err == nil {
		k.keys = keys
		k.fetchedAt = k.cfg.now()
		k.resolvedURL = usedURL
	}
	usable := k.usableLocked()
	k.mu.Unlock()

	close(done)

	if err != nil {
		if usable {
			k.cfg.logger.Warnf("[rp.Keys] refresh JWKS fail, keep previous keys, err:%v", err)
			if fallbackErr := k.loadFallback(); fallbackErr == nil {
				k.cfg.logger.Warnf("[rp.Keys] loaded JWKS from fallback file")
			}
		}
		return err
	}
	k.cfg.logger.Debugf("[rp.Keys] JWKS refreshed, keys:%d", len(keys))
	if k.cfg.fallbackFile != "" {
		k.persistFallback()
	}
	return nil
}

// usableLocked 判断当前缓存是否仍在 maxKeyAge 内（调用方需持有 k.mu）。
func (k *Keys) usableLocked() bool {
	if len(k.keys) == 0 || k.fetchedAt.IsZero() {
		return false
	}
	return k.cfg.now().Sub(k.fetchedAt) <= k.cfg.maxKeyAge
}

func (k *Keys) backgroundRefresh() {
	ticker := time.NewTicker(k.cfg.ttl)
	defer ticker.Stop()
	for {
		select {
		case <-k.stop:
			return
		case <-ticker.C:
			if err := k.refresh(context.Background(), true); err != nil {
				k.cfg.logger.Warnf("[rp.Keys] background JWKS refresh fail, err:%v", err)
			}
		}
	}
}

// fetch 依次尝试候选端点，返回首个能解析出可用 RSA key 的端点及其密钥集。
//
// 端点一旦被确定（resolvedURL 非空）就只打这一个地址：后续刷新不该再为
// 「猜端点」付出额外请求，也不该因某次候选顺序变化而换端点。
func (k *Keys) fetch(ctx context.Context) (map[string]*rsa.PublicKey, string, error) {
	k.mu.RLock()
	resolved := k.resolvedURL
	k.mu.RUnlock()

	urls := k.urls
	if resolved != "" {
		urls = []string{resolved}
	}
	failures := make([]string, 0, len(urls))
	for _, url := range urls {
		keys, err := k.fetchOne(ctx, url)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", url, err))
			continue
		}
		if url != resolved {
			k.cfg.logger.Debugf("[rp.Keys] JWKS endpoint resolved: %s", url)
		}
		return keys, url, nil
	}
	return nil, "", errors.New(strings.Join(failures, "; "))
}

// fetchOne 拉取并解析单个端点的 JWKS。
func (k *Keys) fetchOne(ctx context.Context, url string) (map[string]*rsa.PublicKey, error) {
	body, err := k.get(ctx, url)
	if err != nil {
		return nil, err
	}
	keys, err := parseJWKS(body)
	if err != nil {
		return nil, err
	}
	if len(keys) == 0 {
		return nil, errors.New("rp: JWKS contains no usable RSA key")
	}
	return keys, nil
}

// get 发起 GET 并限制响应体大小（JWKS 不可能很大，防被喂大响应）。
func (k *Keys) get(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := k.cfg.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("rp: JWKS endpoint %s returned %d", url, resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 1<<20))
}

// jwksURLCandidates 把配置值展开为按优先级排列的候选 JWKS 端点。
//
// 配置值有两种形态：
//   - **显式端点**（末段为 keys/jwks，或以 .json 结尾，如 {issuer}/keys、
//     .../jwks.json）：原样作为唯一候选，不做任何推导；
//   - **issuer**（如 http://localhost:8100/oidc）：先取标准 OIDC discovery 文档
//     声明的 jwks_uri，失败再回退 {issuer}/keys 与 {issuer}/.well-known/jwks.json。
//
// 为什么 issuer 形态不能只认 {issuer}/.well-known/jwks.json：zitadel/oidc
// （本仓 OP 使用的库）默认把 JWKS 发布在 {issuer}/keys，discovery 的 jwks_uri
// 也指向它；写死另一条路径会让启动预取拿到 404，进而 fail-fast 拒绝启动。
func jwksURLCandidates(ctx context.Context, raw string, cfg *keysConfig) []string {
	raw = strings.TrimRight(strings.TrimSpace(raw), "/")
	if raw == "" {
		return nil
	}
	if looksLikeJWKSEndpoint(raw) {
		return []string{raw}
	}

	out := make([]string, 0, 3)
	if resolved, err := DiscoveryJWKSURL(ctx, raw, cfg.httpClient); err == nil && resolved != "" {
		out = append(out, strings.TrimRight(resolved, "/"))
	} else if err != nil {
		cfg.logger.Warnf("[rp.Keys] resolve jwks_uri via discovery fail, issuer:%s, err:%v", raw, err)
	}
	for _, candidate := range []string{raw + keysEndpointPath, raw + wellKnownJWKSPath} {
		if !slices.Contains(out, candidate) {
			out = append(out, candidate)
		}
	}
	return out
}

// looksLikeJWKSEndpoint 判断配置值是否已是 JWKS 端点而非 issuer：
// 末段为 keys/jwks（如 {issuer}/keys），或以 .json 结尾（如 .../jwks.json）。
func looksLikeJWKSEndpoint(raw string) bool {
	lower := strings.ToLower(raw)
	if strings.HasSuffix(lower, ".json") {
		return true
	}
	return strings.HasSuffix(lower, "/keys") || strings.HasSuffix(lower, "/jwks")
}

// DiscoveryJWKSURL 通过 OIDC discovery 文档解析 JWKS 端点（标准 OIDC issuer 场景）。
// 本仓 OP 的 issuer 形如 http://host/oidc，其 discovery 文档声明
// jwks_uri = {issuer}/keys（zitadel/oidc 默认端点）。
func DiscoveryJWKSURL(ctx context.Context, issuer string, hc *http.Client) (string, error) {
	issuer = strings.TrimRight(issuer, "/")
	if issuer == "" {
		return "", errors.New("rp: empty issuer")
	}
	if hc == nil {
		hc = &http.Client{Timeout: DefaultHTTPTimeout}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, issuer+discoveryPath, nil)
	if err != nil {
		return "", err
	}
	resp, err := hc.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("rp: discovery endpoint %s returned %d", issuer+discoveryPath, resp.StatusCode)
	}
	var doc struct {
		JWKSURI string `json:"jwks_uri"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&doc); err != nil {
		return "", err
	}
	if doc.JWKSURI == "" {
		return "", errors.New("rp: discovery document has no jwks_uri")
	}
	return doc.JWKSURI, nil
}

// jwkSet 是 JWKS 文档的 wire 形态。
type jwkSet struct {
	Keys []jwk `json:"keys"`
}

// jwk 是 JWK 文档中的单个密钥（只解析 RSA 所需字段）。
type jwk struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// parseJWKS 解析 JWKS 文档为 kid → RSA 公钥（忽略非 RSA / 用于加密的密钥）。
func parseJWKS(raw []byte) (map[string]*rsa.PublicKey, error) {
	var set jwkSet
	if err := json.Unmarshal(raw, &set); err != nil {
		return nil, fmt.Errorf("rp: decode JWKS: %w", err)
	}
	out := make(map[string]*rsa.PublicKey, len(set.Keys))
	for _, k := range set.Keys {
		if k.Kid == "" {
			continue // 无 kid 的 key 在按 kid 取键的验签路径上不可用
		}
		if !strings.EqualFold(k.Kty, "RSA") {
			continue
		}
		if k.Use != "" && k.Use != "sig" {
			continue
		}
		pub, err := rsaPublicKeyFromJWK(k)
		if err != nil {
			return nil, fmt.Errorf("rp: parse JWK %s: %w", k.Kid, err)
		}
		out[k.Kid] = pub
	}
	return out, nil
}

// rsaPublicKeyFromJWK 由 JWK 的 n/e 组装 RSA 公钥（自研约 20 行，避免引入 jose 依赖）。
func rsaPublicKeyFromJWK(k jwk) (*rsa.PublicKey, error) {
	if k.N == "" || k.E == "" {
		return nil, errors.New("missing n or e")
	}
	nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, fmt.Errorf("decode n: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, fmt.Errorf("decode e: %w", err)
	}
	if len(eBytes) == 0 || len(eBytes) > 8 {
		return nil, errors.New("invalid exponent length")
	}
	var e int
	for _, b := range eBytes {
		e = e<<8 | int(b)
	}
	if e < 3 {
		return nil, errors.New("invalid exponent")
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: e}, nil
}

// loadFallback 读取落盘兜底文件并替换内存密钥集（仅在配置了 WithJWKSFallback 时生效）。
func (k *Keys) loadFallback() error {
	if k.cfg.fallbackFile == "" {
		return errors.New("rp: no fallback file configured")
	}
	raw, err := os.ReadFile(k.cfg.fallbackFile)
	if err != nil {
		return err
	}
	keys, err := parseJWKS(raw)
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		return errors.New("rp: fallback JWKS contains no usable RSA key")
	}
	k.mu.Lock()
	k.keys = keys
	k.fetchedAt = k.cfg.now()
	k.mu.Unlock()
	return nil
}

// persistFallback 把最近一次成功的 JWKS 写入落盘兜底文件（0600）。
func (k *Keys) persistFallback() {
	if k.cfg.fallbackFile == "" {
		return
	}
	k.mu.RLock()
	raw, err := json.Marshal(jwkSet{Keys: k.jwksLocked()})
	k.mu.RUnlock()
	if err != nil {
		k.cfg.logger.Warnf("[rp.Keys] marshal fallback JWKS fail, err:%v", err)
		return
	}
	if dir := filepath.Dir(k.cfg.fallbackFile); dir != "." && dir != "" {
		if mkErr := os.MkdirAll(dir, 0o700); mkErr != nil {
			k.cfg.logger.Warnf("[rp.Keys] create fallback dir fail, err:%v", mkErr)
			return
		}
	}
	if wErr := os.WriteFile(k.cfg.fallbackFile, raw, 0o600); wErr != nil {
		k.cfg.logger.Warnf("[rp.Keys] write fallback JWKS fail, err:%v", wErr)
	}
}

// jwksLocked 把内存密钥集还原为 JWK 列表（供落盘兜底使用）。
func (k *Keys) jwksLocked() []jwk {
	out := make([]jwk, 0, len(k.keys))
	for kid, pub := range k.keys {
		out = append(out, jwk{
			Kty: "RSA",
			Kid: kid,
			Alg: "RS256",
			Use: "sig",
			N:   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
			E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
		})
	}
	return out
}

func cloneKeyMap(in map[string]*rsa.PublicKey) map[string]*rsa.PublicKey {
	out := make(map[string]*rsa.PublicKey, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
