package ctroauthclient

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/dto/dtooauthclient"
	"github.com/morehao/ark-iam/iam/internal/service/svcoauthclient"
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
