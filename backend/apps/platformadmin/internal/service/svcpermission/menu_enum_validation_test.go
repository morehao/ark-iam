package svcpermission

import (
	"testing"

	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/object/objpermission"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtopermission"
	"github.com/morehao/ark-iam/platformadmin/testutil"
	"github.com/morehao/golib/gerror"
)

// TestMenuSwitchEnumValidation 隐藏/外链/缓存三个开关已改为具名枚举：
// 合法值原样落库，未提供的空串归一为 disable（不得落空字符串），非法值返回既有功能级错误码且不改写数据。
func TestMenuSwitchEnumValidation(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.MenuEntity{}, &model.ApplicationEntity{})
	app := &model.ApplicationEntity{Code: "app_enum", Name: "枚举应用", Source: model.AppSourceThirdParty, Status: model.AppStatusEnable}
	if err := db.Create(app).Error; err != nil {
		t.Fatalf("seed application: %v", err)
	}
	svc := NewMenuSvc()
	ctx := newGinCtx("1", "0")

	base := func() objpermission.MenuBaseInfo {
		return objpermission.MenuBaseInfo{
			AppID: app.ID, Name: "开关菜单", Code: "switch_menu",
			Type: model.MenuTypeMenu, Visibility: model.MenuVisibilityAdmin, Status: model.MenuStatusEnable,
		}
	}

	// 合法值原样落库
	valid := base()
	valid.Hidden = model.MenuHiddenFlagEnable
	valid.ExternalLink = model.MenuExternalLinkFlagEnable
	valid.KeepAlive = model.MenuKeepAliveFlagEnable
	resp, err := svc.Create(ctx, &dtopermission.MenuCreateReq{MenuBaseInfo: valid})
	if err != nil {
		t.Fatalf("合法开关应允许创建: %v", err)
	}
	got, err := dao.NewMenuDao().GetByID(ctx, resp.MenuID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Hidden != model.MenuHiddenFlagEnable || got.ExternalLink != model.MenuExternalLinkFlagEnable || got.KeepAlive != model.MenuKeepAliveFlagEnable {
		t.Fatalf("合法开关未落库: %+v", got)
	}

	// 空串（未提供）归一为 disable：更新走 UpdateMap，不得把空字符串写进列
	if err := svc.Update(ctx, &dtopermission.MenuUpdateReq{MenuID: resp.MenuID, MenuBaseInfo: base()}); err != nil {
		t.Fatalf("空串开关应允许更新: %v", err)
	}
	got, err = dao.NewMenuDao().GetByID(ctx, resp.MenuID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Hidden != model.MenuHiddenFlagDisable || got.ExternalLink != model.MenuExternalLinkFlagDisable || got.KeepAlive != model.MenuKeepAliveFlagDisable {
		t.Fatalf("空串开关应归一为 disable: %+v", got)
	}

	// 非法值：创建路径
	badHidden := base()
	badHidden.Hidden = model.MenuHiddenFlag("yes")
	if _, err := svc.Create(ctx, &dtopermission.MenuCreateReq{MenuBaseInfo: badHidden}); err == nil || gerror.GetCode(err) != int(code.MenuCreateError) {
		t.Fatalf("非法 hidden 应返回 MenuCreateError, got %v", err)
	}
	badExternal := base()
	badExternal.ExternalLink = model.MenuExternalLinkFlag("true")
	if _, err := svc.Create(ctx, &dtopermission.MenuCreateReq{MenuBaseInfo: badExternal}); err == nil || gerror.GetCode(err) != int(code.MenuCreateError) {
		t.Fatalf("非法 externalLink 应返回 MenuCreateError, got %v", err)
	}
	badKeepAlive := base()
	badKeepAlive.KeepAlive = model.MenuKeepAliveFlag("1")
	if _, err := svc.Create(ctx, &dtopermission.MenuCreateReq{MenuBaseInfo: badKeepAlive}); err == nil || gerror.GetCode(err) != int(code.MenuCreateError) {
		t.Fatalf("非法 keepAlive 应返回 MenuCreateError, got %v", err)
	}

	// 非法值：更新路径，拒绝后不得改写
	badUpdate := base()
	badUpdate.Hidden = model.MenuHiddenFlag("on")
	if err := svc.Update(ctx, &dtopermission.MenuUpdateReq{MenuID: resp.MenuID, MenuBaseInfo: badUpdate}); err == nil || gerror.GetCode(err) != int(code.MenuUpdateError) {
		t.Fatalf("非法 hidden 更新应返回 MenuUpdateError, got %v", err)
	}
	got, err = dao.NewMenuDao().GetByID(ctx, resp.MenuID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Hidden != model.MenuHiddenFlagDisable {
		t.Fatalf("被拒的非法开关不应落库, got %q", got.Hidden)
	}
}
