// Package logout 是 OIDC Back-Channel Logout 的 RP 侧接收端（框架无关，基于 net/http）。
//
// 处理语义（OpenID Connect Back-Channel Logout 1.0 §2.8）：
//   - 成功处理 → 200
//   - token 无效（签名/iss/aud/events/jti/sub/nonce/iat 校验失败）→ 400（无效 token 不重试）
//   - 本地登出回调失败 → 500（OP 按至少一次语义重投，见设计文档实施要点 5）
//
// jti 去重内置：**必须先撤销会话、成功后再记录 jti**——否则"撤销失败返回 500、
// OP 重投被 jti 判为重复"会把这次登出永久丢掉。
package logout

import (
	"context"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/morehao/ark-iam/sdk/contract"
)

// JWKS 是接收端所需的最小密钥来源（与 rp.KeySource 同形，避免包间循环依赖）。
type JWKS interface {
	PublicKey(ctx context.Context, kid string) (*rsa.PublicKey, error)
}

// SessionRevoker 是收到合法 logout_token 后执行本地会话清除的回调。
// 返回 error 时接收端按 500 处理（OP 会重投）；nil 表示已成功处理。
//
// claims 是已通过全部校验的 logout_token 声明；按会话终止优先用 claims.SessionID。
type SessionRevoker func(ctx context.Context, claims *contract.LogoutTokenClaims) error

// JTIStore 记录已处理过的 logout_token jti，用于至少一次投递下的去重。
//
// 默认实现是进程内内存表（单实例足够）；多实例部署由调用方注入落表实现
// （照 authelia 的 jti 表形态：sha256(jti) + 过期时间 + 插入时顺带清理过期行）。
type JTIStore interface {
	// Seen 报告该 jti 是否已被成功处理过且记录尚未过期。
	Seen(ctx context.Context, jti string) bool
	// Record 记录一个已成功处理的 jti，expiresAt 取该 logout_token 自身的 exp。
	Record(ctx context.Context, jti string, expiresAt time.Time) error
}

// MemoryJTIStore 是 JTIStore 的进程内实现（带过期清理）。
type MemoryJTIStore struct {
	mu   sync.Mutex
	seen map[string]time.Time
	now  func() time.Time
}

// NewMemoryJTIStore 构造内存 jti 表。
func NewMemoryJTIStore() *MemoryJTIStore {
	return &MemoryJTIStore{seen: make(map[string]time.Time), now: time.Now}
}

// Seen 实现 JTIStore。
func (s *MemoryJTIStore) Seen(_ context.Context, jti string) bool {
	if s == nil || jti == "" {
		return false
	}
	key := HashJTI(jti)
	s.mu.Lock()
	defer s.mu.Unlock()
	exp, ok := s.seen[key]
	if !ok {
		return false
	}
	if s.now().After(exp) {
		delete(s.seen, key)
		return false
	}
	return true
}

// Record 实现 JTIStore（插入时顺带清理过期行）。
func (s *MemoryJTIStore) Record(_ context.Context, jti string, expiresAt time.Time) error {
	if s == nil || jti == "" {
		return nil
	}
	now := s.now()
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, exp := range s.seen {
		if now.After(exp) {
			delete(s.seen, k)
		}
	}
	s.seen[HashJTI(jti)] = expiresAt
	return nil
}

// HashJTI 返回 jti 的存储键（sha256 十六进制）。
// 落表实现应存该摘要而非原始 jti（与 authelia 的 jti 表形态一致）。
func HashJTI(jti string) string {
	sum := sha256.Sum256([]byte(jti))
	return hex.EncodeToString(sum[:])
}

// Option 配置 Receiver。
type Option func(*receiverConfig)

type receiverConfig struct {
	issuer   string
	clientID string
	keys     JWKS
	revoker  SessionRevoker
	store    JTIStore
	leeway   time.Duration
	now      func() time.Time
	logger   Logger
}

// Logger 是接收端的最小日志接口（SDK 不引入日志框架）。
type Logger interface {
	Warnf(format string, args ...any)
}

type nopLogger struct{}

func (nopLogger) Warnf(string, ...any) {}

// WithLogger 注入日志实现。
func WithLogger(l Logger) Option {
	return func(c *receiverConfig) {
		if l != nil {
			c.logger = l
		}
	}
}

// WithIssuer 设置期望的 OP issuer（必填，fail-closed）。
func WithIssuer(issuer string) Option {
	return func(c *receiverConfig) { c.issuer = issuer }
}

// WithClientID 设置本 RP 的 client_id（必填，用于 aud 校验）。
func WithClientID(clientID string) Option {
	return func(c *receiverConfig) { c.clientID = clientID }
}

// WithKeySource 注入验签密钥来源。
func WithKeySource(keys JWKS) Option {
	return func(c *receiverConfig) { c.keys = keys }
}

// WithSessionRevoker 注入本地会话撤销回调。
func WithSessionRevoker(revoker SessionRevoker) Option {
	return func(c *receiverConfig) { c.revoker = revoker }
}

// WithJTIStore 注入 jti 去重实现（默认内存表）。
func WithJTIStore(store JTIStore) Option {
	return func(c *receiverConfig) { c.store = store }
}

// WithLeeway 设置时钟容差（默认与 rp.DefaultLeeway 一致的 30s）。
func WithLeeway(d time.Duration) Option {
	return func(c *receiverConfig) {
		if d >= 0 {
			c.leeway = d
		}
	}
}

