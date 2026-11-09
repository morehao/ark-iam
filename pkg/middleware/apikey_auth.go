package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/credential"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
)

type ApiKeyAuthMiddleware interface {
	Middleware() gin.HandlerFunc
}

type apiKeyAuthMiddleware struct {
	apiKeyDao *dao.ApiKeyDao
	userDao   *dao.UserDao
	tenantDao *dao.TenantDao
}

// defaultApiKeyAuth 是包级默认实例，供 oidc_auth 的并行鉴权复用。
// DAO 无请求态（DB handle 由 dbclient 在调用时解析），因此可安全跨请求复用，
// 省掉每次回退鉴权都新建 3 个 DAO 的分配。
var defaultApiKeyAuth = newDefaultApiKeyAuth()

func newDefaultApiKeyAuth() *apiKeyAuthMiddleware {
	return &apiKeyAuthMiddleware{
		apiKeyDao: dao.NewApiKeyDao(),
		userDao:   dao.NewUserDao(),
		tenantDao: dao.NewTenantDao(),
	}
}

func NewApiKeyAuthMiddleware() ApiKeyAuthMiddleware {
	return newDefaultApiKeyAuth()
}

func newApiKeyAuthMiddlewareWithDao(apiKeyDao *dao.ApiKeyDao, userDao *dao.UserDao, tenantDao *dao.TenantDao) ApiKeyAuthMiddleware {
	return &apiKeyAuthMiddleware{
		apiKeyDao: apiKeyDao,
		userDao:   userDao,
		tenantDao: tenantDao,
	}
}

func (m *apiKeyAuthMiddleware) Middleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if extractApiKey(ctx) == "" {
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
// 合法则注入租户/用户上下文并返回 true。
//
// 返回 false 时**可能已经写入了 401/500 响应**（缺失凭证除外：那一种不写响应，
// 由调用方决定如何处理）。因此调用方在 false 后必须直接 return，不得再写响应体。
// 供 OIDC 并行鉴权（任一通过即放行）复用同一套校验逻辑。
func (m *apiKeyAuthMiddleware) Authenticate(ctx *gin.Context) bool {
	rawKey := extractApiKey(ctx)
	if rawKey == "" {
		// 缺失 API Key：属"本通道不适用"，不写响应，由调用方决定是否放行/拒绝
		return false
	}

	keyHash := credential.HashSecret(rawKey)

	cond := &dao.ApiKeyCond{
		BaseCond:  &gormdao.BaseCond{Page: 1, PageSize: 1},
		KeyHash:   keyHash,
		RevokedAt: &time.Time{},
	}
	// 按密钥摘要反查归属租户：此刻租户未知，必须**显式**声明"全租户"作用域。
	// 禁止靠"ctx 没有租户作用域"来获得跨租户可见性——那会被 fail-closed 拒绝。
	list, _, err := m.apiKeyDao.GetPageListByCond(dbclient.CrossTenantContext(ctx), cond)
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

	// 租户门禁：密钥从属于某个租户，租户非 active（已挂起）或行已不存在时，其名下全部机器凭证立即失效。
	// 挂起只撤销了成员 refresh token 与 SSO 会话，而 API Key 没有可依赖的 TTL，因此必须在每次请求上校验；
	// 该成本是本请求内的一次主键查询。
	if entity.TenantID == "" {
		writeApiKeyUnauthorized(ctx, http.StatusUnauthorized, "API key tenant missing")
		return false
	}
	tenantEntity, err := m.tenantDao.GetByID(ctx, entity.TenantID)
	if err != nil {
		glog.Errorf(ctx, "[middleware.ApiKeyAuth] tenant GetByID fail, err:%v, tenantID:%s", err, entity.TenantID)
		writeApiKeyUnauthorized(ctx, http.StatusInternalServerError, "internal server error")
		return false
	}
	if tenantEntity == nil || !tenantEntity.IsActive() {
		glog.Warnf(ctx, "[middleware.ApiKeyAuth] reject API key for inactive tenant, tenantID:%s, keyID:%s", entity.TenantID, entity.ID)
		writeApiKeyUnauthorized(ctx, http.StatusUnauthorized, "tenant is not active")
		return false
	}

	// 归属主体解析：密钥代表的是其归属用户（真实用户本人或服务账号），而非创建人。
	// 此处租户已由密钥行确定但尚未写入 ctx，因此用"指定租户"作用域查询归属用户。
	if entity.OwnerUserID == "" {
		writeApiKeyUnauthorized(ctx, http.StatusUnauthorized, "API key owner missing")
		return false
	}
	ownerCtx := dbclient.ExplicitTenantContext(ctx, entity.TenantID)
	owner, err := m.userDao.GetByID(ownerCtx, entity.OwnerUserID)
	if err != nil {
		glog.Errorf(ctx, "[middleware.ApiKeyAuth] owner GetByID fail, err:%v, ownerID:%s", err, entity.OwnerUserID)
		writeApiKeyUnauthorized(ctx, http.StatusInternalServerError, "internal server error")
		return false
	}
	if owner == nil {
		writeApiKeyUnauthorized(ctx, http.StatusUnauthorized, "API key owner not found")
		return false
	}
	if owner.TenantID != entity.TenantID {
		glog.Errorf(ctx, "[middleware.ApiKeyAuth] owner tenant mismatch, ownerID:%s, keyID:%s", owner.ID, entity.ID)
		writeApiKeyUnauthorized(ctx, http.StatusUnauthorized, "API key owner mismatch")
		return false
	}
	if owner.Status == model.UserStatusSuspended {
		writeApiKeyUnauthorized(ctx, http.StatusUnauthorized, "API key owner suspended")
		return false
	}

	// 写入租户作用域（类型化值 + gin Keys 投影），此后本请求的数据访问默认按该租户隔离。
	gincontext.SetTenantScope(ctx, gcontext.CurrentScope(entity.TenantID))
	ctx.Set(gcontext.KeyUserID, owner.ID)

	return true
}

// authenticateApiKey 便捷封装：基于默认 DAO 实例校验请求中的 API Key，合法则返回 true。
// 供 oidcauth 等并行鉴权中间件在 OIDC token 校验失败时回退使用。
func authenticateApiKey(ctx *gin.Context) bool {
	return defaultApiKeyAuth.Authenticate(ctx)
}

// extractApiKey 从 x-api-key 头或 Authorization: Bearer 头解析 API Key。
// 供 oidcauth 并行鉴权判断请求是否携带机器凭证。
func extractApiKey(ctx *gin.Context) string {
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
