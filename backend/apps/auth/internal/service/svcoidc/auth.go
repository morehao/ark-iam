package svcoidc

import (
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/internal/core/oidcop"
	"github.com/morehao/ark-iam/auth/internal/dto/dtooidc"
	"github.com/morehao/ark-iam/auth/internal/service/svcauth"
	"github.com/morehao/ark-iam/pkg/audit"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/object/objauth"
	"github.com/morehao/ark-iam/pkg/sso"
	"github.com/morehao/golib/glog"
)

type OIDCAuthSvc interface {
	CompleteLogin(ctx *gin.Context, req *dtooidc.OIDCLoginReq) (*dtooidc.OIDCLoginResp, error)
	SelectTenant(ctx *gin.Context, authRequestID string, tenantID string) (*dtooidc.OIDCLoginResp, error)
	CompleteLoginBySession(ctx *gin.Context, authRequestID string, sessionID string) (string, error)
	RegisterPerson(ctx *gin.Context, req *dtooidc.RegisterPersonReq) (*dtooidc.RegisterPersonResp, error)
	CreateTenant(ctx *gin.Context, req *dtooidc.CreateTenantReq) (*dtooidc.CreateTenantResp, error)
	ChangePassword(ctx *gin.Context, req *dtooidc.OIDCChangePasswordReq) error
	LoginConfig(ctx *gin.Context, authRequestID string) (*dtooidc.OIDCLoginConfigResp, error)
}

// errSilentLoginNotAllowed 标记静默登录被拒（prompt=login / max_age 超期），
// 由 SSOLogin 处理为清除 SSO cookie 并跳回登录页重新认证。
var errSilentLoginNotAllowed = errors.New("silent login not allowed")

// ErrSilentLoginNoAppAccess 标记静默登录被拒且原因是「本客户端所属应用下没有任何有准入的租户」。
// 与 errSilentLoginNotAllowed 刻意分开：后者清 cookie 回登录页是正确处置（重新认证可能通过），
// 而准入缺失重新登录仍是同一结论，若同样回登录页会形成登录死循环（AC-5）。
// SSOLogin 用 errors.Is 识别本哨兵并直接渲染无权限提示页，不动 SSO 会话。
var ErrSilentLoginNoAppAccess = errors.New("silent login: no tenant grants access to client app")

// writeAppAccessDeniedAudit 记录一次「因应用准入被拒」的登录尝试（best-effort）。
//
// 为什么必须落审计而不只打日志：AC-4 要求"过滤后为空时有明确文案 + 审计"，且灰度 P3 的退出
// 判据是"准入拒绝次数持平于预期"——该指标的唯一可查来源就是这张表（detail 里带
// reason:no app access，可按动作 + 结果 + detail 聚合）。写库失败由 audit.WriteAudit 自行兜底，
// 不影响拒绝结果本身。
func writeAppAccessDeniedAudit(ctx *gin.Context, personID, clientID, tenantID string) {
	audit.WriteAudit(ctx, audit.AuditEntry{
		Action:     audit.ActionLogin,
		TenantID:   tenantID,
		Result:     model.AuditResultFailure,
		TargetType: model.AuditTargetTypePerson,
		TargetID:   personID,
		ClientID:   clientID,
		Detail:     fmt.Sprintf("personID:%s, clientID:%s, reason:no app access", personID, clientID),
	})
}

// mapAuthRequestError 区分"授权票据不存在"与"存储故障"：
// 前者返回 OIDCSessionNotFound（客户端可重试登录），后者返回 OIDCServerError（服务端问题）。
func mapAuthRequestError(err error) error {
	if errors.Is(err, oidcop.ErrStoreUnavailable) {
		return code.GetError(code.OIDCServerError)
	}
	return code.GetError(code.OIDCSessionNotFound)
}

// containsPrompt 判断 prompt 参数是否包含指定值。
// findTenantOption 在租户选项集合中定位指定租户。
func findTenantOption(tenants []objauth.TenantOption, tenantID string) (objauth.TenantOption, bool) {
	for _, option := range tenants {
		if option.TenantID == tenantID {
			return option, true
		}
	}
	return objauth.TenantOption{}, false
}

