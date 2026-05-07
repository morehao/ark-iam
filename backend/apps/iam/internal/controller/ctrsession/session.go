package ctrsession

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/dto/dtouser"
	"github.com/morehao/ark-iam/iam/internal/service/svcsession"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type SessionCtr interface {
	List(ctx *gin.Context)
	Revoke(ctx *gin.Context)
	RevokeAll(ctx *gin.Context)
}

type sessionCtr struct {
	sessionSvc svcsession.SessionSvc
}

var _ SessionCtr = (*sessionCtr)(nil)

func NewSessionCtr() SessionCtr {
	return &sessionCtr{
		sessionSvc: svcsession.NewSessionSvc(),
	}
}

// @Tags 会话管理
// @Summary 会话列表
// @accept application/json
// @Produce application/json
// @Param req query dtouser.SessionListReq true "会话列表"
// @Success 200 {object} gincontext.DtoRender{data=dtouser.SessionListResp}
// @Router /v1/iam/user/sessions [get]
func (ctr *sessionCtr) List(ctx *gin.Context) {
	var req dtouser.SessionListReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}

	userID := gincontext.GetUserID(ctx)
	res, err := ctr.sessionSvc.List(ctx, &req, userID)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 会话管理
// @Summary 撤销会话
// @accept application/json
// @Produce application/json
// @Param req path dtouser.SessionRevokeReq true "撤销会话"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/user/sessions/{sessionId} [delete]
func (ctr *sessionCtr) Revoke(ctx *gin.Context) {
	var req dtouser.SessionRevokeReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}

	userID := gincontext.GetUserID(ctx)
	if err := ctr.sessionSvc.Revoke(ctx, &req, userID); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "撤销成功")
}

// @Tags 会话管理
// @Summary 撤销所有会话
// @accept application/json
// @Produce application/json
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/user/sessions [delete]
func (ctr *sessionCtr) RevokeAll(ctx *gin.Context) {
	userID := gincontext.GetUserID(ctx)
	if err := ctr.sessionSvc.RevokeAll(ctx, userID); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "撤销成功")
}