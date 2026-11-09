// Package oidckit 封装本项目内置应用（gin 侧）所需的 OIDC 能力。
//
// 职责有二：
//   - OP 公钥来源装配：把 oidc.* 配置解析为 sdk/rp 的验签 KeySource（见 keys.go）；
//   - OIDC Back-Channel Logout（OIDC Back-Channel Logout 1.0）的 RP 侧接收端：
//     供各业务应用（RP）挂载 back-channel logout 端点，接收由 auth（OP）在用户
//     登出后推送的 logout_token，并执行本地会话清除。
//
// 边界：RP 侧 OIDC 能力的**唯一事实源**是独立 module `backend/sdk`
// （sdk/contract 契约 + sdk/rp 验签/M2M/登出）。本包只做"配置解析 + Gin 挂载 +
// 类型别名"三件事，不得在此新增第二套验签或协议实现（AGENTS.md §OIDC 分层约定）。
//
// back-channel logout 的校验逻辑单一真源在
// github.com/morehao/ark-iam/sdk/rp/logout（框架无关、基于 net/http）；
// 本包只做"Gin 挂载 + 最近记录（调试/e2e 断言）"，不再自行实现 token 校验。
package oidckit

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/morehao/ark-iam/sdk/contract"
	"github.com/morehao/ark-iam/sdk/rp/logout"
)

// 契约别名：调用方引用 oidckit.Xxx 与 sdk 侧完全同一类型。
const BackChannelLogoutEventURI = contract.BackChannelLogoutEventURI

type (
	// LogoutTokenClaims 是 logout_token 的标准声明（真源 sdk/contract）。
	LogoutTokenClaims = contract.LogoutTokenClaims
	// JTIStore 是 jti 去重实现接口。
	JTIStore = logout.JTIStore
)

// SessionRevoker 是收到合法 logout_token 后执行本地会话清除的回调。
// 返回 error 时接收端按 500 处理（OP 会重投）；nil 表示已成功处理。
type SessionRevoker = logout.SessionRevoker

// MemoryJTIStore 是 JTIStore 的进程内实现。
type MemoryJTIStore = logout.MemoryJTIStore

// NewMemoryJTIStore 构造内存 jti 表。
func NewMemoryJTIStore() *MemoryJTIStore { return logout.NewMemoryJTIStore() }

// HashJTI 返回 jti 的存储键（sha256 十六进制）。
func HashJTI(jti string) string { return logout.HashJTI(jti) }

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

// BackChannelLogoutHandler 是 back-channel logout 接收端的 Gin 处理器。
type BackChannelLogoutHandler struct {
	// Receiver 是 SDK 的 net/http 接收端（校验 + jti 去重的唯一实现）。
	Receiver *logout.Receiver
	// KeySource / Issuer / ClientID 保留导出字段以兼容既有调用方与测试。
	KeySource KeySource
	Issuer    string
	ClientID  string
	OnLogout  SessionRevoker

	// 最近接收的 logout_token 记录（内存，供调试/可观测/e2e 断言）。
	recentMu sync.Mutex
	recent   []RecentLogoutToken
}

// NewBackChannelLogoutHandler 构造接收端处理器。
//
// keys/issuer/clientID 缺一不可：任一缺失时接收端对所有请求返回 400（fail-closed），
// 绝不"因为配不出来"就放行。
func NewBackChannelLogoutHandler(keys KeySource, issuer, clientID string, onLogout SessionRevoker) *BackChannelLogoutHandler {
	return &BackChannelLogoutHandler{
		Receiver:  newSDKReceiver(keys, issuer, clientID, onLogout),
		KeySource: keys,
		Issuer:    issuer,
		ClientID:  clientID,
		OnLogout:  onLogout,
		recent:    make([]RecentLogoutToken, 0, maxRecentTokens),
	}
}

func newSDKReceiver(keys KeySource, issuer, clientID string, onLogout SessionRevoker) *logout.Receiver {
	opts := []logout.Option{
		logout.WithJTIStore(logout.NewMemoryJTIStore()),
	}
	if issuer != "" {
		opts = append(opts, logout.WithIssuer(issuer))
	}
	if clientID != "" {
		opts = append(opts, logout.WithClientID(clientID))
	}
	if keys != nil {
		opts = append(opts, logout.WithKeySource(keys))
	}
	if onLogout != nil {
		opts = append(opts, logout.WithSessionRevoker(onLogout))
	}
	return logout.NewReceiver(opts...)
}

// ParseLogoutToken 解析并校验 logout_token（委托 SDK，保留本包历史导出名）。
//
// 校验项：RS256 + kid 必填 + 拒绝头部 jwk/jku/x5u、iss、aud、exp(必填)、iat(必填)、
// events(必含 backchannel-logout)、jti(必填)、nonce(必须不存在)、sub(必填)。
func ParseLogoutToken(tokenStr string, keys KeySource, issuer, clientID string) (*LogoutTokenClaims, error) {
	if keys == nil {
		return nil, errors.New("oidc logout: key source not initialized")
	}
	return newSDKReceiver(keys, issuer, clientID, nil).Parse(tokenStr)
}

// Handler 返回 Gin HandlerFunc：POST back_channel_logout_uri?logout_token=...
//
// 处理语义（对齐 OIDC Back-Channel Logout 1.0 §2.2）：
//   - 成功处理 → 200（重复投递同样 200，幂等）
//   - token 无效（签名/声明校验失败）→ 400（OP 不重试无效 token）
//   - 本地登出回调失败 → 500（OP 将按重试策略重发；此时不记 jti，保证重投有效）
func (h *BackChannelLogoutHandler) Handler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		tokenStr := ctx.PostForm("logout_token")
		claims, parseErr := h.Receiver.Parse(tokenStr)
		// 观测优先：先记录本次投递（含无效），再交给 SDK 执行副作用。
		h.record(RecentLogoutToken{
			JTI:      claimsJTI(claims),
			Sub:      claimsSub(claims),
			SID:      claimsSID(claims),
			ClientID: h.ClientID,
			Received: time.Now(),
			Valid:    parseErr == nil,
			ParseErr: errString(parseErr),
		})

		// 语义（校验 → 幂等去重 → 撤销 → 记 jti）只有 SDK 一份实现：
		// 撤销失败返回 500 且不记 jti，保证 OP 重投能真正重试。
		status, msg := h.Receiver.Handle(ctx, tokenStr)
		if status == http.StatusOK {
			ctx.Status(http.StatusOK)
			return
		}
		ctx.AbortWithStatusJSON(status, gin.H{"code": status, "msg": msg})
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
func RegisterReceiverRoutes(group *gin.RouterGroup, basePath string, keys KeySource, issuer, clientID string, onLogout SessionRevoker) *BackChannelLogoutHandler {
	if basePath == "" {
		basePath = "/oidc/bc-logout"
	}
	h := NewBackChannelLogoutHandler(keys, issuer, clientID, onLogout)
	group.POST(basePath, h.Handler())
	group.GET(basePath+"/recent", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"recent": h.Recent()})
	})
	return h
}
