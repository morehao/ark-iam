package svcoidc

import (
	"os"
	"sort"
	"testing"
	"time"

	"github.com/morehao/ark-iam/pkg/core/application"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/object/objauth"
	"github.com/stretchr/testify/require"
)

// AC-11 性能证据（P8-6）：门禁打开后，登录/签发路径新增的判定成本必须 ≤ 5ms（P99）。
//
// 为什么测这里而不是压整个 token 端点：门禁对**每一次**人类令牌签发只新增一段固定工作——
// 「按 client_id 解析应用 + 判定式 CheckUsable（3 个点查）」，其余链路（验签、Redis 元数据、
// 令牌落库）与开关无关，不构成增量。因此在**真实 Postgres** 上直接量这段增量，既覆盖 AC-11 的
// 口径，又不受本机 HTTP/连接池波动干扰；端到端 token 端点的 P99 对比留到 P9 灰度前的活栈复测。
//
// 默认跳过（性能用例不进常规回归，避免 CI 机器负载导致的假失败）：运行方式
//
//	IAM_AC11_MEASURE=1 go test ./apps/auth/internal/service/svcoidc/ -run TestAppAccessGateLatency -v
func TestAppAccessGateLatencyMeasurement(t *testing.T) {
	if os.Getenv("IAM_AC11_MEASURE") != "1" {
		t.Skip("需要 IAM_AC11_MEASURE=1（真实 Postgres 上的性能测量，默认不跑）")
	}

	env := newAppAccessGateEnv(t)
	appID, clientCode := env.seedApp(t)
	seed := env.seedMemberTenant(t, appID, model.UserApplicationStatusEnable)
	option := objauth.TenantOption{TenantID: seed.tenantID, Name: "granted", UserID: seed.memberID}
	svc := &oidcAuthSvc{provider: env.provider}

	const iterations = 200

	// 基线：开关关闭时门禁是**纯函数短路**，不产生任何查询
	offEnv := newAppAccessGateEnvWithFlag(t, false)
	offSvc := &oidcAuthSvc{provider: offEnv.provider}
	offDurations := measure(t, iterations, func() {
		allowed, err := offSvc.canUseClientApp(offEnv.ctx, clientCode, option)
		require.NoError(t, err)
		require.True(t, allowed)
	})

	// 开门禁：client_id → 应用 → CheckUsable（user_application/application/tenant_application 三个点查）
	onDurations := measure(t, iterations, func() {
		allowed, err := svc.canUseClientApp(requestCtx(env.ctx), clientCode, option)
		require.NoError(t, err)
		require.True(t, allowed)
	})

	// 领域判定本身的成本（令牌签发兜底门禁与登录早拒共用同一份实现）
	judgeDurations := measure(t, iterations, func() {
		usable, err := application.CheckUsable(env.ctx, seed.tenantID, seed.memberID, appID)
		require.NoError(t, err)
		require.True(t, usable)
	})

	off := summarize(offDurations)
	on := summarize(onDurations)
	judge := summarize(judgeDurations)
	t.Logf("AC-11 增量测量（真实 Postgres，%d 次/组）：", iterations)
	t.Logf("  门禁关闭（纯短路）      P50=%.3fms P95=%.3fms P99=%.3fms max=%.3fms", off.p50, off.p95, off.p99, off.max)
	t.Logf("  门禁开启（client→app+判定）P50=%.3fms P95=%.3fms P99=%.3fms max=%.3fms", on.p50, on.p95, on.p99, on.max)
	t.Logf("  仅判定式 CheckUsable     P50=%.3fms P95=%.3fms P99=%.3fms max=%.3fms", judge.p50, judge.p95, judge.p99, judge.max)
	t.Logf("  AC-11 增量（开启 P99 − 关闭 P99）= %.3fms，阈值 5ms", on.p99-off.p99)

	require.Less(t, on.p99-off.p99, 5.0, "AC-11：门禁开启相对关闭的 P99 增量必须小于 5ms")
}

type latencyStat struct{ p50, p95, p99, max float64 }

// measure 跑 iterations 次并返回每次耗时（毫秒）。前 10 次作为预热丢弃。
func measure(t *testing.T, iterations int, fn func()) []float64 {
	t.Helper()
	const warmup = 10
	durations := make([]float64, 0, iterations)
	for i := 0; i < iterations+warmup; i++ {
		start := time.Now()
		fn()
		elapsed := float64(time.Since(start).Microseconds()) / 1000.0
		if i >= warmup {
			durations = append(durations, elapsed)
		}
	}
	return durations
}

func summarize(durations []float64) latencyStat {
	sorted := append([]float64(nil), durations...)
	sort.Float64s(sorted)
	pick := func(p float64) float64 {
		if len(sorted) == 0 {
			return 0
		}
		idx := int(p * float64(len(sorted)-1))
		return sorted[idx]
	}
	return latencyStat{p50: pick(0.50), p95: pick(0.95), p99: pick(0.99), max: sorted[len(sorted)-1]}
}
