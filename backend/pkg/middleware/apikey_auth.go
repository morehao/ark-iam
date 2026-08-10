package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
)

type ApiKeyAuthMiddleware interface {
	Middleware() gin.HandlerFunc
}

type apiKeyAuthMiddleware struct {
	apiKeyDao *dao.ApiKeyDao
}

func NewApiKeyAuthMiddleware() ApiKeyAuthMiddleware {
	return &apiKeyAuthMiddleware{
		apiKeyDao: dao.NewApiKeyDao(),
	}
}

func newApiKeyAuthMiddlewareWithDao(apiKeyDao *dao.ApiKeyDao) ApiKeyAuthMiddleware {
	return &apiKeyAuthMiddleware{
		apiKeyDao: apiKeyDao,
	}
}

func (m *apiKeyAuthMiddleware) Middleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if ExtractApiKey(ctx) == "" {
			writeApiKeyUnauthorized(ctx, http.StatusUnauthorized, "missing or invalid API key")
			return
		}
		if !m.Authenticate(ctx) {
			return
		}
		ctx.Next()
	}
}

// Authenticate 校验当前请求携带的 API Key（支持 x-api-key 或 Authorization: Bearer），
// 合法则注入租户/用户上下文并返回 true；非法或缺失时不写响应返回 false，交由调用方决定行为。
// 供 OIDC 并行鉴权（任一通过即放行）复用同一套校验逻辑。
func (m *apiKeyAuthMiddleware) Authenticate(ctx *gin.Context) bool {
	rawKey := ExtractApiKey(ctx)
	if rawKey == "" {
		// 缺失 API Key：由调用方决定是否放行/拒绝，此处不写响应
		return false
	}

	keyHash := hashApiKey(rawKey)

	cond := &dao.ApiKeyCond{
		BaseCond:  &gormdao.BaseCond{Page: 1, PageSize: 1},
		KeyHash:   keyHash,
		RevokedAt: &time.Time{},
	}
	list, _, err := m.apiKeyDao.GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[middleware.ApiKeyAuth] GetPageListByCond fail, err:%v", err)
		writeApiKeyUnauthorized(ctx, http.StatusInternalServerError, "internal server error")
		return false
	}
	if len(list) == 0 {
		writeApiKeyUnauthorized(ctx, http.StatusUnauthorized, "invalid API key")
		return false
	}

	entity := list[0]

	if entity.ExpiredAt != nil && entity.ExpiredAt.Before(time.Now()) {
		writeApiKeyUnauthorized(ctx, http.StatusUnauthorized, "API key has expired")
		return false
	}

	if entity.RevokedAt != nil && !entity.RevokedAt.IsZero() {
		writeApiKeyUnauthorized(ctx, http.StatusUnauthorized, "API key has been revoked")
		return false
	}

	ctx.Set(gcontext.KeyTenantID, entity.TenantID)
	ctx.Set(gcontext.KeyUserID, entity.CreatedBy)

	go func() {
		updateCtx := ctx.Copy()
		if err := m.apiKeyDao.UpdateMap(updateCtx, entity.ID, map[string]any{"last_used_at": time.Now()}); err != nil {
			glog.Errorf(updateCtx, "[middleware.ApiKeyAuth] UpdateMap fail, err:%v, id:%d", err, entity.ID)
		}
	}()

	return true
}

// AuthenticateApiKey 便捷封装：基于默认 DAO 校验请求中的 API Key，合法则返回 true。
// 供 oidcauth 等并行鉴权中间件在 OIDC token 校验失败时回退使用。
func AuthenticateApiKey(ctx *gin.Context) bool {
	return (&apiKeyAuthMiddleware{apiKeyDao: dao.NewApiKeyDao()}).Authenticate(ctx)
}

// ExtractApiKey 从 x-api-key 头或 Authorization: Bearer 头解析 API Key。
// 供 oidcauth 并行鉴权判断请求是否携带机器凭证。
func ExtractApiKey(ctx *gin.Context) string {
	if key := ctx.GetHeader("x-api-key"); key != "" {
		return key
	}
	authHeader := ctx.GetHeader("Authorization")
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}
	return ""
}

func writeApiKeyUnauthorized(ctx *gin.Context, status int, message string) {
	ctx.AbortWithStatusJSON(status, gin.H{"code": status, "message": message})
}

// ApiKeyAuth is a convenience function that creates a default middleware instance
// and returns its handler. For use in router registration without DI.
func ApiKeyAuth() gin.HandlerFunc {
	return NewApiKeyAuthMiddleware().Middleware()
}

func hashApiKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}
