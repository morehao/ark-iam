package svcoidc

import (
	"context"
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/internal/oidcop"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/iam/sso"
	"github.com/zitadel/oidc/v3/pkg/op"
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

// resolveAllowPersonCreateTenant reports whether the app backing the oauth client
// allows a zero-tenant person to self-create a tenant. Person with >=1 tenant => false.
func (svc *oidcAuthSvc) resolveAllowPersonCreateTenant(ctx *gin.Context, clientID string, tenantCount int) bool {
	if clientID == "" || tenantCount > 0 || svc.applicationClientDao == nil || svc.applicationDao == nil {
		return false
	}
	client, err := svc.applicationClientDao().GetByCond(ctx, &dao.ApplicationClientCond{Code: clientID})
	if err != nil || client == nil || client.AppID == "" {
		return false
	}
	app, err := svc.applicationDao().GetByID(ctx, client.AppID)
	if err != nil || app == nil || app.ID == "" {
		return false
	}
	var policy model.TenantPolicy
	if len(app.TenantPolicy) == 0 {
		return false
	}
	if err := json.Unmarshal(app.TenantPolicy, &policy); err != nil || policy.AllowPersonCreateTenant == nil {
		return false
	}
	return *policy.AllowPersonCreateTenant
}
