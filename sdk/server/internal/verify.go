package internal

import (
	"github.com/morehao/ark-iam/sdk/rp"
)

// newVerifier 构造 contract.Verifier（包 .sdk/rp 的验签实现），供注入 pkg/middleware。
//
// 解环关键：pkg 侧只依赖 contract.Verifier 接口，真实验签实现由本装配层（internal）
// new 出并注入，从而 pkg 不再 import sdk（F1 修正）。
func newVerifier(keySource KeySource) Verifier {
	return rp.NewVerifier(keySource)
}
