package svcaudit

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/dao"
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/glog"
)

const (
	ActionLogin                   = "login"
	ActionTenantSwitch            = "tenant.switch"
	ActionTenantCreate            = "tenant.create"
	ActionApplicationCreate       = "application.create"
	ActionOAuthClientCreate       = "oauth_client.create"
	ActionOAuthClientCreateSecret = "oauth_client.create_secret"
	ActionApiKeyCreate            = "api_key.create"
	ActionApiKeyRevoke            = "api_key.revoke"
)

type AuditEntry struct {
	Action     string
	TenantID   uint
	TargetType string
	TargetID   uint
	Result     string // success / failure
	Detail     string
	ClientID   string
}

var newAuditLogDao = func() *dao.AuditLogDao { return dao.NewAuditLogDao() }

func WriteAudit(ctx *gin.Context, e AuditEntry) {
	if ctx == nil || ctx.Request == nil {
		return
	}
	entity := &model.AuditLogEntity{
		ActorPersonID: gincontext.GetPersonID(ctx),
		ActorUserID:   gincontext.GetUserID(ctx),
		TenantID:      e.TenantID,
		ClientID:      e.ClientID,
		Action:        e.Action,
		TargetType:    e.TargetType,
		TargetID:      e.TargetID,
		Result:        e.Result,
		IP:            gincontext.GetClientIP(ctx),
		UserAgent:     ctx.GetHeader("User-Agent"),
		Detail:        e.Detail,
		CreatedBy:     gincontext.GetUserID(ctx),
	}
	if err := newAuditLogDao().Insert(context.Background(), entity); err != nil {
		glog.Errorf(ctx, "[svcaudit.WriteAudit] failed, action:%s, err:%v", e.Action, err)
	}
}
