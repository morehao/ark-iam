package ctrpermission

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtopermission"
	"github.com/morehao/ark-iam/platformadmin/internal/service/svcpermission"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type ScopeCtr interface {
	Create(ctx *gin.Context)
	Delete(ctx *gin.Context)
	Update(ctx *gin.Context)
	Detail(ctx *gin.Context)
	PageList(ctx *gin.Context)
}

type scopeCtr struct {
	scopeSvc svcpermission.ScopeSvc
}

var _ ScopeCtr = (*scopeCtr)(nil)

func NewScopeCtr() ScopeCtr {
	return &scopeCtr{
		scopeSvc: svcpermission.NewScopeSvc(),
	}
}

// @Tags 权限范围
// @Summary 创建权限范围
// @accept application/json
// @Produce application/json
// @Param req body dtopermission.ScopeCreateReq true "创建权限范围"
// @Success 200 {object} gincontext.DtoRender{data=dtopermission.ScopeCreateResp}
// @Router /v1/platform/scopes [post]
func (ctr *scopeCtr) Create(ctx *gin.Context) {
	var req dtopermission.ScopeCreateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.scopeSvc.Create(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 权限范围
// @Summary 删除权限范围
// @accept application/json
// @Produce application/json
// @Param scopeID path int true "scopeID"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/platform/scopes/{scopeID} [delete]
func (ctr *scopeCtr) Delete(ctx *gin.Context) {
	var req dtopermission.ScopeDeleteReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.scopeSvc.Delete(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}

// @Tags 权限范围
// @Summary 修改权限范围
// @accept application/json
// @Produce application/json
// @Param scopeID path int true "scopeID"
// @Param req body dtopermission.ScopeUpdateReq true "修改权限范围"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/platform/scopes/{scopeID} [put]
func (ctr *scopeCtr) Update(ctx *gin.Context) {
	var req dtopermission.ScopeUpdateReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.scopeSvc.Update(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "修改成功")
}

// @Tags 权限范围
// @Summary 权限范围详情
// @accept application/json
// @Produce application/json
// @Param scopeID path int true "scopeID"
// @Success 200 {object} gincontext.DtoRender{data=dtopermission.ScopeDetailResp}
// @Router /v1/platform/scopes/{scopeID} [get]
func (ctr *scopeCtr) Detail(ctx *gin.Context) {
	var req dtopermission.ScopeDetailReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.scopeSvc.Detail(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 权限范围
// @Summary 权限范围列表分页
// @accept application/json
// @Produce application/json
// @Param req query dtopermission.ScopePageListReq true "权限范围列表分页"
// @Success 200 {object} gincontext.DtoRender{data=dtopermission.ScopePageListResp}
// @Router /v1/platform/scopes [get]
func (ctr *scopeCtr) PageList(ctx *gin.Context) {
	var req dtopermission.ScopePageListReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.scopeSvc.PageList(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}
