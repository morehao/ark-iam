package middleware

import (
	"context"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/morehao/ark-iam/pkg/config"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/sdk/contract"
	"github.com/morehao/ark-iam/sdk/rp"

	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/glog"
)

const (
	AuthHeaderKey = "Authorization"
	AuthBearer    = "Bearer "
)

// ContextKeyClientID 是 access token 中 client_id 声明在 gin 上下文里的键。
// 它标识"本次请求是经由哪个 OIDC 客户端进来的"，是应用级入口策略
// （pkg/core/application 的两个开关）唯一的解析入口。
// golib 的 gcontext 没有对应常量，因此在本包内定义并配套 ClientIDFromContext 读取。
const ContextKeyClientID = "oidcClientID"

// ContextKeyOIDCIdentity 是完整 rp.Identity（含 Scopes）在 gin 上下文里的键，
// 供下游做策略判定（鉴权已完成，此处只回答"该身份是否有权做这件事"）。
const ContextKeyOIDCIdentity = "oidcIdentity"

// ClientIDFromContext 读取鉴权中间件注入的 client_id；未注入（如 skip path、
// API Key 通道）时返回空字符串，调用方应对空值 fail-closed。
func ClientIDFromContext(ctx *gin.Context) string {
	if ctx == nil {
		return ""
	}
	return ctx.GetString(ContextKeyClientID)
}

// KeySource 是 OP 公钥来源的导出别名（真源 sdk/rp.KeySource）。
type KeySource = rp.KeySource

// TokenClaims 是 access token 私有 claim 的兼容别名（真源在 sdk/contract）。
// 保留该导出名是为了不破坏既有调用方的类型断言。
type TokenClaims = contract.TokenClaims

type authConfig struct {
	skipPaths       []string
	validateOIDCSSO func(ctx *gin.Context, personID string, isMachineToken bool) bool
	// issuer 为 OP 的 issuer，配置后强制校验 access token 的 iss（H3）。
	issuer string
	// audiences 为本应用（RP）认可的 aud 集合，配置后强制校验 access token 的 aud，
	// 防止同一 OP 下其它 client 的 token 串用本应用接口。
	audiences []string
	// keySource 提供按 kid 取公钥的能力（支持 OP 多 key 轮换）。
	keySource rp.KeySource
	// disableLegacyUserLookup 关闭"token 未携带 user_id 时反查 tenant_user"的兼容分支。
	// 生产不应打开；仅用于验证严格模式。
	disableLegacyUserLookup bool
}

type AuthOption func(*authConfig)

func WithAuthSkipPaths(paths ...string) AuthOption {
	return func(c *authConfig) {
		c.skipPaths = append(c.skipPaths, paths...)
	}
}

// WithOIDCIssuer 注入期望的 OP issuer。设置后 access token 的 iss 必须精确匹配，
// 否则按无效 token 拒绝。
func WithOIDCIssuer(issuer string) AuthOption {
	return func(c *authConfig) {
		c.issuer = issuer
	}
}

// WithOIDCAudiences 注入本应用认可的 aud 集合（通常是本应用的 OIDC client_id）。
// 设置后 access token 的 aud 必须包含其中之一，否则按无效 token 拒绝。
func WithOIDCAudiences(audiences ...string) AuthOption {
	return func(c *authConfig) {
		c.audiences = append(c.audiences, audiences...)
	}
}

// WithOIDCKeySource 注入按 kid 取公钥的密钥来源（sdk/rp.KeySource）。
// 这是支持 OP 多 key 轮换的正路：未知 kid 时 SDK 会限速刷新 JWKS，
// 而不是像旧实现那样把公钥在启动时钉死（轮换后必然 401）。
func WithOIDCKeySource(source rp.KeySource) AuthOption {
	return func(c *authConfig) {
		if source != nil {
			c.keySource = source
		}
	}
}

