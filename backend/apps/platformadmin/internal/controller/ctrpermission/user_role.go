package ctrpermission

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtopermission"
	"github.com/morehao/ark-iam/platformadmin/internal/service/svcpermission"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type UserRoleCtr interface {
	Create(ctx *gin.Context)
	Delete(ctx *gin.Context)
	PageList(ctx *gin.Context)
}

type userRoleCtr struct {
	userRoleSvc svcpermission.UserRoleSvc
}

var _ UserRoleCtr = (*userRoleCtr)(nil)

func NewUserRoleCtr() UserRoleCtr {
	return &userRoleCtr{
		userRoleSvc: svcpermission.NewUserRoleSvc(),
	}
}

// @Tags 用户角色
// @Summary 创建用户角色
// @accept application/json
// @Produce application/json
// @Param req body dtopermission.UserRoleCreateReq true "创建用户角色"
// @Success 200 {object} gincontext.DtoRender{data=dtopermission.UserRoleCreateResp}
// @Router /v1/platform/userRole/create [post]
func (ctr *userRoleCtr) Create(ctx *gin.Context) {
	var req dtopermission.UserRoleCreateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.userRoleSvc.Create(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 用户角色
// @Summary 删除用户角色
// @accept application/json
// @Produce application/json
// @Param req body dtopermission.UserRoleDeleteReq true "删除用户角色"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/platform/userRole/delete [post]
func (ctr *userRoleCtr) Delete(ctx *gin.Context) {
	var req dtopermission.UserRoleDeleteReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.userRoleSvc.Delete(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}

// @Tags 用户角色
// @Summary 用户角色列表分页
// @accept application/json
// @Produce application/json
// @Param req body dtopermission.UserRolePageListReq true "用户角色列表分页"
// @Success 200 {object} gincontext.DtoRender{data=dtopermission.UserRolePageListResp}
// @Router /v1/platform/userRole/pageList [post]
func (ctr *userRoleCtr) PageList(ctx *gin.Context) {
	var req dtopermission.UserRolePageListReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.userRoleSvc.PageList(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}
