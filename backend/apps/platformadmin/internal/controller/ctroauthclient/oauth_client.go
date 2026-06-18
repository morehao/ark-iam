package ctroauthclient

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtooauthclient"
	"github.com/morehao/ark-iam/platformadmin/internal/service/svcoauthclient"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type OAuthClientCtr interface {
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
	svc svcoauthclient.OAuthClientSvc
}

var _ OAuthClientCtr = (*oAuthClientCtr)(nil)

func NewOAuthClientCtr() OAuthClientCtr {
	return &oAuthClientCtr{
		svc: svcoauthclient.NewOAuthClientSvc(),
	}
}

// @Tags OAuth客户端
// @Summary 创建OAuth客户端
// @accept application/json
// @Produce application/json
// @Param req body dtooauthclient.CreateReq true "创建OAuth客户端"
// @Success 200 {object} gincontext.DtoRender{data=dtooauthclient.CreateResp}
// @Router /v1/iam/oauthClient/create [post]
func (ctr *oAuthClientCtr) Create(ctx *gin.Context) {
	var req dtooauthclient.CreateReq
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
// @Param req body dtooauthclient.DeleteReq true "删除OAuth客户端"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/oauthClient/delete [post]
func (ctr *oAuthClientCtr) Delete(ctx *gin.Context) {
	var req dtooauthclient.DeleteReq
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
// @Param req body dtooauthclient.UpdateReq true "修改OAuth客户端"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/oauthClient/update [post]
func (ctr *oAuthClientCtr) Update(ctx *gin.Context) {
	var req dtooauthclient.UpdateReq
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
// @Param req query dtooauthclient.DetailReq true "查看OAuth客户端详情"
// @Success 200 {object} gincontext.DtoRender{data=dtooauthclient.DetailResp}
// @Router /v1/iam/oauthClient/detail [get]
func (ctr *oAuthClientCtr) Detail(ctx *gin.Context) {
	var req dtooauthclient.DetailReq
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
// @Param req body dtooauthclient.PageListReq true "查看OAuth客户端列表"
// @Success 200 {object} gincontext.DtoRender{data=dtooauthclient.PageListResp}
// @Router /v1/iam/oauthClient/pageList [post]
func (ctr *oAuthClientCtr) PageList(ctx *gin.Context) {
	var req dtooauthclient.PageListReq
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
// @Param req query dtooauthclient.SecretListReq true "查看OAuth客户端密钥列表"
// @Success 200 {object} gincontext.DtoRender{data=dtooauthclient.SecretListResp}
// @Router /v1/iam/oauthClient/secrets [get]
func (ctr *oAuthClientCtr) ListSecrets(ctx *gin.Context) {
	var req dtooauthclient.SecretListReq
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
// @Param req body dtooauthclient.CreateSecretReq true "创建OAuth客户端密钥"
// @Success 200 {object} gincontext.DtoRender{data=dtooauthclient.CreateSecretResp}
// @Router /v1/iam/oauthClient/secrets [post]
func (ctr *oAuthClientCtr) CreateSecret(ctx *gin.Context) {
	var req dtooauthclient.CreateSecretReq
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
// @Router /v1/iam/oauthClient/secrets/{secretId} [delete]
func (ctr *oAuthClientCtr) DeleteSecret(ctx *gin.Context) {
	var req dtooauthclient.DeleteSecretReq
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
