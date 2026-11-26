package svcperson

import (
	"context"
	"github.com/morehao/ark-iam/apps/auth/dto/dtoperson"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/core/person"
	"github.com/morehao/ark-iam/pkg/credential"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/gcrypto"
	"github.com/morehao/golib/glog"
)

type personProfileSvc struct {
	db gormdao.DBGetter
}

var _ PersonProfileSvc = (*personProfileSvc)(nil)

func NewPersonProfileSvc(db gormdao.DBGetter) PersonProfileSvc {
	if db == nil {
		db = dbclient.IamDB
	}
	return &personProfileSvc{db: db}
}

func (svc *personProfileSvc) Detail(ctx context.Context, req *dtoperson.PersonDetailReq) (*dtoperson.PersonDetailResp, error) {
	personID := req.Operator.PersonID
	if personID == "" {
		return nil, code.GetError(code.UserNotExistError)
	}

	personDao := dao.NewPersonDao(dao.WithDBGetter(svc.db))
	personEntity, err := personDao.GetByID(ctx, personID)
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
		Status:       personEntity.Status,
	}, nil
}

func (svc *personProfileSvc) UpdatePassword(ctx context.Context, req *dtoperson.PersonUpdatePasswordReq) error {
	personID := req.Operator.PersonID
	if personID == "" {
		return code.GetError(code.UserNotExistError)
	}

	personDao := dao.NewPersonDao(dao.WithDBGetter(svc.db))
	personEntity, err := personDao.GetByID(ctx, personID)
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

	if err := credential.ValidateStrength(req.NewPassword); err != nil {
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

	// 自助改密同时清除"首次登录强制改密"标记：用户已亲自设置口令，临时密码的约束随之解除
	// （正常路径下持临时密码者登录即被拦截，改密只能走 /oidc/login/changePassword；
	// 此处兜底覆盖标记在会话有效期内被置位的边界场景）。
	if err := person.UpdatePassword(ctx, svc.db, personID, newHash, model.PasswordStatusNormal, personID, personEntity.PasswordEncrypted); err != nil {
		glog.Errorf(ctx, "[svcperson.UpdatePassword] update password and revoke sessions fail, err:%v, personID:%s", err, personID)
		return code.GetError(code.UserUpdateError)
	}

	return nil
}

// 注意：密码强度规则统一走公共包 pkg/credential（8~128 位，含大小写数字），
// 与注册流程（svcauth.validatePasswordStrength）保持一致。
