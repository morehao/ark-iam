package ctrperson

import (
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/backend/apps/auth/dto/dtoperson"
	"github.com/morehao/ark-iam/backend/apps/auth/service/svcperson"
	"github.com/morehao/ark-iam/pkg/identity"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type PersonCtr interface {
	Detail(ctx *gin.Context)
	UpdatePassword(ctx *gin.Context)
}

type personCtr struct {
	personSvc svcperson.PersonProfileSvc
}

var _ PersonCtr = (*personCtr)(nil)

func NewPersonCtr(db gormdao.DBGetter) PersonCtr {
	return &personCtr{
		personSvc: svcperson.NewPersonProfileSvc(db),
	}
}

// @Tags 自然人管理
// @Summary 获取自然人详情
// @accept application/json
// @Produce application/json
// @Param req query dtoperson.PersonDetailReq true "获取自然人详情"
// @Success 200 {object} gincontext.DtoRender{data=dtoperson.PersonDetailResp}
// @Router /v1/auth/me [get]
func (ctr *personCtr) Detail(ctx *gin.Context) {
	var req dtoperson.PersonDetailReq
	req.Operator = identity.OperatorFromGin(ctx)
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.personSvc.Detail(ctx.Request.Context(), &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 自然人管理
// @Summary 更新自然人密码
// @accept application/json
// @Produce application/json
// @Param req body dtoperson.PersonUpdatePasswordReq true "更新自然人密码"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/auth/me/changePassword [post]
func (ctr *personCtr) UpdatePassword(ctx *gin.Context) {
	var req dtoperson.PersonUpdatePasswordReq
	req.Operator = identity.OperatorFromGin(ctx)
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.personSvc.UpdatePassword(ctx.Request.Context(), &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "密码更新成功")
}
