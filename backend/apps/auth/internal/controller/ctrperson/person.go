package ctrperson

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/internal/dto/dtoperson"
	"github.com/morehao/ark-iam/auth/internal/service/svcperson"
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

func NewPersonCtr(personSvc svcperson.PersonProfileSvc) PersonCtr {
	return &personCtr{
		personSvc: personSvc,
	}
}

// @Tags 自然人管理
// @Summary 获取自然人详情
// @accept application/json
// @Produce application/json
// @Param req query dtoperson.PersonDetailReq true "获取自然人详情"
// @Success 200 {object} gincontext.DtoRender{data=dtoperson.PersonDetailResp}
// @Router /v1/person/detail [get]
func (ctr *personCtr) Detail(ctx *gin.Context) {
	var req dtoperson.PersonDetailReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.personSvc.Detail(ctx, &req)
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
// @Router /v1/person/updatePassword [post]
func (ctr *personCtr) UpdatePassword(ctx *gin.Context) {
	var req dtoperson.PersonUpdatePasswordReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.personSvc.UpdatePassword(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "密码更新成功")
}