// WithOIDCSSOValidation 注入 OIDC 访问令牌的 SSO 会话校验器。
// 校验 OIDC 令牌有效后，如果该校验器返回 false（该自然人不再有有效的 SSO 会话，
// 例如已在其他应用全局登出），则本次请求按未认证处理，返回 401。
// isMachineToken 标识该令牌是否为机器凭证（client_credentials/API Key）签发，
// 机器凭证不依赖浏览器 SSO 会话活性，校验器可据此直接放行。
func WithOIDCSSOValidation(validate func(ctx *gin.Context, personID string, isMachineToken bool) bool) AuthOption {
	return func(c *authConfig) {
		c.validateOIDCSSO = validate
	}
}

// WithOIDCLegacyUserLookup 控制"person token 未携带 user_id 时反查 tenant_user"
// 的兼容分支（默认开启）。保留它是为了让尚未升级的存量 token 在切换期仍可用；
// 待 OP 全面下发 user_id 后可关闭。
func WithOIDCLegacyUserLookup(enable bool) AuthOption {
	return func(c *authConfig) {
		c.disableLegacyUserLookup = !enable
	}
}

// OIDCCompatibleAuth 构造 OIDC 鉴权中间件。
//
// getOIDCPublicKey 是历史签名（返回 OP 公钥）。它被包装为"单 key KeySource"，
// 因此调用方无需改动即可获得 kid 匹配能力；但**无法感知轮换**——新部署请改用
// WithOIDCKeySource / NewKeySourceFromConfig。
//
// Deprecated: 只保留给仓外既有调用方做零改动迁移。单 key 快照在 OP 轮换签名密钥后
// 需要重启本进程才生效（例行轮换会中断）；新代码一律走 WithOIDCKeySource +
// NewKeySourceFromConfig（JWKS 预取 + kid 查表 + 未知 kid 限速刷新）。本仓已无调用方。
func OIDCCompatibleAuth(getOIDCPublicKey func() *rsa.PublicKey, opts ...AuthOption) gin.HandlerFunc {
	if getOIDCPublicKey != nil {
		opts = append([]AuthOption{WithOIDCKeySource(staticKeySource(getOIDCPublicKey))}, opts...)
	}
	return OIDCAuth(opts...)
}

// OIDCAuth 构造完全由 AuthOption 驱动的 OIDC 鉴权中间件。
func OIDCAuth(opts ...AuthOption) gin.HandlerFunc {
	cfg := &authConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	verifier, vErr := newVerifier(cfg)

	return func(ctx *gin.Context) {
		if isSkippedPath(ctx.Request.URL.Path, cfg.skipPaths) {
			ctx.Next()
			return
		}

		// 并行鉴权：若请求显式携带 x-api-key，则任一通过即可（OIDC 或 API Key）。
		// 机器凭证（API Key）不依赖浏览器 SSO 会话活性，见设计文档 §4.4。
		// 仅以 x-api-key 头作为机器凭证通道，避免与 Authorization: Bearer 的 OIDC 通道冲突。
		if ctx.GetHeader("x-api-key") != "" {
			if AuthenticateApiKey(ctx) {
				ctx.Next()
				return
			}
			// API Key 非法/过期/吊销：Authenticate 已写入 401 响应，直接终止
			return
		}

		tokenStr := extractToken(ctx)
		if tokenStr == "" {
			glog.Errorf(ctx, "[oidcauth] missing auth token")
			abortUnauthorized(ctx, "missing auth token")
			return
		}
		if vErr != nil {
			// 密钥来源不可用：fail-closed，绝不因为"配不出来"就放行。
			glog.Errorf(ctx, "[oidcauth] verifier not ready, err:%v", vErr)
			abortUnauthorized(ctx, "auth verifier unavailable")
			return
		}

		identity, err := verifier.Verify(ctx, tokenStr)
		if err == nil {
			if cfg.validateOIDCSSO != nil && !cfg.validateOIDCSSO(ctx, identity.PersonID, identity.IsMachine) {
				glog.Warnf(ctx, "[oidcauth] sso session revoked, personID:%s", identity.PersonID)
				abortUnauthorized(ctx, "session expired")
				return
			}
			if err := setOIDCContext(ctx, identity, tokenStr, !cfg.disableLegacyUserLookup); err != nil {
				abortUnauthorized(ctx, "invalid token")
				return
			}
			ctx.Next()
			return
		}

		glog.Errorf(ctx, "[oidcauth] oidc access token validation fail, err:%v", err)
		abortUnauthorized(ctx, "invalid token")
	}
}

