package appassembly

// newVerifier 通过注入的验签实现（Config.VerifierBuilder）构造 contract.Verifier，供注入
// pkg/middleware 与 apps/auth。
//
// 解环关键：pkg 侧只依赖 contract.Verifier 接口，真实验签实现由调用方（组合根）注入 sdk/rp
// 的实现，从而 pkg 不再 import sdk（F1 修正 / 终局 DAG）。
//
// builder 为 nil 时返回 nil（中间件对任何 token fail-closed 401，绝不裸放行）。
func (a *App) newVerifier(keySource KeySource, issuer string, audiences []string) Verifier {
	if a.cfg.VerifierBuilder == nil {
		return nil
	}
	return a.cfg.VerifierBuilder(keySource, issuer, audiences)
}
