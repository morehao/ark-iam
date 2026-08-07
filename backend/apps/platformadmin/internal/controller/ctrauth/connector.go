package ctrauth

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtoauth"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtoconnector"
	"github.com/morehao/ark-iam/platformadmin/internal/service/svcauth"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type ConnectorCtr interface {
	Create(ctx *gin.Context)
	Delete(ctx *gin.Context)
	Update(ctx *gin.Context)
	Detail(ctx *gin.Context)
	PageList(ctx *gin.Context)
	GetFactoryList(ctx *gin.Context)
	TestConnector(ctx *gin.Context)
	Authorize(ctx *gin.Context)
	Callback(ctx *gin.Context)
}

type connectorCtr struct {
	connectorSvc svcauth.ConnectorSvc
}

var _ ConnectorCtr = (*connectorCtr)(nil)

func NewConnectorCtr() ConnectorCtr {
	return &connectorCtr{
		connectorSvc: svcauth.NewConnectorSvc(),
	}
}

// @Tags 连接器
// @Summary 创建连接器
// @accept application/json
// @Produce application/json
// @Param req body dtoauth.ConnectorCreateReq true "创建连接器"
// @Success 200 {object} gincontext.DtoRender{data=dtoauth.ConnectorCreateResp}
// @Router /v1/iam/connector/create [post]
func (ctr *connectorCtr) Create(ctx *gin.Context) {
	var req dtoauth.ConnectorCreateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.connectorSvc.Create(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 连接器
// @Summary 删除连接器
// @accept application/json
// @Produce application/json
// @Param req body dtoauth.ConnectorDeleteReq true "删除连接器"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/connector/delete [post]
func (ctr *connectorCtr) Delete(ctx *gin.Context) {
	var req dtoauth.ConnectorDeleteReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.connectorSvc.Delete(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}

// @Tags 连接器
// @Summary 修改连接器
// @accept application/json
// @Produce application/json
// @Param req body dtoauth.ConnectorUpdateReq true "修改连接器"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/connector/update [post]
func (ctr *connectorCtr) Update(ctx *gin.Context) {
	var req dtoauth.ConnectorUpdateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.connectorSvc.Update(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "修改成功")
}

// @Tags 连接器
// @Summary 连接器详情
// @accept application/json
// @Produce application/json
// @Param req query dtoauth.ConnectorDetailReq true "连接器详情"
// @Success 200 {object} gincontext.DtoRender{data=dtoauth.ConnectorDetailResp}
// @Router /v1/iam/connector/detail [get]
func (ctr *connectorCtr) Detail(ctx *gin.Context) {
	var req dtoauth.ConnectorDetailReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.connectorSvc.Detail(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 连接器
// @Summary 连接器列表分页
// @accept application/json
// @Produce application/json
// @Param req body dtoauth.ConnectorPageListReq true "连接器列表分页"
// @Success 200 {object} gincontext.DtoRender{data=dtoauth.ConnectorPageListResp}
// @Router /v1/iam/connector/pageList [post]
func (ctr *connectorCtr) PageList(ctx *gin.Context) {
	var req dtoauth.ConnectorPageListReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.connectorSvc.PageList(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 连接器
// @Summary 连接器工厂列表
// @accept application/json
// @Produce application/json
// @Param req body dtoconnector.ConnectorFactoryListReq false "连接器工厂列表"
// @Success 200 {object} gincontext.DtoRender{data=dtoconnector.ConnectorFactoryListResp}
// @Router /v1/iam/connector/getFactoryList [post]
func (ctr *connectorCtr) GetFactoryList(ctx *gin.Context) {
	var req dtoconnector.ConnectorFactoryListReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.connectorSvc.GetFactoryList(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 连接器
// @Summary 测试连接器
// @accept application/json
// @Produce application/json
// @Param connectorId path int true "连接器ID"
// @Success 200 {object} gincontext.DtoRender{data=dtoconnector.TestConnectorResp}
// @Router /v1/iam/connector/{connectorId}/test [post]
func (ctr *connectorCtr) TestConnector(ctx *gin.Context) {
	var req dtoconnector.ConnectorIDReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.connectorSvc.TestConnector(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 连接器
// @Summary 连接器授权
// @accept application/json
// @Produce application/json
// @Param connectorId path int true "连接器ID"
// @Param req body dtoconnector.ConnectorAuthorizeReq true "连接器授权"
// @Success 200 {object} gincontext.DtoRender{data=dtoconnector.ConnectorAuthorizeResp}
// @Router /v1/iam/connector/{connectorId}/authorize [post]
func (ctr *connectorCtr) Authorize(ctx *gin.Context) {
	var uriReq struct {
		ConnectorID uint `uri:"connectorId" binding:"required"`
	}
	if err := ctx.ShouldBindUri(&uriReq); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	var bodyReq struct {
		RedirectURI  string `json:"redirectUri" binding:"required"`
		State        string `json:"state"`
		LoginHint    string `json:"loginHint"`
		ResponseMode string `json:"responseMode"`
	}
	if err := ctx.ShouldBindJSON(&bodyReq); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	req := dtoconnector.ConnectorAuthorizeReq{
		ConnectorID:  uriReq.ConnectorID,
		RedirectURI:  bodyReq.RedirectURI,
		State:        bodyReq.State,
		LoginHint:    bodyReq.LoginHint,
		ResponseMode: bodyReq.ResponseMode,
	}
	res, err := ctr.connectorSvc.Authorize(ctx, &req, req.ConnectorID)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 连接器
// @Summary 连接器回调
// @accept application/json
// @Produce application/json
// @Param connectorId query int false "连接器ID"
// @Param code query string true "授权码"
// @Param state query string true "状态"
// @Success 200 {object} gincontext.DtoRender{data=dtoauth.LoginResp}
// @Router /v1/iam/connector/callback [get]
func (ctr *connectorCtr) Callback(ctx *gin.Context) {
	var req struct {
		ConnectorID uint   `form:"connectorId"`
		Code        string `form:"code" binding:"required"`
		State       string `form:"state" binding:"required"`
	}
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.connectorSvc.Callback(ctx, &dtoconnector.ConnectorCallbackReq{
		ConnectorID: req.ConnectorID,
		Code:        req.Code,
		State:       req.State,
	})
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}
