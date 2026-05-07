package ctrauth

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/dto/dtoauth"
	"github.com/morehao/ark-iam/iam/internal/service/svcauth"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type ConnectorCtr interface {
	Create(ctx *gin.Context)
	Delete(ctx *gin.Context)
	Update(ctx *gin.Context)
	Detail(ctx *gin.Context)
	PageList(ctx *gin.Context)
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