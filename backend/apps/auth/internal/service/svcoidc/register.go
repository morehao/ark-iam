package svcoidc

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/morehao/ark-iam/auth/internal/core/oidcop"
	"github.com/morehao/ark-iam/auth/internal/dto/dtooidc"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/iam/password"
	"github.com/morehao/ark-iam/pkg/iam/person"
	"github.com/morehao/ark-iam/pkg/iam/tenant"

	"github.com/morehao/golib/gcrypto"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"

	"gorm.io/gorm"
)

// RegisterPerson 在 OIDC 认证流程内注册 person（应用需允许自主注册）。
// person find-or-create（不存在建、存在复用，绝不覆盖既有密码）；随后前端按需
// 进入 selectTenant（有租户）或 createTenant（零租户且应用允许）。
// 本方法仅绑定 person 到 authRequest（done=false），不发 code，认证收尾交 selectTenant。
func (svc *oidcAuthSvc) RegisterPerson(ctx *gin.Context, req *dtooidc.RegisterPersonReq) (*dtooidc.RegisterPersonResp, error) {
	if err := password.ValidateStrength(req.Password); err != nil {
		return nil, code.GetError(code.PasswordValidationError)
	}
	if req.Username == "" && req.PrimaryEmail == "" && req.PrimaryPhone == "" {
		return nil, code.GetError(code.AuthIdentifierRequiredError)
	}

	authReq, err := svc.provider.Storage.AuthRequestByID(ctx.Request.Context(), req.AuthRequestID)
	if err != nil {
		return nil, mapAuthRequestError(err)
	}
	if authReq.Done() {
		return nil, code.GetError(code.OIDCSessionNotFound)
	}
	clientID := clientIDFromAuthRequest(authReq)
	if !svc.appAllowsPersonCreateTenant(ctx, clientID) {
		glog.Warnf(ctx, "[oidcAuthSvc.RegisterPerson] app disallows person register, clientID:%s, req:%s", clientID, gutil.ToJsonString(req))
		return nil, code.GetError(code.AuthTenantRegisterNotAllowedError)
	}

	passwordHash, err := gcrypto.GeneratePasswordHash(req.Password)
	if err != nil {
		glog.Errorf(ctx, "[oidcAuthSvc.RegisterPerson] hash password fail, err:%v", err)
		return nil, code.GetError(code.PasswordHashError)
	}

	var personEntity *model.PersonEntity
	personCreated := false
	txErr := dbclient.IamDB(ctx.Request.Context()).Transaction(func(tx *gorm.DB) error {
		p, created, fErr := person.FindOrCreate(ctx.Request.Context(), tx, &person.FindOrCreateReq{
			Username:          req.Username,
			PrimaryEmail:      req.PrimaryEmail,
			PrimaryPhone:      req.PrimaryPhone,
			PasswordEncrypted: passwordHash,
			PasswordMethod:    "bcrypt",
			Name:              req.Name,
			CreatedBy:         "",
		})
		if fErr != nil {
			return fErr
		}
		personEntity = p
		personCreated = created
		return nil
	})
	if txErr != nil {
		if errors.Is(txErr, gorm.ErrDuplicatedKey) {
			return nil, svcResolvePersonConflict(ctx, req)
		}
		glog.Errorf(ctx, "[oidcAuthSvc.RegisterPerson] person find-or-create fail, err:%v, req:%s", txErr, gutil.ToJsonString(req))
		return nil, code.GetError(code.AuthRegisterFailedError)
	}

	// C1：已存在的 person 不通过注册免密绑定——要求走密码登录（/oidc/login）。
	// 否则攻击者以任意密码复用他人 person 即可冒用其身份。已存在时不得
	// 返回既有租户、不得绑定 authRequest、不得进入选租户/建租户。
	if !personCreated {
		return &dtooidc.RegisterPersonResp{
			PersonID:              personEntity.ID,
			RequiresPasswordLogin: true,
		}, nil
	}

	tenants, tErr := svc.authSvc.TenantsForPerson(ctx, personEntity.ID)
	if tErr != nil {
		glog.Errorf(ctx, "[oidcAuthSvc.RegisterPerson] TenantsForPerson fail, err:%v, personID:%s", tErr, personEntity.ID)
		return nil, code.GetError(code.AuthRegisterFailedError)
	}
	if cErr := svc.provider.Storage.CompleteAuthRequest(ctx.Request.Context(), req.AuthRequestID,
		oidcop.BuildSubject(personEntity.ID), time.Now(), []string{"pwd"}, "", "", false); cErr != nil {
		return nil, mapAuthRequestError(cErr)
	}

	allowCreate := svc.appAllowsPersonCreateTenant(ctx, clientID) && len(tenants) == 0
	return &dtooidc.RegisterPersonResp{
		PersonID:                personEntity.ID,
		RequiresTenantSelection: len(tenants) > 0,
		Tenants:                 tenants,
		AllowPersonCreateTenant: allowCreate,
	}, nil
}

