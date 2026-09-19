package rp

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/morehao/ark-iam/sdk/contract"
)

// Identity 是校验通过的令牌身份（框架无关）。
//
// 字段口径（见设计文档「分层与接入点」）：
//   - PersonID：sub=person:<id> 解析出的自然人 ID；机器令牌为空
//   - TenantID：claim tenant_id，租户作用域与审计隔离键
//   - UserID：claim user_id = tenant_user.id，**审计主键**（与 IAM 自身 created_by 口径一致）
//   - ClientID：aud 命中的本应用 client_id
//   - IsMachine：token_usage=machine
//   - SessionID：sid，用于按会话作废与 back-channel 登出匹配
//   - Scopes：标准 scope 声明（空格分隔），用于目录 API 等 M2M 权限判定
//   - ActorID：act.sub，机器凭证的代操作人
type Identity struct {
	PersonID  string
	TenantID  string
	UserID    string
	ClientID  string
	IsMachine bool
	SessionID string
	Scopes    []string
	ActorID   string
}

// HasScope 报告该身份是否携带指定 scope。
func (i *Identity) HasScope(scope string) bool {
	if i == nil {
		return false
	}
	for _, s := range i.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// identityContextKey 是身份在 context 中的私有键类型。
type identityContextKey struct{}

// WithIdentity 把已验证身份放入 context（消费者侧中间件使用）。
func WithIdentity(ctx context.Context, identity *Identity) context.Context {
	if ctx == nil || identity == nil {
		return ctx
	}
	return context.WithValue(ctx, identityContextKey{}, identity)
}

// IdentityFrom 从 context 取回已验证身份；未注入时返回 nil。
func IdentityFrom(ctx context.Context) *Identity {
	if ctx == nil {
		return nil
	}
	identity, _ := ctx.Value(identityContextKey{}).(*Identity)
	return identity
}

// verifierConfig 是 Verifier 的可选配置。
type verifierConfig struct {
	issuer    string
	audiences []string
	// requireAudience 显式要求 aud 必须命中；未配置 issuer/aud 时默认关闭以保持兼容。
	requireAudience bool
	leeway          time.Duration
	algorithms      []string
	keySource       KeySource
	now             func() time.Time
}

// VerifierOption 配置 Verifier。
type VerifierOption func(*verifierConfig)

// WithIssuer 设置期望的 OP issuer；设置后 iss 必须精确匹配。
func WithIssuer(issuer string) VerifierOption {
	return func(c *verifierConfig) { c.issuer = issuer }
}

// WithAudiences 设置本应用认可的 aud 集合；设置后 aud 必须包含其中之一。
func WithAudiences(audiences ...string) VerifierOption {
	return func(c *verifierConfig) {
		c.audiences = append(c.audiences, audiences...)
		c.requireAudience = len(c.audiences) > 0
	}
}

// WithLeeway 设置时钟容差（exp/nbf）。
//
// 默认 DefaultLeeway = 30s：主流实现都有显式偏移（zitadel WithIssuedAtOffset、
// keycloak allowedClockSkew），本 SDK 不再依赖签发端回拨 iat 来"掩盖"校验端零容差。
func WithLeeway(d time.Duration) VerifierOption {
	return func(c *verifierConfig) {
		if d >= 0 {
			c.leeway = d
		}
	}
}

// WithAlgorithms 设置允许的签名算法（白名单，默认仅 RS256）。
// 走允许列表而非硬编码：将来加 ES256 不必发 SDK 版本。
func WithAlgorithms(algs ...string) VerifierOption {
	return func(c *verifierConfig) {
		if len(algs) > 0 {
			c.algorithms = append([]string(nil), algs...)
		}
	}
}

// WithKeySource 注入密钥来源。
func WithKeySource(ks KeySource) VerifierOption {
	return func(c *verifierConfig) { c.keySource = ks }
}

func withVerifierNow(f func() time.Time) VerifierOption {
	return func(c *verifierConfig) {
		if f != nil {
			c.now = f
		}
	}
}

// DefaultLeeway 是默认时钟容差（30s，覆盖验收标准 3 的 ±30s 用例）。
const DefaultLeeway = 30 * time.Second

// forbiddenHeaderKeys 是必须拒绝的 JOSE 头：它们让令牌自带密钥分发信息，
// 是 SSRF 与密钥替换的入口（tinyauth 就发 jku，属反例）。
var forbiddenHeaderKeys = []string{"jwk", "jku", "x5u"}

// Verifier 是框架无关的 access token 校验器。
type Verifier struct {
	cfg     *verifierConfig
	keyFunc jwt.Keyfunc

	// ctxMu/ctx 保存本次 Verify 的调用上下文，供 jwt.Keyfunc（不接收 context）
	// 在触发未知 kid 刷新时沿用请求的取消与超时。Keyfunc 与 ParseWithClaims 同步执行，
	// 加锁只是为了让 race detector 满意（Verify 可被并发调用）。
	ctxMu sync.Mutex
	ctx   context.Context
}

// NewVerifier 构造校验器。必须注入 KeySource，否则所有校验都返回
// contract.ErrKeySourceUnavailable（fail-closed）。
func NewVerifier(ks KeySource, opts ...VerifierOption) *Verifier {
	cfg := &verifierConfig{
		leeway:     DefaultLeeway,
		algorithms: []string{jwt.SigningMethodRS256.Alg()},
		keySource:  ks,
		now:        time.Now,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	v := &Verifier{cfg: cfg}
	v.keyFunc = v.resolveKey
	return v
}

// Verify 校验原始令牌串并返回身份。
//
// 失败时返回的错误可用 errors.Is 与 contract 的 sentinel 错误比较
// （ErrExpired/ErrBadSignature/ErrBadIssuer/ErrBadAudience/ErrUnknownKid/
// ErrUnsupportedAlg/ErrMalformedToken/ErrMissingClaim/ErrUnauthorizedHeader）。
func (v *Verifier) Verify(ctx context.Context, rawToken string) (*Identity, error) {
	if v == nil || v.cfg == nil || v.cfg.keySource == nil {
		return nil, contract.ErrKeySourceUnavailable
	}
	if strings.TrimSpace(rawToken) == "" {
		return nil, contract.ErrMalformedToken
	}
	v.setVerifyContext(ctx)
	defer v.clearVerifyContext()
	claims := &contract.TokenClaims{}
	parserOpts := []jwt.ParserOption{
		jwt.WithValidMethods(v.cfg.algorithms),
		jwt.WithLeeway(v.cfg.leeway),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(v.cfg.now),
	}
	if v.cfg.issuer != "" {
		parserOpts = append(parserOpts, jwt.WithIssuer(v.cfg.issuer))
	}
	if len(v.cfg.audiences) > 0 {
		parserOpts = append(parserOpts, jwt.WithAudience(v.cfg.audiences...))
	}
	token, err := jwt.ParseWithClaims(rawToken, claims, v.keyFunc, parserOpts...)
	if err != nil {
		return nil, classifyJWTError(err)
	}
	if !token.Valid {
		return nil, contract.ErrBadSignature
	}
	if err := checkForbiddenHeaders(token); err != nil {
		return nil, err
	}
	return v.identityFromClaims(claims)
}

// verifyHeader 在验签之前检查保护头策略：kid 必填 + 禁止 jwk/jku/x5u。
//
// 头检查与签名算法校验都在 keyFunc 内完成，因为 jwt/v5 只在取键回调里
// 暴露 token.Header。
func checkForbiddenHeaders(token *jwt.Token) error {
	if token == nil {
		return contract.ErrMalformedToken
	}
	for _, key := range forbiddenHeaderKeys {
		if _, ok := token.Header[key]; ok {
			return fmt.Errorf("%w: %s", contract.ErrUnauthorizedHeader, key)
		}
	}
	return nil
}

// resolveKey 是 jwt.Keyfunc：做头部策略检查 + 按 kid 取公钥。
func (v *Verifier) resolveKey(token *jwt.Token) (any, error) {
	if err := checkForbiddenHeaders(token); err != nil {
		return nil, err
	}
	kid, _ := token.Header["kid"].(string)
	if strings.TrimSpace(kid) == "" {
		return nil, contract.ErrMissingKid
	}
	key, err := v.cfg.keySource.PublicKey(v.verifyContext(), kid)
	if err != nil {
		return nil, err
	}
	if key == nil {
		return nil, fmt.Errorf("%w: %s", contract.ErrUnknownKid, kid)
	}
	return key, nil
}

// setVerifyContext / verifyContext / clearVerifyContext 在 Keyfunc 与 Verify 之间
// 传递请求上下文（jwt/v5 的 Keyfunc 签名不接收 context）。
func (v *Verifier) setVerifyContext(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	v.ctxMu.Lock()
	v.ctx = ctx
	v.ctxMu.Unlock()
}

func (v *Verifier) verifyContext() context.Context {
	v.ctxMu.Lock()
	defer v.ctxMu.Unlock()
	if v.ctx == nil {
		return context.Background()
	}
	return v.ctx
}

func (v *Verifier) clearVerifyContext() {
	v.ctxMu.Lock()
	v.ctx = nil
	v.ctxMu.Unlock()
}

// identityFromClaims 由 claims 构建身份，并做人/机器令牌的必需 claim 校验。
func (v *Verifier) identityFromClaims(claims *contract.TokenClaims) (*Identity, error) {
	// 既非 person 也非 machine 的 token（如仅 client_id 的 client_credentials）拒绝。
	if !claims.HasPerson() && !claims.IsMachine() {
		return nil, fmt.Errorf("%w: neither person nor machine token", contract.ErrMissingClaim)
	}
	// tenant_id **不是**令牌有效性条件，而是「租户作用域」这一上下文条件：
	// 签发侧在 person→租户映射不唯一时本就不写该 claim，若在这里拒绝，
	// 需要租户的接口只能拿到 401（无法区分"令牌坏"与"缺上下文"）。
	// 因此按声明式语义处理：验签只回答"令牌是否可信、主体是谁"；
	// 需要租户的调用方（如目录 API）自行判定并返回 403。
	return &Identity{
		PersonID:  claims.PersonID(),
		TenantID:  claims.TenantID,
		UserID:    claims.UserID,
		ClientID:  claims.ClientID,
		IsMachine: claims.IsMachine(),
		SessionID: claims.SessionID,
		Scopes:    parseScopes(claims),
		ActorID:   parseActorID(claims),
	}, nil
}

// parseScopes 解析标准 scope 声明（RFC 6749 §3.3，空格分隔）。
// 缺失时返回 nil，调用方按"无 scope" fail-closed 判定。
func parseScopes(claims *contract.TokenClaims) []string {
	if claims == nil {
		return nil
	}
	if strings.TrimSpace(claims.Scope) == "" {
		return nil
	}
	fields := strings.Fields(claims.Scope)
	if len(fields) == 0 {
		return nil
	}
	return fields
}

// parseActorID 读取 act.sub（代操作主体）。
func parseActorID(claims *contract.TokenClaims) string {
	if claims == nil || claims.Actor == nil {
		return ""
	}
	if sub, ok := claims.Actor["sub"].(string); ok {
		return sub
	}
	return ""
}

// classifyJWTError 把 jwt/v5 的错误映射为契约 sentinel 错误，
// 让消费者用 errors.Is 分支而不是解析错误字符串。
func classifyJWTError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, jwt.ErrTokenExpired):
		return contract.ErrExpired
	case errors.Is(err, jwt.ErrTokenNotValidYet):
		return contract.ErrNotYetValid
	case errors.Is(err, jwt.ErrTokenInvalidIssuer):
		return contract.ErrBadIssuer
	case errors.Is(err, jwt.ErrTokenInvalidAudience):
		return contract.ErrBadAudience
	case errors.Is(err, jwt.ErrTokenMalformed):
		return contract.ErrMalformedToken
	case errors.Is(err, contract.ErrMissingKid),
		errors.Is(err, contract.ErrUnknownKid),
		errors.Is(err, contract.ErrUnauthorizedHeader),
		errors.Is(err, contract.ErrKeySourceUnavailable):
		return err
	case errors.Is(err, jwt.ErrTokenSignatureInvalid):
		// 算法不在允许列表时 jwt/v5 也报 ErrTokenSignatureInvalid，这里按更精确的语义拆分：
		// keyfunc 成功返回说明 alg 已通过白名单，故仅剩签名不匹配。
		return contract.ErrBadSignature
	case errors.Is(err, jwt.ErrTokenRequiredClaimMissing):
		return fmt.Errorf("%w: %v", contract.ErrMissingClaim, err)
	case errors.Is(err, jwt.ErrTokenUnverifiable):
		return contract.ErrBadSignature
	default:
		return fmt.Errorf("%w: %v", contract.ErrMalformedToken, err)
	}
}
