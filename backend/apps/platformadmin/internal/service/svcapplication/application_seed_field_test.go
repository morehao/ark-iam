package svcapplication

import (
	"testing"

	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtoapplication"
	"github.com/morehao/ark-iam/platformadmin/testutil"
	"github.com/morehao/golib/gerror"
)

// TestUpdateBuiltInApplicationRejectsSeedOwnedFields 内置应用的名称/描述是种子收敛字段
// （字段权威矩阵 reconcile），控制台必须拒写；启停与排序是 create_only（归运维），照常可改。
// 回归背景：种子会收敛 name/description，若控制台同时可写，运维改完重启就被收回（双写者）。
func TestUpdateBuiltInApplicationRejectsSeedOwnedFields(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ApplicationEntity{})
	builtin := &model.ApplicationEntity{
		Code:        "platform_admin",
		Name:        "平台管理后台",
		Description: "平台管理后台应用",
		Source:      model.AppSourceBuiltin,
		Status:      model.AppStatusEnable,
	}
	if err := db.Create(builtin).Error; err != nil {
		t.Fatalf("seed application: %v", err)
	}
	svc := NewApplicationSvc()
	ctx := newDeleteCtx("0")

	err := svc.Update(ctx, &dtoapplication.ApplicationUpdateReq{
		AppID:       builtin.ID,
		Name:        "运维自定名",
		Description: builtin.Description,
		Status:      model.AppStatusEnable,
		Sort:        builtin.Sort,
	})
	if err == nil {
		t.Fatal("内置应用改名必须被拒绝")
	}
	if gerror.GetCode(err) != int(code.ApplicationBuiltInFieldImmutableError) {
		t.Fatalf("期望 ApplicationBuiltInFieldImmutableError, got %v", err)
	}
	got, err := dao.NewApplicationDao().GetByID(ctx, builtin.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "平台管理后台" {
		t.Fatalf("被拒的改名不得落库, got %q", got.Name)
	}

	// 启停/排序归运维：不触发拒写
	if err := svc.Update(ctx, &dtoapplication.ApplicationUpdateReq{
		AppID:       builtin.ID,
		Name:        builtin.Name,
		Description: builtin.Description,
		Status:      model.AppStatusDisable,
		Sort:        9,
	}); err != nil {
		t.Fatalf("内置应用的启停/排序应可改: %v", err)
	}
	got, err = dao.NewApplicationDao().GetByID(ctx, builtin.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Status != model.AppStatusDisable || got.Sort != 9 {
		t.Fatalf("启停/排序未落库: status=%q sort=%d", got.Status, got.Sort)
	}

	// 第三方应用不受该规则约束
	thirdParty := &model.ApplicationEntity{Code: "customer_app", Name: "客户应用", Source: model.AppSourceThirdParty, Status: model.AppStatusEnable}
	if err := db.Create(thirdParty).Error; err != nil {
		t.Fatalf("seed third party application: %v", err)
	}
	if err := svc.Update(ctx, &dtoapplication.ApplicationUpdateReq{
		AppID: thirdParty.ID, Name: "客户应用改名", Description: "新描述", Status: model.AppStatusEnable,
	}); err != nil {
		t.Fatalf("第三方应用改名应放行: %v", err)
	}
}
