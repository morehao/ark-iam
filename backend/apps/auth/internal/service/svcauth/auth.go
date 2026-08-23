package svcauth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/internal/dto/dtoauth"
	"github.com/morehao/ark-iam/auth/internal/service/svcloginguard"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/iam/audit"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/iam/object/objauth"
	"github.com/morehao/ark-iam/pkg/iam/password"
	"github.com/morehao/ark-iam/pkg/iam/sso"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/gconstant"
	"github.com/morehao/golib/gcrypto"
	"github.com/morehao/golib/glog"
	"gorm.io/gorm"
)

type authUserStore interface {
	GetByID(ctx context.Context, id string) (*model.UserEntity, error)
	GetByCond(ctx context.Context, cond gormdao.Cond) (*model.UserEntity, error)
	GetListByCond(ctx context.Context, cond gormdao.Cond) (model.UserEntityList, error)
	Insert(ctx context.Context, entity *model.UserEntity) error
	UpdateMap(ctx context.Context, id string, updateMap map[string]any) error
}

type authPersonStore interface {
	GetByID(ctx context.Context, id string) (*model.PersonEntity, error)
	GetByCond(ctx context.Context, cond gormdao.Cond) (*model.PersonEntity, error)
	Insert(ctx context.Context, entity *model.PersonEntity) error
}

type authTenantStore interface {
	GetByID(ctx context.Context, id string) (*model.TenantEntity, error)
	GetPageListByCond(ctx context.Context, cond gormdao.Cond) (model.TenantEntityList, int64, error)
	GetListByCond(ctx context.Context, cond gormdao.Cond) (model.TenantEntityList, error)
}

type authRefreshTokenStore interface {
	GetByCond(ctx context.Context, cond gormdao.Cond) (*model.RefreshTokenEntity, error)
	Insert(ctx context.Context, entity *model.RefreshTokenEntity) error
	Delete(ctx context.Context, id, userID string) error
	RevokeByPersonID(ctx context.Context, personID string) error
}

var newAuthUserStore = func() authUserStore {
	return dao.NewUserDao()
}

var newAuthPersonStore = func() authPersonStore {
	return dao.NewPersonDao()
}

var newAuthTenantStore = func() authTenantStore {
	return dao.NewTenantDao()
}

var newAuthRefreshTokenStore = func() authRefreshTokenStore {
	return dao.NewRefreshTokenDao()
}

var authLoginRecorder = func(ctx *gin.Context, tenantID, userID string, success bool) {
	defaultRecordLoginLog(ctx, tenantID, userID, success)
}

type AuthSvc interface {
	AuthenticatePassword(ctx *gin.Context, identifier, password string) (*model.PersonEntity, *model.UserEntity, []objauth.TenantOption, error)
	TenantsForPerson(ctx *gin.Context, personID string) ([]objauth.TenantOption, error)
	MyTenants(ctx *gin.Context, req *dtoauth.MyTenantsReq) (*dtoauth.MyTenantsResp, error)
	JoinTenant(ctx *gin.Context, req *dtoauth.JoinTenantReq) (*dtoauth.JoinTenantResp, error)
	Logout(ctx *gin.Context, req *dtoauth.LogoutReq) error
	LogoutAll(ctx *gin.Context, req *dtoauth.LogoutAllReq) error
	Userinfo(ctx *gin.Context, req *dtoauth.UserinfoReq) (*dtoauth.UserinfoResp, error)
}

type authSvc struct {
}

var _ AuthSvc = (*authSvc)(nil)

func NewAuthSvc() AuthSvc {
	return &authSvc{}
}

func (svc *authSvc) AuthenticatePassword(ctx *gin.Context, identifier, password string) (*model.PersonEntity, *model.UserEntity, []objauth.TenantOption, error) {
	personDao := newAuthPersonStore()
	personEntity, userEntity, tenants, err := svc.resolvePersonLogin(ctx, personDao, identifier)
	if err != nil {
		return nil, nil, nil, err
	}
	personEntity, userEntity, err = svc.authenticateResolvedPerson(ctx, personEntity, userEntity, password)
	if err != nil {
		return nil, nil, nil, err
	}
	return personEntity, userEntity, tenants, nil
}

func (svc *authSvc) TenantsForPerson(ctx *gin.Context, personID string) ([]objauth.TenantOption, error) {
	_, tenants, err := svc.listPersonTenants(ctx, personID)
	if err != nil {
		return nil, err
	}
	return tenants, nil
}

