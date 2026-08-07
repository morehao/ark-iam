package ctrtenant

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/platformadmin/internal/service/svctenant"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type SystemCtr interface {
	Create(ctx *gin.Context)
	Delete(ctx *gin.Context)
	Update(ctx *gin.Context)
	Detail(ctx *gin.Context)
	PageList(ctx *gin.Context)
}

type systemCtr struct {
	systemSvc svctenant.SystemSvc
}

var _ SystemCtr = (*systemCtr)(nil)

func NewSystemCtr() SystemCtr {
	return &systemCtr{
		systemSvc: svctenant.NewSystemSvc(),
	}
}

// @Tags 系统配置
// @Summary 创建系统配置
// @accept application/json
// @Produce application/json
// @Param req body dtotenant.SystemCreateReq true "创建系统配置"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.SystemCreateResp}
// @Router /v1/iam/system/create [post]
func (ctr *systemCtr) Create(ctx *gin.Context) {
	var req dtotenant.SystemCreateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.systemSvc.Create(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 系统配置
// @Summary 删除系统配置
// @accept application/json
// @Produce application/json
// @Param req body dtotenant.SystemDeleteReq true "删除系统配置"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/system/delete [post]
func (ctr *systemCtr) Delete(ctx *gin.Context) {
	var req dtotenant.SystemDeleteReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.systemSvc.Delete(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}

// @Tags 系统配置
// @Summary 修改系统配置
// @accept application/json
// @Produce application/json
// @Param req body dtotenant.SystemUpdateReq true "修改系统配置"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/system/update [post]
func (ctr *systemCtr) Update(ctx *gin.Context) {
	var req dtotenant.SystemUpdateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.systemSvc.Update(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "修改成功")
}

// @Tags 系统配置
// @Summary 系统配置详情
// @accept application/json
// @Produce application/json
// @Param req query dtotenant.SystemDetailReq true "系统配置详情"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.SystemDetailResp}
// @Router /v1/iam/system/detail [get]
func (ctr *systemCtr) Detail(ctx *gin.Context) {
	var req dtotenant.SystemDetailReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.systemSvc.Detail(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 系统配置
// @Summary 系统配置列表分页
// @accept application/json
// @Produce application/json
// @Param req body dtotenant.SystemPageListReq true "系统配置列表分页"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.SystemPageListResp}
// @Router /v1/iam/system/pageList [post]
func (ctr *systemCtr) PageList(ctx *gin.Context) {
	var req dtotenant.SystemPageListReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.systemSvc.PageList(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}