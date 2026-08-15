package oidcauth

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/morehao/ark-iam/pkg/iam/object/objauth"
	"github.com/morehao/ark-iam/pkg/middleware"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/glog"
)

const (
	AuthHeaderKey = "Authorization"
	AuthBearer    = "Bearer "
)

type authConfig struct {
	skipPaths       []string
	validateOIDCSSO func(ctx *gin.Context, personID uint, isMachineToken bool) bool
	// issuer 为 OP 的 issuer，配置后强制校验 access token 的 iss（H3）。
	issuer string
	// audiences 为本应用（RP）认可的 aud 集合，配置后强制校验 access token 的 aud，
	// 防止同一 OP 下其它 client 的 token 串用本应用接口。
	audiences []string
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

// WithOIDCSSOValidation 注入 OIDC 访问令牌的 SSO 会话校验器。
// 校验 OIDC 令牌有效后，如果该校验器返回 false（该自然人不再有有效的 SSO 会话，
// 例如已在其他应用全局登出），则本次请求按未认证处理，返回 401。
// isMachineToken 标识该令牌是否为机器凭证（client_credentials/API Key）签发，
// 机器凭证不依赖浏览器 SSO 会话活性，校验器可据此直接放行。
func WithOIDCSSOValidation(validate func(ctx *gin.Context, personID uint, isMachineToken bool) bool) AuthOption {
	return func(c *authConfig) {
		c.validateOIDCSSO = validate
	}
}

func OIDCCompatibleAuth(getOIDCPublicKey func() *rsa.PublicKey, opts ...AuthOption) gin.HandlerFunc {
	cfg := &authConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	return func(ctx *gin.Context) {
		if isSkippedPath(ctx.Request.URL.Path, cfg.skipPaths) {
			ctx.Next()
			return
		}

		// 并行鉴权：若请求显式携带 x-api-key，则任一通过即可（OIDC 或 API Key）。
		// 机器凭证（API Key）不依赖浏览器 SSO 会话活性，见设计文档 §4.4。
		// 仅以 x-api-key 头作为机器凭证通道，避免与 Authorization: Bearer 的 OIDC 通道冲突。
		if ctx.GetHeader("x-api-key") != "" {
			if middleware.AuthenticateApiKey(ctx) {
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

		oidcPublicKey := getOIDCPublicKey()
		claims, err := validateOIDCAccessToken(tokenStr, oidcPublicKey, cfg.issuer, cfg.audiences)
		if err == nil {
			isMachine := claims.IsMachine()
			personID := claims.PersonID()
			if cfg.validateOIDCSSO != nil && !cfg.validateOIDCSSO(ctx, personID, isMachine) {
				glog.Warnf(ctx, "[oidcauth] sso session revoked, personID:%d", personID)
				abortUnauthorized(ctx, "session expired")
				return
			}
			setOIDCContext(ctx, claims, tokenStr)
			ctx.Next()
			return
		}

		glog.Errorf(ctx, "[oidcauth] oidc access token validation fail, err:%v", err)
		abortUnauthorized(ctx, "invalid token")
	}
}

// validateOIDCAccessToken 校验 OIDC access token 的签名与核心声明（H3）：
//   - 算法必须为 RS256（拒绝 HS256 等对称算法混淆）；
//   - 配置了 issuer 时，iss 必须精确匹配 OP issuer；
//   - 配置了 audiences 时，aud 必须包含本应用 client_id（防跨 client 串用）。
func validateOIDCAccessToken(tokenStr string, publicKey *rsa.PublicKey, issuer string, audiences []string) (*objauth.TokenClaims, error) {
	if publicKey == nil {
		return nil, errors.New("oidc public key not initialized")
	}
	claims := &objauth.TokenClaims{}
	parserOpts := []jwt.ParserOption{
		jwt.WithLeeway(0),
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}),
	}
	if issuer != "" {
		parserOpts = append(parserOpts, jwt.WithIssuer(issuer))
	}
	if len(audiences) > 0 {
		parserOpts = append(parserOpts, jwt.WithAudience(audiences...))
	}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return publicKey, nil
	}, parserOpts...)
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid oidc token")
	}
	// 人 token 必须携带 person sub 与 tenant_id；机器凭证必须携带可知的 token_usage。
	// 既非 person 也非 machine 的 token（如仅 client_id 的 client_credentials）拒绝。
	if claims.HasPerson() {
		if claims.TenantID == 0 {
			return nil, errors.New("missing required oidc claim: tenant_id")
		}
		return claims, nil
	}
	if !claims.IsMachine() {
		return nil, errors.New("missing required oidc claims")
	}
	return claims, nil
}

func setOIDCContext(ctx *gin.Context, claims *objauth.TokenClaims, tokenStr string) {
	personID := claims.PersonID()
	tenantID := uint(claims.TenantID)

	ctx.Set(gcontext.KeyPersonID, personID)
	ctx.Set(gcontext.KeyTenantID, tenantID)
	ctx.Set(gcontext.KeyAuthToken, tokenStr)
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
