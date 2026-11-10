// Package logout 是 OIDC Back-Channel Logout 的 RP 侧接收端（对外 SDK 面）。
//
// 校验/去重/撤销的**单一实现已下沉 contract/logout**（框架无关、基于 net/http、
// 仅依赖 stdlib + jwt + contract，使 pkg 侧可依赖它而不 import sdk）。
// 本包只做类型/函数别名，保持对外 `sdk/rp/logout` 的历史 API 完全不变。
package logout

import (
	"github.com/morehao/ark-iam/contract/logout"
)

// 类型别名：调用方引用 sdk/rp/logout.Xxx 与 contract/logout 完全同一类型。
type (
	// JWKS 是接收端所需的最小密钥来源。
	JWKS = logout.JWKS
	// SessionRevoker 是收到合法 logout_token 后执行本地会话清除的回调。
	SessionRevoker = logout.SessionRevoker
	// JTIStore 记录已处理过的 logout_token jti。
	JTIStore = logout.JTIStore
	// MemoryJTIStore 是 JTIStore 的进程内实现。
	MemoryJTIStore = logout.MemoryJTIStore
	// Option 是接收端可选配置。
	Option = logout.Option
	// Logger 是接收端最小日志接口。
	Logger = logout.Logger
	// Receiver 是接收端。
	Receiver = logout.Receiver
)

// NewMemoryJTIStore 构造内存 jti 表。
var NewMemoryJTIStore = logout.NewMemoryJTIStore

// HashJTI 返回 jti 的存储键（sha256 十六进制）。
var HashJTI = logout.HashJTI

// NewReceiver 构造接收端。
var NewReceiver = logout.NewReceiver

// —— Option 构造器透出 ——
var (
	WithLogger         = logout.WithLogger
	WithIssuer         = logout.WithIssuer
	WithClientID       = logout.WithClientID
	WithKeySource      = logout.WithKeySource
	WithSessionRevoker = logout.WithSessionRevoker
	WithJTIStore       = logout.WithJTIStore
	WithLeeway         = logout.WithLeeway
)