// newVerifier 依据选项构造 SDK 校验器（无 KeySource 时返回错误，中间件 fail-closed）。
func newVerifier(cfg *authConfig) (*rp.Verifier, error) {
	if cfg.keySource == nil {
		return nil, errors.New("oidc key source not configured")
	}
	opts := []rp.VerifierOption{}
	if cfg.issuer != "" {
		opts = append(opts, rp.WithIssuer(cfg.issuer))
	}
	if len(cfg.audiences) > 0 {
		opts = append(opts, rp.WithAudiences(cfg.audiences...))
	}
	return rp.NewVerifier(cfg.keySource, opts...), nil
}

// staticKeySource 把"启动时钉死的一把公钥"包装为 KeySource：
// 任意 kid 都返回该公钥（兼容旧签名，代价是无法感知轮换）。
func staticKeySource(get func() *rsa.PublicKey) rp.KeySource {
	return keySourceFunc(func(ctx context.Context, kid string) (*rsa.PublicKey, error) {
		key := get()
		if key == nil {
			return nil, contract.ErrKeySourceUnavailable
		}
		return key, nil
	})
}

// keySourceFunc 把函数适配为 rp.KeySource（SDK 不提供该适配器以保持接口最小）。
type keySourceFunc func(ctx context.Context, kid string) (*rsa.PublicKey, error)

func (f keySourceFunc) PublicKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	return f(ctx, kid)
}

// NewKeySourceFromConfig 依据配置构造 OP 公钥来源，供内置应用（RP）校验 access token。
//
// 取值优先级：
//  1. OIDC.JWKSURL（显式端点）或由 issuer 解析出的 JWKS 端点——
//     issuer 形态走「标准 discovery 的 jwks_uri → {issuer}/keys →
//     {issuer}/.well-known/jwks.json」，是本仓 OP 与外部 OP 都适用的
//     "可轮换"来源，支持未知 kid 限速刷新；生产多进程部署应走这条；
//  2. 本地配置的签名密钥（signingPrivateKeyPath / signingPrivateKeyPEM / oidc.keys）
//     ——导出全部公钥构造进程内 key set。**注意：这是启动时快照，OP 轮换后需要
//     重启本进程**，因此它只作为"没有可用的 JWKS 端点时的兜底"，不是推荐路径。
//
// 两者都没有时返回 error（调用方应 fail-closed，不得退化为不校验）。
func NewKeySourceFromConfig(conf *config.Config) (rp.KeySource, error) {
	if conf != nil && strings.TrimSpace(conf.OIDC.JWKSURL) != "" {
		return rp.NewKeys(context.Background(), strings.TrimSpace(conf.OIDC.JWKSURL))
	}
	if conf != nil && conf.OIDC.Issuer != "" {
		// issuer 形如 {op}/oidc：JWKS 端点由 SDK 解析（discovery 优先，
		// 本仓 OP 的 jwks_uri = {issuer}/keys），并同步预取（失败即 fail-fast，
		// 不静默降级为不校验）。
		return rp.NewKeys(context.Background(), conf.OIDC.Issuer)
	}
	if keys := localPublicKeys(conf); len(keys) > 0 {
		glog.Warnf(context.Background(), "[middleware.NewKeySourceFromConfig] 使用本地签名密钥快照作为 JWKS 来源：OP 轮换签名密钥后本进程需重启才会生效；生产建议配置 oidc.jwksURL")
		return rp.NewKeysFromSet(keys), nil
	}
	return nil, errors.New("oidc key source unavailable: set oidc.jwksURL, oidc.issuer, or local signing key")
}

