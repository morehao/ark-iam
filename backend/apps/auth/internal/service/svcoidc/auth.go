package svcoidc

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/internal/dto/dtooidc"
	"github.com/morehao/ark-iam/auth/internal/oidcop"
	"github.com/morehao/ark-iam/auth/internal/service/svcauth"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/iam/object/objauth"
	"github.com/morehao/ark-iam/pkg/iam/sso"
	"github.com/morehao/ark-iam/pkg/iam/svcaudit"
	"github.com/morehao/golib/glog"
)

type OIDCAuthSvc interface {
	CompleteLogin(ctx *gin.Context, req *dtooidc.OIDCLoginReq) (*dtooidc.OIDCLoginResp, error)
	SelectTenant(ctx *gin.Context, authRequestID string, tenantID string) (*dtooidc.OIDCLoginResp, error)
	CompleteLoginBySession(ctx *gin.Context, authRequestID string, sessionID string) (string, error)
}

type passwordAuthenticator interface {
	AuthenticatePassword(ctx *gin.Context, identifier, password string) (*model.PersonEntity, *model.UserEntity, []objauth.TenantOption, error)
	TenantsForPerson(ctx *gin.Context, personID string) ([]objauth.TenantOption, error)
}

type oidcAuthSvc struct {
	provider             *OIDCProvider
	authSvc              passwordAuthenticator
	ssoSessionStore      sso.SSOSessionStore
	applicationClientDao func() *dao.ApplicationClientDao
	applicationDao       func() *dao.ApplicationDao
}

func NewOIDCAuthSvc(provider *OIDCProvider) OIDCAuthSvc {
	return &oidcAuthSvc{
		provider:             provider,
		authSvc:              svcauth.NewAuthSvc(),
		ssoSessionStore:      sso.NewSSOSessionStore(),
		applicationClientDao: func() *dao.ApplicationClientDao { return dao.NewApplicationClientDao() },
		applicationDao:       func() *dao.ApplicationDao { return dao.NewApplicationDao() },
	}
}

func (svc *oidcAuthSvc) CompleteLogin(ctx *gin.Context, req *dtooidc.OIDCLoginReq) (*dtooidc.OIDCLoginResp, error) {
	authReq, err := svc.provider.Storage.AuthRequestByID(ctx.Request.Context(), req.AuthRequestID)
	if err != nil {
		return nil, code.GetError(code.OIDCSessionNotFound)
	}
	personEntity, userEntity, tenants, err := svc.authSvc.AuthenticatePassword(ctx, req.Identifier, req.Password)
	if err != nil {
		return nil, err
	}
	authTime := time.Now()
	subject := oidcop.BuildSubject(personEntity.ID)
	// 优先尊重 ?tenant hint（如 SSO 会话过期后回退到密码登录时），但仅当 hint 是 person 的成员租户时才采用
	resolvedTenant := ""
	if ar, ok := authReq.(*oidcop.AuthRequest); ok {
		if hint := ar.GetTenantID(); hint != "" {
			for _, t := range tenants {
				if t.TenantID == hint {
					resolvedTenant = hint
					break
				}
			}
		}
	}
	// 多租户：除非有合法的 tenant hint，否则暂不 done、不发 code，需用户先选租户
	if resolvedTenant == "" && len(tenants) > 1 {
		if err := svc.provider.Storage.CompleteAuthRequest(ctx.Request.Context(), req.AuthRequestID, subject, authTime, []string{"pwd"}, "", "", false); err != nil {
			return nil, code.GetError(code.OIDCSessionNotFound)
		}
		return &dtooidc.OIDCLoginResp{
			RequiresTenantSelection: true,
			Tenants:                 tenants,
		}, nil
	}
	// 单租户（或在多租户但 hint 命中成员租户）：自动选租户，done，发 code，并建 SSO 会话
	tenantID := resolvedTenant
	if tenantID == "" {
		tenantID = userEntity.TenantID
	}
	if err := svc.provider.Storage.CompleteAuthRequest(ctx.Request.Context(), req.AuthRequestID, subject, authTime, []string{"pwd"}, "", tenantID, true); err != nil {
		return nil, code.GetError(code.OIDCSessionNotFound)
	}

	allowPersonCreateTenant := false
	if cid := clientIDFromAuthRequest(authReq); cid != "" {
		allowPersonCreateTenant = svc.resolveAllowPersonCreateTenant(ctx, cid, len(tenants))
	}

	resp := &dtooidc.OIDCLoginResp{
		ContinueURL:             svc.provider.BuildAuthCallbackURL(ctx.Request.Context(), req.AuthRequestID),
		TenantID:                tenantID,
		Tenants:                 tenants,
		AllowPersonCreateTenant: allowPersonCreateTenant,
	}

	if svc.ssoSessionStore != nil {
		sessionID, err := svc.ssoSessionStore.CreateSession(sessionAuditContext(ctx.Request.Context(), tenantID), personEntity.ID, []string{"pwd"})
		if err != nil {
			glog.Warnf(ctx, "[oidcAuthSvc.CompleteLogin] failed to create sso session: %v", err)
		} else {
			resp.SessionID = sessionID
			if aErr := svc.provider.Storage.AssociateSession(ctx.Request.Context(), req.AuthRequestID, sessionID); aErr != nil {
				glog.Warnf(ctx, "[oidcAuthSvc.CompleteLogin] associate session fail, err:%v, authRequestID:%s, sessionID:%s", aErr, req.AuthRequestID, sessionID)
			}
		}
	}

	return resp, nil
}

