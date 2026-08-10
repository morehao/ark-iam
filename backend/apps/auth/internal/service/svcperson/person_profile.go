package svcperson

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/auth/internal/dto/dtoperson"
	"github.com/morehao/ark-iam/pkg/code"
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
	personID := gincontext.GetPersonID(ctx)
	if personID == 0 {
		return nil, code.GetError(code.UserNotExistError)
	}

	personDao := dao.NewPersonDao()
	personEntity, err := personDao.GetByID(ctx.Request.Context(), personID)
	if err != nil {
		glog.Errorf(ctx, "[svcperson.Detail] dao GetByID fail, err:%v, personID:%d", err, personID)
		return nil, code.GetError(code.UserGetDetailError)
	}
	if personEntity == nil || personEntity.ID == 0 {
		return nil, code.GetError(code.UserNotExistError)
	}

	return &dtoperson.PersonDetailResp{
		PersonID:     personEntity.ID,
		Username:     personEntity.Username,
		PrimaryEmail: personEntity.PrimaryEmail,
		PrimaryPhone: personEntity.PrimaryPhone,
		Name:         personEntity.Name,
		Avatar:       personEntity.Avatar,
		IsSuspended:  personEntity.IsSuspended,
	}, nil
}

func (svc *personProfileSvc) UpdatePassword(ctx *gin.Context, req *dtoperson.PersonUpdatePasswordReq) error {
	personID := gincontext.GetPersonID(ctx)
	if personID == 0 {
		return code.GetError(code.UserNotExistError)
	}

	personDao := dao.NewPersonDao()
	personEntity, err := personDao.GetByID(ctx.Request.Context(), personID)
	if err != nil {
		glog.Errorf(ctx, "[svcperson.UpdatePassword] dao GetByID fail, err:%v, personID:%d", err, personID)
		return code.GetError(code.UserGetDetailError)
	}
	if personEntity == nil || personEntity.ID == 0 {
		return code.GetError(code.UserNotExistError)
	}

	if err := gcrypto.ComparePasswordHash(personEntity.PasswordEncrypted, req.OldPassword); err != nil {
		glog.Errorf(ctx, "[svcperson.UpdatePassword] old password mismatch, personID:%d", personID)
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
		glog.Errorf(ctx, "[svcperson.UpdatePassword] dao UpdateMap fail, err:%v, personID:%d", err, personID)
		return code.GetError(code.UserUpdateError)
	}

	return nil
}