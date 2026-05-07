package ctrauth

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/dto/dtoauth"
	"github.com/morehao/ark-iam/iam/internal/dto/dtosso"
	"github.com/morehao/ark-iam/iam/internal/service/svcauth"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type SsoConnectorCtr interface {
	Create(ctx *gin.Context)
	Delete(ctx *gin.Context)
	Update(ctx *gin.Context)
	Detail(ctx *gin.Context)
	PageList(ctx *gin.Context)
	ListProviders(ctx *gin.Context)
	GetIdpConfig(ctx *gin.Context)
	UpdateIdpConfig(ctx *gin.Context)
}

type ssoConnectorCtr struct {
	ssoConnectorSvc svcauth.SsoConnectorSvc
}

var _ SsoConnectorCtr = (*ssoConnectorCtr)(nil)

func NewSsoConnectorCtr() SsoConnectorCtr {
	return &ssoConnectorCtr{
		ssoConnectorSvc: svcauth.NewSsoConnectorSvc(),
	}
}

// @Tags SSO连接器
// @Summary 创建SSO连接器
// @accept application/json
// @Produce application/json
// @Param req body dtoauth.SsoConnectorCreateReq true "创建SSO连接器"
// @Success 200 {object} gincontext.DtoRender{data=dtoauth.SsoConnectorCreateResp}
// @Router /v1/iam/ssoConnector/create [post]
func (ctr *ssoConnectorCtr) Create(ctx *gin.Context) {
	var req dtoauth.SsoConnectorCreateReq
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

// @Tags SSO连接器
// @Summary 删除SSO连接器
// @accept application/json
// @Produce application/json
// @Param req body dtoauth.SsoConnectorDeleteReq true "删除SSO连接器"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/ssoConnector/delete [post]
func (ctr *ssoConnectorCtr) Delete(ctx *gin.Context) {
	var req dtoauth.SsoConnectorDeleteReq
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

// @Tags SSO连接器
// @Summary 修改SSO连接器
// @accept application/json
// @Produce application/json
// @Param req body dtoauth.SsoConnectorUpdateReq true "修改SSO连接器"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/ssoConnector/update [post]
func (ctr *ssoConnectorCtr) Update(ctx *gin.Context) {
	var req dtoauth.SsoConnectorUpdateReq
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

// @Tags SSO连接器
// @Summary SSO连接器详情
// @accept application/json
// @Produce application/json
// @Param req query dtoauth.SsoConnectorDetailReq true "SSO连接器详情"
// @Success 200 {object} gincontext.DtoRender{data=dtoauth.SsoConnectorDetailResp}
// @Router /v1/iam/ssoConnector/detail [get]
func (ctr *ssoConnectorCtr) Detail(ctx *gin.Context) {
	var req dtoauth.SsoConnectorDetailReq
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

// @Tags SSO连接器
// @Summary SSO连接器列表分页
// @accept application/json
// @Produce application/json
// @Param req body dtoauth.SsoConnectorPageListReq true "SSO连接器列表分页"
// @Success 200 {object} gincontext.DtoRender{data=dtoauth.SsoConnectorPageListResp}
// @Router /v1/iam/ssoConnector/pageList [post]
func (ctr *ssoConnectorCtr) PageList(ctx *gin.Context) {
	var req dtoauth.SsoConnectorPageListReq
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

// @Tags SSO连接器
// @Summary SSO提供商列表
// @accept application/json
// @Produce application/json
// @Success 200 {object} gincontext.DtoRender{data=dtosso.SsoProviderListResp}
// @Router /v1/iam/ssoConnector/providers [get]
func (ctr *ssoConnectorCtr) ListProviders(ctx *gin.Context) {
	res, err := ctr.ssoConnectorSvc.ListProviders(ctx)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags SSO连接器
// @Summary 获取IdP配置
// @accept application/json
// @Produce application/json
// @Param connectorId path int true "连接器ID"
// @Success 200 {object} gincontext.DtoRender{data=dtosso.SsoIdpConfigResp}
// @Router /v1/iam/ssoConnector/{connectorId}/idp-config [get]
func (ctr *ssoConnectorCtr) GetIdpConfig(ctx *gin.Context) {
	var req dtosso.SsoConnectorIDReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.ssoConnectorSvc.GetIdpConfig(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags SSO连接器
// @Summary 更新IdP配置
// @accept application/json
// @Produce application/json
// @Param connectorId path int true "连接器ID"
// @Param req body dtosso.SsoIdpConfigReq true "IdP配置"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/ssoConnector/{connectorId}/idp-config [put]
func (ctr *ssoConnectorCtr) UpdateIdpConfig(ctx *gin.Context) {
	var req dtosso.SsoConnectorIDReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	var configReq dtosso.SsoIdpConfigReq
	if err := ctx.ShouldBindJSON(&configReq); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.ssoConnectorSvc.UpdateIdpConfig(ctx, &req, &configReq); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "更新成功")
}