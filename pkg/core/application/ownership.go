package application

import (
	"context"
	"errors"

	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/glog"
)

// 归属相关的领域哨兵错误（领域层不依赖 pkg/code，由应用层映射为功能级错误码）。
var (
	// ErrAppNotOwned 目标应用不属于当前租户（非 owner）或不存在。
	// 调用方应映射为"不存在"语义的功能级错误码，避免泄露其它租户应用的存在性。
	ErrAppNotOwned = errors.New("application: 目标应用不属于当前租户或不存在")
	// ErrNoCurrentTenantScope 当前 ctx 不是一个"具体当前租户"作用域（AllScope/未声明/空租户）。
	// 归属判等必须 fail-closed，不得把空作用域当成某租户。
	ErrNoCurrentTenantScope = errors.New("application: 当前上下文无具体租户作用域")
)

// AssertCurrentTenantScope 证明当前 ctx 是一个"具体当前租户"作用域并返回该租户 ID。
//
// 安全要点：不能用 dbclient.TenantIDFromContext 的返回值直接做归属判等——它把 AllScope /
// ExplicitScope / 未声明 都折叠成空串，会让 owner 校验在空作用域下把"空"判成某归属租户（静默
// 越权）。因此这里强制要求 Kind==Current 且非空，否则 fail-closed 返回 ErrNoCurrentTenantScope。
func AssertCurrentTenantScope(ctx context.Context) (string, error) {
	tenantID, ok := dbclient.CurrentScopeTenantID(ctx)
	if !ok {
		return "", ErrNoCurrentTenantScope
	}
	return tenantID, nil
}

// LoadOwnedApp 加载并校验：appID 对应的应用存在且归属当前租户（owner_type=tenant 且
// owner_tenant_id==currentTenant）。任何不满足（不存在 / 非租户归属 / 非本租户）一律返回
// ErrAppNotOwned，不区分"存在但不属于你"，防枚举探测。
//
// 这是租户侧菜单写/管理读的唯一归属判定入口：iam_application 是全局表（租户插件不注入
// tenant_id），归属校验是唯一安全边界，必须显式判 owner，不能靠作用域过滤。
func LoadOwnedApp(ctx context.Context, appID, currentTenant string) (*model.ApplicationEntity, error) {
	if appID == "" || currentTenant == "" {
		return nil, ErrAppNotOwned
	}
	app, err := dao.NewApplicationDao().GetByID(ctx, appID)
	if err != nil {
		glog.Errorf(ctx, "[application.LoadOwnedApp] dao GetByID fail, err:%v, appID:%s", err, appID)
		return nil, err
	}
	if app == nil || app.ID == "" {
		return nil, ErrAppNotOwned
	}
	if app.OwnerType != model.AppOwnerTypeTenant || app.OwnerTenantID != currentTenant {
		return nil, ErrAppNotOwned
	}
	return app, nil
}

// ListOwnedApps 返回归属当前租户的全部启用应用（owner_type=tenant 且 owner_tenant_id==currentTenant）。
// 供租户侧菜单管理的应用选择器使用。
func ListOwnedApps(ctx context.Context, currentTenant string) ([]model.ApplicationEntity, error) {
	if currentTenant == "" {
		return nil, ErrNoCurrentTenantScope
	}
	list, _, err := dao.NewApplicationDao().GetPageListByCond(ctx, &dao.ApplicationCond{
		OwnerType:     model.AppOwnerTypeTenant,
		OwnerTenantID: currentTenant,
	})
	if err != nil {
		glog.Errorf(ctx, "[application.ListOwnedApps] dao GetPageListByCond fail, err:%v, tenantID:%s", err, currentTenant)
		return nil, err
	}
	return list, nil
}
