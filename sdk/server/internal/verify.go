package internal

import (
	"github.com/morehao/ark-iam/sdk/rp"
)

// newVerifier 构造 contract.Verifier（包 sdk/rp 的验签实现），供注入 pkg/middleware。
//
// 解环关键：pkg 侧只依赖 contract.Verifier 接口，真实验签实现由本装配层（internal）
// new 出并注入，从而 pkg 不再 import sdk（F1 修正）。
//
// issuer/audiences 用于构建 sdk/rp 校验器（iss/aud 收紧），与 pkg 中间件选项语义一致。
func newVerifier(keySource KeySource, issuer string, audiences []string) Verifier {
	opts := []rp.VerifierOption{}
	if issuer != "" {
		opts = append(opts, rp.WithIssuer(issuer))
	}
	if len(audiences) > 0 {
		opts = append(opts, rp.WithAudiences(audiences...))
	}
	return rp.NewVerifier(keySource, opts...)
}
