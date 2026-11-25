package svcapplicationclient

import (
	"testing"

	"github.com/morehao/ark-iam/apps/platformadmin/dto/dtoapplicationclient"
	"github.com/morehao/ark-iam/apps/platformadmin/testutil"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/gerror"
)

// TestListSecretsRejectsCrossTenantClient 回归 P2-14：
// ListSecrets 此前未做归属校验，可凭任意 clientID 列出他租户客户端的密钥元数据。
// 现与 Detail/DeleteSecret 同口径：跨租户客户端一律按「客户端不存在」拒绝。
func TestListSecretsRejectsCrossTenantClient(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ApplicationClientEntity{}, &model.ApplicationClientSecretEntity{})

	// 客户端属于租户 2；调用方上下文是租户 1。
	foreign := newTestClientEntity("Foreign", "foreign-client", model.ApplicationClientSourceThirdParty)
	foreign.TenantID = "2"
	if err := db.Create(foreign).Error; err != nil {
		t.Fatalf("seed foreign client: %v", err)
	}

	svc := NewApplicationClientSvc(nil)
	_, err := svc.ListSecrets(requestCtx(newOAuthDeleteCtx("1", "0")), &dtoapplicationclient.SecretListReq{
		ApplicationClientID: foreign.ID,
	})
	if err == nil {
		t.Fatal("cross-tenant ListSecrets must be rejected")
	}
	if gerror.GetCode(err) != int(code.ApplicationClientNotExistError) {
		t.Fatalf("want ApplicationClientNotExistError, got %v", err)
	}

	// 本租户客户端：正常返回（空列表即可）。
	own := newTestClientEntity("Own", "own-client", model.ApplicationClientSourceThirdParty)
	own.TenantID = "1"
	if err := db.Create(own).Error; err != nil {
		t.Fatalf("seed own client: %v", err)
	}
	if _, err := svc.ListSecrets(requestCtx(newOAuthDeleteCtx("1", "0")), &dtoapplicationclient.SecretListReq{
		ApplicationClientID: own.ID,
	}); err != nil {
		t.Fatalf("same-tenant ListSecrets should succeed, got %v", err)
	}
}