// svcResolvePersonConflict 并发唯一冲突时回查哪个标识已存在，返回对应错误码。
func svcResolvePersonConflict(ctx *gin.Context, req *dtooidc.RegisterPersonReq) error {
	personDao := dao.NewPersonDao()
	if req.Username != "" {
		if p, qErr := personDao.GetByCond(ctx.Request.Context(), &dao.PersonCond{Username: req.Username}); qErr == nil && p != nil && p.ID != "" {
			return code.GetError(code.UsernameAlreadyExistsError)
		}
	}
	if req.PrimaryEmail != "" {
		if p, qErr := personDao.GetByCond(ctx.Request.Context(), &dao.PersonCond{PrimaryEmail: req.PrimaryEmail}); qErr == nil && p != nil && p.ID != "" {
			return code.GetError(code.EmailAlreadyExistsError)
		}
	}
	if req.PrimaryPhone != "" {
		if p, qErr := personDao.GetByCond(ctx.Request.Context(), &dao.PersonCond{PrimaryPhone: req.PrimaryPhone}); qErr == nil && p != nil && p.ID != "" {
			return code.GetError(code.PhoneAlreadyExistsError)
		}
	}
	return code.GetError(code.AuthRegisterFailedError)
}

// CreateTenant 为 OIDC 流程内已注册、零租户且应用允许的 person 开通租户并自任 owner。
// 仅建 person+tenant+owner 成员；认证收尾（complete + 发 code + SSO 会话）由前端调 selectTenant 完成。
// I1：createTenant 无幂等、不置 done，同一 authRequest 可被重复调用批量建多个租户，但这是
// 新建 person 的自建动作（C1 修复后仅新建 person 可达此），非跨用户/冒用风险，判定可接受。
func (svc *oidcAuthSvc) CreateTenant(ctx *gin.Context, req *dtooidc.CreateTenantReq) (*dtooidc.CreateTenantResp, error) {
	authReq, err := svc.provider.Storage.AuthRequestByID(ctx.Request.Context(), req.AuthRequestID)
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
	clientID := clientIDFromAuthRequest(authReq)
	if !svc.appAllowsPersonCreateTenant(ctx, clientID) {
		glog.Warnf(ctx, "[oidcAuthSvc.CreateTenant] app disallows tenant create, clientID:%s, req:%s", clientID, gutil.ToJsonString(req))
		return nil, code.GetError(code.AuthTenantRegisterNotAllowedError)
	}
	tenants, tErr := svc.authSvc.TenantsForPerson(ctx, personID)
	if tErr != nil {
		glog.Errorf(ctx, "[oidcAuthSvc.CreateTenant] TenantsForPerson fail, err:%v", tErr)
		return nil, code.GetError(code.AuthRegisterFailedError)
	}
	if len(tenants) > 0 {
		return nil, code.GetError(code.AlreadyJoinedError)
	}

	tenantCode := strings.TrimSpace(req.TenantCode)
	if tenantCode == "" {
		tenantCode = "tenant-" + uuid.NewString()
	}

	var tenantEntity *model.TenantEntity
	txErr := dbclient.IamDB(ctx.Request.Context()).Transaction(func(tx *gorm.DB) error {
		var tErr error
		tenantEntity, tErr = tenant.CreateWithRootOrg(ctx.Request.Context(), tx, &tenant.CreateWithRootOrgReq{
			Code:      tenantCode,
			Name:      req.TenantName,
			Type:      model.TenantTypeCustomer,
			CreatedBy: personID,
		})
		if tErr != nil {
			return tErr
		}
		now := time.Now()
		owner := &model.UserEntity{
			TenantID:   tenantEntity.ID,
			PersonID:   personID,
			Name:       "",
			Profile:    json.RawMessage(`{}`),
			CustomData: json.RawMessage(`{}`),
			IsOwner:    true,
			JoinedAt:   &now,
			CreatedBy:  personID,
		}
		if uErr := dao.NewUserDao().WithTx(tx).Insert(ctx.Request.Context(), owner); uErr != nil {
			return uErr
		}
		return nil
	})
	if txErr != nil {
		if errors.Is(txErr, gorm.ErrDuplicatedKey) {
			glog.Warnf(ctx, "[oidcAuthSvc.CreateTenant] tenant code duplicate, req:%s", gutil.ToJsonString(req))
			return nil, code.GetError(code.AuthRegisterFailedError)
		}
		glog.Errorf(ctx, "[oidcAuthSvc.CreateTenant] transaction fail, err:%v, req:%s", txErr, gutil.ToJsonString(req))
		return nil, code.GetError(code.AuthRegisterFailedError)
	}

	return &dtooidc.CreateTenantResp{
		TenantID: tenantEntity.ID,
		PersonID: personID,
	}, nil
}