func containsPrompt(prompts []string, target string) bool {
	for _, p := range prompts {
		if p == target {
			return true
		}
	}
	return false
}

type passwordAuthenticator interface {
	AuthenticatePassword(ctx *gin.Context, identifier, password string) (*model.PersonEntity, *model.UserEntity, []objauth.TenantOption, error)
	TenantsForPerson(ctx *gin.Context, personID string) ([]objauth.TenantOption, error)
}

type oidcAuthSvc struct {
	provider        *OIDCProvider
	authSvc         passwordAuthenticator
	ssoSessionStore sso.SSOSessionStore
}

func NewOIDCAuthSvc(provider *OIDCProvider) OIDCAuthSvc {
	return &oidcAuthSvc{
		provider:        provider,
		authSvc:         svcauth.NewAuthSvc(),
		ssoSessionStore: sso.NewSSOSessionStore(),
	}
}

func (svc *oidcAuthSvc) CompleteLogin(ctx *gin.Context, req *dtooidc.OIDCLoginReq) (*dtooidc.OIDCLoginResp, error) {
	authReq, err := svc.provider.Storage.AuthRequestByID(ctx, req.AuthRequestID)
	if err != nil {
		return nil, mapAuthRequestError(err)
	}
	personEntity, userEntity, tenants, err := svc.authSvc.AuthenticatePassword(ctx, req.Identifier, req.Password)
	if err != nil {
		return nil, err
	}
	// 首次登录强制改密（D3）：持临时密码的自然人（password_status=must_change）一律先改密，
	// 不完成授权（done=false）、不发 code、不建 SSO 会话，只把 subject 绑到授权票据，
	// 供 POST /oidc/login/changePassword 识别身份。该判定刻意放在租户分支之前：
	// 无论零租户、单租户还是多租户，"临时密码必须先改"都成立。
	//
	// 由此得到一个不变式：SSO 会话只可能建立在已完成密码登录之上，而完成登录要求
	// password_status=normal，因此无需在 SSO/静默登录路径重复该判定。
	if personEntity != nil && personEntity.PasswordStatus == model.PasswordStatusMustChange {
		if cErr := svc.provider.Storage.CompleteAuthRequest(ctx, req.AuthRequestID,
			oidcop.BuildSubject(personEntity.ID), time.Now(), []string{"pwd"}, "", "", false); cErr != nil {
			return nil, mapAuthRequestError(cErr)
		}
		return &dtooidc.OIDCLoginResp{
			PersonID:               personEntity.ID,
			RequiresPasswordChange: true,
			Tenants:                tenants,
		}, nil
	}
	// 零租户已验密 person（person 存在但无租户成员）：绑定 authRequest(done=false)，
	// 返回"可建租户"信号，不发 code、不建 SSO 会话（无租户上下文无法发 token）。
	// 在访问 userEntity.TenantID 之前返回，避免 nil 解引用 panic。
	if personEntity != nil && userEntity == nil && len(tenants) == 0 {
		if cErr := svc.provider.Storage.CompleteAuthRequest(ctx, req.AuthRequestID,
			oidcop.BuildSubject(personEntity.ID), time.Now(), []string{"pwd"}, "", "", false); cErr != nil {
			return nil, mapAuthRequestError(cErr)
		}
		allow := false
		if cid := clientIDFromAuthRequest(authReq); cid != "" {
			allow = svc.appAllowsPersonCreateTenant(ctx, cid)
		}
		return &dtooidc.OIDCLoginResp{
			PersonID:                personEntity.ID,
			Tenants:                 []objauth.TenantOption{},
			AllowPersonCreateTenant: allow,
		}, nil
	}
	// 应用准入（P5-4/P5-5）：只把「本次登录客户端所属应用下已被准入」的租户呈现给用户，
	// 避免选中一个必然失败的租户；一个都没有时当场早拒（不发 code、不建 SSO 会话）。
	clientID := clientIDFromAuthRequest(authReq)
	usableTenants, fErr := svc.filterTenantsByClientApp(ctx, clientID, tenants)
	if fErr != nil {
		glog.Errorf(ctx, "[oidcAuthSvc.CompleteLogin] filter tenants by app access fail, err:%v, clientID:%s", fErr, clientID)
		return nil, code.GetError(code.OIDCServerError)
	}
	if svc.appAccessEnforced() && len(usableTenants) == 0 {
		glog.Warnf(ctx, "[oidcAuthSvc.CompleteLogin] no tenant grants access to client app, clientID:%s, tenantCount:%d", clientID, len(tenants))
		writeAppAccessDeniedAudit(ctx, personEntity.ID, clientID, "")
		return nil, code.GetError(code.AuthAppAccessDeniedError)
	}
	tenants = usableTenants

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
		if err := svc.provider.Storage.CompleteAuthRequest(ctx, req.AuthRequestID, subject, authTime, []string{"pwd"}, "", "", false); err != nil {
			return nil, mapAuthRequestError(err)
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
	// 落地租户必须在本应用下有准入：单租户场景上面已过滤，hint 场景已收窄到可用集合，
	// 但 userEntity.TenantID 可能指向可用集合之外的租户（多成员关系下取"当前成员行"），故再点查一次。
	if svc.appAccessEnforced() {
		option, ok := findTenantOption(usableTenants, tenantID)
		if !ok {
			glog.Warnf(ctx, "[oidcAuthSvc.CompleteLogin] resolved tenant has no app access, clientID:%s, tenantID:%s", clientID, tenantID)
			writeAppAccessDeniedAudit(ctx, personEntity.ID, clientID, tenantID)
			return nil, code.GetError(code.AuthAppAccessDeniedError)
		}
		usable, aErr := svc.canUseClientApp(ctx, clientID, option)
		if aErr != nil {
			glog.Errorf(ctx, "[oidcAuthSvc.CompleteLogin] check app access fail, err:%v, clientID:%s, tenantID:%s", aErr, clientID, tenantID)
			return nil, code.GetError(code.OIDCServerError)
		}
		if !usable {
			writeAppAccessDeniedAudit(ctx, personEntity.ID, clientID, tenantID)
			return nil, code.GetError(code.AuthAppAccessDeniedError)
		}
	}
	if err := svc.provider.Storage.CompleteAuthRequest(ctx, req.AuthRequestID, subject, authTime, []string{"pwd"}, "", tenantID, true); err != nil {
		return nil, mapAuthRequestError(err)
	}
	// 补记密码登录主路径的审计（SelectTenant / CompleteLoginBySession 均已记录）
	audit.WriteAudit(ctx, audit.AuditEntry{
		Action:     audit.ActionLogin,
		TenantID:   tenantID,
		Result:     model.AuditResultSuccess,
		TargetType: model.AuditTargetTypePerson,
		TargetID:   personEntity.ID,
	})
	// 登录日志（T4）：授权票据已 done=true（即将签发授权码）才算成功，此处是「授权码登录」的落点。
	// best-effort，写失败不影响登录（详见 recordOIDCLoginLog）。
	recordOIDCLoginLog(ctx, tenantID, userEntity.ID, personEntity.ID)

	allowPersonCreateTenant := false
	if cid := clientIDFromAuthRequest(authReq); cid != "" {
		allowPersonCreateTenant = svc.resolveAllowPersonCreateTenant(ctx, cid, len(tenants))
	}

	resp := &dtooidc.OIDCLoginResp{
		ContinueURL:             svc.provider.BuildAuthCallbackURL(ctx, req.AuthRequestID),
		TenantID:                tenantID,
		Tenants:                 tenants,
		AllowPersonCreateTenant: allowPersonCreateTenant,
	}

	if svc.ssoSessionStore != nil {
		sessionID, err := svc.ssoSessionStore.CreateSession(sessionAuditContext(ctx, tenantID), personEntity.ID, []string{"pwd"})
		if err != nil {
			glog.Warnf(ctx, "[oidcAuthSvc.CompleteLogin] failed to create sso session: %v", err)
		} else {
			resp.SessionID = sessionID
			if aErr := svc.provider.Storage.AssociateSession(ctx, req.AuthRequestID, sessionID); aErr != nil {
				glog.Warnf(ctx, "[oidcAuthSvc.CompleteLogin] associate session fail, err:%v, authRequestID:%s, sessionID:%s", aErr, req.AuthRequestID, sessionID)
			}
		}
	}

	return resp, nil
}

func (svc *oidcAuthSvc) SelectTenant(ctx *gin.Context, authRequestID string, tenantID string) (*dtooidc.OIDCLoginResp, error) {
	reqCtx := ctx
	authReq, err := svc.provider.Storage.AuthRequestByID(reqCtx, authRequestID)
	if err != nil {
		return nil, mapAuthRequestError(err)
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
	// 应用准入（P5-4）：所选租户必须在本客户端所属应用下被准入，否则早拒（不发 code、不建 SSO 会话）。
	// 每次选择都重新判定——租户选项可能是登录页较早渲染的旧列表。
	if svc.appAccessEnforced() {
		option, found := findTenantOption(tenants, tenantID)
		if !found {
			return nil, code.GetError(code.TenantNotExistError)
		}
		usable, aErr := svc.canUseClientApp(ctx, authReq.GetClientID(), option)
		if aErr != nil {
			glog.Errorf(ctx, "[oidcAuthSvc.SelectTenant] check app access fail, err:%v, clientID:%s, tenantID:%s", aErr, authReq.GetClientID(), tenantID)
			return nil, code.GetError(code.OIDCServerError)
		}
		if !usable {
			glog.Warnf(ctx, "[oidcAuthSvc.SelectTenant] tenant has no app access, clientID:%s, tenantID:%s", authReq.GetClientID(), tenantID)
			writeAppAccessDeniedAudit(ctx, personID, authReq.GetClientID(), tenantID)
			return nil, code.GetError(code.AuthAppAccessDeniedError)
		}
	}
	if err := svc.provider.Storage.CompleteAuthRequest(reqCtx, authRequestID, authReq.GetSubject(), authReq.GetAuthTime(), authReq.GetAMR(), "", tenantID, true); err != nil {
		return nil, mapAuthRequestError(err)
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
	audit.WriteAudit(ctx, audit.AuditEntry{
		Action:     audit.ActionTenantSwitch,
		TenantID:   tenantID,
		Result:     model.AuditResultSuccess,
		TargetType: model.AuditTargetTypeTenant,
		TargetID:   tenantID,
	})
	// 登录日志（T4）：多租户登录由 CompleteLogin(done=false) + 本方法共同完成，成功的落点在
	// 这里（上一步校验过 tenantID 必在 tenants 内，故 UserID 必然可得）。
	// CompleteLogin 在该分支刻意不写，避免同一次登录重复落账。
	selected, _ := findTenantOption(tenants, tenantID)
	recordOIDCLoginLog(ctx, tenantID, selected.UserID, personID)
	return resp, nil
}

func (svc *oidcAuthSvc) CompleteLoginBySession(ctx *gin.Context, authRequestID string, sessionID string) (string, error) {
	reqCtx := ctx
	personID, err := svc.ssoSessionStore.ValidateSession(reqCtx, sessionID)
	if err != nil {
		return "", err
	}

	authReq, err := svc.provider.Storage.AuthRequestByID(reqCtx, authRequestID)
	if err != nil {
		return "", err
	}

	tenantID := ""
	loginUserID := ""
	tenants, tErr := svc.authSvc.TenantsForPerson(ctx, personID)
	if tErr == nil {
		clientID := clientIDFromAuthRequest(authReq)
		// 应用准入（P5-6）：只在「本客户端所属应用下有准入」的租户里选，避免静默登录落到
		// 一个必然被令牌门禁拒绝的租户；一个都没有时返回专用哨兵，由 SSOLogin 渲染无权限页。
		usableTenants, fErr := svc.filterTenantsByClientApp(ctx, clientID, tenants)
		if fErr != nil {
			glog.Errorf(ctx, "[oidcAuthSvc.CompleteLoginBySession] filter tenants by app access fail, err:%v, clientID:%s", fErr, clientID)
			return "", fErr
		}
		if svc.appAccessEnforced() && len(usableTenants) == 0 {
			glog.Warnf(ctx, "[oidcAuthSvc.CompleteLoginBySession] no tenant grants access to client app, clientID:%s, personID:%s", clientID, personID)
			writeAppAccessDeniedAudit(ctx, personID, clientID, "")
			return "", ErrSilentLoginNoAppAccess
		}
		if ar, ok := authReq.(*oidcop.AuthRequest); ok {
			if hint := ar.GetTenantID(); hint != "" {
				for _, t := range usableTenants {
					if t.TenantID == hint {
						tenantID = hint
						loginUserID = t.UserID
						break
					}
				}
			}
		}
		// membership safety: never issue a token for a tenant hinted but not owned by the user
		if tenantID == "" && len(usableTenants) > 0 {
			tenantID = usableTenants[0].TenantID
			loginUserID = usableTenants[0].UserID
		}
	}

	authTime := svc.ssoSessionStore.SessionAuthTime(reqCtx, sessionID)
	if authTime.IsZero() {
		authTime = time.Now()
	}
	// H5：OIDC Core §3.1.2.1 —— prompt=login 强制重新认证；max_age 超期后不得静默续登。
	// 任一命中即拒绝静默完成，由 SSOLogin 清 cookie 并跳转登录页重新认证。
	if ar, ok := authReq.(*oidcop.AuthRequest); ok {
		if containsPrompt(ar.Prompt, "login") {
			glog.Warnf(ctx, "[oidcAuthSvc.CompleteLoginBySession] prompt=login requires re-auth, authRequestID:%s", authRequestID)
			return "", errSilentLoginNotAllowed
		}
		if ar.MaxAge != nil && authTime.Add(time.Duration(*ar.MaxAge)*time.Second).Before(time.Now()) {
			glog.Warnf(ctx, "[oidcAuthSvc.CompleteLoginBySession] max_age exceeded, re-auth required, authRequestID:%s", authRequestID)
			return "", errSilentLoginNotAllowed
		}
	}
	// L7：amr 还原会话创建时的原始认证方法（如 ["pwd"]），不再使用非标准的 "sso"。
	amr := svc.ssoSessionStore.SessionAMR(reqCtx, sessionID)
	if len(amr) == 0 {
		amr = []string{"pwd"}
	}
	if err := svc.provider.Storage.CompleteAuthRequest(reqCtx, authRequestID, oidcop.BuildSubject(personID), authTime, amr, "", tenantID, true); err != nil {
		return "", err
	}

	audit.WriteAudit(ctx, audit.AuditEntry{
		Action:     audit.ActionLogin,
		TenantID:   tenantID,
		Result:     model.AuditResultSuccess,
		TargetType: model.AuditTargetTypePerson,
		TargetID:   personID,
	})

	// 登录日志（T4）：prompt=none 静默 SSO 是货真价实的登录成功（票据已 done=true），
	// 且这条路径没有密码校验、上游不会写 password 行，故必须在此补记。
	recordOIDCLoginLog(ctx, tenantID, loginUserID, personID)

	if aErr := svc.provider.Storage.AssociateSession(reqCtx, authRequestID, sessionID); aErr != nil {
		glog.Warnf(ctx, "[oidcAuthSvc.CompleteLoginBySession] associate session fail, err:%v, authRequestID:%s, sessionID:%s", aErr, authRequestID, sessionID)
	}

	return svc.provider.BuildAuthCallbackURL(reqCtx, authRequestID), nil
}
