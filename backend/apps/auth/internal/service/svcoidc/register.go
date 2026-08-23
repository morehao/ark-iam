package svcoidc

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/internal/core/oidcop"
	"github.com/morehao/ark-iam/auth/internal/dto/dtooidc"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/iam/password"
	"github.com/morehao/ark-iam/pkg/iam/person"
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
	txErr := dbclient.IamDB(ctx.Request.Context()).Transaction(func(tx *gorm.DB) error {
		p, _, fErr := person.FindOrCreate(ctx.Request.Context(), tx, &person.FindOrCreateReq{
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
		return nil
	})
	if txErr != nil {
		if errors.Is(txErr, gorm.ErrDuplicatedKey) {
			return nil, svcResolvePersonConflict(ctx, req)
		}
		glog.Errorf(ctx, "[oidcAuthSvc.RegisterPerson] person find-or-create fail, err:%v, req:%s", txErr, gutil.ToJsonString(req))
		return nil, code.GetError(code.AuthRegisterFailedError)
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

// CreateTenant 完整实现在 Task4；此处为满足接口编译的最小骨架。
func (svc *oidcAuthSvc) CreateTenant(ctx *gin.Context, req *dtooidc.CreateTenantReq) (*dtooidc.CreateTenantResp, error) {
	return nil, code.GetError(code.AuthRegisterFailedError)
}
