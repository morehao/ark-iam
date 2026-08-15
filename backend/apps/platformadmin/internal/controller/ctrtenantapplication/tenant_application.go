package ctrtenantapplication

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtotenantapplication"
	"github.com/morehao/ark-iam/platformadmin/internal/service/svctenantapplication"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type TenantApplicationCtr interface {
	Create(ctx *gin.Context)
	Delete(ctx *gin.Context)
	Update(ctx *gin.Context)
	Detail(ctx *gin.Context)
	PageList(ctx *gin.Context)
}

type tenantApplicationCtr struct {
	svc svctenantapplication.TenantApplicationSvc
}

var _ TenantApplicationCtr = (*tenantApplicationCtr)(nil)

func NewTenantApplicationCtr() TenantApplicationCtr {
	return &tenantApplicationCtr{
		svc: svctenantapplication.NewTenantApplicationSvc(),
	}
}

// @Tags 租户应用订阅
// @Summary 创建租户应用订阅
// @accept application/json
// @Produce application/json
// @Param req body dtotenantapplication.TenantApplicationCreateReq true "创建租户应用订阅"
// @Success 200 {object} gincontext.DtoRender{data=dtotenantapplication.TenantApplicationCreateResp}
// @Router /v1/platform/tenantApplication/create [post]
func (ctr *tenantApplicationCtr) Create(ctx *gin.Context) {
	var req dtotenantapplication.TenantApplicationCreateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.svc.Create(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 租户应用订阅
// @Summary 删除租户应用订阅
// @accept application/json
// @Produce application/json
// @Param req body dtotenantapplication.TenantApplicationDeleteReq true "删除租户应用订阅"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/platform/tenantApplication/delete [post]
func (ctr *tenantApplicationCtr) Delete(ctx *gin.Context) {
	var req dtotenantapplication.TenantApplicationDeleteReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.svc.Delete(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}

// @Tags 租户应用订阅
// @Summary 修改租户应用订阅
// @accept application/json
// @Produce application/json
// @Param req body dtotenantapplication.TenantApplicationUpdateReq true "修改租户应用订阅"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/platform/tenantApplication/update [post]
func (ctr *tenantApplicationCtr) Update(ctx *gin.Context) {
	var req dtotenantapplication.TenantApplicationUpdateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.svc.Update(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "修改成功")
}

// @Tags 租户应用订阅
// @Summary 查看租户应用订阅详情
// @accept application/json
// @Produce application/json
// @Param req query dtotenantapplication.TenantApplicationDetailReq true "查看租户应用订阅详情"
// @Success 200 {object} gincontext.DtoRender{data=dtotenantapplication.TenantApplicationDetailResp}
// @Router /v1/platform/tenantApplication/detail [get]
func (ctr *tenantApplicationCtr) Detail(ctx *gin.Context) {
	var req dtotenantapplication.TenantApplicationDetailReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.svc.Detail(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 租户应用订阅
// @Summary 查看租户应用订阅列表
// @accept application/json
// @Produce application/json
// @Param req body dtotenantapplication.TenantApplicationPageListReq true "查看租户应用订阅列表"
// @Success 200 {object} gincontext.DtoRender{data=dtotenantapplication.TenantApplicationPageListResp}
// @Router /v1/platform/tenantApplication/pageList [post]
func (ctr *tenantApplicationCtr) PageList(ctx *gin.Context) {
	var req dtotenantapplication.TenantApplicationPageListReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.svc.PageList(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}