// ResolveKeySource 返回本应用可用的 OP 公钥来源。
//
// injected 非空时直接使用：**gateway 单体部署**下 auth 的 OP 把自己的已发布公钥
// 作为进程内 key set 注入同进程的其它应用（platformadmin/tenantadmin/rpapi），
// 零网络调用、自动跟随多 key 轮换，也避免"进程启动期 HTTP 请求自己还没监听的
// JWKS 端点"这种必然失败的自举。injected 为 nil 时按本应用配置自建
// （见 NewKeySourceFromConfig），此时才可能发生网络预取。
func ResolveKeySource(injected KeySource, conf *config.Config) (KeySource, error) {
	if injected != nil {
		return injected, nil
	}
	return NewKeySourceFromConfig(conf)
}

// localPublicKeys 从本地配置的签名密钥（单 key 或 keys 列表）导出全部公钥。
func localPublicKeys(conf *config.Config) map[string]*rsa.PublicKey {
	if conf == nil {
		return nil
	}
	out := map[string]*rsa.PublicKey{}
	for i, item := range conf.OIDC.EffectiveSigningKeys() {
		var pemData []byte
		switch {
		case item.PrivateKeyPEM != "":
			pemData = []byte(item.PrivateKeyPEM)
		case item.PrivateKeyPath != "":
			data, err := os.ReadFile(item.PrivateKeyPath)
			if err != nil {
				continue
			}
			pemData = data
		default:
			continue
		}
		key, err := parseRSAPrivateKeyPEM(pemData)
		if err != nil {
			glog.Warnf(context.Background(), "[middleware.localPublicKeys] parse signing key fail, index:%d, err:%v", i, err)
			continue
		}
		kid := item.Kid
		if kid == "" {
			kid = conf.OIDC.SigningKeyID
		}
		if kid == "" {
			// 与 auth 侧一致的 kid 派生（sha256(N)[:16] → base64url）：
			// kid 必须与 OP 实际签发的 kid 完全一致，否则全部请求 401。
			kid = deriveKeyID(&key.PublicKey)
		}
		out[kid] = &key.PublicKey
	}
	return out
}

// deriveKeyID 与 auth 侧 svcoidc.DeriveKeyID 保持同一算法（此处重复实现以避免
// pkg → apps 的反向依赖）。改动其一必须同步另一处。
func deriveKeyID(pub *rsa.PublicKey) string {
	if pub == nil {
		return ""
	}
	sum := sha256.Sum256(pub.N.Bytes())
	return base64.RawURLEncoding.EncodeToString(sum[:16])
}

// LoadSigningPublicKey 保持历史签名：返回 OP 签名公钥的闭包。
//
// Deprecated: 过渡 API——它从**本地**配置的签名密钥导出公钥，属于"启动钉死一把公钥"，
// 感知不到 OP 的密钥轮换。新代码请用 NewKeySourceFromConfig + WithOIDCKeySource。
func LoadSigningPublicKey(conf *config.Config) func() *rsa.PublicKey {
	var publicKey *rsa.PublicKey
	if conf != nil {
		for _, key := range localPublicKeys(conf) {
			publicKey = key
			break
		}
	}
	return func() *rsa.PublicKey { return publicKey }
}

// parseRSAPrivateKeyPEM 宽容解析 RSA 私钥 PEM（PKCS#8 优先、PKCS#1 兜底）。
func parseRSAPrivateKeyPEM(pemData []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, errors.New("invalid RSA private key PEM")
	}
	if block.Type != "RSA PRIVATE KEY" && block.Type != "PRIVATE KEY" {
		return nil, fmt.Errorf("unsupported PEM block type: %s", block.Type)
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		parsed, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse RSA private key: %w", err)
		}
	}
	rsaKey, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("key is not RSA: %T", parsed)
	}
	return rsaKey, nil
}