func (svc *authSvc) authenticateResolvedPerson(ctx *gin.Context, personEntity *model.PersonEntity, userEntity *model.UserEntity, password string) (*model.PersonEntity, *model.UserEntity, error) {
	ip := gincontext.GetClientIP(ctx)
	if svcloginguard.Check(ctx, ip, personEntity.ID) {
		audit.WriteAudit(ctx, audit.AuditEntry{
			Action:     audit.ActionLogin,
			Result:     "failure",
			TargetType: "person",
			Detail:     fmt.Sprintf("personID:%s, reason:login locked", personEntity.ID),
		})
		return nil, nil, code.GetError(code.LoginLockedError)
	}

	if personEntity.IsSuspended {
		audit.WriteAudit(ctx, audit.AuditEntry{
			Action:     audit.ActionLogin,
			Result:     "failure",
			TargetType: "person",
			Detail:     fmt.Sprintf("personID:%s, reason:suspended", personEntity.ID),
		})
		return nil, nil, code.GetError(code.UserSuspendedError)
	}

	if personEntity.PasswordEncrypted == "" {
		audit.WriteAudit(ctx, audit.AuditEntry{
			Action:     audit.ActionLogin,
			Result:     "failure",
			TargetType: "person",
			Detail:     fmt.Sprintf("personID:%s, reason:password not set", personEntity.ID),
		})
		return nil, nil, code.GetError(code.PasswordNotSetError)
	}

	if err := gcrypto.ComparePasswordHash(personEntity.PasswordEncrypted, password); err != nil {
		authLoginRecorder(ctx, userEntity.TenantID, userEntity.ID, false)
		svcloginguard.RecordFailure(ctx, ip, personEntity.ID)
		glog.Errorf(ctx, "[svcauth.authenticateResolvedPerson] password mismatch, personID:%s", personEntity.ID)
		// H8：密码错误与用户不存在统一错误码，避免用户名枚举
		return nil, nil, code.GetError(code.AuthLoginFailedError)
	}

	svcloginguard.RecordSuccess(ctx, ip, personEntity.ID)
	authLoginRecorder(ctx, userEntity.TenantID, userEntity.ID, true)
	return personEntity, userEntity, nil
}

func (svc *authSvc) MyTenants(ctx *gin.Context, req *dtoauth.MyTenantsReq) (*dtoauth.MyTenantsResp, error) {
	personID := gincontext.GetPersonIDString(ctx)
	if personID == "" {
		return nil, code.GetError(gconstant.UnauthorizedErr)
	}

	_, tenants, err := svc.listPersonTenants(ctx, personID)
	if err != nil {
		return nil, err
	}

	return &dtoauth.MyTenantsResp{List: tenants}, nil
}

func (svc *authSvc) JoinTenant(ctx *gin.Context, req *dtoauth.JoinTenantReq) (*dtoauth.JoinTenantResp, error) {
	// 通道 B：凭邀请加入已有租户（非 owner）。
	// 落哪个租户由邀请码决定（租户侧授权，禁止裸 tenantID 直入）。
	personID := gincontext.GetPersonIDString(ctx)
	if personID == "" {
		return nil, code.GetError(gconstant.UnauthorizedErr)
	}
	if req.InviteCode == "" {
		return nil, code.GetError(code.AuthJoinNotAllowedError)
	}

	// 1. 解析邀请
	inviteEntity, err := dao.NewInviteDao().GetByCond(ctx.Request.Context(), &dao.InviteCond{Code: req.InviteCode})
	if err != nil {
		glog.Errorf(ctx, "[svcauth.JoinTenant] invite dao GetByCond fail, err:%v, code:%s", err, req.InviteCode)
		return nil, code.GetError(code.InviteGetDetailError)
	}
	if inviteEntity == nil || inviteEntity.ID == "" || inviteEntity.Status != model.InviteStatusPending {
		return nil, code.GetError(code.InviteInvalidError)
	}
	if inviteEntity.ExpiresAt != nil && time.Now().After(*inviteEntity.ExpiresAt) {
		return nil, code.GetError(code.InviteExpiredError)
	}
	tenantID := inviteEntity.TenantID

	userDao := newAuthUserStore()
	existingUser, err := userDao.GetByCond(ctx.Request.Context(), &dao.UserCond{PersonID: personID, TenantID: tenantID})
	if err != nil {
		glog.Errorf(ctx, "[svcauth.JoinTenant] user dao GetByCond fail, err:%v", err)
		return nil, code.GetError(code.UserGetDetailError)
	}
	if existingUser != nil && existingUser.ID != "" {
		return nil, code.GetError(code.AlreadyJoinedError)
	}

	// 2. 建成员 user（非 owner）+ 标记邀请已用
	var userID string
	txErr := dbclient.IamDB(ctx.Request.Context()).Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		userEntity := &model.UserEntity{
			TenantID:   tenantID,
			PersonID:   personID,
			Name:       "",
			Profile:    json.RawMessage(`{}`),
			CustomData: json.RawMessage(`{}`),
			IsOwner:    false,
			JoinedAt:   &now,
			CreatedBy:  personID,
		}
		if uErr := dao.NewUserDao().WithTx(tx).Insert(ctx.Request.Context(), userEntity); uErr != nil {
			return uErr
		}
		userID = userEntity.ID
		// 标记邀请已使用
		if uErr := dao.NewInviteDao().WithTx(tx).UpdateMap(ctx.Request.Context(), inviteEntity.ID, map[string]any{
			"status":     string(model.InviteStatusAccepted),
			"updated_by": personID,
		}); uErr != nil {
			return uErr
		}
		return nil
	})
	if txErr != nil {
		glog.Errorf(ctx, "[svcauth.JoinTenant] transaction fail, err:%v, code:%s", txErr, req.InviteCode)
		return nil, code.GetError(code.UserCreateError)
	}

	return &dtoauth.JoinTenantResp{
		UserID: userID,
	}, nil
}

