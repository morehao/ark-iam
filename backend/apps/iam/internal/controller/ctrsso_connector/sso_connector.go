package ctrsso_connector

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/dto/dtosso_connector"
	"github.com/morehao/ark-iam/iam/internal/service/svcsso_connector"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type SsoConnectorCtr interface {
	Create(ctx *gin.Context)
	Delete(ctx *gin.Context)
	Update(ctx *gin.Context)
	Detail(ctx *gin.Context)
	PageList(ctx *gin.Context)
}

type ssoConnectorCtr struct {
	ssoConnectorSvc svcsso_connector.SsoConnectorSvc
}

var _ SsoConnectorCtr = (*ssoConnectorCtr)(nil)

func NewSsoConnectorCtr() SsoConnectorCtr {
	return &ssoConnectorCtr{
		ssoConnectorSvc: svcsso_connector.NewSsoConnectorSvc(),
	}
}

func (ctr *ssoConnectorCtr) Create(ctx *gin.Context) {
	var req dtosso_connector.SsoConnectorCreateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.ssoConnectorSvc.Create(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *ssoConnectorCtr) Delete(ctx *gin.Context) {
	var req dtosso_connector.SsoConnectorDeleteReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.ssoConnectorSvc.Delete(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}

func (ctr *ssoConnectorCtr) Update(ctx *gin.Context) {
	var req dtosso_connector.SsoConnectorUpdateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.ssoConnectorSvc.Update(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "修改成功")
}

func (ctr *ssoConnectorCtr) Detail(ctx *gin.Context) {
	var req dtosso_connector.SsoConnectorDetailReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.ssoConnectorSvc.Detail(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *ssoConnectorCtr) PageList(ctx *gin.Context) {
	var req dtosso_connector.SsoConnectorPageListReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.ssoConnectorSvc.PageList(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}