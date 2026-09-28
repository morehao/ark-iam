package svcoidc

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/zitadel/oidc/v3/pkg/op"

	"github.com/morehao/ark-iam/auth/internal/core/oidcop"
	"github.com/morehao/ark-iam/pkg/core/application"
	"github.com/morehao/ark-iam/pkg/object/objauth"
	"github.com/morehao/ark-iam/pkg/sso"
	"github.com/morehao/golib/glog"
)

// sessionAuditContext 将已解析的租户写入 context，供 CreateSession 落库 session 审计时读取 tenant_id。
func sessionAuditContext(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, sso.ContextKeyTenantID, tenantID)
}

func clientIDFromAuthRequest(authReq op.AuthRequest) string {
	if ar, ok := authReq.(*oidcop.AuthRequest); ok {
		return ar.GetClientID()
	}
	return ""
}

// appAccessEnforced 应用准入门禁是否开启（来源为持久化存储的开关，**默认关闭**）。
// 关闭时所有准入读取点直接放行，行为等价于门禁上线前（AC-13）；写入路径不受开关影响。
func (svc *oidcAuthSvc) appAccessEnforced() bool {
	return svc.provider != nil && svc.provider.Storage != nil && svc.provider.Storage.AppAccessEnforced()
}

// canUseClientApp 判定「该成员能否进入本次登录客户端所属的应用」，与令牌签发门禁共用
// pkg/core/application 的同一份判定式（订阅启用 ∧ 应用启用 ∧ 准入启用）。
// fail-closed：client_id 为空、客户端未绑定应用、应用行缺失一律 false；
// 查询失败如实上抛（系统错误不伪装成"无权限"）。
func (svc *oidcAuthSvc) canUseClientApp(ctx *gin.Context, clientID string, option objauth.TenantOption) (bool, error) {
	if !svc.appAccessEnforced() {
		return true, nil
	}
	appEntity, err := application.GetByClientID(ctx, clientID)
	if err != nil {
		glog.Errorf(ctx, "[oidcAuthSvc.canUseClientApp] resolve client app fail, clientID:%s, err:%v", clientID, err)
		return false, err
	}
	if appEntity == nil {
		glog.Warnf(ctx, "[oidcAuthSvc.canUseClientApp] client has no bound application, deny, clientID:%s", clientID)
		return false, nil
	}
	usable, err := application.CheckUsable(ctx, option.TenantID, option.UserID, appEntity.ID)
	if err != nil {
		glog.Errorf(ctx, "[oidcAuthSvc.canUseClientApp] check app access fail, clientID:%s, tenantID:%s, userID:%s, appID:%s, err:%v",
			clientID, option.TenantID, option.UserID, appEntity.ID, err)
		return false, err
	}
	return usable, nil
}

// filterTenantsByClientApp 只保留「该成员在本客户端所属应用下已被准入」的租户选项，
// 使用户不会在多租户选择里选中一个必然失败的租户。开关关闭时原样返回（不查库）。
func (svc *oidcAuthSvc) filterTenantsByClientApp(ctx *gin.Context, clientID string, tenants []objauth.TenantOption) ([]objauth.TenantOption, error) {
	if !svc.appAccessEnforced() {
		return tenants, nil
	}
	usableTenants := make([]objauth.TenantOption, 0, len(tenants))
	for _, option := range tenants {
		usable, err := svc.canUseClientApp(ctx, clientID, option)
		if err != nil {
			return nil, err
		}
		if usable {
			usableTenants = append(usableTenants, option)
		}
	}
	return usableTenants, nil
}

// appAllowsPersonCreateTenant 仅判定应用策略层面是否允许 person 注册/建租户（不看 person 租户数）。
// 供 RegisterPerson/CreateTenant 作为"注册入口是否开放"的门禁。
// client_id → 应用的解析与应用策略读取统一在 pkg/core/application，
// 与通道 B 的 AllowJoinByInvite 共用同一份实现（禁止各写一套）。
func (svc *oidcAuthSvc) appAllowsPersonCreateTenant(ctx *gin.Context, clientID string) bool {
	appEntity, err := application.GetByClientID(ctx, clientID)
	if err != nil {
		return false
	}
	return application.AllowsPersonCreateTenant(appEntity)
}

// resolveAllowPersonCreateTenant reports whether the app backing the oauth client
// allows a zero-tenant person to self-create a tenant. Person with >=1 tenant => false.
func (svc *oidcAuthSvc) resolveAllowPersonCreateTenant(ctx *gin.Context, clientID string, tenantCount int) bool {
	if clientID == "" || tenantCount > 0 {
		return false
	}
	return svc.appAllowsPersonCreateTenant(ctx, clientID)
}
