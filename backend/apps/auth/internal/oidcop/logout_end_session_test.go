package oidcop

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/iam/sso"
	"github.com/morehao/ark-iam/pkg/testsetup"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/oidc/v3/pkg/op"
)

// sloQueueContended 检测共享 SLO 队列是否正被运行中的服务（logout worker）持续消费。
// 若被外部消费，队列内容无法稳定断言（任务会被立即取走），测试应跳过而非误报失败。
// 仅把连接存活超过 contentionMinAge 的 BRPop 消费者视为外部 worker：
// 本测试套件自身的短超时 BRPop（几秒内）不算，避免跨包并行时误判。
func sloQueueContended(t *testing.T) bool {
	t.Helper()
	const contentionMinAge = 5 * time.Second
	clients, err := dbclient.RedisCli.ClientList(context.Background()).Result()
	if err != nil {
		return true // 无法探测时保守跳过
	}
	for _, line := range strings.Split(clients, "\n") {
		if !strings.Contains(line, "cmd=brpop") {
			continue
		}
		if age, ok := parseClientAge(line); ok && age > contentionMinAge {
			return true
		}
	}
	return false
}

// parseClientAge 从 redis CLIENT LIST 单行中解析 age=<秒> 字段。
func parseClientAge(line string) (time.Duration, bool) {
	for _, field := range strings.Fields(line) {
		if strings.HasPrefix(field, "age=") {
			secs, err := strconv.ParseInt(strings.TrimPrefix(field, "age="), 10, 64)
			if err != nil {
				return 0, false
			}
			return time.Duration(secs) * time.Second, true
		}
	}
	return 0, false
}

func TestTerminateSessionFromRequestEnqueuesPreciseSidJobs(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)
	// 共享 SLO 队列被运行中的 logout worker 消费时任务会被立即取走，无法稳定断言（环境问题）。
	if sloQueueContended(t) {
		t.Skip("shared SLO queue is being consumed by a running logout worker; skip queue assertion")
	}
	ctx := context.Background()

	users := []model.UserEntity{
		{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "80"}}, TenantID: "1", PersonID: "88"},
	}
	storage, _ := newTenantClaimTestStore(t, users)

	// 登记两个会话：sid-a 与 sid-b，person 88
	sloStore := sso.NewSLOStore()
	regA := sso.LogoutRegistration{OIDCSessionID: "at-a", ClientID: "client-a", UserID: "person:88", BackChannelLogoutURI: "https://a/bc"}
	regB := sso.LogoutRegistration{OIDCSessionID: "at-b", ClientID: "client-b", UserID: "person:88", BackChannelLogoutURI: "https://b/bc"}
	if err := sloStore.Register(ctx, "sid-a", regA); err != nil {
		t.Fatalf("register sid-a: %v", err)
	}
	if err := sloStore.Register(ctx, "sid-b", regB); err != nil {
		t.Fatalf("register sid-b: %v", err)
	}

	// 带 sid 的登出：精准注销 sid-a，只应入队 client-a
	endSession := &op.EndSessionRequest{
		UserID: "person:88",
		IDTokenHintClaims: &oidc.IDTokenClaims{
			SessionID: "sid-a",
		},
		RedirectURI: "https://rp.example.com/logged-out",
	}
	if _, err := storage.TerminateSessionFromRequest(ctx, endSession); err != nil {
		t.Fatalf("TerminateSessionFromRequest failed: %v", err)
	}

	// 队列应有一条 sid-a 的任务
	job, ok, err := sso.DequeueLogout(ctx, 200*time.Millisecond)
	if err != nil {
		t.Fatalf("dequeue: %v", err)
	}
	if !ok {
		t.Fatal("expected a logout job")
	}
	if job.SessionID != "sid-a" {
		t.Fatalf("expected job SessionID sid-a, got %q", job.SessionID)
	}
	if job.ClientID != "client-a" {
		t.Fatalf("expected job client client-a, got %q", job.ClientID)
	}

	// 不应再有多余任务
	if _, ok, _ = sso.DequeueLogout(ctx, 200*time.Millisecond); ok {
		t.Fatal("expected no additional logout job")
	}
}
