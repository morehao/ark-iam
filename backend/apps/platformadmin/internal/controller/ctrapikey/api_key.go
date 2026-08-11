package ctrapikey

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtoapikey"
	"github.com/morehao/ark-iam/platformadmin/internal/service/svcapikey"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type ApiKeyCtr interface {
	Create(ctx *gin.Context)
	PageList(ctx *gin.Context)
	Revoke(ctx *gin.Context)
	Delete(ctx *gin.Context)
}

type apiKeyCtr struct {
	apiKeySvc svcapikey.CreateApiKeySvc
}

var _ ApiKeyCtr = (*apiKeyCtr)(nil)

func NewApiKeyCtr() ApiKeyCtr {
	return &apiKeyCtr{
		apiKeySvc: svcapikey.NewCreateApiKeySvc(),
	}
}

// @Tags API密钥管理
// @Summary 创建API密钥
// @accept application/json
// @Produce application/json
// @Param req body dtoapikey.CreateApiKeyReq true "创建API密钥"
// @Success 200 {object} gincontext.DtoRender{data=dtoapikey.CreateApiKeyResp}
// @Router /v1/iam/apiKey/create [post]
func (ctr *apiKeyCtr) Create(ctx *gin.Context) {
	var req dtoapikey.CreateApiKeyReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	tenantID := gincontext.GetTenantID(ctx)
	res, err := ctr.apiKeySvc.Create(ctx, tenantID, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags API密钥管理
// @Summary API密钥列表分页
// @accept application/json
// @Produce application/json
// @Param req body dtoapikey.ApiKeyPageListReq true "API密钥列表分页"
// @Success 200 {object} gincontext.DtoRender{data=dtoapikey.ApiKeyPageListResp}
// @Router /v1/iam/apiKey/pageList [post]
func (ctr *apiKeyCtr) PageList(ctx *gin.Context) {
	var req dtoapikey.ApiKeyPageListReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	tenantID := gincontext.GetTenantID(ctx)
	res, err := ctr.apiKeySvc.PageList(ctx, tenantID, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags API密钥管理
// @Summary 撤销API密钥
// @accept application/json
// @Produce application/json
// @Param req body dtoapikey.RevokeApiKeyReq true "撤销API密钥"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/apiKey/revoke [post]
func (ctr *apiKeyCtr) Revoke(ctx *gin.Context) {
	var req dtoapikey.RevokeApiKeyReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	tenantID := gincontext.GetTenantID(ctx)
	if err := ctr.apiKeySvc.Revoke(ctx, tenantID, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "撤销成功")
}

// @Tags API密钥管理
// @Summary 删除API密钥
// @accept application/json
// @Produce application/json
// @Param req body dtoapikey.DeleteApiKeyReq true "删除API密钥"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/apiKey/delete [post]
func (ctr *apiKeyCtr) Delete(ctx *gin.Context) {
	var req dtoapikey.DeleteApiKeyReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	tenantID := gincontext.GetTenantID(ctx)
	if err := ctr.apiKeySvc.Delete(ctx, tenantID, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}
