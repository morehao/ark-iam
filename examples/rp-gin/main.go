// Package main 演示外部项目如何用 ark-iam/sdk 在 Gin 中接入 RP 侧本地验签。
//
// 依赖：github.com/morehao/ark-iam/sdk（框架无关，不共享 IAM 的库/Redis）。
// 运行时只需在进程启动时构建一次 Keys + Verifier，之后每个请求本地验签，
// 业务请求路径上绝不访问 OP。
//
// 启动前设置环境变量：
//
//	export ARK_IAM_ISSUER="http://localhost:8080"   # OP 的 issuer（如内嵌 IAM）
//	export ARK_IAM_AUDIENCE="my-app"                # 本应用的 client_id
//	go run .
//
// 请求携带 Authorization: Bearer <access_token> 即可通过。
package main

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/morehao/ark-iam/sdk/rp"
)

// newVerifier 构建一次性的验证器：从 issuer 拉取 JWKS 并组装 Verifier。
// 真实项目里应在 main 启动时调用一次并复用（Keys 内部做 JWKS 预取与限频刷新）。
func newVerifier(ctx context.Context, issuer, audience string) (*rp.Verifier, error) {
	keys, err := rp.NewKeys(ctx, issuer)
	if err != nil {
		return nil, err
	}
	return rp.NewVerifier(keys,
		rp.WithIssuer(issuer),
		rp.WithAudiences(audience),
		rp.WithLeeway(30*time.Second),
	), nil
}

// authMiddleware 是 20 行的 Gin 鉴权中间件：本地验签 → 注入 Identity → 放行。
// fail-closed：任何验签失败都落到 401，绝不静默放行。
func authMiddleware(v *rp.Verifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := strings.TrimSpace(strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer "))
		if raw == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": "MISSING_TOKEN"})
			return
		}
		identity, err := v.Verify(c.Request.Context(), raw)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": "INVALID_TOKEN"})
			return
		}
		// 把已验证身份放进 request context，后续 handler 用 rp.IdentityFrom 取回。
		c.Request = c.Request.WithContext(rp.WithIdentity(c.Request.Context(), identity))
		c.Next()
	}
}

func main() {
	ctx := context.Background()
	issuer := os.Getenv("ARK_IAM_ISSUER")
	audience := os.Getenv("ARK_IAM_AUDIENCE")
	if issuer == "" || audience == "" {
		panic("请设置 ARK_IAM_ISSUER 与 ARK_IAM_AUDIENCE")
	}

	verifier, err := newVerifier(ctx, issuer, audience)
	if err != nil {
		panic(err)
	}

	r := gin.Default()
	api := r.Group("/api", authMiddleware(verifier))
	api.GET("/whoami", func(c *gin.Context) {
		id := rp.IdentityFrom(c.Request.Context())
		c.JSON(http.StatusOK, gin.H{
			"user_id":   id.UserID,   // 审计主键
			"tenant_id": id.TenantID, // 租户作用域
			"client_id": id.ClientID,
			"machine":   id.IsMachine,
			"scopes":    id.Scopes,
		})
	})

	_ = r.Run(":8098")
}