func (svc *authSvc) Logout(ctx *gin.Context, req *dtoauth.LogoutReq) error {
	personID := gincontext.GetPersonIDString(ctx)
	// 全局登出语义：撤销该 person 的全部 refresh token + SSO 会话，实现"一处登出、处处登出"。
	// access token 依赖其短 TTL 失效（见设计文档 §2.5），此处不维护 HS256 黑名单。
	if personID != "" {
		// H13：登出动作记录审计
		audit.WriteAudit(ctx, audit.AuditEntry{
			Action:     audit.ActionLogout,
			Result:     "success",
			TargetType: "person",
			TargetID:   personID,
		})
		if err := newAuthRefreshTokenStore().RevokeByPersonID(ctx.Request.Context(), personID); err != nil {
			glog.Errorf(ctx, "[svcauth.Logout] RevokeByPersonID fail, personID:%s, err:%v", personID, err)
		}
		// 先入队 back-channel 通知（ListByPersonID 依赖 sso_user_sessions 索引），再撤销 SSO 会话。
		svc.enqueueBackChannelLogouts(ctx, personID)
		if err := sso.RevokeSSOSessionsByPersonID(ctx.Request.Context(), personID); err != nil {
			glog.Errorf(ctx, "[svcauth.Logout] RevokeSSOSessionsByPersonID fail, personID:%s, err:%v", personID, err)
		}
	}
	return nil
}

// enqueueBackChannelLogouts 将该 person 已登记的全部 client 的 back-channel logout
// 任务入队（在撤销 SSO 会话前调用，此时会话索引仍可用），使其它已登录应用（含第三方 RP）
// 即时收到 logout_token。入队失败仅告警，不影响登出主流程；任务由 oidcop 的 logoutWorker 异步消费。
func (svc *authSvc) enqueueBackChannelLogouts(ctx *gin.Context, personID string) {
	slo := sso.NewSLOStore()
	regs, err := slo.ListByPersonID(ctx.Request.Context(), personID)
	if err != nil {
		glog.Warnf(ctx, "[svcauth.Logout] list logout registrations fail, personID:%s, err:%v", personID, err)
		return
	}
	for _, reg := range regs {
		if err := sso.EnqueueLogout(ctx.Request.Context(), sso.LogoutJob{
			SessionID:            reg.SessionID,
			PersonID:             personID,
			OIDCSessionID:        reg.OIDCSessionID,
			ClientID:             reg.ClientID,
			UserID:               reg.UserID,
			BackChannelLogoutURI: reg.BackChannelLogoutURI,
		}); err != nil {
			glog.Warnf(ctx, "[svcauth.Logout] enqueue back-channel logout fail, clientID:%s, err:%v", reg.ClientID, err)
		}
	}
}

func (svc *authSvc) LogoutAll(ctx *gin.Context, req *dtoauth.LogoutAllReq) error {
	// LogoutAll 与 Logout 语义一致（person 级全局登出）
	return svc.Logout(ctx, &dtoauth.LogoutReq{})
}

