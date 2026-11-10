package contract

import "context"

// Identity 是校验通过的令牌身份（框架无关，纯数据）。
//
// 字段口径（见设计文档「分层与接入点」）：
//   - PersonID：sub=person:<id> 解析出的自然人 ID；机器令牌为空
//   - TenantID：claim tenant_id，租户作用域与审计隔离键
//   - UserID：claim user_id = user.id，**审计主键**（与 IAM 自身 created_by 口径一致）
//   - ClientID：aud 命中的本应用 client_id
//   - IsMachine：token_usage=machine
//   - SessionID：sid，用于按会话作废与 back-channel 登出匹配
//   - Scopes：标准 scope 声明（空格分隔），用于目录 API 等 M2M 权限判定
//   - ActorID：act.sub，机器凭证的代操作人
type Identity struct {
	PersonID  string
	TenantID  string
	UserID    string
	ClientID  string
	IsMachine bool
	SessionID string
	Scopes    []string
	ActorID   string
}

// HasScope 报告该身份是否携带指定 scope。
func (i *Identity) HasScope(scope string) bool {
	if i == nil {
		return false
	}
	for _, s := range i.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// identityContextKey 是身份在 context 中的私有键类型。
type identityContextKey struct{}

// WithIdentity 把已验证身份放入 context（消费者侧中间件使用）。
func WithIdentity(ctx context.Context, identity *Identity) context.Context {
	if ctx == nil || identity == nil {
		return ctx
	}
	return context.WithValue(ctx, identityContextKey{}, identity)
}

// IdentityFrom 从 context 取回已验证身份；未注入时返回 nil。
func IdentityFrom(ctx context.Context) *Identity {
	if ctx == nil {
		return nil
	}
	identity, _ := ctx.Value(identityContextKey{}).(*Identity)
	return identity
}
