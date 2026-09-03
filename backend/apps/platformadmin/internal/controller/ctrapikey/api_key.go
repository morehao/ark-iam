package ctrapikey

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtoapikey"
	"github.com/morehao/ark-iam/platformadmin/internal/service/svcapikey"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

// ApiKeyCtr 平台侧 API 密钥控制器：仅保留跨租户只读监督视图。
// 平台不再提供任何密钥写动作（创建/吊销/删除），密钥管理归属租户自服务控制台。
type ApiKeyCtr interface {
	PageListSupervision(ctx *gin.Context)
}

type apiKeyCtr struct {
	apiKeySvc svcapikey.ApiKeySupervisionSvc
}

var _ ApiKeyCtr = (*apiKeyCtr)(nil)

func NewApiKeyCtr() ApiKeyCtr {
	return &apiKeyCtr{
		apiKeySvc: svcapikey.NewApiKeySupervisionSvc(),
	}
}

// @Tags API密钥监督
// @Summary 全租户 API 密钥只读监督列表（平台排查视角，忽略上下文租户）
// @accept application/json
// @Produce application/json
// @Param req query dtoapikey.ApiKeySupervisionPageListReq true "API密钥监督列表"
// @Success 200 {object} gincontext.DtoRender{data=dtoapikey.ApiKeySupervisionPageListResp}
// @Router /v1/platform/api-keys/supervision [get]
func (ctr *apiKeyCtr) PageListSupervision(ctx *gin.Context) {
	var req dtoapikey.ApiKeySupervisionPageListReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.apiKeySvc.PageListSupervision(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}
