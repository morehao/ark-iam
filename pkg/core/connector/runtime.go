package connector

import "context"

// runtimeContext 保证下游拿到非 nil 的 context（context.Context 是接口，nil 会被
// glog / gorm 解引用 panic）。
func runtimeContext(ctx context.Context) context.Context {
	if ctx != nil {
		return ctx
	}
	return context.Background()
}