func (svc *oidcAuthSvc) SelectTenant(ctx *gin.Context, authRequestID string, tenantID string) (*dtooidc.OIDCLoginResp, error) {
	reqCtx := ctx.Request.Context()
	authReq, err := svc.provider.Storage.AuthRequestByID(reqCtx, authRequestID)
	if err != nil {
		return nil, code.GetError(code.OIDCSessionNotFound)
	}
	if authReq.Done() {
		return nil, code.GetError(code.OIDCSessionNotFound)
	}
	personID, perr := oidcop.ParseSubject(authReq.GetSubject())
	if perr != nil {
		return nil, code.GetError(code.OIDCSessionNotFound)
	}
	tenants, terr := svc.authSvc.TenantsForPerson(ctx, personID)
	if terr != nil {
		return nil, code.GetError(code.OIDCSessionNotFound)
	}
	ok := false
	for _, t := range tenants {
		if t.TenantID == tenantID {
			ok = true
			break
		}
	}
	if !ok {
		return nil, code.GetError(code.TenantNotExistError)
	}
	if err := svc.provider.Storage.CompleteAuthRequest(reqCtx, authRequestID, authReq.GetSubject(), authReq.GetAuthTime(), authReq.GetAMR(), "", tenantID, true); err != nil {
		return nil, code.GetError(code.OIDCSessionNotFound)
	}
	allowPersonCreateTenant := false
	if cid := authReq.GetClientID(); cid != "" {
		allowPersonCreateTenant = svc.resolveAllowPersonCreateTenant(ctx, cid, len(tenants))
	}
	resp := &dtooidc.OIDCLoginResp{
		ContinueURL:             svc.provider.BuildAuthCallbackURL(reqCtx, authRequestID),
		TenantID:                tenantID,
		Tenants:                 tenants,
		AllowPersonCreateTenant: allowPersonCreateTenant,
	}
	if svc.ssoSessionStore != nil {
		if sessionID, sErr := svc.ssoSessionStore.CreateSession(sessionAuditContext(reqCtx, tenantID), personID, authReq.GetAMR()); sErr == nil {
			resp.SessionID = sessionID
			if aErr := svc.provider.Storage.AssociateSession(reqCtx, authRequestID, sessionID); aErr != nil {
				glog.Warnf(ctx, "[oidcAuthSvc.SelectTenant] associate session fail, err:%v, authRequestID:%s, sessionID:%s", aErr, authRequestID, sessionID)
			}
		} else {
			glog.Warnf(ctx, "[oidcAuthSvc.SelectTenant] failed to create sso session: %v", sErr)
		}
	}
	svcaudit.WriteAudit(ctx, svcaudit.AuditEntry{
		Action:     svcaudit.ActionTenantSwitch,
		TenantID:   tenantID,
		Result:     "success",
		TargetType: "tenant",
		TargetID:   tenantID,
	})
	return resp, nil
}

func (svc *oidcAuthSvc) CompleteLoginBySession(ctx *gin.Context, authRequestID string, sessionID string) (string, error) {
	reqCtx := ctx.Request.Context()
	personID, err := svc.ssoSessionStore.ValidateSession(reqCtx, sessionID)
	if err != nil {
		return "", err
	}

	authReq, err := svc.provider.Storage.AuthRequestByID(reqCtx, authRequestID)
	if err != nil {
		return "", err
	}

	tenantID := ""
	tenants, tErr := svc.authSvc.TenantsForPerson(ctx, personID)
	if tErr == nil {
		if ar, ok := authReq.(*oidcop.AuthRequest); ok {
			if hint := ar.GetTenantID(); hint != "" {
				for _, t := range tenants {
					if t.TenantID == hint {
						tenantID = hint
						break
					}
				}
			}
		}
		// membership safety: never issue a token for a tenant hinted but not owned by the user
		if tenantID == "" && len(tenants) > 0 {
			tenantID = tenants[0].TenantID
		}
	}

	authTime := time.Now()
	// L7：amr 还原会话创建时的原始认证方法（如 ["pwd"]），不再使用非标准的 "sso"。
	amr := svc.ssoSessionStore.SessionAMR(reqCtx, sessionID)
	if len(amr) == 0 {
		amr = []string{"pwd"}
	}
	if err := svc.provider.Storage.CompleteAuthRequest(reqCtx, authRequestID, oidcop.BuildSubject(personID), authTime, amr, "", tenantID, true); err != nil {
		return "", err
	}

	svcaudit.WriteAudit(ctx, svcaudit.AuditEntry{
		Action:     svcaudit.ActionLogin,
		TenantID:   tenantID,
		Result:     "success",
		TargetType: "person",
		TargetID:   personID,
	})

	if aErr := svc.provider.Storage.AssociateSession(reqCtx, authRequestID, sessionID); aErr != nil {
		glog.Warnf(ctx, "[oidcAuthSvc.CompleteLoginBySession] associate session fail, err:%v, authRequestID:%s, sessionID:%s", aErr, authRequestID, sessionID)
	}

	return svc.provider.BuildAuthCallbackURL(reqCtx, authRequestID), nil
}
