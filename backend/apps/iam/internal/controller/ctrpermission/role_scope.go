package ctrpermission

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/dto/dtopermission"
	"github.com/morehao/ark-iam/iam/internal/service/svcpermission"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type RoleScopeCtr interface {
	Create(ctx *gin.Context)
	Delete(ctx *gin.Context)
	PageList(ctx *gin.Context)
}

type roleScopeCtr struct {
	roleScopeSvc svcpermission.RoleScopeSvc
}

var _ RoleScopeCtr = (*roleScopeCtr)(nil)

func NewRoleScopeCtr() RoleScopeCtr {
	return &roleScopeCtr{
		roleScopeSvc: svcpermission.NewRoleScopeSvc(),
	}
}

// @Tags 角色权限范围
// @Summary 创建角色权限范围
// @accept application/json
// @Produce application/json
// @Param req body dtopermission.RoleScopeCreateReq true "创建角色权限范围"
// @Success 200 {object} gincontext.DtoRender{data=dtopermission.RoleScopeCreateResp}
// @Router /v1/iam/roleScope/create [post]
func (ctr *roleScopeCtr) Create(ctx *gin.Context) {
	var req dtopermission.RoleScopeCreateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.roleScopeSvc.Create(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 角色权限范围
// @Summary 删除角色权限范围
// @accept application/json
// @Produce application/json
// @Param req body dtopermission.RoleScopeDeleteReq true "删除角色权限范围"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/roleScope/delete [post]
func (ctr *roleScopeCtr) Delete(ctx *gin.Context) {
	var req dtopermission.RoleScopeDeleteReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.roleScopeSvc.Delete(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}

// @Tags 角色权限范围
// @Summary 角色权限范围列表分页
// @accept application/json
// @Produce application/json
// @Param req body dtopermission.RoleScopePageListReq true "角色权限范围列表分页"
// @Success 200 {object} gincontext.DtoRender{data=dtopermission.RoleScopePageListResp}
// @Router /v1/iam/roleScope/pageList [post]
func (ctr *roleScopeCtr) PageList(ctx *gin.Context) {
	var req dtopermission.RoleScopePageListReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.roleScopeSvc.PageList(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}