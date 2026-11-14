package svctenant

import (
	"context"
	"errors"
	"testing"

	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/notify"
)

// fakeNotifySender 记录投递并可配置返回错误（模拟渠道可用/不可用）。
type fakeNotifySender struct {
	to   []string
	err  error
}

func (f *fakeNotifySender) Send(_ context.Context, msg notify.Message) error {
	f.to = append(f.to, msg.To)
	return f.err
}

// 通道启用 + 投递成功：管理员初始密码不回显、delivered=true。
func TestTenantCreateDeliversAndHidesPlaintext(t *testing.T) {
	setupTenantCreateEnv(t)
	ctx := newTenantScopeGinCtx("")

	sender := &fakeNotifySender{}
	svc := &tenantSvc{db: dbclient.IamDB, notifier: notify.NewForTest(sender, nil, true)}
	resp, err := svc.Create(requestCtx(ctx), newTenantCreateReq("Acme Corp", "admin@acme.com"))
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if resp.AdminInitialPassword != "" {
		t.Errorf("adminInitialPassword = %q, want empty when delivered", resp.AdminInitialPassword)
	}
	if !resp.Delivered {
		t.Error("delivered = false, want true")
	}
	if len(sender.to) != 1 || sender.to[0] != "admin@acme.com" {
		t.Errorf("delivery recipients = %v, want [admin@acme.com]", sender.to)
	}
}

// 通道启用 + 投递失败 + requireDelivery=true：返回 NotifyDeliverError，账号已建（不重复建）。
func TestTenantCreateDeliveryFailClosed(t *testing.T) {
	setupTenantCreateEnv(t)
	ctx := newTenantScopeGinCtx("")

	sender := &fakeNotifySender{err: errors.Join(notify.ErrDeliveryFailed, errors.New("smtp down"))}
	svc := &tenantSvc{db: dbclient.IamDB, notifier: notify.NewForTest(sender, nil, true)}
	_, err := svc.Create(requestCtx(ctx), newTenantCreateReq("Acme Corp", "admin@acme.com"))
	if err == nil {
		t.Fatal("want NotifyDeliverError, got nil")
	}
	if !errors.Is(err, code.GetError(code.NotifyDeliverError)) {
		t.Fatalf("want NotifyDeliverError, got %v", err)
	}
}

// 通道启用 + 投递失败 + requireDelivery=false：回退为回显明文、delivered=false。
func TestTenantCreateDeliveryFailOpen(t *testing.T) {
	setupTenantCreateEnv(t)
	ctx := newTenantScopeGinCtx("")

	sender := &fakeNotifySender{err: errors.Join(notify.ErrDeliveryFailed, errors.New("smtp down"))}
	svc := &tenantSvc{db: dbclient.IamDB, notifier: notify.NewForTest(sender, nil, false)}
	resp, err := svc.Create(requestCtx(ctx), newTenantCreateReq("Acme Corp", "admin@acme.com"))
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if resp.AdminInitialPassword == "" {
		t.Error("adminInitialPassword empty, want fallback plaintext")
	}
	if resp.Delivered {
		t.Error("delivered = true, want false on fallback")
	}
}

// 通道未启用（nil notifier）：行为与现状一致——回显明文、delivered 缺省 false。
func TestTenantCreateNotifyDisabledKeepsEcho(t *testing.T) {
	setupTenantCreateEnv(t)
	ctx := newTenantScopeGinCtx("")

	svc := &tenantSvc{db: dbclient.IamDB} // notifier nil = 通道未启用
	resp, err := svc.Create(requestCtx(ctx), newTenantCreateReq("Acme Corp", "admin@acme.com"))
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if resp.AdminInitialPassword == "" {
		t.Error("adminInitialPassword empty, want echo when channel disabled")
	}
	if resp.Delivered {
		t.Error("delivered = true, want false when channel disabled")
	}
}
