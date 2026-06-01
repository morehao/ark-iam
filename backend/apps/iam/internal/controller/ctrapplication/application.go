package ctrapplication

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/dto/dtoapplication"
	"github.com/morehao/ark-iam/iam/internal/service/svcapplication"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type ApplicationCtr interface {
	Create(ctx *gin.Context)
	Update(ctx *gin.Context)
	Delete(ctx *gin.Context)
	Detail(ctx *gin.Context)
	PageList(ctx *gin.Context)
}

type applicationCtr struct {
	svc svcapplication.ApplicationSvc
}

var _ ApplicationCtr = (*applicationCtr)(nil)

func NewApplicationCtr() ApplicationCtr {
	return &applicationCtr{svc: svcapplication.NewApplicationSvc()}
}

// @Tags 应用管理
// @Summary 创建应用
// @accept application/json
// @Produce application/json
// @Param req body dtoapplication.CreateReq true "创建应用"
// @Success 200 {object} gincontext.DtoRender{data=dtoapplication.CreateResp}
// @Router /v1/iam/application/create [post]
func (ctr *applicationCtr) Create(ctx *gin.Context) {
	var req dtoapplication.CreateReq
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

// @Tags 应用管理
// @Summary 修改应用
// @accept application/json
// @Produce application/json
// @Param req body dtoapplication.UpdateReq true "修改应用"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/application/update [post]
func (ctr *applicationCtr) Update(ctx *gin.Context) {
	var req dtoapplication.UpdateReq
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

// @Tags 应用管理
// @Summary 删除应用
// @accept application/json
// @Produce application/json
// @Param req body dtoapplication.DeleteReq true "删除应用"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/application/delete [post]
func (ctr *applicationCtr) Delete(ctx *gin.Context) {
	var req dtoapplication.DeleteReq
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

// @Tags 应用管理
// @Summary 查看应用详情
// @accept application/json
// @Produce application/json
// @Param req query dtoapplication.DetailReq true "查看应用详情"
// @Success 200 {object} gincontext.DtoRender{data=dtoapplication.DetailResp}
// @Router /v1/iam/application/detail [get]
func (ctr *applicationCtr) Detail(ctx *gin.Context) {
	var req dtoapplication.DetailReq
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

// @Tags 应用管理
// @Summary 查看应用列表
// @accept application/json
// @Produce application/json
// @Param req query dtoapplication.PageListReq true "查看应用列表"
// @Success 200 {object} gincontext.DtoRender{data=dtoapplication.PageListResp}
// @Router /v1/iam/application/pageList [get]
func (ctr *applicationCtr) PageList(ctx *gin.Context) {
	var req dtoapplication.PageListReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
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
