package ctrapplicationclient

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtoapplicationclient"
	"github.com/morehao/ark-iam/platformadmin/internal/service/svcapplicationclient"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type ApplicationClientCtr interface {
	Create(ctx *gin.Context)
	Delete(ctx *gin.Context)
	Update(ctx *gin.Context)
	Detail(ctx *gin.Context)
	PageList(ctx *gin.Context)
	ListSecrets(ctx *gin.Context)
	CreateSecret(ctx *gin.Context)
	DeleteSecret(ctx *gin.Context)
}

type oAuthClientCtr struct {
	svc svcapplicationclient.ApplicationClientSvc
}

var _ ApplicationClientCtr = (*oAuthClientCtr)(nil)

func NewApplicationClientCtr() ApplicationClientCtr {
	return &oAuthClientCtr{
		svc: svcapplicationclient.NewApplicationClientSvc(),
	}
}

// @Tags OAuth客户端
// @Summary 创建OAuth客户端
// @accept application/json
// @Produce application/json
// @Param req body dtoapplicationclient.ApplicationClientCreateReq true "创建OAuth客户端"
// @Success 200 {object} gincontext.DtoRender{data=dtoapplicationclient.ApplicationClientCreateResp}
// @Router /v1/platform/applicationClient/create [post]
func (ctr *oAuthClientCtr) Create(ctx *gin.Context) {
	var req dtoapplicationclient.ApplicationClientCreateReq
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

// @Tags OAuth客户端
// @Summary 删除OAuth客户端
// @accept application/json
// @Produce application/json
// @Param req body dtoapplicationclient.ApplicationClientDeleteReq true "删除OAuth客户端"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/platform/applicationClient/delete [post]
func (ctr *oAuthClientCtr) Delete(ctx *gin.Context) {
	var req dtoapplicationclient.ApplicationClientDeleteReq
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

// @Tags OAuth客户端
// @Summary 修改OAuth客户端
// @accept application/json
// @Produce application/json
// @Param req body dtoapplicationclient.ApplicationClientUpdateReq true "修改OAuth客户端"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/platform/applicationClient/update [post]
func (ctr *oAuthClientCtr) Update(ctx *gin.Context) {
	var req dtoapplicationclient.ApplicationClientUpdateReq
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

// @Tags OAuth客户端
// @Summary 查看OAuth客户端详情
// @accept application/json
// @Produce application/json
// @Param req query dtoapplicationclient.ApplicationClientDetailReq true "查看OAuth客户端详情"
// @Success 200 {object} gincontext.DtoRender{data=dtoapplicationclient.ApplicationClientDetailResp}
// @Router /v1/platform/applicationClient/detail [get]
func (ctr *oAuthClientCtr) Detail(ctx *gin.Context) {
	var req dtoapplicationclient.ApplicationClientDetailReq
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

// @Tags OAuth客户端
// @Summary 查看OAuth客户端列表
// @accept application/json
// @Produce application/json
// @Param req body dtoapplicationclient.ApplicationClientPageListReq true "查看OAuth客户端列表"
// @Success 200 {object} gincontext.DtoRender{data=dtoapplicationclient.ApplicationClientPageListResp}
// @Router /v1/platform/applicationClient/pageList [post]
func (ctr *oAuthClientCtr) PageList(ctx *gin.Context) {
	var req dtoapplicationclient.ApplicationClientPageListReq
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

// @Tags OAuth客户端密钥
// @Summary 查看OAuth客户端密钥列表
// @accept application/json
// @Produce application/json
// @Param req query dtoapplicationclient.SecretListReq true "查看OAuth客户端密钥列表"
// @Success 200 {object} gincontext.DtoRender{data=dtoapplicationclient.SecretListResp}
// @Router /v1/platform/applicationClient/secrets [get]
func (ctr *oAuthClientCtr) ListSecrets(ctx *gin.Context) {
	var req dtoapplicationclient.SecretListReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.svc.ListSecrets(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags OAuth客户端密钥
// @Summary 创建OAuth客户端密钥
// @accept application/json
// @Produce application/json
// @Param req body dtoapplicationclient.SecretCreateReq true "创建OAuth客户端密钥"
// @Success 200 {object} gincontext.DtoRender{data=dtoapplicationclient.SecretCreateResp}
// @Router /v1/platform/applicationClient/secrets [post]
func (ctr *oAuthClientCtr) CreateSecret(ctx *gin.Context) {
	var req dtoapplicationclient.SecretCreateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.svc.CreateSecret(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags OAuth客户端密钥
// @Summary 删除OAuth客户端密钥
// @accept application/json
// @Produce application/json
// @Param secretID path int true "密钥ID"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/platform/applicationClient/secrets/{secretID} [delete]
func (ctr *oAuthClientCtr) DeleteSecret(ctx *gin.Context) {
	var req dtoapplicationclient.SecretDeleteReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.svc.DeleteSecret(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}