func (svc *authSvc) Userinfo(ctx *gin.Context, req *dtoauth.UserinfoReq) (*dtoauth.UserinfoResp, error) {
	userDao := newAuthUserStore()

	userID := gincontext.GetUserIDString(ctx)
	personID := gincontext.GetPersonIDString(ctx)
	tenantID := gincontext.GetTenantIDString(ctx)

	var userEntity *model.UserEntity
	var err error

	if userID != "" {
		userEntity, err = userDao.GetByID(ctx.Request.Context(), userID)
	} else if personID != "" && tenantID != "" {
		userEntity, err = userDao.GetByCond(ctx.Request.Context(), &dao.UserCond{PersonID: personID, TenantID: tenantID})
	} else {
		return nil, code.GetError(gconstant.UnauthorizedErr)
	}

	if err != nil {
		glog.Errorf(ctx, "[svcauth.Userinfo] dao query fail, err:%v, userID:%s, personID:%s, tenantID:%s", err, userID, personID, tenantID)
		return nil, code.GetError(code.UserGetDetailError)
	}
	if userEntity == nil || userEntity.ID == "" {
		return nil, code.GetError(code.UserNotExistError)
	}

	if personID == "" {
		personID = userEntity.PersonID
	}

	personInfo := objauth.PersonInfo{}
	if personID != "" {
		personDao := newAuthPersonStore()
		personEntity, personErr := personDao.GetByID(ctx.Request.Context(), personID)
		if personErr != nil {
			// 不再静默吞错：DB 故障时如实报错，避免返回残缺 personInfo 误导调用方
			glog.Errorf(ctx, "[svcauth.Userinfo] person dao GetByID fail, err:%v, personID:%s", personErr, personID)
			return nil, code.GetError(code.UserGetDetailError)
		}
		if personEntity != nil && personEntity.ID != "" {
			personInfo = objauth.PersonInfo{
				PersonID: personEntity.ID,
				Name:     personEntity.Name,
				Avatar:   personEntity.Avatar,
			}
		} else {
			personInfo = objauth.PersonInfo{
				PersonID: personID,
			}
		}
	}

	return &dtoauth.UserinfoResp{
		PersonInfo: personInfo,
		UserInfo: objauth.TenantUserInfo{
			UserID:   userEntity.ID,
			TenantID: userEntity.TenantID,
			Name:     userEntity.Name,
			IsOwner:  userEntity.IsOwner,
		},
	}, nil
}

func (svc *authSvc) resolvePersonLogin(ctx *gin.Context, personDao authPersonStore, identifier string) (*model.PersonEntity, *model.UserEntity, []objauth.TenantOption, error) {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return nil, nil, nil, code.GetError(code.AuthIdentifierRequiredError)
	}

	personCond := &dao.PersonCond{}
	if strings.Contains(identifier, "@") {
		personCond.PrimaryEmail = identifier
	} else if len(identifier) >= 11 && strings.HasPrefix(identifier, "1") {
		personCond.PrimaryPhone = identifier
	} else {
		personCond.Username = identifier
	}

	personEntity, err := personDao.GetByCond(ctx.Request.Context(), personCond)
	if err != nil {
		audit.WriteAudit(ctx, audit.AuditEntry{
			Action:     audit.ActionLogin,
			Result:     "failure",
			TargetType: "person",
			Detail:     fmt.Sprintf("identifier:%s, reason:user lookup error", identifier),
		})
		glog.Errorf(ctx, "[svcauth.resolvePersonLogin] person dao GetByCond fail, err:%v", err)
		return nil, nil, nil, code.GetError(code.UserGetDetailError)
	}
	if personEntity == nil || personEntity.ID == "" {
		// H8：未知标识同样计入 IP 维度失败（防口令喷洒绕过 IP 锁）；
		// 审计 detail 不写"user not found"，避免用户名枚举。
		svcloginguard.RecordFailure(ctx, gincontext.GetClientIP(ctx), "")
		audit.WriteAudit(ctx, audit.AuditEntry{
			Action:     audit.ActionLogin,
			Result:     "failure",
			TargetType: "person",
			Detail:     fmt.Sprintf("identifier-hash:%s, reason:auth failed", hashIdentifier(identifier)),
		})
		return nil, nil, nil, code.GetError(code.AuthLoginFailedError)
	}

	userEntity, tenants, err := svc.listPersonTenants(ctx, personEntity.ID)
	if err != nil {
		// listPersonTenants 在无租户成员时返回 UserNotExistError（登录失败统一语义）
		return nil, nil, nil, err
	}
	if userEntity.IsSuspended {
		audit.WriteAudit(ctx, audit.AuditEntry{
			Action:     audit.ActionLogin,
			Result:     "failure",
			TargetType: "person",
			Detail:     fmt.Sprintf("userID:%s, reason:suspended", userEntity.ID),
		})
		return nil, nil, nil, code.GetError(code.UserSuspendedError)
	}
	return personEntity, userEntity, tenants, nil
}

