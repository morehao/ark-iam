package ctruserrole

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/dto/dtopermission"
	"github.com/morehao/ark-iam/iam/internal/service/svcuser"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type UserRoleCtr interface {
	Create(ctx *gin.Context)
	Delete(ctx *gin.Context)
	PageList(ctx *gin.Context)
}

type userRoleCtr struct {
	userRoleSvc svcuser.UserRoleSvc
}

var _ UserRoleCtr = (*userRoleCtr)(nil)

func NewUserRoleCtr() UserRoleCtr {
	return &userRoleCtr{
		userRoleSvc: svcuser.NewUserRoleSvc(),
	}
}

// Create 用户角色关系创建
// @Tags 用户角色关系管理
// @Summary 创建用户角色关系管理
// @accept application/json
// @Produce application/json
// @Param req body dtopermission.UserRoleCreateReq true "创建用户角色关系管理"
// @Success 200 {object} gincontext.DtoRender{data=dtopermission.UserRoleCreateResp} "{"code": 0, "requestID": "xxx", "data": "ok", "msg": "success"}"
// @Router /v1/iam/user/create [post]
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

// Delete 用户角色关系删除
// @Tags 用户角色关系管理
// @Summary 删除用户角色关系管理
// @accept application/json
// @Produce application/json
// @Param req body dtopermission.UserRoleDeleteReq true "删除用户角色关系管理"
// @Success 200 {object} gincontext.DtoRender{data=string} "{"code": 0, "requestID": "xxx", "data": "ok", "msg": "success"}"
// @Router /v1/iam/user/delete [post]
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

// PageList 用户角色关系列表
// @Tags 用户角色关系管理
// @Summary 用户角色关系管理列表分页
// @accept application/json
// @Produce application/json
// @Param req body dtopermission.UserRolePageListReq true "用户角色关系管理列表"
// @Success 200 {object} gincontext.DtoRender{data=dtopermission.UserRolePageListResp} "{"code": 0, "requestID": "xxx", "data": "ok", "msg": "success"}"
// @Router /v1/iam/user/pageList [post]
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