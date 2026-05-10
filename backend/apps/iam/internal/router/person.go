package router

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/config"
	"github.com/morehao/ark-iam/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/controller/ctrperson"
	"github.com/morehao/ark-iam/iam/internal/dto/dtoperson"
	"github.com/morehao/ark-iam/iam/internal/service/svcauth"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

type personSessionSvc struct {
}

func (s *personSessionSvc) Detail(ctx *gin.Context, req *dtoperson.PersonDetailReq) (*dtoperson.PersonDetailResp, error) {
	personID := gincontext.GetPersonID(ctx)
	if personID == 0 {
		return nil, code.GetError(code.UserNotExistError)
	}

	personDao := dao.NewPersonDao()
	personEntity, err := personDao.GetByID(ctx.Request.Context(), personID)
	if err != nil {
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

func (s *personSessionSvc) UpdatePassword(ctx *gin.Context, req *dtoperson.PersonUpdatePasswordReq) error {
	return nil
}

func personRouter(groups *ginserver.RouterGroups) {
	authSvc := svcauth.NewAuthSvc(config.Conf.JWT.SignKey)
	personCtr := ctrperson.NewPersonCtr(
		&personSessionSvc{},
		authSvc,
	)

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.GET("/person/detail", personCtr.Detail)
	v1RouterGroup.POST("/person/updatePassword", personCtr.UpdatePassword)
}