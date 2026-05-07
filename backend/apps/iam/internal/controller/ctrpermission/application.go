package ctrpermission

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/dto/dtoapplication"
	"github.com/morehao/ark-iam/iam/internal/service/svcapplication"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type ApplicationCtr interface {
	Create(ctx *gin.Context)
	Delete(ctx *gin.Context)
	Update(ctx *gin.Context)
	Detail(ctx *gin.Context)
	PageList(ctx *gin.Context)
}

type applicationCtr struct {
	applicationSvc svcapplication.ApplicationSvc
}

var _ ApplicationCtr = (*applicationCtr)(nil)

func NewApplicationCtr() ApplicationCtr {
	return &applicationCtr{
		applicationSvc: svcapplication.NewApplicationSvc(),
	}
}

// @Tags 应用管理
// @Summary 创建应用管理
// @accept application/json
// @Produce application/json
// @Param req body dtoapplication.ApplicationCreateReq true "创建应用管理"
// @Success 200 {object} gincontext.DtoRender{data=dtoapplication.ApplicationCreateResp}
// @Router /v1/iam/application/create [post]
func (ctr *applicationCtr) Create(ctx *gin.Context) {
	var req dtoapplication.ApplicationCreateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.applicationSvc.Create(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 应用管理
// @Summary 删除应用管理
// @accept application/json
// @Produce application/json
// @Param req body dtoapplication.ApplicationDeleteReq true "删除应用管理"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/application/delete [post]
func (ctr *applicationCtr) Delete(ctx *gin.Context) {
	var req dtoapplication.ApplicationDeleteReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.applicationSvc.Delete(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}

// @Tags 应用管理
// @Summary 修改应用管理
// @accept application/json
// @Produce application/json
// @Param req body dtoapplication.ApplicationUpdateReq true "修改应用管理"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/application/update [post]
func (ctr *applicationCtr) Update(ctx *gin.Context) {
	var req dtoapplication.ApplicationUpdateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.applicationSvc.Update(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "修改成功")
}

// @Tags 应用管理
// @Summary 应用管理详情
// @accept application/json
// @Produce application/json
// @Param req query dtoapplication.ApplicationDetailReq true "应用管理详情"
// @Success 200 {object} gincontext.DtoRender{data=dtoapplication.ApplicationDetailResp}
// @Router /v1/iam/application/detail [get]
func (ctr *applicationCtr) Detail(ctx *gin.Context) {
	var req dtoapplication.ApplicationDetailReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.applicationSvc.Detail(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 应用管理
// @Summary 应用管理列表分页
// @accept application/json
// @Produce application/json
// @Param req body dtoapplication.ApplicationPageListReq true "应用管理列表分页"
// @Success 200 {object} gincontext.DtoRender{data=dtoapplication.ApplicationPageListResp}
// @Router /v1/iam/application/pageList [post]
func (ctr *applicationCtr) PageList(ctx *gin.Context) {
	var req dtoapplication.ApplicationPageListReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.applicationSvc.PageList(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}