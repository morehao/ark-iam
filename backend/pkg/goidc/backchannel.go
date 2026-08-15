// Package goidc 提供跨应用共享的 OIDC 能力。
//
// 当前内容为 OIDC Back-Channel Logout（OIDC Back-Channel Logout 1.0）的
// RP 侧接收端：供各业务应用（RP）挂载 back-channel logout 接收端点，
// 接收由 auth（OP）在用户登出后推送的 logout_token，并执行本地会话清除。
// 与 auth/internal/oidcop（OP 侧领域层，含登出登记与 SLO 队列）对应，本包是"接收端"。
// OP 侧领域层代码位于 auth 应用内部，待出现第二个 OP 消费者时再上提至本包。
package goidc

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const (
	// BackChannelLogoutEventURI 是 logout_token 中 events 声明的标准事件 URI。
	BackChannelLogoutEventURI = "http://schemas.openid.net/event/backchannel-logout"
)

// LogoutTokenClaims 是 OIDC Back-Channel Logout logout_token 的标准声明。
//
// 除 JWT 注册声明外，logout_token 必须包含：
//   - events: 包含 BackChannelLogoutEventURI 键
//   - sid:    会话 ID（可选，取决于 OP 是否支持 session 粒度）
//   - sub:    用户标识
//   - aud:    客户端 ID（必须是接收方 RP）
//   - jti:    唯一标识（防重放）
type LogoutTokenClaims struct {
	jwt.RegisteredClaims
	SessionID string         `json:"sid,omitempty"`
	Events    map[string]any `json:"events,omitempty"`
}

// HasBackChannelLogoutEvent 报告 logout_token 是否声明了 back-channel logout 事件。
func (c *LogoutTokenClaims) HasBackChannelLogoutEvent() bool {
	if c == nil || c.Events == nil {
		return false
	}
	_, ok := c.Events[BackChannelLogoutEventURI]
	return ok
}

// ParseLogoutToken 解析并校验 logout_token。
//
// 校验项（对齐 OIDC Back-Channel Logout 1.0 §2.1.2 RP 处理要求）：
//   - 签名：RS256，使用 OP 公钥验签
//   - iss：必须等于 OP issuer
//   - aud：必须包含本 RP 的 client_id
//   - exp：未过期
//   - events：必须包含 backchannel-logout 事件
//   - jti：必须存在（防重放基础；具体去重由调用方基于 jti 完成）
func ParseLogoutToken(tokenStr string, publicKey *rsa.PublicKey, issuer, clientID string) (*LogoutTokenClaims, error) {
	if publicKey == nil {
		return nil, errors.New("oidc logout: public key not initialized")
	}
	if tokenStr == "" {
		return nil, errors.New("oidc logout: empty logout_token")
	}
	claims := &LogoutTokenClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("oidc logout: unexpected signing method: %v", token.Header["alg"])
		}
		return publicKey, nil
	},
		jwt.WithLeeway(0),
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuer(issuer),
		jwt.WithAudience(clientID),
	)
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("oidc logout: invalid logout_token")
	}
	if !claims.HasBackChannelLogoutEvent() {
		return nil, errors.New("oidc logout: missing backchannel-logout event")
	}
	if claims.ID == "" {
		return nil, errors.New("oidc logout: missing jti")
	}
	if claims.Subject == "" {
		return nil, errors.New("oidc logout: missing sub")
	}
	return claims, nil
}

// SessionRevoker 是收到合法 logout_token 后执行本地会话清除的回调。
// 返回 error 时接收端按 500 处理（OP 会重试）；nil 表示已成功处理。
type SessionRevoker func(ctx *gin.Context, claims *LogoutTokenClaims) error

// BackChannelLogoutHandler 是 back-channel logout 接收端的 Gin 处理器。
type BackChannelLogoutHandler struct {
	GetPublicKey func() *rsa.PublicKey
	Issuer       string
	ClientID     string
	OnLogout     SessionRevoker

	// 最近接收的 logout_token 记录（内存，供调试/可观测/e2e 断言）。
	recentMu sync.Mutex
	recent   []RecentLogoutToken
}

// RecentLogoutToken 记录一次接收到的 logout_token 摘要。
type RecentLogoutToken struct {
	JTI      string    `json:"jti"`
	Sub      string    `json:"sub"`
	SID      string    `json:"sid"`
	ClientID string    `json:"clientID"`
	Received time.Time `json:"received"`
	Valid    bool      `json:"valid"`
	ParseErr string    `json:"parseErr,omitempty"`
}

const maxRecentTokens = 64

