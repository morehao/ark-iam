package ctrtenant

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/tenantadmin/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/tenantadmin/internal/service/svctenant"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type ApiKeyCtr interface {
	Create(ctx *gin.Context)
	PageList(ctx *gin.Context)
	Revoke(ctx *gin.Context)
	Delete(ctx *gin.Context)
}

type apiKeyCtr struct {
	apiKeySvc svctenant.ApiKeySvc
}

var _ ApiKeyCtr = (*apiKeyCtr)(nil)

func NewApiKeyCtr() ApiKeyCtr {
	return &apiKeyCtr{
		apiKeySvc: svctenant.NewApiKeySvc(),
	}
}

// @Tags API密钥
// @Summary 创建API密钥(归属本人或服务账号)
// @accept application/json
// @Produce application/json
// @Param req body dtotenant.ApiKeyCreateReq true "创建API密钥"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.ApiKeyCreateResp}
// @Router /v1/tenant/api-keys [post]
func (ctr *apiKeyCtr) Create(ctx *gin.Context) {
	var req dtotenant.ApiKeyCreateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.apiKeySvc.Create(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags API密钥
// @Summary API密钥列表分页(本人/指定服务账号/全租户)
// @accept application/json
// @Produce application/json
// @Param req query dtotenant.ApiKeyPageListReq true "API密钥列表分页"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.ApiKeyPageListResp}
// @Router /v1/tenant/api-keys [get]
func (ctr *apiKeyCtr) PageList(ctx *gin.Context) {
	var req dtotenant.ApiKeyPageListReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.apiKeySvc.PageList(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags API密钥
// @Summary 吊销API密钥
// @accept application/json
// @Produce application/json
// @Param apiKeyID path string true "API密钥ID"
// @Success 200 {object} gincontext.DtoRender
// @Router /v1/tenant/api-keys/{apiKeyID}/revoke [post]
func (ctr *apiKeyCtr) Revoke(ctx *gin.Context) {
	var req dtotenant.ApiKeyRevokeReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.apiKeySvc.Revoke(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, nil)
}

// @Tags API密钥
// @Summary 删除API密钥
// @accept application/json
// @Produce application/json
// @Param apiKeyID path string true "API密钥ID"
// @Success 200 {object} gincontext.DtoRender
// @Router /v1/tenant/api-keys/{apiKeyID} [delete]
func (ctr *apiKeyCtr) Delete(ctx *gin.Context) {
	var req dtotenant.ApiKeyDeleteReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.apiKeySvc.Delete(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, nil)
}
