package ctrrolescope

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

// Create 角色范围关系创建
// @Tags 角色范围关系管理
// @Summary 创建角色范围关系管理
// @accept application/json
// @Produce application/json
// @Param req body dtopermission.RoleScopeCreateReq true "创建角色范围关系管理"
// @Success 200 {object} gincontext.DtoRender{data=dtopermission.RoleScopeCreateResp} "{"code": 0, "requestID": "xxx", "data": "ok", "msg": "success"}"
// @Router /v1/iam/permission/create [post]
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

// Delete 角色范围关系删除
// @Tags 角色范围关系管理
// @Summary 删除角色范围关系管理
// @accept application/json
// @Produce application/json
// @Param req body dtopermission.RoleScopeDeleteReq true "删除角色范围关系管理"
// @Success 200 {object} gincontext.DtoRender{data=string} "{"code": 0, "requestID": "xxx", "data": "ok", "msg": "success"}"
// @Router /v1/iam/permission/delete [post]
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

// PageList 角色范围关系列表
// @Tags 角色范围关系管理
// @Summary 角色范围关系管理列表分页
// @accept application/json
// @Produce application/json
// @Param req body dtopermission.RoleScopePageListReq true "角色范围关系管理列表"
// @Success 200 {object} gincontext.DtoRender{data=dtopermission.RoleScopePageListResp} "{"code": 0, "requestID": "xxx", "data": "ok", "msg": "success"}"
// @Router /v1/iam/permission/pageList [post]
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