// NewBackChannelLogoutHandler 构造接收端处理器。
func NewBackChannelLogoutHandler(getPublicKey func() *rsa.PublicKey, issuer, clientID string, onLogout SessionRevoker) *BackChannelLogoutHandler {
	return &BackChannelLogoutHandler{
		GetPublicKey: getPublicKey,
		Issuer:       issuer,
		ClientID:     clientID,
		OnLogout:     onLogout,
		recent:       make([]RecentLogoutToken, 0, maxRecentTokens),
	}
}

// Handler 返回 Gin HandlerFunc：POST back_channel_logout_uri?logout_token=...
//
// 处理语义（对齐 OIDC Back-Channel Logout 1.0 §2.2）：
//   - 成功处理 → 200
//   - token 无效（签名/声明校验失败）→ 400（OP 不重试无效 token，仅记日志）
//   - 本地登出回调失败 → 500（OP 将按重试策略重发）
func (h *BackChannelLogoutHandler) Handler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		tokenStr := ctx.PostForm("logout_token")
		claims, err := ParseLogoutToken(tokenStr, h.GetPublicKey(), h.Issuer, h.ClientID)
		h.record(RecentLogoutToken{
			JTI:      claimsJTI(claims),
			Sub:      claimsSub(claims),
			SID:      claimsSID(claims),
			ClientID: h.ClientID,
			Received: time.Now(),
			Valid:    err == nil,
			ParseErr: errString(err),
		})
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "invalid logout_token: " + err.Error()})
			return
		}
		if h.OnLogout != nil {
			if lErr := h.OnLogout(ctx, claims); lErr != nil {
				ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "logout processing failed"})
				return
			}
		}
		ctx.Status(http.StatusOK)
	}
}

// Recent 返回最近接收的 logout_token 记录（线程安全副本）。
// 主要用于调试与 e2e 断言：验证 OP 确实向本 RP 推送了合法 logout_token。
func (h *BackChannelLogoutHandler) Recent() []RecentLogoutToken {
	h.recentMu.Lock()
	defer h.recentMu.Unlock()
	out := make([]RecentLogoutToken, len(h.recent))
	copy(out, h.recent)
	return out
}

func (h *BackChannelLogoutHandler) record(t RecentLogoutToken) {
	h.recentMu.Lock()
	defer h.recentMu.Unlock()
	h.recent = append(h.recent, t)
	if len(h.recent) > maxRecentTokens {
		h.recent = h.recent[len(h.recent)-maxRecentTokens:]
	}
}

func claimsJTI(claims *LogoutTokenClaims) string {
	if claims == nil {
		return ""
	}
	return claims.ID
}

func claimsSub(claims *LogoutTokenClaims) string {
	if claims == nil {
		return ""
	}
	return claims.Subject
}

func claimsSID(claims *LogoutTokenClaims) string {
	if claims == nil {
		return ""
	}
	return claims.SessionID
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// ContextKey 用于在 context 中传递注销令牌，供业务应用在 OnLogout 中读取（可选）。
type ContextKey string

// ContextKeyLogoutTokenClaims 注入到请求 context 的 logout_token claims。
const ContextKeyLogoutTokenClaims ContextKey = "iam.logoutTokenClaims"

// SetClaimsOnContext 将解析后的 claims 放入 gin context，便于业务应用在中间件/后续处理器读取。
func SetClaimsOnContext(ctx *gin.Context, claims *LogoutTokenClaims) {
	ctx.Set(string(ContextKeyLogoutTokenClaims), claims)
}

// Receiver 是挂载接收端所需的完整配置与处理器集合。
type Receiver struct {
	Handler *BackChannelLogoutHandler
}

// RegisterReceiverRoutes 在指定 router group 下挂载 back-channel logout 接收端点。
//
// 路由：
//   - POST {basePath}            接收 logout_token（标准 Back-Channel Logout 端点）
//   - GET  {basePath}/recent     最近接收记录（调试/e2e 断言用，dev 环境建议仅本地暴露）
//
// basePath 默认 "/oidc/bc-logout"；调用方传入 group 时请确保路径不会与其它应用冲突
// （gateway 聚合部署时各 app 应使用独立 basePath，如 /oidc/bc-logout/platform）。
func RegisterReceiverRoutes(group *gin.RouterGroup, basePath string, getPublicKey func() *rsa.PublicKey, issuer, clientID string, onLogout SessionRevoker) *BackChannelLogoutHandler {
	if basePath == "" {
		basePath = "/oidc/bc-logout"
	}
	h := NewBackChannelLogoutHandler(getPublicKey, issuer, clientID, onLogout)
	group.POST(basePath, h.Handler())
	group.GET(basePath+"/recent", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"recent": h.Recent()})
	})
	return h
}
