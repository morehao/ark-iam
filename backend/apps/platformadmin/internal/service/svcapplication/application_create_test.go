package svcapplication

import (
	"testing"

	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtoapplication"
	"github.com/morehao/ark-iam/platformadmin/testutil"
	"github.com/morehao/golib/gerror"
)

// TestApplicationCreateRejectsInvalidCode 应用编码规则为下划线连接（model.AppCodePattern：
// 小写字母开头，仅含小写字母/数字/下划线）：连字符、大写、数字开头等非法编码必须在
// service 入口被拒（返回 ApplicationCodeInvalidError），不得落库。
func TestApplicationCreateRejectsInvalidCode(t *testing.T) {
	testutil.SetupSQLite(t, &model.ApplicationEntity{})

	svc := NewApplicationSvc()
	for _, badCode := range []string{"my-app", "My_App", "1app", "app.web", "app web", "_app", ""} {
		_, err := svc.Create(newApplicationCtx(), &dtoapplication.ApplicationCreateReq{Code: badCode, Name: "非法编码应用"})
		if err == nil {
			t.Fatalf("Create(%q) expected error", badCode)
		}
		if gerror.GetCode(err) != int(code.ApplicationCodeInvalidError) {
			t.Fatalf("Create(%q) error code = %d, want %d", badCode, gerror.GetCode(err), code.ApplicationCodeInvalidError)
		}
	}
}

// TestApplicationCreateAcceptsUnderscoreCode 合法编码（下划线形态）落库成功，
// 且控制台创建的应用恒为 third_party。
func TestApplicationCreateAcceptsUnderscoreCode(t *testing.T) {
	testutil.SetupSQLite(t, &model.ApplicationEntity{})

	ctx := newApplicationCtx()
	resp, err := NewApplicationSvc().Create(ctx, &dtoapplication.ApplicationCreateReq{Code: "my_app_2", Name: "合法应用"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if resp.Code != "my_app_2" {
		t.Fatalf("resp code = %q, want my_app_2", resp.Code)
	}

	got, err := dao.NewApplicationDao().GetByID(ctx, resp.AppID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got == nil || got.ID == "" {
		t.Fatalf("application not persisted: %+v", got)
	}
	if got.Code != "my_app_2" {
		t.Errorf("stored code = %q, want my_app_2", got.Code)
	}
	if got.Source != model.AppSourceThirdParty {
		t.Errorf("stored source = %q, want %q", got.Source, model.AppSourceThirdParty)
	}
}
