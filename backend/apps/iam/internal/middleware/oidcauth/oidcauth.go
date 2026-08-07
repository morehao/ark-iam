package oidcauth

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/biz/gobject"
	"github.com/morehao/golib/gauth/jwtauth"
	"github.com/morehao/golib/glog"
)

const (
	AuthHeaderKey = "Authorization"
	AuthBearer    = "Bearer "
)

type authConfig struct {
	skipPaths      []string
	validateOIDCSSO func(ctx *gin.Context, personID uint) bool
}

type AuthOption func(*authConfig)

func WithAuthSkipPaths(paths ...string) AuthOption {
	return func(c *authConfig) {
		c.skipPaths = append(c.skipPaths, paths...)
	}
}

// WithOIDCSSOValidation 注入 OIDC 访问令牌的 SSO 会话校验器。
// 校验 OIDC 令牌有效后，如果该校验器返回 false（该自然人不再有有效的 SSO 会话，
// 例如已在其他应用全局登出），则本次请求按未认证处理，返回 401。
func WithOIDCSSOValidation(validate func(ctx *gin.Context, personID uint) bool) AuthOption {
	return func(c *authConfig) {
		c.validateOIDCSSO = validate
	}
}

func OIDCCompatibleAuth(secretKey string, getOIDCPublicKey func() *rsa.PublicKey, opts ...AuthOption) gin.HandlerFunc {
	cfg := &authConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	return func(ctx *gin.Context) {
		if isSkippedPath(ctx.Request.URL.Path, cfg.skipPaths) {
			ctx.Next()
			return
		}

		tokenStr := extractToken(ctx)
		if tokenStr == "" {
			glog.Errorf(ctx, "[oidcauth] missing auth token")
			abortUnauthorized(ctx, "missing auth token")
			return
		}

		oidcPublicKey := getOIDCPublicKey()
		claims, err := validateOIDCAccessToken(tokenStr, oidcPublicKey)
		if err == nil {
			personID := parsePersonIDFromSub(claims["sub"].(string))
			if cfg.validateOIDCSSO != nil && !cfg.validateOIDCSSO(ctx, personID) {
				glog.Warnf(ctx, "[oidcauth] sso session revoked, personID:%d", personID)
				abortUnauthorized(ctx, "session expired")
				return
			}
			setOIDCContext(ctx, claims, tokenStr)
			ctx.Next()
			return
		}

		auth, err := jwtauth.New[gobject.UserClaims](secretKey)
		if err != nil {
			glog.Errorf(ctx, "[oidcauth] jwtauth.New fail, err:%v", err)
			abortUnauthorized(ctx, "internal server error")
			return
		}

		internalClaims, err := auth.Parse(tokenStr)
		if err != nil {
			glog.Errorf(ctx, "[oidcauth] internal JWT parse fail, err:%v", err)
			abortUnauthorized(ctx, "invalid token")
			return
		}

		ctx.Set(gcontext.KeyOrgID, internalClaims.CustomData.OrgID)
		ctx.Set(gcontext.KeyTenantID, internalClaims.CustomData.TenantID)
		ctx.Set(gcontext.KeyPersonID, internalClaims.CustomData.PersonID)
		ctx.Set(gcontext.KeyUserID, internalClaims.CustomData.UserID)
		ctx.Set(gcontext.KeyDeptID, internalClaims.CustomData.DeptID)
		ctx.Set(gcontext.KeyUserType, internalClaims.CustomData.UserType)
		ctx.Set(gcontext.KeyAuthToken, tokenStr)

		ctx.Next()
	}
}

func validateOIDCAccessToken(tokenStr string, publicKey *rsa.PublicKey) (jwt.MapClaims, error) {
	if publicKey == nil {
		return nil, errors.New("oidc public key not initialized")
	}
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return publicKey, nil
	}, jwt.WithLeeway(0))
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid oidc token")
	}
	_, subOK := claims["sub"].(string)
	_, tenantOK := claims["tenant_id"].(float64)
	if !subOK || !tenantOK {
		return nil, errors.New("missing required oidc claims")
	}
	return claims, nil
}

func setOIDCContext(ctx *gin.Context, claims jwt.MapClaims, tokenStr string) {
	sub, _ := claims["sub"].(string)
	personID := parsePersonIDFromSub(sub)

	tenantIDFloat, _ := claims["tenant_id"].(float64)
	tenantID := uint(tenantIDFloat)

	ctx.Set(gcontext.KeyPersonID, personID)
	ctx.Set(gcontext.KeyTenantID, tenantID)
	ctx.Set(gcontext.KeyAuthToken, tokenStr)
}

func parsePersonIDFromSub(sub string) uint {
	const prefix = "person:"
	if !strings.HasPrefix(sub, prefix) {
		return 0
	}
	raw := strings.TrimPrefix(sub, prefix)
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0
	}
	return uint(id)
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
