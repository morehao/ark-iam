package ctrconnector

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/apps/platformadmin/dto/dtoconnector"
	"github.com/morehao/ark-iam/apps/platformadmin/service/svcconnector"
	"github.com/morehao/ark-iam/pkg/identity"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/dbaccess/gormdao"
)

// ConnectorCtr 连接器「管理面」控制器（平台级全局对象，跨租户运维）。
// 运行时（Authorize/Callback）在 auth/ctrauth，不在本控制器。
type ConnectorCtr interface {
	Create(ctx *gin.Context)
	Delete(ctx *gin.Context)
	Update(ctx *gin.Context)
	Detail(ctx *gin.Context)
	PageList(ctx *gin.Context)
	GetFactoryList(ctx *gin.Context)
	TestConnector(ctx *gin.Context)
}

type connectorCtr struct {
	svc svcconnector.ConnectorSvc
}

var _ ConnectorCtr = (*connectorCtr)(nil)

func NewConnectorCtr(db gormdao.DBGetter) ConnectorCtr {
	return &connectorCtr{
		svc: svcconnector.NewConnectorSvc(db),
	}
}

// @Tags 连接器
// @Summary 创建连接器
// @accept application/json
// @Produce application/json
// @Param req body dtoconnector.ConnectorCreateReq true "创建连接器"
// @Success 200 {object} gincontext.DtoRender{data=dtoconnector.ConnectorCreateResp}
// @Router /v1/platform/connectors [post]
func (ctr *connectorCtr) Create(ctx *gin.Context) {
	var req dtoconnector.ConnectorCreateReq
	req.Operator = identity.OperatorFromGin(ctx)
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.svc.Create(ctx.Request.Context(), &req)
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
// @Param connectorID path string true "connectorID"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/platform/connectors/{connectorID} [delete]
func (ctr *connectorCtr) Delete(ctx *gin.Context) {
	var req dtoconnector.ConnectorDeleteReq
	req.Operator = identity.OperatorFromGin(ctx)
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.svc.Delete(ctx.Request.Context(), &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}

// @Tags 连接器
// @Summary 修改连接器
// @accept application/json
// @Produce application/json
// @Param connectorID path string true "connectorID"
// @Param req body dtoconnector.ConnectorUpdateReq true "修改连接器"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/platform/connectors/{connectorID} [put]
func (ctr *connectorCtr) Update(ctx *gin.Context) {
	var req dtoconnector.ConnectorUpdateReq
	req.Operator = identity.OperatorFromGin(ctx)
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.svc.Update(ctx.Request.Context(), &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "修改成功")
}

// @Tags 连接器
// @Summary 查看连接器详情
// @accept application/json
// @Produce application/json
// @Param connectorID path string true "connectorID"
// @Success 200 {object} gincontext.DtoRender{data=dtoconnector.ConnectorDetailResp}
// @Router /v1/platform/connectors/{connectorID} [get]
func (ctr *connectorCtr) Detail(ctx *gin.Context) {
	var req dtoconnector.ConnectorDetailReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.svc.Detail(ctx.Request.Context(), &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 连接器
// @Summary 查看连接器列表
// @accept application/json
// @Produce application/json
// @Param req query dtoconnector.ConnectorPageListReq true "查看连接器列表"
// @Success 200 {object} gincontext.DtoRender{data=dtoconnector.ConnectorPageListResp}
// @Router /v1/platform/connectors [get]
func (ctr *connectorCtr) PageList(ctx *gin.Context) {
	var req dtoconnector.ConnectorPageListReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.svc.PageList(ctx.Request.Context(), &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 连接器
// @Summary 查看连接器工厂列表
// @accept application/json
// @Produce application/json
// @Param req query dtoconnector.ConnectorFactoryListReq true "查看连接器工厂列表"
// @Success 200 {object} gincontext.DtoRender{data=dtoconnector.ConnectorFactoryListResp}
// @Router /v1/platform/connector-factories [get]
func (ctr *connectorCtr) GetFactoryList(ctx *gin.Context) {
	var req dtoconnector.ConnectorFactoryListReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.svc.ListFactories(ctx.Request.Context(), &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 连接器
// @Summary 测试连接器连通性
// @accept application/json
// @Produce application/json
// @Param connectorID path string true "connectorID"
// @Success 200 {object} gincontext.DtoRender{data=dtoconnector.ConnectorTestResp}
// @Router /v1/platform/connectors/{connectorID}/test [post]
func (ctr *connectorCtr) TestConnector(ctx *gin.Context) {
	var req dtoconnector.ConnectorIDReq
	req.Operator = identity.OperatorFromGin(ctx)
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.svc.TestConnector(ctx.Request.Context(), &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}
