package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/identity"
	"github.com/morehao/ark-iam/pkg/oidckit"
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
	keySource oidckit.KeySource
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

// WithOIDCKeySource 注入按 kid 取公钥的密钥来源（真源 sdk/rp.KeySource，装配见 pkg/oidckit）。
// 这是支持 OP 多 key 轮换的正路：未知 kid 时 SDK 会限速刷新 JWKS，
// 而不是把公钥在启动时钉死（轮换后必然 401）。
func WithOIDCKeySource(source oidckit.KeySource) AuthOption {
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

		ident, err := verifier.Verify(ctx, tokenStr)
		if err == nil {
			if cfg.validateOIDCSSO != nil && !cfg.validateOIDCSSO(ctx, ident.PersonID, ident.IsMachine) {
				glog.Warnf(ctx, "[oidcauth] sso session revoked, personID:%s", ident.PersonID)
				abortUnauthorized(ctx, "session expired")
				return
			}
			if err := setOIDCContext(ctx, ident, tokenStr, !cfg.disableLegacyUserLookup); err != nil {
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

// setOIDCContext 把校验通过的 identity 注入 gin 上下文：
//   - person token：注入 personID + tenantID + authToken + userID + client_id；
//   - 机器凭证（token_usage=machine）：同样注入 user_id（机器主体），不做租户成员反查。
//
// user_id 自「令牌增加 user_id 声明」起由 OP 直接下发，因此不再需要每次请求查库反查
// tenant_user（既省一次查询，也消除了"缓存层一旦读不到就误判非成员"的降级风险）。
// 对尚未携带 user_id 的存量 token，保留一段反查兼容（allowLegacyLookup）。
func setOIDCContext(ctx *gin.Context, ident *rp.Identity, tokenStr string, allowLegacyLookup bool) error {
	personID := ident.PersonID

	ctx.Set(gcontext.KeyPersonID, personID)
	// 租户作用域：类型化值写入请求上下文（协议层 / 异步任务同样可见），
	// 同时投影 gin Keys 的 tenantID 供既有身份读取点使用。
	// 必须写在下面的"租户内用户反查"之前，该查询依赖租户隔离。
	gincontext.SetTenantScope(ctx, gcontext.CurrentScope(ident.TenantID))
	ctx.Set(gcontext.KeyAuthToken, tokenStr)
	// client_id 与完整 identity（含 Scopes）交由 pkg/identity 的写侧落盘：
	// 读取方（业务/策略层）只依赖该契约包，不必依赖本中间件包。
	identity.SetClientID(ctx, ident.ClientID)
	identity.SetOIDCIdentity(ctx, ident)

	if ident.UserID != "" {
		ctx.Set(gcontext.KeyUserID, ident.UserID)
		return nil
	}
	// 机器凭证不隶属租户成员，不做反查。
	if ident.IsMachine || personID == "" || !allowLegacyLookup {
		return nil
	}

	// 兼容分支：token 未携带 user_id（存量 token 或旧 OP）。
	userList, err := dao.NewUserDao().GetListByCond(ctx, &dao.UserCond{
		TenantID: ident.TenantID,
		PersonID: personID,
	})
	if err != nil {
		glog.Errorf(ctx, "[oidcauth] resolve tenant user fail, err:%v, tenantID:%s, personID:%s", err, ident.TenantID, personID)
		return fmt.Errorf("resolve tenant user fail: %w", err)
	}
	if len(userList) == 0 {
		glog.Warnf(ctx, "[oidcauth] person has no user in tenant, tenantID:%s, personID:%s", ident.TenantID, personID)
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
