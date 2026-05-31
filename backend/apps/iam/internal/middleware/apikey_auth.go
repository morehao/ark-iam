package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/dao"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/biz/genericdao"
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
		authHeader := ctx.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "missing or invalid Authorization header",
			})
			return
		}

		rawKey := strings.TrimPrefix(authHeader, "Bearer ")
		if rawKey == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "empty API key",
			})
			return
		}

		keyHash := hashApiKey(rawKey)

		cond := &dao.ApiKeyCond{
			BaseCond:  &genericdao.BaseCond{Page: 1, PageSize: 1},
			KeyHash:    keyHash,
			RevokedAt:  &time.Time{},
		}
		list, _, err := m.apiKeyDao.GetPageListByCond(ctx, cond)
		if err != nil {
			glog.Errorf(ctx, "[middleware.ApiKeyAuth] GetPageListByCond fail, err:%v", err)
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "internal server error",
			})
			return
		}
		if len(list) == 0 {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "invalid API key",
			})
			return
		}

		entity := list[0]

		if entity.ExpiredAt != nil && entity.ExpiredAt.Before(time.Now()) {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "API key has expired",
			})
			return
		}

		if entity.RevokedAt != nil && !entity.RevokedAt.IsZero() {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "API key has been revoked",
			})
			return
		}

		ctx.Set(gcontext.KeyTenantID, entity.TenantID)
		ctx.Set(gcontext.KeyUserID, entity.CreatedBy)

		go func() {
			updateCtx := ctx.Copy()
			if err := m.apiKeyDao.UpdateMap(updateCtx, entity.ID, map[string]any{"last_used_at": time.Now()}); err != nil {
				glog.Errorf(updateCtx, "[middleware.ApiKeyAuth] UpdateMap fail, err:%v, id:%d", err, entity.ID)
			}
		}()

		ctx.Next()
	}
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