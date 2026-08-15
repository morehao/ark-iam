package ctrtenant

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/tenantadmin/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/tenantadmin/internal/service/svctenant"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type UserCtr interface {
	PageList(ctx *gin.Context)
}

type userCtr struct {
	userSvc svctenant.UserSvc
}

var _ UserCtr = (*userCtr)(nil)

func NewUserCtr() UserCtr {
	return &userCtr{
		userSvc: svctenant.NewUserSvc(),
	}
}

// @Tags 用户
// @Summary 租户内用户列表分页
// @accept application/json
// @Produce application/json
// @Param req query dtotenant.UserPageListReq true "租户内用户列表分页"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.UserPageListResp}
// @Router /v1/tenant/users [get]
func (ctr *userCtr) PageList(ctx *gin.Context) {
	var req dtotenant.UserPageListReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.userSvc.PageList(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}
