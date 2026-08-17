package svcperson

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/internal/dto/dtoperson"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/iam/password"
	"github.com/morehao/ark-iam/pkg/iam/sso"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/gcrypto"
	"github.com/morehao/golib/glog"
)

type personProfileSvc struct{}

var _ PersonProfileSvc = (*personProfileSvc)(nil)

func NewPersonProfileSvc() PersonProfileSvc {
	return &personProfileSvc{}
}

func (svc *personProfileSvc) Detail(ctx *gin.Context, req *dtoperson.PersonDetailReq) (*dtoperson.PersonDetailResp, error) {
	personID := gincontext.GetPersonIDString(ctx)
	if personID == "" {
		return nil, code.GetError(code.UserNotExistError)
	}

	personDao := dao.NewPersonDao()
	personEntity, err := personDao.GetByID(ctx.Request.Context(), personID)
	if err != nil {
		glog.Errorf(ctx, "[svcperson.Detail] dao GetByID fail, err:%v, personID:%s", err, personID)
		return nil, code.GetError(code.UserGetDetailError)
	}
	if personEntity == nil || personEntity.ID == "" {
		return nil, code.GetError(code.UserNotExistError)
	}

	return &dtoperson.PersonDetailResp{
		PersonID:     personEntity.ID,
		Username:     model.DerefStr(personEntity.Username),
		PrimaryEmail: model.DerefStr(personEntity.PrimaryEmail),
		PrimaryPhone: model.DerefStr(personEntity.PrimaryPhone),
		Name:         personEntity.Name,
		Avatar:       personEntity.Avatar,
		IsSuspended:  personEntity.IsSuspended,
	}, nil
}

func (svc *personProfileSvc) UpdatePassword(ctx *gin.Context, req *dtoperson.PersonUpdatePasswordReq) error {
	personID := gincontext.GetPersonIDString(ctx)
	if personID == "" {
		return code.GetError(code.UserNotExistError)
	}

	personDao := dao.NewPersonDao()
	personEntity, err := personDao.GetByID(ctx.Request.Context(), personID)
	if err != nil {
		glog.Errorf(ctx, "[svcperson.UpdatePassword] dao GetByID fail, err:%v, personID:%s", err, personID)
		return code.GetError(code.UserGetDetailError)
	}
	if personEntity == nil || personEntity.ID == "" {
		return code.GetError(code.UserNotExistError)
	}

	// 连接器注册的账号可能没有本地密码，此时应提示"密码未设置"而非"密码错误"
	if personEntity.PasswordEncrypted == "" {
		return code.GetError(code.PasswordNotSetError)
	}

	if err := password.ValidateStrength(req.NewPassword); err != nil {
		return code.GetError(code.PasswordValidationError)
	}

	if req.OldPassword == req.NewPassword {
		glog.Warnf(ctx, "[svcperson.UpdatePassword] new password equals old password, personID:%s", personID)
		return code.GetError(code.PasswordValidationError)
	}

	if err := gcrypto.ComparePasswordHash(personEntity.PasswordEncrypted, req.OldPassword); err != nil {
		glog.Errorf(ctx, "[svcperson.UpdatePassword] old password mismatch, personID:%s", personID)
		return code.GetError(code.PasswordMismatchError)
	}

	newHash, err := gcrypto.GeneratePasswordHash(req.NewPassword)
	if err != nil {
		glog.Errorf(ctx, "[svcperson.UpdatePassword] GeneratePasswordHash fail, err:%v", err)
		return code.GetError(code.PasswordHashError)
	}

	if err := personDao.UpdateMap(ctx.Request.Context(), personID, map[string]interface{}{
		"password_encrypted": newHash,
	}); err != nil {
		glog.Errorf(ctx, "[svcperson.UpdatePassword] dao UpdateMap fail, err:%v, personID:%s", err, personID)
		return code.GetError(code.UserUpdateError)
	}

	// H7：改密即全局登出——撤销该 person 的全部 SSO 会话与 refresh token，
	// 使已泄露/被盗的旧会话在改密后立即失效（与 Logout 的"一处登出、处处登出"语义一致）。
	if err := sso.RevokeSSOSessionsByPersonID(ctx.Request.Context(), personID); err != nil {
		glog.Warnf(ctx, "[svcperson.UpdatePassword] revoke sso sessions fail, personID:%s, err:%v", personID, err)
	}
	if err := dao.NewRefreshTokenDao().RevokeByPersonID(ctx.Request.Context(), personID); err != nil {
		glog.Warnf(ctx, "[svcperson.UpdatePassword] revoke refresh tokens fail, personID:%s, err:%v", personID, err)
	}

	return nil
}

// 注意：密码强度规则统一走公共包 pkg/iam/password（8~128 位，含大小写数字），
// 与注册流程（svcauth.validatePasswordStrength）保持一致。
