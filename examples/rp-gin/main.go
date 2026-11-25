// Package main 是外部项目接入 ark-iam 的**正式 RP 范本**：
// 演示如何仅用框架无关的 sdk/rp，在 Gin 中完成 RP 侧本地验签并把已验证身份交给业务。
//
// # 职责边界（重要）
//
//   - sdk/rp      = 验签引擎（框架无关）。提供 NewKeys/NewVerifier/Verify 与
//     WithIdentity/IdentityFrom 标准身份通道；不依赖 gin/gorm/redis。
//   - gin 中间件  = 消费方自写（本文件 authMiddleware，约 20 行）。取 Bearer → Verify →
//     rp.WithIdentity 注入 request context → Next；失败一律 401（fail-closed）。
//   - server 装配 = ark-iam 内部（pkg/appassembly），不再作为对外 SDK 发布。
//
// # 业务映射由消费方自加
//
// sdk/rp 只负责“令牌是否有效、其中的身份声明是什么”，返回的 rp.Identity 里是
// 令牌侧口径（string 类型的 PersonID/UserID/TenantID 等）。把令牌身份映射到**本应用
// 自己的本地数据**（例如字符串主体 → 本地 BIGINT 用户/租户主键、查本地用户表、
// 装载本应用的角色/权限）是**消费方自己的职责**，SDK 不替你做，也不感知你的库。
// 见下方 localUserResolver 与 whoami handler 的示范。
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
		id, err := v.Verify(c.Request.Context(), raw)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": "INVALID_TOKEN"})
			return
		}
		// 把已验证身份放进 request context，后续 handler 用 rp.IdentityFrom 取回。
		// 注意：此处只做“验签 + 注入令牌身份”，不查你的本地库。
		c.Request = c.Request.WithContext(rp.WithIdentity(c.Request.Context(), id))
		c.Next()
	}
}

// localUser 是消费方自己的本地用户视图（示例）。
// 真实项目里它通常来自你自己的用户表 / 缓存，主键是你的本地 ID。
type localUser struct {
	LocalID  int64 // 本应用本地主键（与 rp.Identity 的 string 主体解耦）
	Account  string
	TenantID int64
}

// localUserResolver 示范“业务映射由消费方自加”：
// 把 rp.Identity 里的令牌侧口径（string 的 PersonID/UserID/TenantID）映射成本应用
// 的本地数据（本地 BIGINT 主键、本地账号、本应用角色/权限等）。
//
// 这一步 SDK 不做、也不该做——它不知道你的库结构，也不知道你的主键类型。
// 你的中间件或 handler 在这里做一次查库/查缓存即可（可放进中间件里做并写 gcontext，
// 也可像本例这样在 handler 里按需做）。
func localUserResolver(ctx context.Context, id *rp.Identity) (*localUser, bool) {
	// 示例：仅以令牌里的 UserID（机器主体/租户成员主键，string）为例做演示。
	// 真实实现：SELECT ... FROM your_user WHERE subject = id.UserID
	_ = ctx
	if id == nil || id.UserID == "" {
		return nil, false
	}
	return &localUser{LocalID: 42, Account: "demo-user", TenantID: 1}, true
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
		// 令牌侧身份（sdk/rp 提供，全部是令牌声明口径，string）。
		id := rp.IdentityFrom(c.Request.Context())
		resp := gin.H{
			"subject_user_id": id.UserID,   // 令牌主体
			"subject_tenant":  id.TenantID, // 令牌租户作用域
			"client_id":       id.ClientID,
			"machine":         id.IsMachine,
			"scopes":          id.Scopes,
		}
		// 业务映射（消费方自加）：把令牌主体映射到本应用本地用户/主键/权限。
		if u, ok := localUserResolver(c.Request.Context(), id); ok {
			resp["local_user_id"] = u.LocalID // 本应用 BIGINT 主键（审计列写这个）
			resp["local_account"] = u.Account
		}
		c.JSON(http.StatusOK, resp)
	})

	_ = r.Run(":8098")
}
