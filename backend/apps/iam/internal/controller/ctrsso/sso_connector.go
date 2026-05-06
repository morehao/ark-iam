package ctrsso

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/dto/dtoauth"
	"github.com/morehao/ark-iam/iam/internal/service/svcauth"
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
	ssoConnectorSvc svcauth.SsoConnectorSvc
}

var _ SsoConnectorCtr = (*ssoConnectorCtr)(nil)

func NewSsoConnectorCtr() SsoConnectorCtr {
	return &ssoConnectorCtr{
		ssoConnectorSvc: svcauth.NewSsoConnectorSvc(),
	}
}

// Create SSO连接器创建
// @Tags SSO连接器管理
// @Summary 创建SSO连接器管理
// @accept application/json
// @Produce application/json
// @Param req body dtoauth.SsoConnectorCreateReq true "创建SSO连接器管理"
// @Success 200 {object} gincontext.DtoRender{data=dtoauth.SsoConnectorCreateResp} "{"code": 0, "requestID": "xxx", "data": "ok", "msg": "success"}"
// @Router /v1/iam/auth/create [post]
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

// Delete SSO连接器删除
// @Tags SSO连接器管理
// @Summary 删除SSO连接器管理
// @accept application/json
// @Produce application/json
// @Param req body dtoauth.SsoConnectorDeleteReq true "删除SSO连接器管理"
// @Success 200 {object} gincontext.DtoRender{data=string} "{"code": 0, "requestID": "xxx", "data": "ok", "msg": "success"}"
// @Router /v1/iam/auth/delete [post]
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

// Update SSO连接器修改
// @Tags SSO连接器管理
// @Summary 修改SSO连接器管理
// @accept application/json
// @Produce application/json
// @Param req body dtoauth.SsoConnectorUpdateReq true "修改SSO连接器管理"
// @Success 200 {object} gincontext.DtoRender{data=string} "{"code": 0, "requestID": "xxx", "data": "ok", "msg": "修改成功"}"
// @Router /v1/iam/auth/update [post]
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

// Detail SSO连接器详情
// @Tags SSO连接器管理
// @Summary SSO连接器管理详情
// @accept application/json
// @Produce application/json
// @Param req query dtoauth.SsoConnectorDetailReq true "SSO连接器管理详情"
// @Success 200 {object} gincontext.DtoRender{data=dtoauth.SsoConnectorDetailResp} "{"code": 0, "requestID": "xxx", "data": "ok", "msg": "success"}"
// @Router /v1/iam/auth/detail [get]
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

// PageList SSO连接器列表
// @Tags SSO连接器管理
// @Summary SSO连接器管理列表分页
// @accept application/json
// @Produce application/json
// @Param req body dtoauth.SsoConnectorPageListReq true "SSO连接器管理列表"
// @Success 200 {object} gincontext.DtoRender{data=dtoauth.SsoConnectorPageListResp} "{"code": 0, "requestID": "xxx", "data": "ok", "msg": "success"}"
// @Router /v1/iam/auth/pageList [post]
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