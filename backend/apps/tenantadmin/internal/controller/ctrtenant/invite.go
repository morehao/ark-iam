package ctrtenant

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/identity"
	"github.com/morehao/ark-iam/tenantadmin/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/tenantadmin/internal/service/svctenant"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type InviteCtr interface {
	Create(ctx *gin.Context)
	Revoke(ctx *gin.Context)
	PageList(ctx *gin.Context)
}

type inviteCtr struct {
	inviteSvc svctenant.InviteSvc
}

var _ InviteCtr = (*inviteCtr)(nil)

func NewInviteCtr() InviteCtr {
	return &inviteCtr{
		inviteSvc: svctenant.NewInviteSvc(),
	}
}

// @Tags 邀请
// @Summary 生成租户加入邀请
// @accept application/json
// @Produce application/json
// @Param req body dtotenant.InviteCreateReq true "生成租户加入邀请"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.InviteCreateResp}
// @Router /v1/tenant/invites [post]
func (ctr *inviteCtr) Create(ctx *gin.Context) {
	var req dtotenant.InviteCreateReq
	req.Operator = identity.OperatorFromGin(ctx)
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.inviteSvc.Create(ctx.Request.Context(), &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 邀请
// @Summary 撤销租户加入邀请
// @accept application/json
// @Produce application/json
// @Param inviteID path string true "邀请ID"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/tenant/invites/{inviteID} [delete]
func (ctr *inviteCtr) Revoke(ctx *gin.Context) {
	var req dtotenant.InviteRevokeReq
	req.Operator = identity.OperatorFromGin(ctx)
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.inviteSvc.Revoke(ctx.Request.Context(), &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "撤销成功")
}

// @Tags 邀请
// @Summary 租户加入邀请列表分页
// @accept application/json
// @Produce application/json
// @Param req query dtotenant.InvitePageListReq true "租户加入邀请列表分页"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.InvitePageListResp}
// @Router /v1/tenant/invites [get]
func (ctr *inviteCtr) PageList(ctx *gin.Context) {
	var req dtotenant.InvitePageListReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.inviteSvc.PageList(ctx.Request.Context(), &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}
