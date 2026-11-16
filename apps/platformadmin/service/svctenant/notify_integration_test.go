package svctenant

import (
	"testing"

	"github.com/morehao/ark-iam/pkg/dbclient"
)

// 通道未启用（gsetting 无 notify 配置 / enabled=false）：行为与现状一致——
// 回显明文、delivered 缺省 false。
//
// 说明：投递成功 / requireDelivery 收窄等「渠道可用」分支的语义已下沉到 pkg/notify
// （DeliverTempPassword + fake Sender）单测覆盖；service 层只负责把结果映射到 DTO 与审计，
// 这里以「未启用 → 回显」这一确定性路径验证映射正确。
func TestTenantCreateNotifyDisabledKeepsEcho(t *testing.T) {
	setupTenantCreateEnv(t)
	ctx := newTenantScopeGinCtx("")

	svc := &tenantSvc{db: dbclient.IamDB} // 空 notify 配置 = 通道未启用
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