func withNow(f func() time.Time) Option {
	return func(c *receiverConfig) {
		if f != nil {
			c.now = f
		}
	}
}

// Receiver 是 back-channel logout 的 net/http 处理器。
type Receiver struct {
	cfg *receiverConfig
}

// NewReceiver 构造接收端。issuer / clientID / 密钥来源缺失时所有请求 400（fail-closed）。
func NewReceiver(opts ...Option) *Receiver {
	cfg := &receiverConfig{
		store:  NewMemoryJTIStore(),
		leeway: 30 * time.Second,
		now:    time.Now,
		logger: nopLogger{},
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return &Receiver{cfg: cfg}
}

// ServeHTTP 实现 http.Handler。
func (r *Receiver) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		writeError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	status, msg := r.Handle(req.Context(), req.PostForm.Get("logout_token"))
	if status == http.StatusOK {
		w.WriteHeader(status)
		return
	}
	writeError(w, status, msg)
}

// Handle 执行一次完整的 logout_token 处理，返回 HTTP 状态码与错误消息。
//
// ServeHTTP 与各框架适配器（如 Gin）都走这里，保证"校验 → 幂等去重 → 撤销 →
// 记录 jti"的顺序只有一份实现：撤销失败返回 500 且**不记 jti**，让 OP 的重投有效。
func (r *Receiver) Handle(ctx context.Context, tokenStr string) (int, string) {
	claims, err := r.Parse(tokenStr)
	if err != nil {
		return http.StatusBadRequest, "invalid logout_token: " + err.Error()
	}
	// 重复投递：已成功处理过则直接 200（幂等），不再触发本地副作用。
	if claims.ID != "" && r.cfg.store.Seen(ctx, claims.ID) {
		return http.StatusOK, ""
	}
	if r.cfg.revoker != nil {
		if rErr := r.cfg.revoker(ctx, claims); rErr != nil {
			// 关键顺序：撤销失败不记 jti，让 OP 的重投能真正生效。
			return http.StatusInternalServerError, "logout processing failed"
		}
	}
	if claims.ID != "" {
		exp := r.cfg.now().Add(15 * time.Minute)
		if claims.ExpiresAt != nil {
			exp = claims.ExpiresAt.Time
		}
		if rErr := r.cfg.store.Record(ctx, claims.ID, exp); rErr != nil {
			// 去重记录失败不影响本次登出结果（会话已撤销），只记日志。
			r.cfg.logger.Warnf("[logout.Receiver] record logout jti fail, err:%v", rErr)
		}
	}
	return http.StatusOK, ""
}

// Seen 报告该 jti 是否已被成功处理过（供框架适配器做幂等短路与观测）。
func (r *Receiver) Seen(ctx context.Context, jti string) bool {
	if r == nil || jti == "" {
		return false
	}
	return r.cfg.store.Seen(ctx, jti)
}

// Parse 解析并校验 logout_token。
//
// 校验项（对齐 OIDC Back-Channel Logout 1.0 §2.6 与 keycloak 的 RP 校验清单）：
//   - 签名：RS256 + kid 必填 + 拒绝头部 jwk/jku/x5u
//   - iss：必须等于 OP issuer
//   - aud：必须等于本 RP 的 client_id
//   - exp：必填且未过期（带时钟容差）
//   - iat：必填
//   - events：必须包含 backchannel-logout 事件
//   - jti：必填（去重基础）
//   - nonce：必须不存在
//   - sub：必填（本仓 OP 始终下发；sid-only 属放宽项，见设计文档开放问题）
func (r *Receiver) Parse(tokenStr string) (*contract.LogoutTokenClaims, error) {
	if r == nil || r.cfg.keys == nil {
		return nil, errors.New("oidc logout: key source not configured")
	}
	if r.cfg.issuer == "" || r.cfg.clientID == "" {
		return nil, errors.New("oidc logout: issuer or client_id not configured")
	}
	if strings.TrimSpace(tokenStr) == "" {
		return nil, errors.New("oidc logout: empty logout_token")
	}
	claims := &contract.LogoutTokenClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (any, error) {
		for _, h := range []string{"jwk", "jku", "x5u"} {
			if _, ok := token.Header[h]; ok {
				return nil, fmt.Errorf("%w: %s", contract.ErrUnauthorizedHeader, h)
			}
		}
		kid, _ := token.Header["kid"].(string)
		if strings.TrimSpace(kid) == "" {
			return nil, contract.ErrMissingKid
		}
		return r.cfg.keys.PublicKey(context.Background(), kid)
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}),
		jwt.WithLeeway(r.cfg.leeway),
		jwt.WithTimeFunc(r.cfg.now),
		jwt.WithExpirationRequired(),
		jwt.WithIssuer(r.cfg.issuer),
		jwt.WithAudience(r.cfg.clientID),
	)
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("oidc logout: invalid logout_token")
	}
	if !claims.HasBackChannelLogoutEvent() {
		return nil, errors.New("oidc logout: missing backchannel-logout event")
	}
	if claims.ID == "" {
		return nil, errors.New("oidc logout: missing jti")
	}
	if claims.IssuedAt == nil {
		return nil, errors.New("oidc logout: missing iat")
	}
	if claims.Subject == "" {
		return nil, errors.New("oidc logout: missing sub")
	}
	if claims.Nonce != "" {
		return nil, errors.New("oidc logout: nonce must not be present")
	}
	return claims, nil
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(fmt.Sprintf(`{"code":%d,"message":%q}`, status, msg)))
}
