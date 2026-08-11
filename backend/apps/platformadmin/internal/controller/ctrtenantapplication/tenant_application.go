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
// @Param req body dtotenantapplication.CreateReq true "创建租户应用订阅"
// @Success 200 {object} gincontext.DtoRender{data=dtotenantapplication.CreateResp}
// @Router /v1/iam/tenantApplication/create [post]
func (ctr *tenantApplicationCtr) Create(ctx *gin.Context) {
	var req dtotenantapplication.CreateReq
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
// @Param req body dtotenantapplication.DeleteReq true "删除租户应用订阅"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/tenantApplication/delete [post]
func (ctr *tenantApplicationCtr) Delete(ctx *gin.Context) {
	var req dtotenantapplication.DeleteReq
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
// @Param req body dtotenantapplication.UpdateReq true "修改租户应用订阅"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/tenantApplication/update [post]
func (ctr *tenantApplicationCtr) Update(ctx *gin.Context) {
	var req dtotenantapplication.UpdateReq
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
// @Param req query dtotenantapplication.DetailReq true "查看租户应用订阅详情"
// @Success 200 {object} gincontext.DtoRender{data=dtotenantapplication.DetailResp}
// @Router /v1/iam/tenantApplication/detail [get]
func (ctr *tenantApplicationCtr) Detail(ctx *gin.Context) {
	var req dtotenantapplication.DetailReq
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
// @Param req body dtotenantapplication.PageListReq true "查看租户应用订阅列表"
// @Success 200 {object} gincontext.DtoRender{data=dtotenantapplication.PageListResp}
// @Router /v1/iam/tenantApplication/pageList [post]
func (ctr *tenantApplicationCtr) PageList(ctx *gin.Context) {
	var req dtotenantapplication.PageListReq
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