// setOIDCContext 把校验通过的 identity 注入 gin 上下文：
//   - person token：注入 personID + tenantID + authToken + userID + client_id；
//   - 机器凭证（token_usage=machine）：同样注入 user_id（机器主体），不做租户成员反查。
//
// user_id 自「令牌增加 user_id 声明」起由 OP 直接下发，因此不再需要每次请求查库反查
// tenant_user（既省一次查询，也消除了"缓存层一旦读不到就误判非成员"的降级风险）。
// 对尚未携带 user_id 的存量 token，保留一段反查兼容（allowLegacyLookup）。
func setOIDCContext(ctx *gin.Context, identity *rp.Identity, tokenStr string, allowLegacyLookup bool) error {
	personID := identity.PersonID

	ctx.Set(gcontext.KeyPersonID, personID)
	// 租户作用域：类型化值写入请求上下文（协议层 / 异步任务同样可见），
	// 同时投影 gin Keys 的 tenantID 供既有身份读取点使用。
	// 必须写在下面的"租户内用户反查"之前，该查询依赖租户隔离。
	gincontext.SetTenantScope(ctx, gcontext.CurrentScope(identity.TenantID))
	ctx.Set(gcontext.KeyAuthToken, tokenStr)
	ctx.Set(ContextKeyClientID, identity.ClientID)
	// 完整 identity（含 Scopes）留一份供下游做**策略**判定（如目录 API 要求
	// directory.read）。鉴权（验签/租户）由本中间件完成，策略判定不属于它。
	ctx.Set(ContextKeyOIDCIdentity, identity)

	if identity.UserID != "" {
		ctx.Set(gcontext.KeyUserID, identity.UserID)
		return nil
	}
	// 机器凭证不隶属租户成员，不做反查。
	if identity.IsMachine || personID == "" || !allowLegacyLookup {
		return nil
	}

	// 兼容分支：token 未携带 user_id（存量 token 或旧 OP）。
	userList, err := dao.NewUserDao().GetListByCond(ctx, &dao.UserCond{
		TenantID: identity.TenantID,
		PersonID: personID,
	})
	if err != nil {
		glog.Errorf(ctx, "[oidcauth] resolve tenant user fail, err:%v, tenantID:%s, personID:%s", err, identity.TenantID, personID)
		return fmt.Errorf("resolve tenant user fail: %w", err)
	}
	if len(userList) == 0 {
		glog.Warnf(ctx, "[oidcauth] person has no user in tenant, tenantID:%s, personID:%s", identity.TenantID, personID)
		return fmt.Errorf("person not a member of tenant")
	}
	ctx.Set(gcontext.KeyUserID, userList[0].ID)
	return nil
}

func isSkippedPath(path string, skipPaths []string) bool {
	for _, p := range skipPaths {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

// OIDCIdentityFromContext 取出本中间件写入的完整 identity（未经过 OIDC 鉴权时返回 nil）。
//
// 用途是**策略**判定（scope/客户端维度），不是鉴权：鉴权已由 OIDCAuth 完成，
// 因此调用方拿到 nil 时应按"策略不满足"处理，绝不回退成放行。
func OIDCIdentityFromContext(ctx *gin.Context) *rp.Identity {
	if ctx == nil {
		return nil
	}
	value, ok := ctx.Get(ContextKeyOIDCIdentity)
	if !ok {
		return nil
	}
	identity, ok := value.(*rp.Identity)
	if !ok {
		return nil
	}
	return identity
}

func abortUnauthorized(ctx *gin.Context, msg string) {
	ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": msg})
}

func extractToken(ctx *gin.Context) string {
	auth := ctx.GetHeader(AuthHeaderKey)
	if auth == "" {
		return ""
	}
	if strings.HasPrefix(auth, AuthBearer) {
		return strings.TrimPrefix(auth, AuthBearer)
	}
	return auth
}