func (svc *authSvc) listPersonTenants(ctx *gin.Context, personID string) (*model.UserEntity, []objauth.TenantOption, error) {
	userDao := newAuthUserStore()
	tenantDao := newAuthTenantStore()
	// 一次取全部租户成员，按加入时间排序：第一条即"默认租户"（确定性），
	// 消除原先 GetByCond 无 ORDER BY 时默认租户随机的问题。
	joinedUsers, err := userDao.GetListByCond(ctx.Request.Context(), &dao.UserCond{
		BaseCond: &gormdao.BaseCond{OrderField: "joined_at, id"},
		PersonID: personID,
	})
	if err != nil {
		glog.Errorf(ctx, "[svcauth.listPersonTenants] user dao GetListByCond fail, err:%v", err)
		return nil, nil, code.GetError(code.UserGetDetailError)
	}
	if len(joinedUsers) == 0 {
		return nil, nil, code.GetError(code.UserNotExistError)
	}
	// 批量查询租户（消除 N+1）
	tenantIDs := make([]string, 0, len(joinedUsers))
	for _, u := range joinedUsers {
		if u.TenantID != "" {
			tenantIDs = append(tenantIDs, u.TenantID)
		}
	}
	tenantMap := map[string]*model.TenantEntity{}
	if len(tenantIDs) > 0 {
		tenants, qErr := tenantDao.GetListByCond(ctx.Request.Context(), &dao.TenantCond{
			BaseCond: &gormdao.BaseCond{IDs: toAnySlice(tenantIDs)},
		})
		if qErr != nil {
			glog.Errorf(ctx, "[svcauth.listPersonTenants] tenant dao GetListByCond fail, err:%v", qErr)
			return nil, nil, code.GetError(code.UserGetDetailError)
		}
		for i := range tenants {
			tenantMap[tenants[i].ID] = &tenants[i]
		}
	}
	options := make([]objauth.TenantOption, 0, len(joinedUsers))
	for _, joinedUser := range joinedUsers {
		tenantEntity := tenantMap[joinedUser.TenantID]
		if tenantEntity == nil {
			continue
		}
		options = append(options, objauth.TenantOption{TenantID: tenantEntity.ID, Name: tenantEntity.Name, Tag: tenantEntity.Tag, UserID: joinedUser.ID, IsOwner: joinedUser.IsOwner})
	}
	return &joinedUsers[0], options, nil
}

// toAnySlice 把 []string 转为 []any（供 gormdao.BaseCond.IDs 使用）。
func toAnySlice(ids []string) []any {
	out := make([]any, len(ids))
	for i, id := range ids {
		out[i] = id
	}
	return out
}

// validatePasswordStrength 密码强度校验，统一走公共包 pkg/iam/password
// （8~128 位，含大小写数字；上限防 bcrypt 登录 DoS）。
func validatePasswordStrength(rawPassword string) error {
	return password.ValidateStrength(rawPassword)
}

// hashIdentifier 对登录标识做摘要（前 16 位 hex），
// 供审计/日志记录失败来源而不泄露明文用户名/邮箱。
func hashIdentifier(identifier string) string {
	sum := sha256.Sum256([]byte(identifier))
	return hex.EncodeToString(sum[:8])
}

func defaultRecordLoginLog(ctx *gin.Context, tenantID, userID string, success bool) {
	loginIP := gincontext.GetClientIP(ctx)
	userAgent := ctx.GetHeader("User-Agent")

	loginLogEntity := &model.UserLoginLogEntity{
		TenantID:  tenantID,
		UserID:    userID,
		LoginType: "password",
		LoginIP:   loginIP,
		UserAgent: userAgent,
		LoginTime: time.Now(),
		CreatedBy: "",
	}

	if err := dao.NewUserLoginLogDao().Insert(ctx, loginLogEntity); err != nil {
		glog.Errorf(ctx, "[svcauth.defaultRecordLoginLog] insert login log fail, err:%v", err)
	}

	if success {
		userDao := newAuthUserStore()
		if err := userDao.UpdateMap(ctx.Request.Context(), userID, map[string]interface{}{
			"last_sign_in_at": time.Now(),
		}); err != nil {
			glog.Errorf(ctx, "[svcauth.defaultRecordLoginLog] update last_sign_in_at fail, err:%v", err)
		}
	}

	result := "failure"
	if success {
		result = "success"
	}
	audit.WriteAudit(ctx, audit.AuditEntry{
		Action:     audit.ActionLogin,
		TenantID:   tenantID,
		Result:     result,
		TargetType: "person",
		TargetID:   userID,
		Detail:     fmt.Sprintf("userID:%s", userID),
	})
}
