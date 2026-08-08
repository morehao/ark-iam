package ctrapplicationclient

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/dto/dtoapplicationclient"
	"github.com/morehao/ark-iam/iam/internal/service/svcapplicationclient"
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
// @Param req body dtoapplicationclient.CreateReq true "创建OAuth客户端"
// @Success 200 {object} gincontext.DtoRender{data=dtoapplicationclient.CreateResp}
// @Router /v1/iam/applicationClient/create [post]
func (ctr *oAuthClientCtr) Create(ctx *gin.Context) {
	var req dtoapplicationclient.CreateReq
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
// @Param req body dtoapplicationclient.DeleteReq true "删除OAuth客户端"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/applicationClient/delete [post]
func (ctr *oAuthClientCtr) Delete(ctx *gin.Context) {
	var req dtoapplicationclient.DeleteReq
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
// @Param req body dtoapplicationclient.UpdateReq true "修改OAuth客户端"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/applicationClient/update [post]
func (ctr *oAuthClientCtr) Update(ctx *gin.Context) {
	var req dtoapplicationclient.UpdateReq
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
// @Param req query dtoapplicationclient.DetailReq true "查看OAuth客户端详情"
// @Success 200 {object} gincontext.DtoRender{data=dtoapplicationclient.DetailResp}
// @Router /v1/iam/applicationClient/detail [get]
func (ctr *oAuthClientCtr) Detail(ctx *gin.Context) {
	var req dtoapplicationclient.DetailReq
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
// @Param req body dtoapplicationclient.PageListReq true "查看OAuth客户端列表"
// @Success 200 {object} gincontext.DtoRender{data=dtoapplicationclient.PageListResp}
// @Router /v1/iam/applicationClient/pageList [post]
func (ctr *oAuthClientCtr) PageList(ctx *gin.Context) {
	var req dtoapplicationclient.PageListReq
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
// @Router /v1/iam/applicationClient/secrets [get]
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
// @Param req body dtoapplicationclient.CreateSecretReq true "创建OAuth客户端密钥"
// @Success 200 {object} gincontext.DtoRender{data=dtoapplicationclient.CreateSecretResp}
// @Router /v1/iam/applicationClient/secrets [post]
func (ctr *oAuthClientCtr) CreateSecret(ctx *gin.Context) {
	var req dtoapplicationclient.CreateSecretReq
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
// @Param secretId path int true "密钥ID"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/applicationClient/secrets/{secretId} [delete]
func (ctr *oAuthClientCtr) DeleteSecret(ctx *gin.Context) {
	var req dtoapplicationclient.DeleteSecretReq
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
