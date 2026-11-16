package ctrsetting

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/apps/platformadmin/dto/dtosetting"
	"github.com/morehao/ark-iam/apps/platformadmin/service/svcsetting"
	"github.com/morehao/ark-iam/pkg/identity"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/dbaccess/gormdao"
)

// SettingCtr 是运行时配置中心（系统设置）管理面控制器。
type SettingCtr interface {
	Create(ctx *gin.Context)
	Update(ctx *gin.Context)
	UpdateStatus(ctx *gin.Context)
	Delete(ctx *gin.Context)
	Detail(ctx *gin.Context)
	PageList(ctx *gin.Context)
}

type settingCtr struct {
	settingSvc svcsetting.SettingSvc
}

var _ SettingCtr = (*settingCtr)(nil)

// NewSettingCtr 构造控制器。db 参数保留以对齐 DomainFunc 签名（gsetting 走进程级实例）。
func NewSettingCtr(_ gormdao.DBGetter) SettingCtr {
	return &settingCtr{settingSvc: svcsetting.NewSettingSvc()}
}

// @Tags 系统设置
// @Summary 创建配置项
// @accept application/json
// @Produce application/json
// @Param req body dtosetting.SettingCreateReq true "创建配置项"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/platform/settings [post]
func (ctr *settingCtr) Create(ctx *gin.Context) {
	var req dtosetting.SettingCreateReq
	req.Operator = identity.OperatorFromGin(ctx)
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.settingSvc.Create(ctx.Request.Context(), &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "创建成功")
}

// @Tags 系统设置
// @Summary 配置项列表分页
// @accept application/json
// @Produce application/json
// @Param req query dtosetting.SettingPageListReq true "配置项列表分页"
// @Success 200 {object} gincontext.DtoRender{data=dtosetting.SettingPageListResp}
// @Router /v1/platform/settings [get]
func (ctr *settingCtr) PageList(ctx *gin.Context) {
	var req dtosetting.SettingPageListReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.settingSvc.PageList(ctx.Request.Context(), &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 系统设置
// @Summary 配置项详情
// @accept application/json
// @Produce application/json
// @Param settingID path string true "settingID"
// @Success 200 {object} gincontext.DtoRender{data=dtosetting.SettingDetailResp}
// @Router /v1/platform/settings/{settingID} [get]
func (ctr *settingCtr) Detail(ctx *gin.Context) {
	var req dtosetting.SettingDetailReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.settingSvc.Detail(ctx.Request.Context(), &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 系统设置
// @Summary 修改配置项
// @accept application/json
// @Produce application/json
// @Param settingID path string true "settingID"
// @Param req body dtosetting.SettingUpdateReq true "修改配置项"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/platform/settings/{settingID} [put]
func (ctr *settingCtr) Update(ctx *gin.Context) {
	var req dtosetting.SettingUpdateReq
	req.Operator = identity.OperatorFromGin(ctx)
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.settingSvc.Update(ctx.Request.Context(), &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "修改成功")
}

// @Tags 系统设置
// @Summary 启停配置项
// @accept application/json
// @Produce application/json
// @Param settingID path string true "settingID"
// @Param req body dtosetting.SettingUpdateStatusReq true "状态"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/platform/settings/{settingID}/status [patch]
func (ctr *settingCtr) UpdateStatus(ctx *gin.Context) {
	var req dtosetting.SettingUpdateStatusReq
	req.Operator = identity.OperatorFromGin(ctx)
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.settingSvc.UpdateStatus(ctx.Request.Context(), &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "修改成功")
}

// @Tags 系统设置
// @Summary 删除配置项
// @accept application/json
// @Produce application/json
// @Param settingID path string true "settingID"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/platform/settings/{settingID} [delete]
func (ctr *settingCtr) Delete(ctx *gin.Context) {
	var req dtosetting.SettingDeleteReq
	req.Operator = identity.OperatorFromGin(ctx)
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.settingSvc.Delete(ctx.Request.Context(), &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}
